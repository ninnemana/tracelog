package main

import (
	"context"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/ninnemana/tracelog"
)

const (
	instrumentationName    = "github.com/ninnemana"
	instrumentationVersion = "v0.1.0"
)

var (
	tracer = otel.GetTracerProvider().Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(instrumentationVersion),
		trace.WithSchemaURL(semconv.SchemaURL),
	)
)

func installPipeline(ctx context.Context) func() {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatalf("creating stdout exporter: %v", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("stdout-example"),
			semconv.ServiceVersionKey.String("0.0.1"),
		)),
	)
	otel.SetTracerProvider(tracerProvider)

	return func() {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			log.Fatalf("stopping tracer provider: %v", err)
		}
	}
}

func main() {
	ctx := context.Background()

	flush := installPipeline(ctx)
	defer flush()

	ctx, span := tracer.Start(ctx, "sample")
	defer span.End()

	l, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	lg := tracelog.NewLogger(tracelog.WithLogger(l))

	lg = lg.Context(ctx)

	lg.Info(
		"hello world",
		attribute.KeyValue{
			Key:   "trace",
			Value: attribute.StringValue("value"),
		},
		zap.String("log", "value"),
	)
}
