package builders

import (
	"maps"

	"github.com/jdziat/langfuse-go/pkg/types"
)

// ModelParameters is a type alias for Metadata used for model parameters.
type ModelParameters = types.Metadata

// ModelParametersBuilder provides a type-safe way to build model parameters.
//
// Example:
//
//	params := NewModelParameters().
//	    Temperature(0.7).
//	    MaxTokens(150).
//	    TopP(0.9).
//	    Build()
//
//	gen.ModelParameters(params).Create(ctx)
type ModelParametersBuilder struct {
	params Metadata
}

// NewModelParameters creates a new ModelParametersBuilder.
func NewModelParameters() *ModelParametersBuilder {
	return &ModelParametersBuilder{params: make(Metadata)}
}

// Temperature sets the temperature parameter.
func (m *ModelParametersBuilder) Temperature(temp float64) *ModelParametersBuilder {
	m.params["temperature"] = temp
	return m
}

// MaxTokens sets the max_tokens parameter.
func (m *ModelParametersBuilder) MaxTokens(tokens int) *ModelParametersBuilder {
	m.params["max_tokens"] = tokens
	return m
}

// TopP sets the top_p parameter.
func (m *ModelParametersBuilder) TopP(p float64) *ModelParametersBuilder {
	m.params["top_p"] = p
	return m
}

// TopK sets the top_k parameter.
func (m *ModelParametersBuilder) TopK(k int) *ModelParametersBuilder {
	m.params["top_k"] = k
	return m
}

// FrequencyPenalty sets the frequency_penalty parameter.
func (m *ModelParametersBuilder) FrequencyPenalty(penalty float64) *ModelParametersBuilder {
	m.params["frequency_penalty"] = penalty
	return m
}

// PresencePenalty sets the presence_penalty parameter.
func (m *ModelParametersBuilder) PresencePenalty(penalty float64) *ModelParametersBuilder {
	m.params["presence_penalty"] = penalty
	return m
}

// Stop sets the stop sequences.
func (m *ModelParametersBuilder) Stop(sequences ...string) *ModelParametersBuilder {
	m.params["stop"] = sequences
	return m
}

// Seed sets the seed for deterministic outputs.
func (m *ModelParametersBuilder) Seed(seed int) *ModelParametersBuilder {
	m.params["seed"] = seed
	return m
}

// ResponseFormat sets the response format.
func (m *ModelParametersBuilder) ResponseFormat(format string) *ModelParametersBuilder {
	m.params["response_format"] = map[string]string{"type": format}
	return m
}

// Set sets an arbitrary parameter.
func (m *ModelParametersBuilder) Set(key string, value any) *ModelParametersBuilder {
	m.params[key] = value
	return m
}

// Merge merges another parameters map.
func (m *ModelParametersBuilder) Merge(other Metadata) *ModelParametersBuilder {
	maps.Copy(m.params, other)
	return m
}

// Build returns the constructed parameters map.
func (m *ModelParametersBuilder) Build() Metadata {
	return m.params
}
