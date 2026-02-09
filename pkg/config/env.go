package config

import "os"

// Environment variable names for configuration.
const (
	EnvPublicKey = "LANGFUSE_PUBLIC_KEY"
	EnvSecretKey = "LANGFUSE_SECRET_KEY"
	EnvBaseURL   = "LANGFUSE_BASE_URL"
	EnvHost      = "LANGFUSE_HOST"
	EnvRegion    = "LANGFUSE_REGION"
	EnvDebug     = "LANGFUSE_DEBUG"
)

// GetEnvString returns the value of an environment variable or a default.
func GetEnvString(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// GetEnvBool returns true if the env var is "true" or "1".
func GetEnvBool(key string) bool {
	v := os.Getenv(key)
	return v == "true" || v == "1"
}

// GetEnvRegion returns the region from environment or default.
func GetEnvRegion(defaultRegion Region) Region {
	if v := os.Getenv(EnvRegion); v != "" {
		return Region(v)
	}
	return defaultRegion
}
