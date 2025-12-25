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
		name      string
		config    Config
		wantError error
	}{
		{
			name: "valid config",
			config: Config{
				PublicKey: "pk-test",
				SecretKey: "sk-test",
				BaseURL:   "https://api.example.com",
			},
			wantError: nil,
		},
		{
			name: "missing public key",
			config: Config{
				SecretKey: "sk-test",
				BaseURL:   "https://api.example.com",
			},
			wantError: ErrMissingPublicKey,
		},
		{
			name: "missing secret key",
			config: Config{
				PublicKey: "pk-test",
				BaseURL:   "https://api.example.com",
			},
			wantError: ErrMissingSecretKey,
		},
		{
			name: "missing base URL",
			config: Config{
				PublicKey: "pk-test",
				SecretKey: "sk-test",
			},
			wantError: ErrMissingBaseURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
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
