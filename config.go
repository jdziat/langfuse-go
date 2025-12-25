// Package langfuse provides a Go SDK for the Langfuse observability platform.
package langfuse

import (
	"net/http"
	"time"
)

// Region represents a Langfuse cloud region.
type Region string

const (
	// RegionEU is the European cloud region.
	RegionEU Region = "eu"
	// RegionUS is the US cloud region.
	RegionUS Region = "us"
	// RegionHIPAA is the HIPAA-compliant US region.
	RegionHIPAA Region = "hipaa"
)

// regionBaseURLs maps regions to their base URLs.
var regionBaseURLs = map[Region]string{
	RegionEU:    "https://cloud.langfuse.com/api/public",
	RegionUS:    "https://us.cloud.langfuse.com/api/public",
	RegionHIPAA: "https://hipaa.cloud.langfuse.com/api/public",
}

// Config holds the configuration for the Langfuse client.
type Config struct {
	// PublicKey is the Langfuse public key (required).
	PublicKey string

	// SecretKey is the Langfuse secret key (required).
	SecretKey string

	// BaseURL is the base URL for the Langfuse API.
	// If not set, it will be derived from the Region.
	BaseURL string

	// Region is the Langfuse cloud region.
	// Defaults to RegionEU if not set and BaseURL is empty.
	Region Region

	// HTTPClient is the HTTP client to use for requests.
	// If not set, a default client with sensible timeouts will be used.
	HTTPClient *http.Client

	// Timeout is the request timeout.
	// Defaults to 30 seconds if not set.
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts for failed requests.
	// Defaults to 3 if not set.
	MaxRetries int

	// RetryDelay is the initial delay between retry attempts.
	// Defaults to 1 second if not set.
	RetryDelay time.Duration

	// BatchSize is the maximum number of events to send in a single batch.
	// Defaults to 100 if not set.
	BatchSize int

	// FlushInterval is the interval at which to flush pending events.
	// Defaults to 5 seconds if not set.
	FlushInterval time.Duration

	// Debug enables debug logging.
	Debug bool
}

// ConfigOption is a function that modifies a Config.
type ConfigOption func(*Config)

// WithRegion sets the Langfuse cloud region.
func WithRegion(region Region) ConfigOption {
	return func(c *Config) {
		c.Region = region
	}
}

// WithBaseURL sets a custom base URL for the Langfuse API.
func WithBaseURL(baseURL string) ConfigOption {
	return func(c *Config) {
		c.BaseURL = baseURL
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ConfigOption {
	return func(c *Config) {
		c.HTTPClient = client
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) ConfigOption {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// WithMaxRetries sets the maximum number of retry attempts.
func WithMaxRetries(maxRetries int) ConfigOption {
	return func(c *Config) {
		c.MaxRetries = maxRetries
	}
}

// WithRetryDelay sets the initial delay between retry attempts.
func WithRetryDelay(delay time.Duration) ConfigOption {
	return func(c *Config) {
		c.RetryDelay = delay
	}
}

// WithBatchSize sets the maximum batch size for ingestion.
func WithBatchSize(size int) ConfigOption {
	return func(c *Config) {
		c.BatchSize = size
	}
}

// WithFlushInterval sets the flush interval for batched events.
func WithFlushInterval(interval time.Duration) ConfigOption {
	return func(c *Config) {
		c.FlushInterval = interval
	}
}

// WithDebug enables debug logging.
func WithDebug(debug bool) ConfigOption {
	return func(c *Config) {
		c.Debug = debug
	}
}

// applyDefaults sets default values for unset configuration options.
func (c *Config) applyDefaults() {
	if c.BaseURL == "" {
		if c.Region == "" {
			c.Region = RegionEU
		}
		if url, ok := regionBaseURLs[c.Region]; ok {
			c.BaseURL = url
		}
	}

	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}

	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}

	if c.RetryDelay == 0 {
		c.RetryDelay = 1 * time.Second
	}

	if c.BatchSize == 0 {
		c.BatchSize = 100
	}

	if c.FlushInterval == 0 {
		c.FlushInterval = 5 * time.Second
	}

	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{
			Timeout: c.Timeout,
		}
	}
}

// validate checks that the configuration is valid.
func (c *Config) validate() error {
	if c.PublicKey == "" {
		return ErrMissingPublicKey
	}
	if c.SecretKey == "" {
		return ErrMissingSecretKey
	}
	if c.BaseURL == "" {
		return ErrMissingBaseURL
	}
	return nil
}
