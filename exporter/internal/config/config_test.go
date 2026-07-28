package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	for _, name := range []string{"OTLP_LISTEN_ADDR", "ZABBIX_SERVER_ADDR", "ZABBIX_HOST", "ZABBIX_KEY_PREFIX", "ZABBIX_SEND_TIMEOUT", "ZABBIX_BATCH_SIZE", "ZABBIX_FLUSH_INTERVAL", "PROCESSOR_GENERIC_ENABLED", "LOG_LEVEL"} {
		t.Setenv(name, "")
	}
	t.Setenv("OTLP_LISTEN_ADDR", defaultOTLPListenAddr)
	t.Setenv("ZABBIX_SERVER_ADDR", defaultZabbixServerAddr)
	t.Setenv("ZABBIX_HOST", defaultZabbixHost)
	t.Setenv("ZABBIX_KEY_PREFIX", defaultZabbixKeyPrefix)
	t.Setenv("ZABBIX_SEND_TIMEOUT", "5s")
	t.Setenv("ZABBIX_BATCH_SIZE", "100")
	t.Setenv("ZABBIX_FLUSH_INTERVAL", "1s")
	t.Setenv("PROCESSOR_GENERIC_ENABLED", "true")
	t.Setenv("LOG_LEVEL", "info")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OTLPListenAddr != defaultOTLPListenAddr || cfg.ZabbixSendTimeout != 5*time.Second || !cfg.ProcessorGenericEnabled {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadInvalidValues(t *testing.T) {
	tests := map[string]string{
		"ZABBIX_SEND_TIMEOUT":       "later",
		"ZABBIX_BATCH_SIZE":         "many",
		"PROCESSOR_GENERIC_ENABLED": "perhaps",
	}
	for name, value := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv(name, value)
			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("expected error naming %s, got %v", name, err)
			}
		})
	}
}

func TestLoadRejectsRequiredEmptyValue(t *testing.T) {
	t.Setenv("ZABBIX_HOST", " ")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "ZABBIX_HOST") {
		t.Fatalf("expected ZABBIX_HOST error, got %v", err)
	}
}
