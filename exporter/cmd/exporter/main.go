package main

import (
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/otlp-zabbix-exporter/internal/config"
	"github.com/example/otlp-zabbix-exporter/internal/otlp"
	"github.com/example/otlp-zabbix-exporter/internal/pipeline"
	"github.com/example/otlp-zabbix-exporter/internal/processor"
	"github.com/example/otlp-zabbix-exporter/internal/zabbix"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"google.golang.org/grpc"
)

func main() {
	const methodCtx = "main.main"

	if err := run(); err != nil {
		slog.Error(methodCtx+": экспортёр остановлен", "component", "main", "error", err)
		os.Exit(1)
	}
}

func run() error {
	const methodCtx = "main.run"

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	listener, err := net.Listen("tcp", cfg.OTLPListenAddr)
	if err != nil {
		return errors.New("не удалось начать прослушивание адреса OTLP " + cfg.OTLPListenAddr + ": " + err.Error())
	}
	defer listener.Close()

	client := zabbix.NewClient(cfg.ZabbixServerAddr, cfg.ZabbixSendTimeout)
	specialized := []processor.Processor{processor.NewCodexTokenUsage(cfg.ZabbixHost, logger)}
	var generic processor.Processor
	if cfg.ProcessorGenericEnabled {
		generic = processor.NewGeneric(cfg.ZabbixHost, cfg.ZabbixKeyPrefix)
	}
	registry := processor.NewRegistry(specialized, generic)
	processingPipeline := pipeline.New(registry, client, cfg.ZabbixBatchSize, logger)
	receiver := otlp.NewReceiver(processingPipeline)

	server := grpc.NewServer()
	pmetricotlp.RegisterGRPCServer(server, receiver)
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Serve(listener)
	}()
	logger.Info(methodCtx+": экспортёр OTLP запущен", "component", "main", "listen_addr", cfg.OTLPListenAddr, "zabbix_server_addr", cfg.ZabbixServerAddr)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	select {
	case received := <-signals:
		logger.Info(methodCtx+": получен сигнал завершения работы", "component", "main", "signal", received.String())
	case serveErr := <-serverErrors:
		if !errors.Is(serveErr, grpc.ErrServerStopped) {
			return serveErr
		}
		return nil
	}

	stopped := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(10 * time.Second):
		logger.Warn(methodCtx+": превышено время ожидания корректного завершения, выполняется принудительная остановка gRPC", "component", "main")
		server.Stop()
	}
	logger.Info(methodCtx+": экспортёр остановлен", "component", "main")
	return nil
}

func newLogger(levelName string) *slog.Logger {
	level := slog.LevelInfo
	switch levelName {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
