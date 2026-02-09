package config

import (
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

// RegionBaseURLs maps regions to their base URLs.
var RegionBaseURLs = map[Region]string{
	RegionEU:    "https://cloud.langfuse.com/api/public",
	RegionUS:    "https://us.cloud.langfuse.com/api/public",
	RegionHIPAA: "https://hipaa.cloud.langfuse.com/api/public",
}

// BaseURL returns the API base URL for this region.
func (r Region) BaseURL() string {
	if url, ok := RegionBaseURLs[r]; ok {
		return url
	}
	return RegionBaseURLs[RegionEU]
}

// String returns the string representation of the region.
func (r Region) String() string {
	return string(r)
}

// Default configuration values.
const (
	// DefaultTimeout is the default request timeout.
	DefaultTimeout = 30 * time.Second

	// DefaultMaxRetries is the default maximum number of retry attempts.
	DefaultMaxRetries = 3

	// DefaultRetryDelay is the default initial delay between retry attempts.
	DefaultRetryDelay = 1 * time.Second

	// DefaultBatchSize is the default maximum number of events per batch.
	DefaultBatchSize = 100

	// DefaultFlushInterval is the default interval for flushing pending events.
	DefaultFlushInterval = 5 * time.Second

	// DefaultMaxIdleConns is the default maximum number of idle connections.
	DefaultMaxIdleConns = 100

	// DefaultMaxIdleConnsPerHost is the default maximum idle connections per host.
	DefaultMaxIdleConnsPerHost = 10

	// DefaultIdleConnTimeout is the default timeout for idle connections.
	DefaultIdleConnTimeout = 90 * time.Second

	// DefaultShutdownTimeout is the default graceful shutdown timeout.
	// Must be >= DefaultTimeout to allow pending requests to complete.
	DefaultShutdownTimeout = 35 * time.Second

	// DefaultBatchQueueSize is the default size of the background batch queue.
	DefaultBatchQueueSize = 100

	// DefaultBackgroundSendTimeout is the timeout for background batch sends.
	DefaultBackgroundSendTimeout = 30 * time.Second

	// MaxBatchSize is the maximum allowed batch size.
	MaxBatchSize = 10000

	// MaxMaxRetries is the maximum allowed retry count.
	MaxMaxRetries = 100

	// MaxTimeout is the maximum allowed request timeout.
	MaxTimeout = 10 * time.Minute

	// MinFlushInterval is the minimum allowed flush interval.
	MinFlushInterval = 100 * time.Millisecond

	// MinShutdownTimeout is the minimum allowed shutdown timeout.
	MinShutdownTimeout = 1 * time.Second

	// MinKeyLength is the minimum length for API keys.
	MinKeyLength = 8

	// PublicKeyPrefix is the expected prefix for public keys.
	PublicKeyPrefix = "pk-"

	// SecretKeyPrefix is the expected prefix for secret keys.
	SecretKeyPrefix = "sk-"
)
