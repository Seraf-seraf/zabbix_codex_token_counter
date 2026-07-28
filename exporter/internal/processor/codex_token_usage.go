package processor

import (
	"context"
	"log/slog"

	"github.com/example/otlp-zabbix-exporter/internal/zabbix"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

const CodexTokenUsageMetricName = "codex.turn.token_usage"

var knownTokenTypes = map[string]struct{}{
	"input": {}, "cached_input": {}, "cache_write_input": {}, "output": {}, "reasoning_output": {}, "total": {},
}

type CodexTokenUsageProcessor struct {
	host   string
	logger *slog.Logger
}

func NewCodexTokenUsage(host string, logger *slog.Logger) *CodexTokenUsageProcessor {
	return &CodexTokenUsageProcessor{host: host, logger: logger}
}

func (p *CodexTokenUsageProcessor) Name() string { return "codex_token_usage" }
func (p *CodexTokenUsageProcessor) Match(input pmetric.Metric) bool {
	return input.Name() == CodexTokenUsageMetricName || input.Name() == "turn.token_usage"
}

func (p *CodexTokenUsageProcessor) Process(_ context.Context, input pmetric.Metric) ([]zabbix.Metric, error) {
	const methodCtx = "processor.CodexTokenUsageProcessor.Process"

	var result []zabbix.Metric
	switch input.Type() {
	case pmetric.MetricTypeGauge:
		points := input.Gauge().DataPoints()
		for index := 0; index < points.Len(); index++ {
			point := points.At(index)
			result = append(result, p.processNumberPoint(input.Name(), point)...)
		}
	case pmetric.MetricTypeSum:
		points := input.Sum().DataPoints()
		for index := 0; index < points.Len(); index++ {
			point := points.At(index)
			result = append(result, p.processNumberPoint(input.Name(), point)...)
		}
	case pmetric.MetricTypeHistogram:
		points := input.Histogram().DataPoints()
		for index := 0; index < points.Len(); index++ {
			point := points.At(index)
			tokenType, ok := p.tokenType(input.Name(), point.Attributes())
			if !ok {
				continue
			}
			if !point.HasSum() {
				p.logger.Warn(methodCtx+": пропущена точка данных Codex без числового значения", "component", "processor", "processor", p.Name(), "metric_name", input.Name(), "token_type", tokenType)
				continue
			}
			result = append(result, zabbix.Metric{
				Host: p.host, Key: "codex.tokens." + tokenType, Value: point.Sum(), Timestamp: point.Timestamp().AsTime(),
			})
		}
	}
	return result, nil
}

func (p *CodexTokenUsageProcessor) processNumberPoint(metricName string, point pmetric.NumberDataPoint) []zabbix.Metric {
	const methodCtx = "processor.CodexTokenUsageProcessor.processNumberPoint"

	tokenType, ok := p.tokenType(metricName, point.Attributes())
	if !ok {
		return nil
	}
	var value any
	switch point.ValueType() {
	case pmetric.NumberDataPointValueTypeInt:
		value = point.IntValue()
	case pmetric.NumberDataPointValueTypeDouble:
		value = point.DoubleValue()
	default:
		p.logger.Warn(methodCtx+": пропущена точка данных Codex без числового значения", "component", "processor", "processor", p.Name(), "metric_name", metricName, "token_type", tokenType)
		return nil
	}
	return []zabbix.Metric{{
		Host: p.host, Key: "codex.tokens." + tokenType, Value: value, Timestamp: point.Timestamp().AsTime(),
	}}
}

func (p *CodexTokenUsageProcessor) tokenType(metricName string, attributes pcommon.Map) (string, bool) {
	const methodCtx = "processor.CodexTokenUsageProcessor.tokenType"

	value, ok := attributes.Get("token_type")
	if !ok || value.Type() != pcommon.ValueTypeStr {
		p.logger.Warn(methodCtx+": пропущена точка данных Codex без строкового token_type", "component", "processor", "processor", p.Name(), "metric_name", metricName)
		return "", false
	}
	tokenType := value.Str()
	if _, ok := knownTokenTypes[tokenType]; !ok {
		p.logger.Warn(methodCtx+": пропущен неизвестный тип токена Codex", "component", "processor", "processor", p.Name(), "metric_name", metricName, "token_type", tokenType)
		return "", false
	}
	return tokenType, true
}
