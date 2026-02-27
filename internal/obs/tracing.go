package obs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type TracingConfig struct {
	Enabled          bool
	ServiceName      string
	OTLPHTTPEndpoint string
	SampleRatio      float64
}

func InitTracing(ctx context.Context, cfg TracingConfig) (func(context.Context) error, error) {
	if !cfg.Enabled {
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			attribute.String("service.namespace", "ispo"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}

	sampler := sdktrace.TraceIDRatioBased(cfg.SampleRatio)

	var exporter sdktrace.SpanExporter
	if cfg.OTLPHTTPEndpoint != "" {
		exp, err := otlptracehttp.New(ctx,
			otlptracehttp.WithEndpointURL(cfg.OTLPHTTPEndpoint),
			otlptracehttp.WithTimeout(5*time.Second),
		)
		if err != nil {
			return nil, fmt.Errorf("otlp http exporter: %w", err)
		}
		exporter = exp
	} else {
		exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("stdout exporter: %w", err)
		}
		exporter = exp
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	shutdown := func(ctx context.Context) error {
		var outErr error
		if err := provider.Shutdown(ctx); err != nil {
			outErr = errors.Join(outErr, err)
		}
		return outErr
	}
	return shutdown, nil
}
