package tracelog

import (
	"context"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// A TraceLogger wraps the base Logger functionality in logic to tag
// and correlate OpenTelemtry data with the associated log entries.
type TraceLogger struct {
	base *zap.Logger
	ctx  context.Context
}

type LoggerOption func(*TraceLogger)

// WithLogger sets the base logger to use in the TraceLogger.
func WithLogger(lg *zap.Logger) LoggerOption {
	return func(tl *TraceLogger) {
		if tl != nil {
			tl.base = lg
		}
	}
}

// NewLogger instaniates a new instance our of logger.
func NewLogger(opts ...LoggerOption) *TraceLogger {
	tl := &TraceLogger{}
	for _, opt := range opts {
		opt(tl)
	}

	return tl
}

// Named adds a sub-scope to the logger's name. See Logger.Named for details.
func (tl *TraceLogger) Named(name string) *TraceLogger {
	return &TraceLogger{
		base: tl.base.Named(name),
		ctx:  tl.ctx,
	}
}

// SetContext associates the `context.Context` in use with the instance of our logger.
func (tl *TraceLogger) SetContext(ctx context.Context) *TraceLogger {
	l := &TraceLogger{
		base: tl.base,
		ctx:  ctx,
	}

	span := trace.SpanFromContext(l.ctx)
	if span == nil {
		return tl
	}

	spanCtx := span.SpanContext()

	return l.With(
		zap.String("traceID", spanCtx.TraceID().String()),
		zap.String("dd.traceID", spanCtx.TraceID().String()),
		zap.String("spanID", spanCtx.SpanID().String()),
		zap.String("dd.spanID", spanCtx.SpanID().String()),
	)
}

// With adds a variadic number of fields to the logging context. It accepts a
// mix of strongly-typed Field objects.
func (tl *TraceLogger) With(args ...zap.Field) *TraceLogger {
	return &TraceLogger{base: tl.base.With(args...)}
}

// FromRequest retrieves any HTTP Headers on the provided request and associates
// the current TraceLogger's `context.Context`.
func (tl *TraceLogger) FromRequest(r *http.Request) *TraceLogger {
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

	return tl.SetContext(ctx)
}

// WithRequest tags the outgoing `http.Request` with HTTP Headers to associate any downstream
// tracing with the provided `context.Context`.
func (tl *TraceLogger) WithRequest(ctx context.Context, r *http.Request) *http.Request {
	r2 := new(http.Request)
	*r2 = *r

	span := trace.SpanFromContext(r2.Context())
	if span != nil {
		attrs := semconv.NetAttributesFromHTTPRequest("tcp", r2)
		attrs = append(attrs, semconv.EndUserAttributesFromHTTPRequest(r2)...)
		attrs = append(attrs, semconv.HTTPServerAttributesFromHTTPRequest("http.server", r2.URL.String(), r2)...)

		span.SetAttributes(attrs...)
	}

	r2 = r2.WithContext(ctx)

	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r2.Header))

	return r2
}

// Debug uses fmt.Sprint to construct and log a message.
func (tl *TraceLogger) Debug(msg string, args ...interface{}) {
	fields, tags := parseArguments(args...)
	tagSpan(tl.ctx, tags...)
	tl.base.Debug(msg, fields...)
}

// Info uses fmt.Sprint to construct and log a message.
func (tl *TraceLogger) Info(msg string, args ...interface{}) {
	fields, tags := parseArguments(args...)
	tagSpan(tl.ctx, tags...)
	tl.base.Info(msg, fields...)
}

// Warn uses fmt.Sprint to construct and log a message.
func (tl *TraceLogger) Warn(msg string, args ...interface{}) {
	fields, tags := parseArguments(args...)
	tagSpan(tl.ctx, tags...)
	tl.base.Warn(msg, fields...)
}

// Error uses fmt.Sprint to construct and log a message.
func (tl *TraceLogger) Error(msg string, args ...interface{}) {
	fields, tags := parseArguments(args...)
	tagSpan(tl.ctx, tags...)
	tl.base.Error(msg, fields...)
}

// DPanic uses fmt.Sprint to construct and log a message. In development, the
// logger then panics. (See DPanicLevel for details.)
func (tl *TraceLogger) DPanic(msg string, args ...interface{}) {
	fields, tags := parseArguments(args...)
	tagSpan(tl.ctx, tags...)
	tl.base.DPanic(msg, fields...)
}

// Panic uses fmt.Sprint to construct and log a message, then panics.
func (tl *TraceLogger) Panic(msg string, args ...interface{}) {
	fields, tags := parseArguments(args...)
	tagSpan(tl.ctx, tags...)
	tl.base.Panic(msg, fields...)
}

// Fatal uses fmt.Sprint to construct and log a message, then calls os.Exit.
func (tl *TraceLogger) Fatal(msg string, args ...interface{}) {
	fields, tags := parseArguments(args...)
	tagSpan(tl.ctx, tags...)
	tl.base.Fatal(msg, fields...)
}

// Sync flushes any buffered log entries.
func (tl *TraceLogger) Sync() error {
	if err := tl.base.Sync(); err != nil {
		return fmt.Errorf("failed to sync logger: %w", err)
	}

	return nil
}

func parseArguments(args ...interface{}) ([]zap.Field, []attribute.KeyValue) {
	var (
		tags   []attribute.KeyValue
		fields []zap.Field
	)

	for _, arg := range args {
		switch v := arg.(type) {
		case attribute.KeyValue:
			tags = append(tags, v)
		case zap.Field:
			fields = append(fields, v)
		}
	}

	return fields, tags
}

func tagSpan(ctx context.Context, tags ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return
	}

	span.SetAttributes(tags...)
}
