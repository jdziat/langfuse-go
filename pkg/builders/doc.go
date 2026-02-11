// Package builders provides fluent builder patterns and validation for Langfuse data types.
//
// This package contains:
//
// # Data Builders
//
//   - [MetadataBuilder] for constructing type-safe metadata
//   - [TagsBuilder] for constructing tag slices
//   - [UsageBuilder] for constructing token usage stats
//   - [ModelParametersBuilder] for constructing model parameters
//
// # Validation
//
//   - [Validator] for accumulating validation errors in builders
//   - Validation functions: ValidateID, ValidateName, ValidateMetadata, etc.
//   - Constants: MaxNameLength, MaxTagLength, MaxTagCount
//
// # Result Types
//
//   - [BuildResult] generic type for wrapping builder results with errors
//
// # Interfaces
//
//   - [EventQueuer] interface for event submission (used by builders)
//
// These types are re-exported from the root langfuse package for convenience.
//
// Example usage:
//
//	// Build metadata with typed values
//	metadata := builders.BuildMetadata().
//	    String("user_id", "123").
//	    Int("request_count", 5).
//	    Bool("is_premium", true).
//	    Build()
//
//	// Build tags
//	tags := builders.NewTags().
//	    Add("production", "api").
//	    Environment("prod").
//	    Build()
//
//	// Validate input
//	if err := builders.ValidateID("traceId", id); err != nil {
//	    return err
//	}
package builders
