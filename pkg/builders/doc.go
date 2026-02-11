// Package builders provides fluent builder patterns for constructing Langfuse data types.
//
// This package contains helper builders that are independent of the Client:
//   - [MetadataBuilder] for constructing type-safe metadata
//   - [TagsBuilder] for constructing tag slices
//   - [UsageBuilder] for constructing token usage stats
//   - [ModelParametersBuilder] for constructing model parameters
//
// These builders are re-exported from the root langfuse package for convenience.
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
//	// Build token usage
//	usage := builders.NewUsage().
//	    Input(100).
//	    Output(50).
//	    Build()
//
//	// Build model parameters
//	params := builders.NewModelParameters().
//	    Temperature(0.7).
//	    MaxTokens(150).
//	    Build()
package builders
