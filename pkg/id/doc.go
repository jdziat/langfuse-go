// Package id provides unique ID generation for the Langfuse SDK.
//
// The package supports multiple generation strategies:
//   - Crypto-based UUID v4 (default, using crypto/rand)
//   - Fallback IDs (timestamp + counter + process ID when crypto fails)
//
// ID Generation Modes:
//   - IDModeFallback: Uses fallback when crypto/rand fails (default)
//   - IDModeStrict: Returns error when crypto/rand fails
//
// Example usage:
//
//	// Use default generator (fallback mode)
//	defaultID, err := id.GenerateID()
//
//	// Create strict mode generator
//	gen := id.NewIDGenerator(&id.IDGeneratorConfig{
//	    Mode: id.IDModeStrict,
//	})
//	strictID, err := gen.Generate()
//
//	// Check if an ID was generated using fallback
//	if id.IsFallbackID(defaultID) {
//	    log.Println("ID was generated using fallback method")
//	}
package id
