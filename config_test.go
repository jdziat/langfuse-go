package langfuse

import (
	"net/http"
	"testing"
	"time"
)

func TestConfigApplyDefaults(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected Config
	}{
		{
			name:   "empty config gets defaults",
			config: Config{},
			expected: Config{
				Region:        RegionEU,
				BaseURL:       "https://cloud.langfuse.com/api/public",
				Timeout:       30 * time.Second,
				MaxRetries:    3,
				RetryDelay:    1 * time.Second,
				BatchSize:     100,
				FlushInterval: 5 * time.Second,
			},
		},
		{
			name: "US region sets correct URL",
			config: Config{
				Region: RegionUS,
			},
			expected: Config{
				Region:        RegionUS,
				BaseURL:       "https://us.cloud.langfuse.com/api/public",
				Timeout:       30 * time.Second,
				MaxRetries:    3,
				RetryDelay:    1 * time.Second,
				BatchSize:     100,
				FlushInterval: 5 * time.Second,
			},
		},
		{
			name: "HIPAA region sets correct URL",
			config: Config{
				Region: RegionHIPAA,
			},
			expected: Config{
				Region:        RegionHIPAA,
				BaseURL:       "https://hipaa.cloud.langfuse.com/api/public",
				Timeout:       30 * time.Second,
				MaxRetries:    3,
				RetryDelay:    1 * time.Second,
				BatchSize:     100,
				FlushInterval: 5 * time.Second,
			},
		},
		{
			name: "custom BaseURL overrides region",
			config: Config{
				BaseURL: "https://custom.example.com/api",
				Region:  RegionUS,
			},
			expected: Config{
				Region:        RegionUS,
				BaseURL:       "https://custom.example.com/api",
				Timeout:       30 * time.Second,
				MaxRetries:    3,
				RetryDelay:    1 * time.Second,
				BatchSize:     100,
				FlushInterval: 5 * time.Second,
			},
		},
		{
			name: "custom values are preserved",
			config: Config{
				Timeout:       60 * time.Second,
				MaxRetries:    5,
				RetryDelay:    2 * time.Second,
				BatchSize:     50,
				FlushInterval: 10 * time.Second,
			},
			expected: Config{
				Region:        RegionEU,
				BaseURL:       "https://cloud.langfuse.com/api/public",
				Timeout:       60 * time.Second,
				MaxRetries:    5,
				RetryDelay:    2 * time.Second,
				BatchSize:     50,
				FlushInterval: 10 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.config
			cfg.applyDefaults()

			if cfg.Region != tt.expected.Region {
				t.Errorf("Region = %v, want %v", cfg.Region, tt.expected.Region)
			}
			if cfg.BaseURL != tt.expected.BaseURL {
				t.Errorf("BaseURL = %v, want %v", cfg.BaseURL, tt.expected.BaseURL)
			}
			if cfg.Timeout != tt.expected.Timeout {
				t.Errorf("Timeout = %v, want %v", cfg.Timeout, tt.expected.Timeout)
			}
			if cfg.MaxRetries != tt.expected.MaxRetries {
				t.Errorf("MaxRetries = %v, want %v", cfg.MaxRetries, tt.expected.MaxRetries)
			}
			if cfg.RetryDelay != tt.expected.RetryDelay {
				t.Errorf("RetryDelay = %v, want %v", cfg.RetryDelay, tt.expected.RetryDelay)
			}
			if cfg.BatchSize != tt.expected.BatchSize {
				t.Errorf("BatchSize = %v, want %v", cfg.BatchSize, tt.expected.BatchSize)
			}
			if cfg.FlushInterval != tt.expected.FlushInterval {
				t.Errorf("FlushInterval = %v, want %v", cfg.FlushInterval, tt.expected.FlushInterval)
			}
			if cfg.HTTPClient == nil {
				t.Error("HTTPClient should not be nil after applyDefaults")
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name         string
		config       Config
		applyDefault bool // whether to call applyDefaults() before validate()
		wantError    error
	}{
		{
			name: "valid config with defaults",
			config: Config{
				PublicKey: "pk-lf-test-key",
				SecretKey: "sk-lf-test-key",
				BaseURL:   "https://api.example.com",
			},
			applyDefault: true,
			wantError:    nil,
		},
		{
			name: "missing public key",
			config: Config{
				SecretKey: "sk-lf-test-key",
				BaseURL:   "https://api.example.com",
			},
			applyDefault: true,
			wantError:    ErrMissingPublicKey,
		},
		{
			name: "missing secret key",
			config: Config{
				PublicKey: "pk-lf-test-key",
				BaseURL:   "https://api.example.com",
			},
			applyDefault: true,
			wantError:    ErrMissingSecretKey,
		},
		{
			name: "missing base URL before defaults",
			config: Config{
				PublicKey: "pk-lf-test-key",
				SecretKey: "sk-lf-test-key",
			},
			applyDefault: false, // BaseURL not set, defaults not applied
			wantError:    ErrMissingBaseURL,
		},
		{
			name: "base URL set by defaults from region",
			config: Config{
				PublicKey: "pk-lf-test-key",
				SecretKey: "sk-lf-test-key",
				Region:    RegionUS,
			},
			applyDefault: true, // defaults will set BaseURL from Region
			wantError:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.config
			if tt.applyDefault {
				cfg.applyDefaults()
			}
			err := cfg.validate()
			if err != tt.wantError {
				t.Errorf("validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestConfigOptions(t *testing.T) {
	cfg := &Config{}

	WithRegion(RegionUS)(cfg)
	if cfg.Region != RegionUS {
		t.Errorf("WithRegion failed: got %v, want %v", cfg.Region, RegionUS)
	}

	WithBaseURL("https://custom.example.com")(cfg)
	if cfg.BaseURL != "https://custom.example.com" {
		t.Errorf("WithBaseURL failed: got %v, want %v", cfg.BaseURL, "https://custom.example.com")
	}

	customClient := &http.Client{}
	WithHTTPClient(customClient)(cfg)
	if cfg.HTTPClient != customClient {
		t.Error("WithHTTPClient failed: client not set")
	}

	WithTimeout(60 * time.Second)(cfg)
	if cfg.Timeout != 60*time.Second {
		t.Errorf("WithTimeout failed: got %v, want %v", cfg.Timeout, 60*time.Second)
	}

	WithMaxRetries(5)(cfg)
	if cfg.MaxRetries != 5 {
		t.Errorf("WithMaxRetries failed: got %v, want %v", cfg.MaxRetries, 5)
	}

	WithRetryDelay(2 * time.Second)(cfg)
	if cfg.RetryDelay != 2*time.Second {
		t.Errorf("WithRetryDelay failed: got %v, want %v", cfg.RetryDelay, 2*time.Second)
	}

	WithBatchSize(50)(cfg)
	if cfg.BatchSize != 50 {
		t.Errorf("WithBatchSize failed: got %v, want %v", cfg.BatchSize, 50)
	}

	WithFlushInterval(10 * time.Second)(cfg)
	if cfg.FlushInterval != 10*time.Second {
		t.Errorf("WithFlushInterval failed: got %v, want %v", cfg.FlushInterval, 10*time.Second)
	}

	WithDebug(true)(cfg)
	if !cfg.Debug {
		t.Error("WithDebug failed: debug not enabled")
	}
}

func TestRegionBaseURLs(t *testing.T) {
	tests := []struct {
		region      Region
		expectedURL string
	}{
		{RegionEU, "https://cloud.langfuse.com/api/public"},
		{RegionUS, "https://us.cloud.langfuse.com/api/public"},
		{RegionHIPAA, "https://hipaa.cloud.langfuse.com/api/public"},
	}

	for _, tt := range tests {
		t.Run(string(tt.region), func(t *testing.T) {
			url, ok := regionBaseURLs[tt.region]
			if !ok {
				t.Errorf("Region %v not found in regionBaseURLs", tt.region)
				return
			}
			if url != tt.expectedURL {
				t.Errorf("Region %v URL = %v, want %v", tt.region, url, tt.expectedURL)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("pk-lf-test-key", "sk-lf-test-key")

	if cfg.PublicKey != "pk-lf-test-key" {
		t.Errorf("PublicKey = %v, want pk-lf-test-key", cfg.PublicKey)
	}
	if cfg.SecretKey != "sk-lf-test-key" {
		t.Errorf("SecretKey = %v, want sk-lf-test-key", cfg.SecretKey)
	}
	if cfg.Region != RegionEU {
		t.Errorf("Region = %v, want %v", cfg.Region, RegionEU)
	}

	// Apply defaults and validate
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		t.Errorf("DefaultConfig should produce valid config: %v", err)
	}
}

func TestDevelopmentConfig(t *testing.T) {
	cfg := DevelopmentConfig("pk-lf-test-key", "sk-lf-test-key")

	if cfg.PublicKey != "pk-lf-test-key" {
		t.Errorf("PublicKey = %v, want pk-lf-test-key", cfg.PublicKey)
	}
	if cfg.SecretKey != "sk-lf-test-key" {
		t.Errorf("SecretKey = %v, want sk-lf-test-key", cfg.SecretKey)
	}
	if cfg.Region != RegionEU {
		t.Errorf("Region = %v, want %v", cfg.Region, RegionEU)
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

	// Apply defaults and validate
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		t.Errorf("DevelopmentConfig should produce valid config: %v", err)
	}
}

func TestHighThroughputConfig(t *testing.T) {
	cfg := HighThroughputConfig("pk-lf-test-key", "sk-lf-test-key")

	if cfg.PublicKey != "pk-lf-test-key" {
		t.Errorf("PublicKey = %v, want pk-lf-test-key", cfg.PublicKey)
	}
	if cfg.SecretKey != "sk-lf-test-key" {
		t.Errorf("SecretKey = %v, want sk-lf-test-key", cfg.SecretKey)
	}
	if cfg.Region != RegionEU {
		t.Errorf("Region = %v, want %v", cfg.Region, RegionEU)
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

	// Apply defaults and validate
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		t.Errorf("HighThroughputConfig should produce valid config: %v", err)
	}
}

func TestCircuitBreakerConfigOptions(t *testing.T) {
	t.Run("WithCircuitBreaker", func(t *testing.T) {
		cfg := &Config{}
		cbConfig := CircuitBreakerConfig{
			FailureThreshold: 10,
			Timeout:          60 * time.Second,
		}

		WithCircuitBreaker(cbConfig)(cfg)

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
		cfg := &Config{}

		WithDefaultCircuitBreaker()(cfg)

		if cfg.CircuitBreaker == nil {
			t.Error("CircuitBreaker should not be nil")
		}
		// Check that defaults are applied
		defaultCfg := DefaultCircuitBreakerConfig()
		if cfg.CircuitBreaker.FailureThreshold != defaultCfg.FailureThreshold {
			t.Errorf("FailureThreshold = %v, want %v", cfg.CircuitBreaker.FailureThreshold, defaultCfg.FailureThreshold)
		}
	})
}

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
			got := MaskCredential(tt.input)
			if got != tt.expected {
				t.Errorf("MaskCredential(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

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
			got := MaskAuthHeader(tt.input)
			if got != tt.expected {
				t.Errorf("MaskAuthHeader(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestConfigString(t *testing.T) {
	cfg := &Config{
		PublicKey:     "pk-lf-1234567890abcdef",
		SecretKey:     "sk-lf-secretkey1234567",
		BaseURL:       "https://cloud.langfuse.com/api/public",
		Region:        RegionEU,
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
	}

	str := cfg.String()

	// Verify credentials are masked
	if contains(str, "1234567890abcdef") {
		t.Error("Config.String() should not contain full public key")
	}
	if contains(str, "secretkey1234567") {
		t.Error("Config.String() should not contain full secret key")
	}

	// Verify the masked versions are present
	if !contains(str, "pk-lf-") {
		t.Error("Config.String() should contain masked public key prefix")
	}
	if !contains(str, "sk-lf-") {
		t.Error("Config.String() should contain masked secret key prefix")
	}

	// Verify non-sensitive data is present
	if !contains(str, "https://cloud.langfuse.com/api/public") {
		t.Error("Config.String() should contain BaseURL")
	}
	if !contains(str, "eu") {
		t.Error("Config.String() should contain Region")
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
