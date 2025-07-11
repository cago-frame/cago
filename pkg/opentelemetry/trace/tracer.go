package trace

import (
	"context"
	"go.opentelemetry.io/otel/trace/noop"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	// Sample 采样率 0-1 其它数字为跟随父配置
	Sample   float64           `yaml:"sample"`
	Endpoint string            `yaml:"endpoint"`
	UseSSL   bool              `yaml:"useSSL"`
	Type     string            `yaml:"type"`
	Header   map[string]string `yaml:"header"`
}

func NewWithConfig(ctx context.Context, cfg *Config, opts ...Option) (trace.TracerProvider, error) {
	options := &Options{}
	for _, v := range opts {
		v(options)
	}
	var sample tracesdk.Sampler
	if cfg.Sample <= 0 {
		// 总是关闭
		sample = tracesdk.NeverSample()
	} else if cfg.Sample < 1 {
		// 百分比采样，如果父开启了那么会开启
		sample = tracesdk.ParentBased(tracesdk.TraceIDRatioBased(cfg.Sample))
	} else if cfg.Sample == 1 {
		// 总是采样，如果父未开启那么不会开启
		sample = tracesdk.ParentBased(tracesdk.AlwaysSample())
	} else {
		// 总是关闭，如果父开启了那么会开启
		sample = tracesdk.ParentBased(tracesdk.NeverSample())
	}

	var client otlptrace.Client
	switch cfg.Type {
	case "http":
		clientOpts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(cfg.Endpoint),
			otlptracehttp.WithHeaders(cfg.Header),
			otlptracehttp.WithTimeout(10 * time.Second),
		}

		if !cfg.UseSSL {
			clientOpts = append(clientOpts, otlptracehttp.WithInsecure())
		}
		client = otlptracehttp.NewClient(clientOpts...)
	case "noop":
		return noop.NewTracerProvider(), nil
	default:
		clientOpts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.Endpoint),
			otlptracegrpc.WithHeaders(cfg.Header),
			otlptracegrpc.WithTimeout(10 * time.Second),
		}

		if !cfg.UseSSL {
			clientOpts = append(clientOpts, otlptracegrpc.WithInsecure())
		}
		client = otlptracegrpc.NewClient(clientOpts...)
	}

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(context.Background(),
		//resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithHost(),
		//resource.WithContainer(),
		//resource.WithOS(),
		//resource.WithProcess(),
		resource.WithAttributes(
			options.attrs...,
		))
	if err != nil {
		return nil, err
	}

	tp := tracesdk.NewTracerProvider(
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exporter),
		// Record information about this application in a Resource.
		tracesdk.WithResource(res),
		tracesdk.WithSampler(sample),
	)
	return tp, nil
}
