package tracelog

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
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

func WithLogger(lg *zap.Logger) LoggerOption {
	return func(tl *TraceLogger) {
		if tl != nil {
			tl.base = lg
		}
	}
}

func NewLogger(opts ...LoggerOption) *TraceLogger {
	tl := &TraceLogger{}
	for _, opt := range opts {
		opt(tl)
	}

	return tl
}

// Named adds a sub-scope to the logger's name. See Logger.Named for details.
func (l *TraceLogger) Named(name string) *TraceLogger {
	return &TraceLogger{
		base: l.base.Named(name),
		ctx:  context.Background(),
	}
}

func (l *TraceLogger) Context(ctx context.Context) *TraceLogger {
	tl := &TraceLogger{
		base: l.base,
		ctx:  ctx,
	}

	span := trace.SpanFromContext(tl.ctx)
	if span == nil {
		fmt.Println("no span")
		return tl
	}

	spanCtx := span.SpanContext()

	return tl.With(
		zap.String("traceID", spanCtx.TraceID().String()),
		zap.String("dd.traceID", spanCtx.TraceID().String()),
		zap.String("spanID", spanCtx.SpanID().String()),
		zap.String("dd.spanID", spanCtx.SpanID().String()),
	)
}

// With adds a variadic number of fields to the logging context. It accepts a
// mix of strongly-typed Field objects.
func (t *TraceLogger) With(args ...zap.Field) *TraceLogger {
	return &TraceLogger{base: t.base.With(args...)}
}

// Debug uses fmt.Sprint to construct and log a message.
func (t *TraceLogger) Debug(msg string, args ...interface{}) {
	fields, tags := parseArguments(args...)
	tagSpan(t.ctx, tags...)
	t.base.Debug(msg, fields...)
}

// Info uses fmt.Sprint to construct and log a message.
func (t *TraceLogger) Info(msg string, args ...interface{}) {
	fields, tags := parseArguments(args...)
	tagSpan(t.ctx, tags...)
	t.base.Info(msg, fields...)
}

// Warn uses fmt.Sprint to construct and log a message.
func (t *TraceLogger) Warn(msg string, args ...interface{}) {
	fields, tags := parseArguments(args...)
	tagSpan(t.ctx, tags...)
	t.base.Warn(msg, fields...)
}

// Error uses fmt.Sprint to construct and log a message.
func (t *TraceLogger) Error(msg string, args ...interface{}) {
	fields, tags := parseArguments(args...)
	tagSpan(t.ctx, tags...)
	t.base.Error(msg, fields...)
}

// DPanic uses fmt.Sprint to construct and log a message. In development, the
// logger then panics. (See DPanicLevel for details.)
func (t *TraceLogger) DPanic(msg string, args ...interface{}) {
	fields, tags := parseArguments(args...)
	tagSpan(t.ctx, tags...)
	t.base.DPanic(msg, fields...)
}

// Panic uses fmt.Sprint to construct and log a message, then panics.
func (t *TraceLogger) Panic(msg string, args ...interface{}) {
	fields, tags := parseArguments(args...)
	tagSpan(t.ctx, tags...)
	t.base.Panic(msg, fields...)
}

// Fatal uses fmt.Sprint to construct and log a message, then calls os.Exit.
func (t *TraceLogger) Fatal(msg string, args ...interface{}) {
	fields, tags := parseArguments(args...)
	tagSpan(t.ctx, tags...)
	t.base.Fatal(msg, fields...)
}

// Sync flushes any buffered log entries.
func (t *TraceLogger) Sync() error {
	return t.base.Sync()
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
