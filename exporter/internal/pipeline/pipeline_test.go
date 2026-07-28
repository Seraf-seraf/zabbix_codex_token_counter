package pipeline

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/example/otlp-zabbix-exporter/internal/processor"
	"github.com/example/otlp-zabbix-exporter/internal/zabbix"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

type senderStub struct {
	metrics []zabbix.Metric
}

func (s *senderStub) Send(_ context.Context, metrics []zabbix.Metric) error {
	s.metrics = append(s.metrics, metrics...)
	return nil
}

func TestPipelineProcessesPdataAndAddsRussianMethodContext(t *testing.T) {
	metrics := pmetric.NewMetrics()
	scope := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
	scope.Scope().SetName("test-scope")
	input := scope.Metrics().AppendEmpty()
	input.SetName("requests")
	input.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(5)
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	sender := &senderStub{}
	registry := processor.NewRegistry(nil, processor.NewGeneric("host", "otel"))

	New(registry, sender, 10, logger).Process(context.Background(), metrics)

	if len(sender.metrics) != 1 || sender.metrics[0].Key != "otel.requests" || sender.metrics[0].Value != int64(5) {
		t.Fatalf("unexpected metrics: %+v", sender.metrics)
	}
	logText := logs.String()
	if !strings.Contains(logText, "pipeline.Pipeline.Process: метрика обработана") ||
		!strings.Contains(logText, "pipeline.Pipeline.Process: метрики отправлены в Zabbix") {
		t.Fatalf("unexpected logs: %s", logText)
	}
}
