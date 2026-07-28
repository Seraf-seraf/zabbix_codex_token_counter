package processor

import (
	"context"

	"github.com/example/otlp-zabbix-exporter/internal/zabbix"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

type Processor interface {
	Name() string
	Match(pmetric.Metric) bool
	Process(context.Context, pmetric.Metric) ([]zabbix.Metric, error)
}
