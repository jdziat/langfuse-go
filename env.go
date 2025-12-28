package langfuse

import (
	"fmt"
	"os"
)

// Environment variable names for configuration.
const (
	// EnvPublicKey is the environment variable for the Langfuse public key.
	EnvPublicKey = "LANGFUSE_PUBLIC_KEY"
	// EnvSecretKey is the environment variable for the Langfuse secret key.
	EnvSecretKey = "LANGFUSE_SECRET_KEY"
	// EnvBaseURL is the environment variable for the Langfuse API base URL.
	EnvBaseURL = "LANGFUSE_BASE_URL"
	// EnvHost is an alias for EnvBaseURL (for compatibility).
	EnvHost = "LANGFUSE_HOST"
	// EnvRegion is the environment variable for the Langfuse cloud region.
	EnvRegion = "LANGFUSE_REGION"
	// EnvDebug is the environment variable to enable debug mode.
	EnvDebug = "LANGFUSE_DEBUG"
)

// NewFromEnv creates a new client using environment variables for configuration.
// It reads LANGFUSE_PUBLIC_KEY, LANGFUSE_SECRET_KEY, and optionally
// LANGFUSE_BASE_URL (or LANGFUSE_HOST), LANGFUSE_REGION, and LANGFUSE_DEBUG.
//
// Example:
//
//	client, err := langfuse.NewFromEnv()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Shutdown(context.Background())
func NewFromEnv(opts ...ConfigOption) (*Client, error) {
	publicKey := os.Getenv(EnvPublicKey)
	secretKey := os.Getenv(EnvSecretKey)

	if publicKey == "" {
		return nil, fmt.Errorf("langfuse: %s environment variable is required", EnvPublicKey)
	}
	if secretKey == "" {
		return nil, fmt.Errorf("langfuse: %s environment variable is required", EnvSecretKey)
	}

	// Prepend env var options so explicit options can override them
	envOpts := make([]ConfigOption, 0, 4)

	// Check for base URL (LANGFUSE_BASE_URL or LANGFUSE_HOST)
	if baseURL := os.Getenv(EnvBaseURL); baseURL != "" {
		envOpts = append(envOpts, WithBaseURL(baseURL))
	} else if host := os.Getenv(EnvHost); host != "" {
		envOpts = append(envOpts, WithBaseURL(host))
	}

	// Check for region
	if region := os.Getenv(EnvRegion); region != "" {
		envOpts = append(envOpts, WithRegion(Region(region)))
	}

	// Check for debug mode
	if debug := os.Getenv(EnvDebug); debug == "true" || debug == "1" {
		envOpts = append(envOpts, WithDebug(true))
	}

	// Combine env options with explicit options (explicit options take precedence)
	allOpts := append(envOpts, opts...)

	return New(publicKey, secretKey, allOpts...)
}
