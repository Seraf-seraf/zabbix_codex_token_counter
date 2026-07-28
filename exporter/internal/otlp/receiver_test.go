package otlp

import (
	"context"
	"testing"

	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
)

type pipelineStub struct {
	metrics pmetric.Metrics
}

func (p *pipelineStub) Process(_ context.Context, metrics pmetric.Metrics) {
	p.metrics = metrics
}

func TestReceiverPassesPdataMetricsToPipeline(t *testing.T) {
	request := pmetricotlp.NewExportRequest()
	metric := request.Metrics().ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	metric.SetName("requests")
	metric.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(7)
	pipeline := &pipelineStub{}

	response, err := NewReceiver(pipeline).Export(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if pipeline.metrics.MetricCount() != 1 || pipeline.metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Name() != "requests" {
		t.Fatalf("unexpected metrics: count=%d", pipeline.metrics.MetricCount())
	}
	if response.PartialSuccess().RejectedDataPoints() != 0 {
		t.Fatalf("unexpected response: %+v", response)
	}
}
