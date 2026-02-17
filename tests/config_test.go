package langfuse_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	langfuse "github.com/jdziat/langfuse-go"
)

// TestConfigOptions tests the public configuration option functions.
func TestConfigOptions(t *testing.T) {
	cfg := &langfuse.Config{}

	langfuse.WithRegion(langfuse.RegionUS)(cfg)
	if cfg.Region != langfuse.RegionUS {
		t.Errorf("WithRegion failed: got %v, want %v", cfg.Region, langfuse.RegionUS)
	}

	langfuse.WithBaseURL("https://custom.example.com")(cfg)
	if cfg.BaseURL != "https://custom.example.com" {
		t.Errorf("WithBaseURL failed: got %v, want %v", cfg.BaseURL, "https://custom.example.com")
	}

	customClient := &http.Client{}
	langfuse.WithHTTPClient(customClient)(cfg)
	if cfg.HTTPClient != customClient {
		t.Error("WithHTTPClient failed: client not set")
	}

	langfuse.WithTimeout(60 * time.Second)(cfg)
	if cfg.Timeout != 60*time.Second {
		t.Errorf("WithTimeout failed: got %v, want %v", cfg.Timeout, 60*time.Second)
	}

	langfuse.WithMaxRetries(5)(cfg)
	if cfg.MaxRetries != 5 {
		t.Errorf("WithMaxRetries failed: got %v, want %v", cfg.MaxRetries, 5)
	}

	langfuse.WithRetryDelay(2 * time.Second)(cfg)
	if cfg.RetryDelay != 2*time.Second {
		t.Errorf("WithRetryDelay failed: got %v, want %v", cfg.RetryDelay, 2*time.Second)
	}

	langfuse.WithBatchSize(50)(cfg)
	if cfg.BatchSize != 50 {
		t.Errorf("WithBatchSize failed: got %v, want %v", cfg.BatchSize, 50)
	}

	langfuse.WithFlushInterval(10 * time.Second)(cfg)
	if cfg.FlushInterval != 10*time.Second {
		t.Errorf("WithFlushInterval failed: got %v, want %v", cfg.FlushInterval, 10*time.Second)
	}

	langfuse.WithDebug(true)(cfg)
	if !cfg.Debug {
		t.Error("WithDebug failed: debug not enabled")
	}
}

// TestDefaultConfig tests the DefaultConfig constructor.
func TestDefaultConfig(t *testing.T) {
	cfg := langfuse.DefaultConfig("pk-lf-test-key", "sk-lf-test-key")

	if cfg.PublicKey != "pk-lf-test-key" {
		t.Errorf("PublicKey = %v, want pk-lf-test-key", cfg.PublicKey)
	}
	if cfg.SecretKey != "sk-lf-test-key" {
		t.Errorf("SecretKey = %v, want sk-lf-test-key", cfg.SecretKey)
	}
	if cfg.Region != langfuse.RegionEU {
		t.Errorf("Region = %v, want %v", cfg.Region, langfuse.RegionEU)
	}
}

// TestDevelopmentConfig tests the DevelopmentConfig constructor.
func TestDevelopmentConfig(t *testing.T) {
	cfg := langfuse.DevelopmentConfig("pk-lf-test-key", "sk-lf-test-key")

	if cfg.PublicKey != "pk-lf-test-key" {
		t.Errorf("PublicKey = %v, want pk-lf-test-key", cfg.PublicKey)
	}
	if cfg.SecretKey != "sk-lf-test-key" {
		t.Errorf("SecretKey = %v, want sk-lf-test-key", cfg.SecretKey)
	}
	if cfg.Region != langfuse.RegionEU {
		t.Errorf("Region = %v, want %v", cfg.Region, langfuse.RegionEU)
	}
	if !cfg.Debug {
		t.Error("Debug should be enabled in DevelopmentConfig")
	}
	if cfg.BatchSize != 1 {
		t.Errorf("BatchSize = %v, want 1", cfg.BatchSize)
	}
	if cfg.FlushInterval != 1*time.Second {
		t.Errorf("FlushInterval = %v, want 1s", cfg.FlushInterval)
	}
}

// TestHighThroughputConfig tests the HighThroughputConfig constructor.
func TestHighThroughputConfig(t *testing.T) {
	cfg := langfuse.HighThroughputConfig("pk-lf-test-key", "sk-lf-test-key")

	if cfg.PublicKey != "pk-lf-test-key" {
		t.Errorf("PublicKey = %v, want pk-lf-test-key", cfg.PublicKey)
	}
	if cfg.SecretKey != "sk-lf-test-key" {
		t.Errorf("SecretKey = %v, want sk-lf-test-key", cfg.SecretKey)
	}
	if cfg.Region != langfuse.RegionEU {
		t.Errorf("Region = %v, want %v", cfg.Region, langfuse.RegionEU)
	}
	if cfg.BatchSize != 500 {
		t.Errorf("BatchSize = %v, want 500", cfg.BatchSize)
	}
	if cfg.BatchQueueSize != 500 {
		t.Errorf("BatchQueueSize = %v, want 500", cfg.BatchQueueSize)
	}
	if cfg.FlushInterval != 10*time.Second {
		t.Errorf("FlushInterval = %v, want 10s", cfg.FlushInterval)
	}
	if cfg.MaxIdleConns != 200 {
		t.Errorf("MaxIdleConns = %v, want 200", cfg.MaxIdleConns)
	}
	if cfg.MaxIdleConnsPerHost != 50 {
		t.Errorf("MaxIdleConnsPerHost = %v, want 50", cfg.MaxIdleConnsPerHost)
	}
}

// TestCircuitBreakerConfigOptions tests circuit breaker configuration options.
func TestCircuitBreakerConfigOptions(t *testing.T) {
	t.Run("WithCircuitBreaker", func(t *testing.T) {
		cfg := &langfuse.Config{}
		cbConfig := langfuse.CircuitBreakerConfig{
			FailureThreshold: 10,
			Timeout:          60 * time.Second,
		}

		langfuse.WithCircuitBreaker(cbConfig)(cfg)

		if cfg.CircuitBreaker == nil {
			t.Error("CircuitBreaker should not be nil")
		}
		if cfg.CircuitBreaker.FailureThreshold != 10 {
			t.Errorf("FailureThreshold = %v, want 10", cfg.CircuitBreaker.FailureThreshold)
		}
		if cfg.CircuitBreaker.Timeout != 60*time.Second {
			t.Errorf("Timeout = %v, want 60s", cfg.CircuitBreaker.Timeout)
		}
	})

	t.Run("WithDefaultCircuitBreaker", func(t *testing.T) {
		cfg := &langfuse.Config{}

		langfuse.WithDefaultCircuitBreaker()(cfg)

		if cfg.CircuitBreaker == nil {
			t.Error("CircuitBreaker should not be nil")
		}
		// Check that defaults are applied
		defaultCfg := langfuse.DefaultCircuitBreakerConfig()
		if cfg.CircuitBreaker.FailureThreshold != defaultCfg.FailureThreshold {
			t.Errorf("FailureThreshold = %v, want %v", cfg.CircuitBreaker.FailureThreshold, defaultCfg.FailureThreshold)
		}
	})
}

// TestMaskCredential tests the credential masking function.
func TestMaskCredential(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "very short string",
			input:    "abc",
			expected: "****",
		},
		{
			name:     "short string",
			input:    "abcde",
			expected: "****bcde",
		},
		{
			name:     "standard public key",
			input:    "pk-lf-1234567890abcdef",
			expected: "pk-lf-************cdef",
		},
		{
			name:     "standard secret key",
			input:    "sk-lf-abcd1234efgh5678",
			expected: "sk-lf-************5678",
		},
		{
			name:     "key without standard prefix",
			input:    "1234567890abcdef",
			expected: "************cdef",
		},
		{
			name:     "single hyphen prefix",
			input:    "pk-1234567890",
			expected: "*********7890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := langfuse.MaskCredential(tt.input)
			if got != tt.expected {
				t.Errorf("MaskCredential(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestMaskAuthHeader tests the auth header masking function.
func TestMaskAuthHeader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic auth header",
			input:    "Basic cHVibGljOnNlY3JldA==",
			expected: "Basic ********",
		},
		{
			name:     "bearer token",
			input:    "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: "Bearer ********",
		},
		{
			name:     "unknown format",
			input:    "Custom abc123",
			expected: "********",
		},
		{
			name:     "short string",
			input:    "abc",
			expected: "********",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := langfuse.MaskAuthHeader(tt.input)
			if got != tt.expected {
				t.Errorf("MaskAuthHeader(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestConfigString tests the Config.String() method for safe credential masking.
func TestConfigString(t *testing.T) {
	cfg := &langfuse.Config{
		PublicKey:     "pk-lf-1234567890abcdef",
		SecretKey:     "sk-lf-secretkey1234567",
		BaseURL:       "https://cloud.langfuse.com/api/public",
		Region:        langfuse.RegionEU,
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
	}

	str := cfg.String()

	// Verify credentials are masked
	if strings.Contains(str, "1234567890abcdef") {
		t.Error("Config.String() should not contain full public key")
	}
	if strings.Contains(str, "secretkey1234567") {
		t.Error("Config.String() should not contain full secret key")
	}

	// Verify the masked versions are present
	if !strings.Contains(str, "pk-lf-") {
		t.Error("Config.String() should contain masked public key prefix")
	}
	if !strings.Contains(str, "sk-lf-") {
		t.Error("Config.String() should contain masked secret key prefix")
	}

	// Verify non-sensitive data is present
	if !strings.Contains(str, "https://cloud.langfuse.com/api/public") {
		t.Error("Config.String() should contain BaseURL")
	}
	if !strings.Contains(str, "eu") {
		t.Error("Config.String() should contain Region")
	}
}

// TestRegionConstants verifies region constants are accessible.
func TestRegionConstants(t *testing.T) {
	tests := []struct {
		region langfuse.Region
		name   string
	}{
		{langfuse.RegionEU, "eu"},
		{langfuse.RegionUS, "us"},
		{langfuse.RegionHIPAA, "hipaa"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.region) != tt.name {
				t.Errorf("Region %v = %q, want %q", tt.region, string(tt.region), tt.name)
			}
		})
	}
}

// TestBackpressureOptionsMutualExclusivity tests that BlockOnQueueFull and
// DropOnQueueFull cannot both be set to true.
func TestBackpressureOptionsMutualExclusivity(t *testing.T) {
	tests := []struct {
		name             string
		blockOnQueueFull bool
		dropOnQueueFull  bool
		wantErr          bool
	}{
		{
			name:             "both false - valid",
			blockOnQueueFull: false,
			dropOnQueueFull:  false,
			wantErr:          false,
		},
		{
			name:             "block only - valid",
			blockOnQueueFull: true,
			dropOnQueueFull:  false,
			wantErr:          false,
		},
		{
			name:             "drop only - valid",
			blockOnQueueFull: false,
			dropOnQueueFull:  true,
			wantErr:          false,
		},
		{
			name:             "both true - invalid",
			blockOnQueueFull: true,
			dropOnQueueFull:  true,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := langfuse.New(
				"pk-lf-testpublickey123",
				"sk-lf-testsecretkey123",
				langfuse.WithBlockOnQueueFull(tt.blockOnQueueFull),
				langfuse.WithDropOnQueueFull(tt.dropOnQueueFull),
			)

			if tt.wantErr && err == nil {
				t.Error("expected error for mutually exclusive options, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), "mutually exclusive") {
					t.Errorf("error should mention mutual exclusivity, got: %v", err)
				}
			}

			// Clean up client if it was created
			if client != nil {
				client.Close(context.Background())
			}
		})
	}
}

// TestWithMaxBackgroundSenders tests the MaxBackgroundSenders config option.
func TestWithMaxBackgroundSenders(t *testing.T) {
	cfg := &langfuse.Config{}

	langfuse.WithMaxBackgroundSenders(20)(cfg)

	if cfg.MaxBackgroundSenders != 20 {
		t.Errorf("MaxBackgroundSenders = %v, want 20", cfg.MaxBackgroundSenders)
	}
}

// TestMaxBackgroundSendersDefault tests that MaxBackgroundSenders has a default value.
func TestMaxBackgroundSendersDefault(t *testing.T) {
	client, err := langfuse.New(
		"pk-lf-testpublickey123",
		"sk-lf-testsecretkey123",
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close(context.Background())

	// The default is 10, but we can't directly access the internal config
	// This test just ensures the client creates successfully with defaults
}

// TestMaxBackgroundSendersNegativeValidation tests that negative values are rejected.
func TestMaxBackgroundSendersNegativeValidation(t *testing.T) {
	_, err := langfuse.New(
		"pk-lf-testpublickey123",
		"sk-lf-testsecretkey123",
		langfuse.WithMaxBackgroundSenders(-1),
	)
	if err == nil {
		t.Error("expected error for negative MaxBackgroundSenders, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "cannot be negative") {
		t.Errorf("error should mention negative value, got: %v", err)
	}
}

// TestErrBatchDroppedExported tests that ErrBatchDropped is properly exported.
func TestErrBatchDroppedExported(t *testing.T) {
	// Verify the error is accessible and has expected properties
	err := langfuse.ErrBatchDropped
	if err == nil {
		t.Fatal("ErrBatchDropped should not be nil")
	}
	if !strings.Contains(err.Error(), "batch dropped") {
		t.Errorf("ErrBatchDropped.Error() = %q, want substring 'batch dropped'", err.Error())
	}
}
