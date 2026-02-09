package config

import (
	"os"
	"testing"
	"time"
)

// TestRegionBaseURL tests that Region.BaseURL returns correct URLs.
func TestRegionBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		region   Region
		expected string
	}{
		{
			name:     "EU region",
			region:   RegionEU,
			expected: "https://cloud.langfuse.com/api/public",
		},
		{
			name:     "US region",
			region:   RegionUS,
			expected: "https://us.cloud.langfuse.com/api/public",
		},
		{
			name:     "HIPAA region",
			region:   RegionHIPAA,
			expected: "https://hipaa.cloud.langfuse.com/api/public",
		},
		{
			name:     "Unknown region defaults to EU",
			region:   Region("unknown"),
			expected: "https://cloud.langfuse.com/api/public",
		},
		{
			name:     "Empty region defaults to EU",
			region:   Region(""),
			expected: "https://cloud.langfuse.com/api/public",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.region.BaseURL()
			if got != tt.expected {
				t.Errorf("Region.BaseURL() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestRegionString tests that Region.String returns correct strings.
func TestRegionString(t *testing.T) {
	tests := []struct {
		name     string
		region   Region
		expected string
	}{
		{
			name:     "EU region",
			region:   RegionEU,
			expected: "eu",
		},
		{
			name:     "US region",
			region:   RegionUS,
			expected: "us",
		},
		{
			name:     "HIPAA region",
			region:   RegionHIPAA,
			expected: "hipaa",
		},
		{
			name:     "Custom region",
			region:   Region("custom"),
			expected: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.region.String()
			if got != tt.expected {
				t.Errorf("Region.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestDefaultConstants verifies default configuration values.
func TestDefaultConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      any
		expected any
	}{
		{"DefaultTimeout", DefaultTimeout, 30 * time.Second},
		{"DefaultMaxRetries", DefaultMaxRetries, 3},
		{"DefaultRetryDelay", DefaultRetryDelay, 1 * time.Second},
		{"DefaultBatchSize", DefaultBatchSize, 100},
		{"DefaultFlushInterval", DefaultFlushInterval, 5 * time.Second},
		{"DefaultMaxIdleConns", DefaultMaxIdleConns, 100},
		{"DefaultMaxIdleConnsPerHost", DefaultMaxIdleConnsPerHost, 10},
		{"DefaultIdleConnTimeout", DefaultIdleConnTimeout, 90 * time.Second},
		{"DefaultShutdownTimeout", DefaultShutdownTimeout, 35 * time.Second},
		{"DefaultBatchQueueSize", DefaultBatchQueueSize, 100},
		{"DefaultBackgroundSendTimeout", DefaultBackgroundSendTimeout, 30 * time.Second},
		{"MaxBatchSize", MaxBatchSize, 10000},
		{"MaxMaxRetries", MaxMaxRetries, 100},
		{"MaxTimeout", MaxTimeout, 10 * time.Minute},
		{"MinFlushInterval", MinFlushInterval, 100 * time.Millisecond},
		{"MinShutdownTimeout", MinShutdownTimeout, 1 * time.Second},
		{"MinKeyLength", MinKeyLength, 8},
		{"PublicKeyPrefix", PublicKeyPrefix, "pk-"},
		{"SecretKeyPrefix", SecretKeyPrefix, "sk-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

// TestGetEnvString tests the GetEnvString helper function.
func TestGetEnvString(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		setEnv       bool
		expected     string
	}{
		{
			name:         "Returns env value when set",
			key:          "TEST_STRING",
			defaultValue: "default",
			envValue:     "from_env",
			setEnv:       true,
			expected:     "from_env",
		},
		{
			name:         "Returns default when env not set",
			key:          "TEST_STRING_UNSET",
			defaultValue: "default",
			envValue:     "",
			setEnv:       false,
			expected:     "default",
		},
		{
			name:         "Returns env value when empty string is default",
			key:          "TEST_STRING_EMPTY",
			defaultValue: "",
			envValue:     "from_env",
			setEnv:       true,
			expected:     "from_env",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before and after
			os.Unsetenv(tt.key)
			defer os.Unsetenv(tt.key)

			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
			}

			got := GetEnvString(tt.key, tt.defaultValue)
			if got != tt.expected {
				t.Errorf("GetEnvString() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestGetEnvBool tests the GetEnvBool helper function.
func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		envValue string
		setEnv   bool
		expected bool
	}{
		{
			name:     "Returns true for 'true'",
			key:      "TEST_BOOL_TRUE",
			envValue: "true",
			setEnv:   true,
			expected: true,
		},
		{
			name:     "Returns true for '1'",
			key:      "TEST_BOOL_ONE",
			envValue: "1",
			setEnv:   true,
			expected: true,
		},
		{
			name:     "Returns false for 'false'",
			key:      "TEST_BOOL_FALSE",
			envValue: "false",
			setEnv:   true,
			expected: false,
		},
		{
			name:     "Returns false for '0'",
			key:      "TEST_BOOL_ZERO",
			envValue: "0",
			setEnv:   true,
			expected: false,
		},
		{
			name:     "Returns false when not set",
			key:      "TEST_BOOL_UNSET",
			envValue: "",
			setEnv:   false,
			expected: false,
		},
		{
			name:     "Returns false for arbitrary string",
			key:      "TEST_BOOL_RANDOM",
			envValue: "yes",
			setEnv:   true,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before and after
			os.Unsetenv(tt.key)
			defer os.Unsetenv(tt.key)

			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
			}

			got := GetEnvBool(tt.key)
			if got != tt.expected {
				t.Errorf("GetEnvBool() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestGetEnvRegion tests the GetEnvRegion helper function.
func TestGetEnvRegion(t *testing.T) {
	tests := []struct {
		name          string
		envValue      string
		setEnv        bool
		defaultRegion Region
		expected      Region
	}{
		{
			name:          "Returns EU from env",
			envValue:      "eu",
			setEnv:        true,
			defaultRegion: RegionUS,
			expected:      RegionEU,
		},
		{
			name:          "Returns US from env",
			envValue:      "us",
			setEnv:        true,
			defaultRegion: RegionEU,
			expected:      RegionUS,
		},
		{
			name:          "Returns HIPAA from env",
			envValue:      "hipaa",
			setEnv:        true,
			defaultRegion: RegionEU,
			expected:      RegionHIPAA,
		},
		{
			name:          "Returns default when not set",
			envValue:      "",
			setEnv:        false,
			defaultRegion: RegionUS,
			expected:      RegionUS,
		},
		{
			name:          "Returns custom region from env",
			envValue:      "custom",
			setEnv:        true,
			defaultRegion: RegionEU,
			expected:      Region("custom"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before and after
			os.Unsetenv(EnvRegion)
			defer os.Unsetenv(EnvRegion)

			if tt.setEnv {
				os.Setenv(EnvRegion, tt.envValue)
			}

			got := GetEnvRegion(tt.defaultRegion)
			if got != tt.expected {
				t.Errorf("GetEnvRegion() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestRegionBaseURLsMap verifies the RegionBaseURLs map contains expected entries.
func TestRegionBaseURLsMap(t *testing.T) {
	if len(RegionBaseURLs) != 3 {
		t.Errorf("RegionBaseURLs should have 3 entries, got %d", len(RegionBaseURLs))
	}

	expectedURLs := map[Region]string{
		RegionEU:    "https://cloud.langfuse.com/api/public",
		RegionUS:    "https://us.cloud.langfuse.com/api/public",
		RegionHIPAA: "https://hipaa.cloud.langfuse.com/api/public",
	}

	for region, expectedURL := range expectedURLs {
		if url, ok := RegionBaseURLs[region]; !ok {
			t.Errorf("RegionBaseURLs missing entry for %s", region)
		} else if url != expectedURL {
			t.Errorf("RegionBaseURLs[%s] = %s, want %s", region, url, expectedURL)
		}
	}
}

// TestEnvConstants verifies environment variable constant values.
func TestEnvConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"EnvPublicKey", EnvPublicKey, "LANGFUSE_PUBLIC_KEY"},
		{"EnvSecretKey", EnvSecretKey, "LANGFUSE_SECRET_KEY"},
		{"EnvBaseURL", EnvBaseURL, "LANGFUSE_BASE_URL"},
		{"EnvHost", EnvHost, "LANGFUSE_HOST"},
		{"EnvRegion", EnvRegion, "LANGFUSE_REGION"},
		{"EnvDebug", EnvDebug, "LANGFUSE_DEBUG"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}
