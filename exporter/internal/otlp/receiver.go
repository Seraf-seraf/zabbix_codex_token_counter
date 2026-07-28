package otlp

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
)

type Pipeline interface {
	Process(context.Context, pmetric.Metrics)
}

type Receiver struct {
	pmetricotlp.UnimplementedGRPCServer
	pipeline Pipeline
}

func NewReceiver(pipeline Pipeline) *Receiver {
	return &Receiver{pipeline: pipeline}
}

func (r *Receiver) Export(ctx context.Context, request pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
	if err := ctx.Err(); err != nil {
		return pmetricotlp.ExportResponse{}, err
	}

	r.pipeline.Process(ctx, request.Metrics())
	return pmetricotlp.NewExportResponse(), nil
}
