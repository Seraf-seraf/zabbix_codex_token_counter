package processor

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/example/otlp-zabbix-exporter/internal/zabbix"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

type stubProcessor struct {
	name    string
	matches bool
	key     string
}

func (p stubProcessor) Name() string              { return p.name }
func (p stubProcessor) Match(pmetric.Metric) bool { return p.matches }
func (p stubProcessor) Process(context.Context, pmetric.Metric) ([]zabbix.Metric, error) {
	return []zabbix.Metric{{Key: p.key}}, nil
}

func TestRegistryFirstMatchAndFallback(t *testing.T) {
	input := pmetric.NewMetric()
	registry := NewRegistry([]Processor{
		stubProcessor{name: "first", matches: true, key: "first"},
		stubProcessor{name: "second", matches: true, key: "second"},
	}, stubProcessor{name: "generic", matches: true, key: "generic"})
	output, name, err := registry.Process(context.Background(), input)
	if err != nil || name != "first" || output[0].Key != "first" {
		t.Fatalf("unexpected result: output=%+v name=%s err=%v", output, name, err)
	}
	registry = NewRegistry(nil, stubProcessor{name: "generic", matches: true, key: "generic"})
	output, name, _ = registry.Process(context.Background(), input)
	if name != "generic" || output[0].Key != "generic" {
		t.Fatalf("fallback not selected")
	}
	registry = NewRegistry(nil, nil)
	output, name, _ = registry.Process(context.Background(), input)
	if output != nil || name != "" {
		t.Fatalf("expected no match")
	}
}

func TestCodexTokenUsage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	instance := NewCodexTokenUsage("host", logger)
	for _, tokenType := range []string{"input", "cached_input", "cache_write_input", "output", "reasoning_output", "total"} {
		t.Run(tokenType, func(t *testing.T) {
			input := histogramMetric(CodexTokenUsageMetricName, tokenType, 9)
			output, err := instance.Process(context.Background(), input)
			if err != nil || len(output) != 1 || output[0].Key != "codex.tokens."+tokenType {
				t.Fatalf("output=%+v err=%v", output, err)
			}
		})
	}
}

func TestCodexTokenUsageNumberValues(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	instance := NewCodexTokenUsage("host", logger)
	for _, metricType := range []pmetric.MetricType{pmetric.MetricTypeGauge, pmetric.MetricTypeSum} {
		input := numberMetric(CodexTokenUsageMetricName, metricType, "input", 7)
		output, err := instance.Process(context.Background(), input)
		if err != nil || len(output) != 1 || output[0].Value != int64(7) {
			t.Fatalf("type=%s output=%+v err=%v", metricType, output, err)
		}
	}
}

func TestCodexTokenUsageSkipsInvalidPointsAndLogsInRussian(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	instance := NewCodexTokenUsage("host", logger)
	input := pmetric.NewMetric()
	input.SetName(CodexTokenUsageMetricName)
	points := input.SetEmptyHistogram().DataPoints()
	points.AppendEmpty()
	unknown := points.AppendEmpty()
	unknown.Attributes().PutStr("token_type", "future")
	missingValue := points.AppendEmpty()
	missingValue.Attributes().PutStr("token_type", "input")

	output, err := instance.Process(context.Background(), input)
	if err != nil || len(output) != 0 {
		t.Fatalf("output=%+v err=%v", output, err)
	}
	logText := logs.String()
	if !strings.Contains(logText, "processor.CodexTokenUsageProcessor.tokenType:") ||
		!strings.Contains(logText, "пропущен неизвестный тип токена") ||
		!strings.Contains(logText, "processor.CodexTokenUsageProcessor.Process:") {
		t.Fatalf("unexpected logs: %s", logText)
	}
}

func TestGenericNormalizesKeyAndHistogram(t *testing.T) {
	input := histogramMetric("http server/latency", "", 3)
	point := input.Histogram().DataPoints().At(0)
	point.SetCount(2)
	point.SetMin(1)
	point.SetMax(4)
	output, err := NewGeneric("host", "otel").Process(context.Background(), input)
	if err != nil || len(output) != 4 || output[0].Key != "otel.http_server_latency.count" {
		t.Fatalf("output=%+v err=%v", output, err)
	}
}

func TestGenericNumberValues(t *testing.T) {
	for _, metricType := range []pmetric.MetricType{pmetric.MetricTypeGauge, pmetric.MetricTypeSum} {
		input := numberMetric("requests", metricType, "", 12)
		output, err := NewGeneric("host", "otel").Process(context.Background(), input)
		if err != nil || len(output) != 1 || output[0].Key != "otel.requests" || output[0].Value != int64(12) {
			t.Fatalf("type=%s output=%+v err=%v", metricType, output, err)
		}
	}
}

func numberMetric(name string, metricType pmetric.MetricType, tokenType string, value int64) pmetric.Metric {
	input := pmetric.NewMetric()
	input.SetName(name)
	var points pmetric.NumberDataPointSlice
	if metricType == pmetric.MetricTypeGauge {
		points = input.SetEmptyGauge().DataPoints()
	} else {
		points = input.SetEmptySum().DataPoints()
	}
	point := points.AppendEmpty()
	point.SetIntValue(value)
	point.SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(100, 0)))
	if tokenType != "" {
		point.Attributes().PutStr("token_type", tokenType)
	}
	return input
}

func histogramMetric(name, tokenType string, sum float64) pmetric.Metric {
	input := pmetric.NewMetric()
	input.SetName(name)
	point := input.SetEmptyHistogram().DataPoints().AppendEmpty()
	point.SetSum(sum)
	point.SetTimestamp(pcommon.NewTimestampFromTime(time.Unix(100, 0)))
	if tokenType != "" {
		point.Attributes().PutStr("token_type", tokenType)
	}
	return input
}
