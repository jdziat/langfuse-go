package ingestion

import (
	"crypto/rand"
	"fmt"
	"time"
)

// UUID generates a random UUID v4.
func UUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("langfuse: failed to generate UUID: %w", err)
	}

	// Set version (4) and variant bits
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// GenerateID generates a random UUID-like ID.
// This is a fallback-safe version that uses timestamp if crypto fails.
func GenerateID() string {
	id, err := UUID()
	if err != nil {
		// Fallback to timestamp-based ID if crypto fails
		return fmt.Sprintf("%d-%x", time.Now().UnixNano(), time.Now().Unix())
	}
	return id
}

// IsValidUUID checks if a string is a valid UUID format.
// It accepts both standard UUID format (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
// and compact format without hyphens (32 hex characters).
func IsValidUUID(s string) bool {
	// Standard UUID format: 8-4-4-4-12 = 36 characters
	if len(s) == 36 {
		return isValidStandardUUID(s)
	}
	// Compact format: 32 hex characters without hyphens
	if len(s) == 32 {
		return isHexString(s)
	}
	return false
}

// isValidStandardUUID checks if a string is a valid standard UUID format.
func isValidStandardUUID(s string) bool {
	// Check hyphen positions: 8, 13, 18, 23
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	// Check hex segments
	return isHexString(s[0:8]) &&
		isHexString(s[9:13]) &&
		isHexString(s[14:18]) &&
		isHexString(s[19:23]) &&
		isHexString(s[24:36])
}

// isHexString checks if a string contains only hexadecimal characters.
func isHexString(s string) bool {
	for _, c := range s {
		isDigit := c >= '0' && c <= '9'
		isLowerHex := c >= 'a' && c <= 'f'
		isUpperHex := c >= 'A' && c <= 'F'
		if !isDigit && !isLowerHex && !isUpperHex {
			return false
		}
	}
	return true
}
