package tracelog

import (
	"context"
	"fmt"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

const (
	_oddNumberErrMsg    = "Ignored key without a value."
	_nonStringKeyErrMsg = "Ignored key-value pairs with non-string keys."
)

// Named adds a sub-scope to the logger's name. See Logger.Named for details.
func (l *TraceLogger) Named(name string) *TraceLogger {
	return &TraceLogger{
		base: l.base.Named(name),
		ctx:  context.Background(),
	}
}

func (l *TraceLogger) Context(ctx context.Context) *TraceLogger {
	return &TraceLogger{
		base: l.base,
		ctx:  ctx,
	}
}

// With adds a variadic number of fields to the logging context. It accepts a
// mix of strongly-typed Field objects and loosely-typed key-value pairs. When
// processing pairs, the first element of the pair is used as the field key
// and the second as the field value.
//
// For example,
//   TraceLogger.With(
//     "hello", "world",
//     "failure", errors.New("oh no"),
//     Stack(),
//     "count", 42,
//     "user", User{Name: "alice"},
//  )
// is the equivalent of
//   unsugared.With(
//     String("hello", "world"),
//     String("failure", "oh no"),
//     Stack(),
//     Int("count", 42),
//     Object("user", User{Name: "alice"}),
//   )
//
// Note that the keys in key-value pairs should be strings. In development,
// passing a non-string key panics. In production, the logger is more
// forgiving: a separate error is logged, but the key-value pair is skipped
// and execution continues. Passing an orphaned key triggers similar behavior:
// panics in development and errors in production.
func (s *TraceLogger) With(args ...interface{}) *TraceLogger {
	return &TraceLogger{base: s.base.With(s.sweetenFields(args)...)}
}

// Debug uses fmt.Sprint to construct and log a message.
func (s *TraceLogger) Debug(args ...interface{}) {
	s.log(zap.DebugLevel, "", args, nil)
}

// Info uses fmt.Sprint to construct and log a message.
func (s *TraceLogger) Info(args ...interface{}) {
	s.log(zap.InfoLevel, "", args, nil)
}

// Warn uses fmt.Sprint to construct and log a message.
func (s *TraceLogger) Warn(args ...interface{}) {
	s.log(zap.WarnLevel, "", args, nil)
}

// Error uses fmt.Sprint to construct and log a message.
func (s *TraceLogger) Error(args ...interface{}) {
	s.log(zap.ErrorLevel, "", args, nil)
}

// DPanic uses fmt.Sprint to construct and log a message. In development, the
// logger then panics. (See DPanicLevel for details.)
func (s *TraceLogger) DPanic(args ...interface{}) {
	s.log(zap.DPanicLevel, "", args, nil)
}

// Panic uses fmt.Sprint to construct and log a message, then panics.
func (s *TraceLogger) Panic(args ...interface{}) {
	s.log(zap.PanicLevel, "", args, nil)
}

// Fatal uses fmt.Sprint to construct and log a message, then calls os.Exit.
func (s *TraceLogger) Fatal(args ...interface{}) {
	s.log(zap.FatalLevel, "", args, nil)
}

// Debugf uses fmt.Sprintf to log a templated message.
func (s *TraceLogger) Debugf(template string, args ...interface{}) {
	s.log(zap.DebugLevel, template, args, nil)
}

// Infof uses fmt.Sprintf to log a templated message.
func (s *TraceLogger) Infof(template string, args ...interface{}) {
	s.log(zap.InfoLevel, template, args, nil)
}

// Warnf uses fmt.Sprintf to log a templated message.
func (s *TraceLogger) Warnf(template string, args ...interface{}) {
	s.log(zap.WarnLevel, template, args, nil)
}

// Errorf uses fmt.Sprintf to log a templated message.
func (s *TraceLogger) Errorf(template string, args ...interface{}) {
	s.log(zap.ErrorLevel, template, args, nil)
}

// DPanicf uses fmt.Sprintf to log a templated message. In development, the
// logger then panics. (See DPanicLevel for details.)
func (s *TraceLogger) DPanicf(template string, args ...interface{}) {
	s.log(zap.DPanicLevel, template, args, nil)
}

// Panicf uses fmt.Sprintf to log a templated message, then panics.
func (s *TraceLogger) Panicf(template string, args ...interface{}) {
	s.log(zap.PanicLevel, template, args, nil)
}

// Fatalf uses fmt.Sprintf to log a templated message, then calls os.Exit.
func (s *TraceLogger) Fatalf(template string, args ...interface{}) {
	s.log(zap.FatalLevel, template, args, nil)
}

// Debugw logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
//
// When debug-level logging is disabled, this is much faster than
//  s.With(keysAndValues).Debug(msg)
func (s *TraceLogger) Debugw(msg string, keysAndValues ...interface{}) {
	s.log(zap.DebugLevel, msg, nil, keysAndValues)
}

// Infow logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func (s *TraceLogger) Infow(msg string, keysAndValues ...interface{}) {
	s.log(zap.InfoLevel, msg, nil, keysAndValues)
}

// Warnw logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func (s *TraceLogger) Warnw(msg string, keysAndValues ...interface{}) {
	s.log(zap.WarnLevel, msg, nil, keysAndValues)
}

// Errorw logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func (s *TraceLogger) Errorw(msg string, keysAndValues ...interface{}) {
	s.log(zap.ErrorLevel, msg, nil, keysAndValues)
}

// DPanicw logs a message with some additional context. In development, the
// logger then panics. (See DPanicLevel for details.) The variadic key-value
// pairs are treated as they are in With.
func (s *TraceLogger) DPanicw(msg string, keysAndValues ...interface{}) {
	s.log(zap.DPanicLevel, msg, nil, keysAndValues)
}

// Panicw logs a message with some additional context, then panics. The
// variadic key-value pairs are treated as they are in With.
func (s *TraceLogger) Panicw(msg string, keysAndValues ...interface{}) {
	s.log(zap.PanicLevel, msg, nil, keysAndValues)
}

// Fatalw logs a message with some additional context, then calls os.Exit. The
// variadic key-value pairs are treated as they are in With.
func (s *TraceLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	s.log(zap.FatalLevel, msg, nil, keysAndValues)
}

// Sync flushes any buffered log entries.
func (s *TraceLogger) Sync() error {
	return s.base.Sync()
}

func (s *TraceLogger) log(lvl zapcore.Level, template string, fmtArgs []interface{}, context []interface{}) {
	// If logging at this level is completely disabled, skip the overhead of
	// string formatting.
	if lvl < zap.DPanicLevel && !s.base.Core().Enabled(lvl) {
		return
	}

	msg := getMessage(template, fmtArgs)
	if ce := s.base.Check(lvl, msg); ce != nil {
		ce.Write(s.sweetenFields(context)...)
	}
}

// getMessage format with Sprint, Sprintf, or neither.
func getMessage(template string, fmtArgs []interface{}) string {
	if len(fmtArgs) == 0 {
		return template
	}

	if template != "" {
		return fmt.Sprintf(template, fmtArgs...)
	}

	if len(fmtArgs) == 1 {
		if str, ok := fmtArgs[0].(string); ok {
			return str
		}
	}
	return fmt.Sprint(fmtArgs...)
}

func (s *TraceLogger) sweetenFields(args []interface{}) []zap.Field {
	if len(args) == 0 {
		return nil
	}

	// Allocate enough space for the worst case; if users pass only structured
	// fields, we shouldn't penalize them with extra allocations.
	fields := make([]zap.Field, 0, len(args))
	var invalid invalidPairs

	for i := 0; i < len(args); {
		// This is a strongly-typed field. Consume it and move on.
		if f, ok := args[i].(zap.Field); ok {
			fields = append(fields, f)
			i++
			continue
		}

		// Make sure this element isn't a dangling key.
		if i == len(args)-1 {
			s.base.Error(_oddNumberErrMsg, zap.Any("ignored", args[i]))
			break
		}

		// Consume this value and the next, treating them as a key-value pair. If the
		// key isn't a string, add this pair to the slice of invalid pairs.
		key, val := args[i], args[i+1]
		if keyStr, ok := key.(string); !ok {
			// Subsequent errors are likely, so allocate once up front.
			if cap(invalid) == 0 {
				invalid = make(invalidPairs, 0, len(args)/2)
			}
			invalid = append(invalid, invalidPair{i, key, val})
		} else {
			fields = append(fields, zap.Any(keyStr, val))
		}
		i += 2
	}

	// If we encountered any invalid key-value pairs, log an error.
	if len(invalid) > 0 {
		s.base.Error(_nonStringKeyErrMsg, zap.Array("invalid", invalid))
	}
	return fields
}

type invalidPair struct {
	position   int
	key, value interface{}
}

func (p invalidPair) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt64("position", int64(p.position))
	zap.Any("key", p.key).AddTo(enc)
	zap.Any("value", p.value).AddTo(enc)
	return nil
}

type invalidPairs []invalidPair

func (ps invalidPairs) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	var err error
	for i := range ps {
		err = multierr.Append(err, enc.AppendObject(ps[i]))
	}
	return err
}
