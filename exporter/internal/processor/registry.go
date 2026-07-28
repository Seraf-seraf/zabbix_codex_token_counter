package processor

import (
	"context"
	"fmt"

	"github.com/example/otlp-zabbix-exporter/internal/zabbix"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

type Registry struct {
	specialized []Processor
	generic     Processor
}

func NewRegistry(specialized []Processor, generic Processor) *Registry {
	return &Registry{specialized: append([]Processor(nil), specialized...), generic: generic}
}

func (r *Registry) Process(ctx context.Context, input pmetric.Metric) ([]zabbix.Metric, string, error) {
	if err := ctx.Err(); err != nil {
        return nil, "", err
    }

	for _, candidate := range r.specialized {
		if candidate.Match(input) {
			result, err := candidate.Process(ctx, input)
			return result, candidate.Name(), wrapProcessorError(candidate.Name(), err)
		}
	}
	if r.generic != nil && r.generic.Match(input) {
		result, err := r.generic.Process(ctx, input)
		return result, r.generic.Name(), wrapProcessorError(r.generic.Name(), err)
	}
	return nil, "", nil
}

func wrapProcessorError(name string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("обработчик %s: %w", name, err)
}
