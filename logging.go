package langfuse

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"time"
)

// Logger is a minimal logging interface for the SDK.
// It's compatible with standard library log.Logger and popular logging frameworks.
//
// Deprecated: Use StructuredLogger instead, which provides leveled logging.
// You can wrap a printf-style logger using WrapPrintfLogger():
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithStructuredLogger(langfuse.WrapPrintfLogger(log.Default())),
//	)
type Logger interface {
	// Printf logs a formatted message.
	Printf(format string, v ...any)
}

// StructuredLogger provides structured logging support for the SDK.
// This is the preferred logging interface and is compatible with Go 1.21's
// slog package and similar structured logging libraries.
//
// Use WithStructuredLogger() to configure:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithStructuredLogger(langfuse.NewSlogAdapter(slog.Default())),
//	)
//
// Or wrap a standard logger:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithStructuredLogger(langfuse.WrapPrintfLogger(log.Default())),
//	)
type StructuredLogger interface {
	// Debug logs a debug-level message with optional key-value pairs.
	Debug(msg string, args ...any)
	// Info logs an info-level message with optional key-value pairs.
	Info(msg string, args ...any)
	// Warn logs a warning-level message with optional key-value pairs.
	Warn(msg string, args ...any)
	// Error logs an error-level message with optional key-value pairs.
	Error(msg string, args ...any)
}

// printfLoggerWrapper wraps a printf-style logger to implement StructuredLogger.
type printfLoggerWrapper struct {
	logger Logger
}

// WrapPrintfLogger wraps a printf-style Logger (like *log.Logger) to implement
// StructuredLogger. All messages are logged at the same level with formatted
// key-value pairs appended.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithStructuredLogger(langfuse.WrapPrintfLogger(log.Default())),
//	)
func WrapPrintfLogger(l Logger) StructuredLogger {
	return &printfLoggerWrapper{logger: l}
}

// WrapStdLogger wraps a standard library *log.Logger to implement StructuredLogger.
// This is a convenience function equivalent to WrapPrintfLogger(l).
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithStructuredLogger(langfuse.WrapStdLogger(log.Default())),
//	)
func WrapStdLogger(l *log.Logger) StructuredLogger {
	return &printfLoggerWrapper{logger: &defaultLogger{logger: l}}
}

func (w *printfLoggerWrapper) Debug(msg string, args ...any) {
	w.logger.Printf("[DEBUG] " + msg + formatArgs(args))
}

func (w *printfLoggerWrapper) Info(msg string, args ...any) {
	w.logger.Printf("[INFO] " + msg + formatArgs(args))
}

func (w *printfLoggerWrapper) Warn(msg string, args ...any) {
	w.logger.Printf("[WARN] " + msg + formatArgs(args))
}

func (w *printfLoggerWrapper) Error(msg string, args ...any) {
	w.logger.Printf("[ERROR] " + msg + formatArgs(args))
}

// Ensure printfLoggerWrapper implements StructuredLogger.
var _ StructuredLogger = (*printfLoggerWrapper)(nil)

// Metrics is an optional interface for SDK telemetry.
type Metrics interface {
	// IncrementCounter increments a counter metric.
	IncrementCounter(name string, value int64)
	// RecordDuration records a duration metric.
	RecordDuration(name string, duration time.Duration)
	// SetGauge sets a gauge metric.
	SetGauge(name string, value float64)
}

// defaultLogger wraps the standard library logger.
type defaultLogger struct {
	logger *log.Logger
}

func (l *defaultLogger) Printf(format string, v ...any) {
	l.logger.Printf(format, v...)
}

// formatArgs formats structured logging arguments as a string.
func formatArgs(args []any) string {
	if len(args) == 0 {
		return ""
	}
	result := " |"
	for i := 0; i < len(args)-1; i += 2 {
		key := args[i]
		var value any
		if i+1 < len(args) {
			value = args[i+1]
		}
		result += fmt.Sprintf(" %v=%v", key, value)
	}
	return result
}

// NopLogger is a logger that discards all log messages.
// Use this to disable logging entirely.
type NopLogger struct{}

// Printf implements Logger.Printf.
func (NopLogger) Printf(format string, v ...any) {}

// Debug implements StructuredLogger.Debug.
func (NopLogger) Debug(msg string, args ...any) {}

// Info implements StructuredLogger.Info.
func (NopLogger) Info(msg string, args ...any) {}

// Warn implements StructuredLogger.Warn.
func (NopLogger) Warn(msg string, args ...any) {}

// Error implements StructuredLogger.Error.
func (NopLogger) Error(msg string, args ...any) {}

// Ensure NopLogger implements both interfaces.
var (
	_ Logger           = NopLogger{}
	_ StructuredLogger = NopLogger{}
)

// MaskCredential masks a credential string for safe logging.
// It preserves the prefix and shows only the last 4 characters.
// For short strings, it returns a fully masked version.
//
// Examples:
//
//	MaskCredential("pk-lf-1234567890abcdef") => "pk-lf-************cdef"
//	MaskCredential("sk-lf-abcd1234efgh5678") => "sk-lf-************5678"
//	MaskCredential("short") => "****t"
func MaskCredential(s string) string {
	if s == "" {
		return ""
	}

	const (
		visibleSuffix = 4
		minMaskLength = 8
	)

	// For very short strings, mask most of it
	if len(s) <= visibleSuffix {
		return "****"
	}

	// Find prefix (up to first hyphen after the type prefix)
	// e.g., "pk-lf-" or "sk-lf-"
	prefixEnd := 0
	hyphenCount := 0
	for i, c := range s {
		if c == '-' {
			hyphenCount++
			if hyphenCount == 2 {
				prefixEnd = i + 1
				break
			}
		}
	}

	// If no valid prefix found, just mask the middle
	if prefixEnd == 0 {
		if len(s) <= minMaskLength {
			return "****" + s[len(s)-visibleSuffix:]
		}
		maskLen := len(s) - visibleSuffix
		return repeat('*', maskLen) + s[len(s)-visibleSuffix:]
	}

	// Mask everything between prefix and last 4 chars
	prefix := s[:prefixEnd]
	suffix := s[len(s)-visibleSuffix:]
	maskLen := len(s) - prefixEnd - visibleSuffix
	if maskLen < 0 {
		maskLen = 0
	}

	return prefix + repeat('*', maskLen) + suffix
}

// repeat creates a string of the given rune repeated n times.
func repeat(r rune, n int) string {
	if n <= 0 {
		return ""
	}
	result := make([]rune, n)
	for i := range result {
		result[i] = r
	}
	return string(result)
}

// MaskAuthHeader masks a Basic auth header for safe logging.
// It replaces the base64 encoded credentials with asterisks.
func MaskAuthHeader(header string) string {
	if len(header) > 6 && header[:6] == "Basic " {
		return "Basic ********"
	}
	if len(header) > 7 && header[:7] == "Bearer " {
		return "Bearer ********"
	}
	return "********"
}

// ============================================================================
// Slog Adapter
// ============================================================================

// SlogAdapter adapts a slog.Logger to the StructuredLogger interface.
// This allows seamless integration with Go 1.21+'s structured logging.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithStructuredLogger(langfuse.NewSlogAdapter(slog.Default())),
//	)
//
//	// Or with a custom logger:
//	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithStructuredLogger(langfuse.NewSlogAdapter(logger)),
//	)
type SlogAdapter struct {
	logger *slog.Logger
}

// NewSlogAdapter creates a new SlogAdapter wrapping the given slog.Logger.
// If logger is nil, slog.Default() is used.
func NewSlogAdapter(logger *slog.Logger) *SlogAdapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &SlogAdapter{logger: logger}
}

// Debug implements StructuredLogger.Debug.
func (a *SlogAdapter) Debug(msg string, args ...any) {
	a.logger.Debug(msg, args...)
}

// Info implements StructuredLogger.Info.
func (a *SlogAdapter) Info(msg string, args ...any) {
	a.logger.Info(msg, args...)
}

// Warn implements StructuredLogger.Warn.
func (a *SlogAdapter) Warn(msg string, args ...any) {
	a.logger.Warn(msg, args...)
}

// Error implements StructuredLogger.Error.
func (a *SlogAdapter) Error(msg string, args ...any) {
	a.logger.Error(msg, args...)
}

// Printf implements Logger.Printf for backward compatibility.
// Logs at Info level.
func (a *SlogAdapter) Printf(format string, v ...any) {
	a.logger.Info(format, v...)
}

// WithContext returns a new SlogAdapter that uses a logger with the given context.
// This is useful for propagating trace context through logs.
func (a *SlogAdapter) WithContext(ctx context.Context) *SlogAdapter {
	return &SlogAdapter{
		logger: a.logger,
	}
}

// WithGroup returns a new SlogAdapter with a log group prefix.
func (a *SlogAdapter) WithGroup(name string) *SlogAdapter {
	return &SlogAdapter{
		logger: a.logger.WithGroup(name),
	}
}

// With returns a new SlogAdapter with the given attributes added.
func (a *SlogAdapter) With(args ...any) *SlogAdapter {
	return &SlogAdapter{
		logger: a.logger.With(args...),
	}
}
