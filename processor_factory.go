package processor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
)

const (
	// TypeStr is the unique identifier for the metrics processor
	TypeStr = "metricsprocessor"
)

// processorFactory is the factory for the metrics processor
type processorFactory struct{}

// NewFactory creates a new factory for the metrics processor
func NewFactory() component.Factory {
	return &processorFactory{}
}

// Type returns the type of the factory
func (f *processorFactory) Type() component.Type {
	return component.Type(TypeStr)
}

// CreateDefaultConfig creates the default configuration for the processor
func (f *processorFactory) CreateDefaultConfig() component.Config {
	return &Config{
		WindowSize:       100,
		AnomalyThreshold: 3.0,
		MinDataPoints:    10,
	}
}

// CreateMetricsProcessor creates a metrics processor based on the given config
func (f *processorFactory) CreateMetricsProcessor(
	ctx context.Context,
	params interface{},
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (consumer.Metrics, error) {
	pCfg, ok := cfg.(*Config)
	if !ok {
		return nil, component.ErrDataTypeIsNotSupported
	}
	return newMetricsProcessor(nextConsumer, pCfg), nil
}
