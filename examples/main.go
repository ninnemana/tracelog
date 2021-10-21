package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
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
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return func() {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			log.Fatalf("stopping tracer provider: %v", err)
		}
	}
}

func server(l *zap.Logger) string {
	lg := tracelog.NewLogger(
		tracelog.WithLogger(l),
	)
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lg = lg.FromRequest(r)

		ctx, span := tracer.Start(r.Context(), "http.client")
		defer span.End()

		l := lg.SetContext(ctx)
		l.Info("handling HTTP request")
	}))

	return svr.URL
}

func main() {
	ctx := context.Background()

	l, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	flush := installPipeline(ctx)
	defer flush()

	ctx, span := tracer.Start(ctx, "sample")
	defer span.End()

	lg := tracelog.NewLogger(
		tracelog.WithLogger(l),
	)

	lg = lg.SetContext(ctx)

	route, err := url.Parse(server(l))
	if err != nil {
		lg.Fatal("failed to parse test server address", zap.Error(err))

		return
	}

	req, err := http.NewRequest(http.MethodPost, route.String(), nil)
	if err != nil {
		lg.Fatal("failed to create HTTP request", zap.Error(err))
	}

	request := lg.WithRequest(ctx, req)

	lg.Info("created request")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		lg.Fatal("failed to execute HTTP request", zap.Error(err))

		return
	}

	defer resp.Body.Close()
}
