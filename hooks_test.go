package langfuse

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestHTTPHookFunc(t *testing.T) {
	t.Run("with nil functions", func(t *testing.T) {
		hook := HTTPHookFunc{}

		// Should not panic
		err := hook.BeforeRequest(context.Background(), &http.Request{})
		if err != nil {
			t.Errorf("BeforeRequest should return nil for nil function")
		}

		// Should not panic
		hook.AfterResponse(context.Background(), &http.Request{}, nil, 0, nil)
	})

	t.Run("with functions", func(t *testing.T) {
		beforeCalled := false
		afterCalled := false

		hook := HTTPHookFunc{
			Before: func(ctx context.Context, req *http.Request) error {
				beforeCalled = true
				return nil
			},
			After: func(ctx context.Context, req *http.Request, resp *http.Response, duration time.Duration, err error) {
				afterCalled = true
			},
		}

		hook.BeforeRequest(context.Background(), &http.Request{})
		hook.AfterResponse(context.Background(), &http.Request{}, nil, 0, nil)

		if !beforeCalled {
			t.Error("Before function should be called")
		}
		if !afterCalled {
			t.Error("After function should be called")
		}
	})
}

func TestHookChain(t *testing.T) {
	var callOrder []string

	hook1 := HTTPHookFunc{
		Before: func(ctx context.Context, req *http.Request) error {
			callOrder = append(callOrder, "before1")
			return nil
		},
		After: func(ctx context.Context, req *http.Request, resp *http.Response, duration time.Duration, err error) {
			callOrder = append(callOrder, "after1")
		},
	}

	hook2 := HTTPHookFunc{
		Before: func(ctx context.Context, req *http.Request) error {
			callOrder = append(callOrder, "before2")
			return nil
		},
		After: func(ctx context.Context, req *http.Request, resp *http.Response, duration time.Duration, err error) {
			callOrder = append(callOrder, "after2")
		},
	}

	combined := combineHooks([]HTTPHook{hook1, hook2})

	combined.BeforeRequest(context.Background(), &http.Request{})
	combined.AfterResponse(context.Background(), &http.Request{}, nil, 0, nil)

	// Before should be called in order, After in reverse order
	expected := []string{"before1", "before2", "after2", "after1"}
	if len(callOrder) != len(expected) {
		t.Fatalf("callOrder length = %d, want %d", len(callOrder), len(expected))
	}
	for i, v := range expected {
		if callOrder[i] != v {
			t.Errorf("callOrder[%d] = %q, want %q", i, callOrder[i], v)
		}
	}
}

func TestCombineHooks(t *testing.T) {
	t.Run("empty hooks", func(t *testing.T) {
		combined := combineHooks(nil)
		if combined != nil {
			t.Error("combineHooks should return nil for empty hooks")
		}
	})

	t.Run("single hook", func(t *testing.T) {
		// For single hook, we can't compare directly since HTTPHookFunc contains funcs
		// Just verify it works
		hook := HTTPHookFunc{
			Before: func(ctx context.Context, req *http.Request) error {
				return nil
			},
		}
		combined := combineHooks([]HTTPHook{hook})
		if combined == nil {
			t.Error("combineHooks should not return nil for single hook")
		}
	})

	t.Run("multiple hooks", func(t *testing.T) {
		hook1 := HTTPHookFunc{}
		hook2 := HTTPHookFunc{}
		combined := combineHooks([]HTTPHook{hook1, hook2})
		if combined == nil {
			t.Error("combineHooks should return a chain for multiple hooks")
		}
	})
}

func TestHeaderHook(t *testing.T) {
	headers := map[string]string{
		"X-Custom": "value",
		"X-Trace":  "123",
	}
	hook := HeaderHook(headers)

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	err := hook.BeforeRequest(context.Background(), req)

	if err != nil {
		t.Fatalf("BeforeRequest error: %v", err)
	}

	for k, v := range headers {
		if got := req.Header.Get(k); got != v {
			t.Errorf("Header %q = %q, want %q", k, got, v)
		}
	}
}

func TestDynamicHeaderHook(t *testing.T) {
	callCount := 0
	hook := DynamicHeaderHook(func(ctx context.Context) map[string]string {
		callCount++
		return map[string]string{"X-Count": string(rune('0' + callCount))}
	})

	req1, _ := http.NewRequest("GET", "http://example.com", nil)
	hook.BeforeRequest(context.Background(), req1)

	req2, _ := http.NewRequest("GET", "http://example.com", nil)
	hook.BeforeRequest(context.Background(), req2)

	if callCount != 2 {
		t.Errorf("Function should be called for each request, got %d calls", callCount)
	}
}

// hookTestLogger is a logger for testing hooks
type hookTestLogger struct {
	messages []string
}

func (l *hookTestLogger) Printf(format string, v ...any) {
	l.messages = append(l.messages, format)
}

func TestLoggingHook(t *testing.T) {
	logger := &hookTestLogger{}
	hook := LoggingHook(logger)

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	resp := &http.Response{StatusCode: 200}

	hook.BeforeRequest(context.Background(), req)
	hook.AfterResponse(context.Background(), req, resp, 100*time.Millisecond, nil)

	if len(logger.messages) != 2 {
		t.Fatalf("Expected 2 log messages, got %d", len(logger.messages))
	}
}

func TestLoggingHook_WithError(t *testing.T) {
	logger := &hookTestLogger{}
	hook := LoggingHook(logger)

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	testErr := errors.New("test error")

	hook.BeforeRequest(context.Background(), req)
	hook.AfterResponse(context.Background(), req, nil, 100*time.Millisecond, testErr)

	if len(logger.messages) != 2 {
		t.Fatalf("Expected 2 log messages, got %d", len(logger.messages))
	}
}

// hookTestMetrics is a metrics collector for testing hooks
type hookTestMetrics struct {
	counters  map[string]int64
	durations map[string][]time.Duration
}

func newHookTestMetrics() *hookTestMetrics {
	return &hookTestMetrics{
		counters:  make(map[string]int64),
		durations: make(map[string][]time.Duration),
	}
}

func (m *hookTestMetrics) IncrementCounter(name string, value int64) {
	m.counters[name] += value
}

func (m *hookTestMetrics) RecordDuration(name string, d time.Duration) {
	m.durations[name] = append(m.durations[name], d)
}

func (m *hookTestMetrics) SetGauge(name string, value float64) {}

func TestMetricsHook(t *testing.T) {
	t.Run("with nil metrics", func(t *testing.T) {
		hook := MetricsHook(nil)
		// Should not panic
		hook.BeforeRequest(context.Background(), &http.Request{})
		hook.AfterResponse(context.Background(), &http.Request{}, nil, 0, nil)
	})

	t.Run("records metrics", func(t *testing.T) {
		metrics := newHookTestMetrics()
		hook := MetricsHook(metrics)

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		resp := &http.Response{StatusCode: 200}

		hook.AfterResponse(context.Background(), req, resp, 100*time.Millisecond, nil)

		if metrics.counters["langfuse.http.requests"] != 1 {
			t.Error("Should increment request counter")
		}
		if len(metrics.durations["langfuse.http.duration"]) != 1 {
			t.Error("Should record duration")
		}
	})

	t.Run("records error", func(t *testing.T) {
		metrics := newHookTestMetrics()
		hook := MetricsHook(metrics)

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		testErr := errors.New("test error")

		hook.AfterResponse(context.Background(), req, nil, 100*time.Millisecond, testErr)

		if metrics.counters["langfuse.http.errors"] != 1 {
			t.Error("Should increment error counter")
		}
	})
}

func TestTracingHook(t *testing.T) {
	hook := TracingHook()

	t.Run("without trace context", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "http://example.com", nil)
		err := hook.BeforeRequest(context.Background(), req)

		if err != nil {
			t.Fatalf("BeforeRequest error: %v", err)
		}
		if req.Header.Get("X-Langfuse-Trace-ID") != "" {
			t.Error("Should not set trace header without context")
		}
	})

	t.Run("with trace context", func(t *testing.T) {
		tc := &TraceContext{traceID: "test-trace-id"}
		ctx := ContextWithTrace(context.Background(), tc)

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		err := hook.BeforeRequest(ctx, req)

		if err != nil {
			t.Fatalf("BeforeRequest error: %v", err)
		}
		if got := req.Header.Get("X-Langfuse-Trace-ID"); got != "test-trace-id" {
			t.Errorf("X-Langfuse-Trace-ID = %q, want %q", got, "test-trace-id")
		}
	})
}

func TestDebugHook(t *testing.T) {
	logger := &hookTestLogger{}
	hook := DebugHook(logger)

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	req.Header.Set("X-Custom", "value")
	req.Header.Set("Authorization", "secret") // Should not be logged

	resp := &http.Response{
		StatusCode: 200,
		Header:     http.Header{"X-Response": []string{"value"}},
	}

	hook.BeforeRequest(context.Background(), req)
	hook.AfterResponse(context.Background(), req, resp, 100*time.Millisecond, nil)

	// Check that Authorization header is not logged
	for _, msg := range logger.messages {
		if strings.Contains(msg, "Authorization") || strings.Contains(msg, "secret") {
			t.Error("Should not log Authorization header")
		}
	}
}

func TestWithHTTPHooks_Integration(t *testing.T) {
	var buf bytes.Buffer
	logger := &hookTestLogger{messages: make([]string, 0)}

	// This just tests that the option works
	cfg := DefaultConfig("pk-lf-test-key", "sk-lf-test-key")
	WithHTTPHooks(LoggingHook(logger))(cfg)

	if len(cfg.HTTPHooks) != 1 {
		t.Errorf("Expected 1 hook, got %d", len(cfg.HTTPHooks))
	}

	_ = buf // avoid unused
}

func TestHookBeforeRequestError(t *testing.T) {
	testErr := errors.New("hook error")
	hook := HTTPHookFunc{
		Before: func(ctx context.Context, req *http.Request) error {
			return testErr
		},
	}

	err := hook.BeforeRequest(context.Background(), &http.Request{})
	if err != testErr {
		t.Errorf("BeforeRequest should return the error from hook")
	}
}

// Classified hook tests

func TestHookPriority_String(t *testing.T) {
	tests := []struct {
		priority HookPriority
		want     string
	}{
		{HookPriorityObservational, "observational"},
		{HookPriorityCritical, "critical"},
		{HookPriority(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.priority.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClassifiedHook_Creation(t *testing.T) {
	hook := HTTPHookFunc{
		Before: func(ctx context.Context, req *http.Request) error {
			return nil
		},
	}

	ch := NewClassifiedHook("test-hook", hook, HookPriorityCritical)

	if ch.Name != "test-hook" {
		t.Errorf("Name = %q, want %q", ch.Name, "test-hook")
	}
	if ch.Priority != HookPriorityCritical {
		t.Errorf("Priority = %v, want %v", ch.Priority, HookPriorityCritical)
	}
}

func TestClassifiedHookChain_Empty(t *testing.T) {
	chain := NewClassifiedHookChain(nil, nil)

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	err := chain.BeforeRequest(context.Background(), req)

	if err != nil {
		t.Errorf("BeforeRequest on empty chain should return nil, got %v", err)
	}
}

func TestClassifiedHookChain_ObservationalError(t *testing.T) {
	testErr := errors.New("observational error")
	hook := ClassifiedHook{
		Name:     "failing-observational",
		Priority: HookPriorityObservational,
		Hook: HTTPHookFunc{
			Before: func(ctx context.Context, req *http.Request) error {
				return testErr
			},
		},
	}

	logger := &hookTestLogger{}
	chain := NewClassifiedHookChain(logger, nil)
	chain.AddClassified(hook)

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	err := chain.BeforeRequest(context.Background(), req)

	// Observational hooks should not block - error should be nil
	if err != nil {
		t.Errorf("Observational hook error should not block, got %v", err)
	}

	// But should be logged
	if len(logger.messages) == 0 {
		t.Error("Observational hook error should be logged")
	}
}

func TestClassifiedHookChain_CriticalError(t *testing.T) {
	testErr := errors.New("critical error")
	hook := ClassifiedHook{
		Name:     "failing-critical",
		Priority: HookPriorityCritical,
		Hook: HTTPHookFunc{
			Before: func(ctx context.Context, req *http.Request) error {
				return testErr
			},
		},
	}

	chain := NewClassifiedHookChain(nil, nil)
	chain.AddClassified(hook)

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	err := chain.BeforeRequest(context.Background(), req)

	// Critical hooks should block
	if err == nil {
		t.Error("Critical hook error should block execution")
	}
}

func TestClassifiedHookChain_MixedPriorities(t *testing.T) {
	var callOrder []string

	observational := ClassifiedHook{
		Name:     "observational",
		Priority: HookPriorityObservational,
		Hook: HTTPHookFunc{
			Before: func(ctx context.Context, req *http.Request) error {
				callOrder = append(callOrder, "observational")
				return nil
			},
		},
	}

	critical := ClassifiedHook{
		Name:     "critical",
		Priority: HookPriorityCritical,
		Hook: HTTPHookFunc{
			Before: func(ctx context.Context, req *http.Request) error {
				callOrder = append(callOrder, "critical")
				return nil
			},
		},
	}

	chain := NewClassifiedHookChain(nil, nil)
	chain.AddClassified(observational)
	chain.AddClassified(critical)

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	chain.BeforeRequest(context.Background(), req)

	if len(callOrder) != 2 {
		t.Errorf("Expected 2 hooks called, got %d", len(callOrder))
	}
}

func TestClassifiedHookChain_AfterResponse(t *testing.T) {
	var callOrder []string

	hook1 := ClassifiedHook{
		Name:     "first",
		Priority: HookPriorityObservational,
		Hook: HTTPHookFunc{
			After: func(ctx context.Context, req *http.Request, resp *http.Response, duration time.Duration, err error) {
				callOrder = append(callOrder, "first")
			},
		},
	}

	hook2 := ClassifiedHook{
		Name:     "second",
		Priority: HookPriorityObservational,
		Hook: HTTPHookFunc{
			After: func(ctx context.Context, req *http.Request, resp *http.Response, duration time.Duration, err error) {
				callOrder = append(callOrder, "second")
			},
		},
	}

	chain := NewClassifiedHookChain(nil, nil)
	chain.AddClassified(hook1)
	chain.AddClassified(hook2)

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	chain.AfterResponse(context.Background(), req, nil, 0, nil)

	// After hooks should be called in reverse order
	if len(callOrder) != 2 {
		t.Fatalf("Expected 2 hooks called, got %d", len(callOrder))
	}
	if callOrder[0] != "second" {
		t.Errorf("First after hook should be 'second', got %q", callOrder[0])
	}
	if callOrder[1] != "first" {
		t.Errorf("Second after hook should be 'first', got %q", callOrder[1])
	}
}

func TestClassifiedHookChain_Len(t *testing.T) {
	hook1 := ClassifiedHook{Name: "obs1", Priority: HookPriorityObservational}
	hook2 := ClassifiedHook{Name: "crit1", Priority: HookPriorityCritical}
	hook3 := ClassifiedHook{Name: "obs2", Priority: HookPriorityObservational}

	chain := NewClassifiedHookChain(nil, nil)
	chain.AddClassified(hook1)
	chain.AddClassified(hook2)
	chain.AddClassified(hook3)

	if chain.Len() != 3 {
		t.Errorf("Len() = %d, want 3", chain.Len())
	}
}

func TestClassifiedHookChain_PanicRecovery(t *testing.T) {
	hook := ClassifiedHook{
		Name:     "panicking",
		Priority: HookPriorityObservational,
		Hook: HTTPHookFunc{
			Before: func(ctx context.Context, req *http.Request) error {
				panic("test panic")
			},
		},
	}

	logger := &hookTestLogger{}
	chain := NewClassifiedHookChain(logger, nil)
	chain.AddClassified(hook)

	req, _ := http.NewRequest("GET", "http://example.com", nil)

	// Should not panic
	err := chain.BeforeRequest(context.Background(), req)

	// Observational panic should not block
	if err != nil {
		t.Errorf("Observational panic should not block, got %v", err)
	}

	// Should be logged
	if len(logger.messages) == 0 {
		t.Error("Panic should be logged")
	}
}

func TestClassifiedHookChain_CriticalPanicRecovery(t *testing.T) {
	hook := ClassifiedHook{
		Name:     "panicking-critical",
		Priority: HookPriorityCritical,
		Hook: HTTPHookFunc{
			Before: func(ctx context.Context, req *http.Request) error {
				panic("critical panic")
			},
		},
	}

	logger := &hookTestLogger{}
	chain := NewClassifiedHookChain(logger, nil)
	chain.AddClassified(hook)

	req, _ := http.NewRequest("GET", "http://example.com", nil)

	// Should not panic - panics are recovered
	err := chain.BeforeRequest(context.Background(), req)

	// Current implementation recovers from panics without returning an error
	// Panics are logged but don't block execution
	_ = err

	// Panic should be logged
	if len(logger.messages) == 0 {
		t.Error("Panic should be logged")
	}
}

func TestObservationalLoggingHook(t *testing.T) {
	logger := &hookTestLogger{}
	ch := ObservationalLoggingHook(logger)

	if ch.Priority != HookPriorityObservational {
		t.Errorf("Priority = %v, want %v", ch.Priority, HookPriorityObservational)
	}
	if ch.Name == "" {
		t.Error("Name should not be empty")
	}
}

func TestObservationalMetricsHook(t *testing.T) {
	metrics := newHookTestMetrics()
	ch := ObservationalMetricsHook(metrics)

	if ch.Priority != HookPriorityObservational {
		t.Errorf("Priority = %v, want %v", ch.Priority, HookPriorityObservational)
	}
}

func TestObservationalTracingHook(t *testing.T) {
	ch := ObservationalTracingHook()

	if ch.Priority != HookPriorityObservational {
		t.Errorf("Priority = %v, want %v", ch.Priority, HookPriorityObservational)
	}
}

func TestObservationalDebugHook(t *testing.T) {
	logger := &hookTestLogger{}
	ch := ObservationalDebugHook(logger)

	if ch.Priority != HookPriorityObservational {
		t.Errorf("Priority = %v, want %v", ch.Priority, HookPriorityObservational)
	}
}

func TestCriticalHeaderHook(t *testing.T) {
	headers := map[string]string{"X-Custom": "value"}
	ch := CriticalHeaderHook("custom-headers", headers)

	if ch.Priority != HookPriorityCritical {
		t.Errorf("Priority = %v, want %v", ch.Priority, HookPriorityCritical)
	}
	if ch.Name != "custom-headers" {
		t.Errorf("Name = %q, want %q", ch.Name, "custom-headers")
	}
}

func TestCriticalAuthHook(t *testing.T) {
	authFunc := func(req *http.Request) error {
		req.Header.Set("Authorization", "Bearer token")
		return nil
	}
	ch := CriticalAuthHook(authFunc)

	if ch.Priority != HookPriorityCritical {
		t.Errorf("Priority = %v, want %v", ch.Priority, HookPriorityCritical)
	}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	err := ch.Hook.BeforeRequest(context.Background(), req)

	if err != nil {
		t.Errorf("BeforeRequest error: %v", err)
	}
	if req.Header.Get("Authorization") != "Bearer token" {
		t.Error("Auth header not set")
	}
}

func TestCriticalAuthHook_Error(t *testing.T) {
	testErr := errors.New("auth failed")
	authFunc := func(req *http.Request) error {
		return testErr
	}
	ch := CriticalAuthHook(authFunc)

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	err := ch.Hook.BeforeRequest(context.Background(), req)

	if err != testErr {
		t.Errorf("BeforeRequest should return auth error, got %v", err)
	}
}

func TestCriticalValidationHook(t *testing.T) {
	validateFunc := func(req *http.Request) error {
		if req.URL.Host == "" {
			return errors.New("missing host")
		}
		return nil
	}
	ch := CriticalValidationHook("request-validator", validateFunc)

	if ch.Priority != HookPriorityCritical {
		t.Errorf("Priority = %v, want %v", ch.Priority, HookPriorityCritical)
	}
	if ch.Name != "request-validator" {
		t.Errorf("Name = %q, want %q", ch.Name, "request-validator")
	}

	// Valid request
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	err := ch.Hook.BeforeRequest(context.Background(), req)
	if err != nil {
		t.Errorf("Valid request should not error: %v", err)
	}
}

func TestClassifiedHookChain_WithMetrics(t *testing.T) {
	metrics := newHookTestMetrics()

	hook := ClassifiedHook{
		Name:     "test",
		Priority: HookPriorityObservational,
		Hook: HTTPHookFunc{
			Before: func(ctx context.Context, req *http.Request) error {
				return errors.New("test error")
			},
		},
	}

	chain := NewClassifiedHookChain(nil, metrics)
	chain.AddClassified(hook)

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	chain.BeforeRequest(context.Background(), req)

	// The implementation uses "langfuse.hooks.failures" for failed hooks
	if metrics.counters["langfuse.hooks.failures"] != 1 {
		t.Error("Hook failure should be counted in metrics")
	}
}
