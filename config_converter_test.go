package langfuse

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	pkgclient "github.com/jdziat/langfuse-go/pkg/client"
	pkghttp "github.com/jdziat/langfuse-go/pkg/http"
)

// fullyPopulatedRootConfig returns a root Config where every field that the
// converter is expected to forward is set to a non-zero value. It is the input
// for the drift guard: after conversion, each mapped pkgclient.Config field
// must be observably non-zero.
func fullyPopulatedRootConfig() *Config {
	return &Config{
		PublicKey:           "pk-lf-drift-guard-key",
		SecretKey:           "sk-lf-drift-guard-key",
		BaseURL:             "https://example.invalid",
		APIPathPrefix:       "/api/public",
		Region:              RegionUS,
		HTTPClient:          &http.Client{},
		Timeout:             7 * time.Second,
		MaxRetries:          4,
		RetryDelay:          2 * time.Second,
		BatchSize:           42,
		FlushInterval:       3 * time.Second,
		Debug:               true,
		ErrorHandler:        func(error) {},
		Logger:              NopLogger{},
		StructuredLogger:    NopLogger{},
		Metrics:             &noopMetrics{},
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 25,
		IdleConnTimeout:     30 * time.Second,
		ShutdownTimeout:     15 * time.Second,
		BatchQueueSize:      99,
		RetryStrategy:       NewExponentialBackoff(),
		CircuitBreaker:      func() *CircuitBreakerConfig { cb := DefaultCircuitBreakerConfig(); return &cb }(),
		OnBatchFlushed:      func(BatchResult) {},
		HTTPHooks:           []HTTPHook{HeaderHook(map[string]string{"X-Test": "1"})},
		IdleWarningDuration: 5 * time.Minute,
		IDGenerationMode:    IDModeStrict,
		ClassifiedHooks: []ClassifiedHook{
			NewClassifiedHook("drift", HeaderHook(map[string]string{"X-Drift": "1"}), HookPriorityObservational),
		},
		BackpressureConfig: &BackpressureHandlerConfig{},
		OnBackpressure:     func(QueueState) {},
		// Both queue-full booleans are set so the drift guard can observe that the
		// converter forwards each one. They are normally mutually exclusive, but this
		// fixture is passed directly to convertToPkgClientConfig (which does not run
		// validation), so the exclusivity rule does not apply here.
		BlockOnQueueFull:     true,
		DropOnQueueFull:      true,
		MaxBackgroundSenders: 8,
	}
}

// noopMetrics is a minimal Metrics implementation used so the Metrics field is
// observably non-zero after conversion.
type noopMetrics struct{}

func (noopMetrics) IncrementCounter(string, int64)       {}
func (noopMetrics) RecordDuration(string, time.Duration) {}
func (noopMetrics) SetGauge(string, float64)             {}

var _ Metrics = (*noopMetrics)(nil)

// TestConvertToPkgClientConfig_ClassifiedHooks asserts that ClassifiedHooks set
// on the root Config reach the pkgclient.Config via the converter. This test
// fails if the ClassifiedHooks wiring is removed from convertToPkgClientConfig.
func TestConvertToPkgClientConfig_ClassifiedHooks(t *testing.T) {
	root := &Config{
		PublicKey: "pk-lf-test-key",
		SecretKey: "sk-lf-test-key",
		ClassifiedHooks: []ClassifiedHook{
			NewClassifiedHook("auth", HeaderHook(map[string]string{"X-Auth": "1"}), HookPriorityCritical),
			NewClassifiedHook("log", HeaderHook(map[string]string{"X-Log": "1"}), HookPriorityObservational),
		},
	}

	pkgCfg := convertToPkgClientConfig(root)

	if len(pkgCfg.ClassifiedHooks) != len(root.ClassifiedHooks) {
		t.Fatalf("ClassifiedHooks length = %d, want %d", len(pkgCfg.ClassifiedHooks), len(root.ClassifiedHooks))
	}
	if pkgCfg.ClassifiedHooks[0].Name != "auth" {
		t.Errorf("ClassifiedHooks[0].Name = %q, want %q", pkgCfg.ClassifiedHooks[0].Name, "auth")
	}
	if pkgCfg.ClassifiedHooks[0].Priority != HookPriorityCritical {
		t.Errorf("ClassifiedHooks[0].Priority = %v, want %v", pkgCfg.ClassifiedHooks[0].Priority, HookPriorityCritical)
	}
	if pkgCfg.ClassifiedHooks[1].Name != "log" {
		t.Errorf("ClassifiedHooks[1].Name = %q, want %q", pkgCfg.ClassifiedHooks[1].Name, "log")
	}
}

// driftGuardAllowList enumerates pkgclient.Config fields that convertToPkgClientConfig
// intentionally does NOT populate from a fully-populated root Config. Each entry MUST
// document why the field is unmapped. The guard test below fails if a field is neither
// populated by the converter nor present here, so a newly added pkgclient.Config field
// cannot be silently forgotten.
//
// Currently empty: every pkgclient.Config field has a direct root Config counterpart
// (or, for Version, is sourced from the package-level Version var) and is forwarded by
// the converter. If a future field is genuinely intentionally-unmapped, add it here with
// a reason comment.
var driftGuardAllowList = map[string]string{
	// (intentionally empty - see doc comment above)
}

// TestConvertToPkgClientConfig_DriftGuard enumerates every pkgclient.Config field via
// reflection and fails if any field is neither populated by convertToPkgClientConfig
// (detected as non-zero after converting a fully-populated root Config) nor on the
// documented allow-list. This ensures a newly added pkgclient.Config field cannot be
// silently dropped by the converter without a deliberate decision.
func TestConvertToPkgClientConfig_DriftGuard(t *testing.T) {
	root := fullyPopulatedRootConfig()
	pkgCfg := convertToPkgClientConfig(root)

	v := reflect.ValueOf(*pkgCfg)
	typ := v.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			// Unexported fields cannot be set by the converter from another package.
			continue
		}

		fieldValue := v.Field(i)
		if !fieldValue.IsZero() {
			// Field was populated by the converter - good.
			if reason, ok := driftGuardAllowList[field.Name]; ok {
				t.Errorf("field %q is on the allow-list (%q) but was populated by the converter; remove it from driftGuardAllowList", field.Name, reason)
			}
			continue
		}

		// Field is zero after conversion. It must be explicitly allow-listed.
		if _, ok := driftGuardAllowList[field.Name]; !ok {
			t.Errorf("pkgclient.Config field %q is not populated by convertToPkgClientConfig and is not on the drift-guard allow-list; either wire it in convertToPkgClientConfig or add it to driftGuardAllowList with a documented reason", field.Name)
		}
	}
}

// TestConvertToPkgClientConfig_Version asserts the converter forwards the package
// Version into pkgclient.Config.Version (which has no direct root Config field).
func TestConvertToPkgClientConfig_Version(t *testing.T) {
	root := &Config{
		PublicKey: "pk-lf-test-key",
		SecretKey: "sk-lf-test-key",
	}
	pkgCfg := convertToPkgClientConfig(root)
	if pkgCfg.Version != Version {
		t.Errorf("Version = %q, want %q", pkgCfg.Version, Version)
	}
}

// Compile-time assertions documenting the type relationships the converter relies on:
// root ClassifiedHook, pkgclient.ClassifiedHook, and pkghttp.ClassifiedHook are all the
// same underlying type, so []ClassifiedHook is directly assignable across packages.
var (
	_ = func(h []ClassifiedHook) []pkgclient.ClassifiedHook { return h }
	_ = func(h []ClassifiedHook) []pkghttp.ClassifiedHook { return h }
)
