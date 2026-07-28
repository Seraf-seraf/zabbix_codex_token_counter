package processor

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/example/otlp-zabbix-exporter/internal/zabbix"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var invalidKeyCharacters = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

type GenericProcessor struct {
	host   string
	prefix string
}

func NewGeneric(host, prefix string) *GenericProcessor {
	return &GenericProcessor{host: host, prefix: strings.Trim(prefix, ".")}
}

func (p *GenericProcessor) Name() string { return "generic" }

func (p *GenericProcessor) Match(input pmetric.Metric) bool {
	return input.Type() == pmetric.MetricTypeGauge || input.Type() == pmetric.MetricTypeSum || input.Type() == pmetric.MetricTypeHistogram
}

func (p *GenericProcessor) Process(_ context.Context, input pmetric.Metric) ([]zabbix.Metric, error) {
	base := invalidKeyCharacters.ReplaceAllString(input.Name(), "_")
	base = strings.Trim(base, ".")
	if base == "" {
		return nil, fmt.Errorf("имя метрики %q преобразуется в пустой ключ Zabbix", input.Name())
	}
	if p.prefix != "" {
		base = p.prefix + "." + base
	}

	var result []zabbix.Metric
	switch input.Type() {
	case pmetric.MetricTypeGauge:
		result = append(result, p.numberMetrics(base, input.Gauge().DataPoints())...)
	case pmetric.MetricTypeSum:
		result = append(result, p.numberMetrics(base, input.Sum().DataPoints())...)
	case pmetric.MetricTypeHistogram:
		points := input.Histogram().DataPoints()
		for index := 0; index < points.Len(); index++ {
			point := points.At(index)
			timestamp := point.Timestamp().AsTime()
			result = append(result, zabbix.Metric{Host: p.host, Key: base + ".count", Value: point.Count(), Timestamp: timestamp})
			if point.HasSum() {
				result = append(result, zabbix.Metric{Host: p.host, Key: base + ".sum", Value: point.Sum(), Timestamp: timestamp})
			}
			if point.HasMin() {
				result = append(result, zabbix.Metric{Host: p.host, Key: base + ".min", Value: point.Min(), Timestamp: timestamp})
			}
			if point.HasMax() {
				result = append(result, zabbix.Metric{Host: p.host, Key: base + ".max", Value: point.Max(), Timestamp: timestamp})
			}
		}
	}
	return result, nil
}

func (p *GenericProcessor) numberMetrics(key string, points pmetric.NumberDataPointSlice) []zabbix.Metric {
	result := make([]zabbix.Metric, 0, points.Len())
	for index := 0; index < points.Len(); index++ {
		point := points.At(index)
		var value any
		switch point.ValueType() {
		case pmetric.NumberDataPointValueTypeInt:
			value = point.IntValue()
		case pmetric.NumberDataPointValueTypeDouble:
			value = point.DoubleValue()
		default:
			continue
		}
		result = append(result, zabbix.Metric{Host: p.host, Key: key, Value: value, Timestamp: point.Timestamp().AsTime()})
	}
	return result
}
