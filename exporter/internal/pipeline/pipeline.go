package pipeline

import (
	"context"
	"log/slog"
	"time"

	"github.com/example/otlp-zabbix-exporter/internal/processor"
	"github.com/example/otlp-zabbix-exporter/internal/zabbix"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

type Pipeline struct {
	registry  *processor.Registry
	sender    zabbix.Sender
	batchSize int
	logger    *slog.Logger
}

func New(registry *processor.Registry, sender zabbix.Sender, batchSize int, logger *slog.Logger) *Pipeline {
	return &Pipeline{registry: registry, sender: sender, batchSize: batchSize, logger: logger}
}

func (p *Pipeline) Process(ctx context.Context, metrics pmetric.Metrics) {
	const methodCtx = "pipeline.Pipeline.Process"

	var batch []zabbix.Metric
	flush := func() {
		if len(batch) == 0 {
			return
		}
		started := time.Now()
		if err := p.sender.Send(ctx, batch); err != nil {
			p.logger.Error(methodCtx+": не удалось отправить метрики в Zabbix", "component", "pipeline", "failed", len(batch), "duration", time.Since(started), "error", err)
		} else {
			p.logger.Debug(methodCtx+": метрики отправлены в Zabbix", "component", "pipeline", "processed", len(batch), "failed", 0, "duration", time.Since(started))
		}
		batch = batch[:0]
	}

	resourceMetrics := metrics.ResourceMetrics()
	for resourceIndex := 0; resourceIndex < resourceMetrics.Len(); resourceIndex++ {
		scopeMetrics := resourceMetrics.At(resourceIndex).ScopeMetrics()
		for scopeIndex := 0; scopeIndex < scopeMetrics.Len(); scopeIndex++ {
			scope := scopeMetrics.At(scopeIndex)
			inputMetrics := scope.Metrics()
			for metricIndex := 0; metricIndex < inputMetrics.Len(); metricIndex++ {
				input := inputMetrics.At(metricIndex)
				output, processorName, err := p.registry.Process(ctx, input)
				if err != nil {
					p.logger.Error(methodCtx+": не удалось обработать метрику", "component", "pipeline", "processor", processorName, "metric_name", input.Name(), "metric_type", input.Type().String(), "error", err)
					continue
				}
				p.logger.Debug(methodCtx+": метрика обработана", "component", "pipeline", "processor", processorName, "metric_name", input.Name(), "metric_type", input.Type().String(), "scope_name", scope.Scope().Name(), "data_point_count", dataPointCount(input), "zabbix_metric_count", len(output))
				for len(output) > 0 {
					room := p.batchSize - len(batch)
					if room > len(output) {
						room = len(output)
					}
					batch = append(batch, output[:room]...)
					output = output[room:]
					if len(batch) == p.batchSize {
						flush()
					}
				}
			}
		}
	}
	flush()
}

func dataPointCount(input pmetric.Metric) int {
	switch input.Type() {
	case pmetric.MetricTypeGauge:
		return input.Gauge().DataPoints().Len()
	case pmetric.MetricTypeSum:
		return input.Sum().DataPoints().Len()
	case pmetric.MetricTypeHistogram:
		return input.Histogram().DataPoints().Len()
	case pmetric.MetricTypeExponentialHistogram:
		return input.ExponentialHistogram().DataPoints().Len()
	case pmetric.MetricTypeSummary:
		return input.Summary().DataPoints().Len()
	default:
		return 0
	}
}
