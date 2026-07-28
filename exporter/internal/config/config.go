package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultOTLPListenAddr   = "0.0.0.0:4317"
	defaultZabbixServerAddr = "zabbix-server:10051"
	defaultZabbixHost       = "codex-wrapper"
	defaultZabbixKeyPrefix  = "otel"
	defaultSendTimeout      = 5 * time.Second
	defaultBatchSize        = 100
	defaultFlushInterval    = time.Second
	defaultLogLevel         = "info"
)

type Config struct {
	OTLPListenAddr          string
	ZabbixServerAddr        string
	ZabbixHost              string
	ZabbixKeyPrefix         string
	ZabbixSendTimeout       time.Duration
	ZabbixBatchSize         int
	ZabbixFlushInterval     time.Duration
	ProcessorGenericEnabled bool
	LogLevel                string
}

func Load() (Config, error) {
	timeout, err := durationEnv("ZABBIX_SEND_TIMEOUT", defaultSendTimeout)
	if err != nil {
		return Config{}, err
	}
	batchSize, err := intEnv("ZABBIX_BATCH_SIZE", defaultBatchSize)
	if err != nil {
		return Config{}, err
	}
	if batchSize <= 0 {
		return Config{}, fmt.Errorf("ZABBIX_BATCH_SIZE должен быть больше нуля")
	}
	flushInterval, err := durationEnv("ZABBIX_FLUSH_INTERVAL", defaultFlushInterval)
	if err != nil {
		return Config{}, err
	}
	generic, err := boolEnv("PROCESSOR_GENERIC_ENABLED", true)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		OTLPListenAddr:          stringEnv("OTLP_LISTEN_ADDR", defaultOTLPListenAddr),
		ZabbixServerAddr:        stringEnv("ZABBIX_SERVER_ADDR", defaultZabbixServerAddr),
		ZabbixHost:              stringEnv("ZABBIX_HOST", defaultZabbixHost),
		ZabbixKeyPrefix:         strings.Trim(stringEnv("ZABBIX_KEY_PREFIX", defaultZabbixKeyPrefix), "."),
		ZabbixSendTimeout:       timeout,
		ZabbixBatchSize:         batchSize,
		ZabbixFlushInterval:     flushInterval,
		ProcessorGenericEnabled: generic,
		LogLevel:                strings.ToLower(stringEnv("LOG_LEVEL", defaultLogLevel)),
	}

	for name, value := range map[string]string{
		"OTLP_LISTEN_ADDR":   cfg.OTLPListenAddr,
		"ZABBIX_SERVER_ADDR": cfg.ZabbixServerAddr,
		"ZABBIX_HOST":        cfg.ZabbixHost,
		"ZABBIX_KEY_PREFIX":  cfg.ZabbixKeyPrefix,
	} {
		if strings.TrimSpace(value) == "" {
			return Config{}, fmt.Errorf("%s не должен быть пустым", name)
		}
	}
	if timeout <= 0 || flushInterval <= 0 {
		return Config{}, fmt.Errorf("ZABBIX_SEND_TIMEOUT и ZABBIX_FLUSH_INTERVAL должны быть больше нуля")
	}
	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return Config{}, fmt.Errorf("LOG_LEVEL должен иметь одно из значений: debug, info, warn, error")
	}
	return cfg, nil
}

func stringEnv(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return strings.TrimSpace(value)
	}
	return fallback
}

func durationEnv(name string, fallback time.Duration) (time.Duration, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s: некорректная длительность %q: %w", name, value, err)
	}
	return parsed, nil
}

func intEnv(name string, fallback int) (int, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s: некорректное целое число %q: %w", name, value, err)
	}
	return parsed, nil
}

func boolEnv(name string, fallback bool) (bool, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s: некорректное логическое значение %q: %w", name, value, err)
	}
	return parsed, nil
}
