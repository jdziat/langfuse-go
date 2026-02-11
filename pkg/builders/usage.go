package builders

import (
	"github.com/jdziat/langfuse-go/pkg/types"
)

// Usage is a type alias for pkg/types.Usage.
type Usage = types.Usage

// UsageBuilder provides a type-safe way to build token usage.
//
// Example:
//
//	usage := NewUsage().
//	    Input(100).
//	    Output(50).
//	    InputCost(0.001).
//	    OutputCost(0.002).
//	    Build()
//
//	gen.Usage(usage).Create(ctx)
type UsageBuilder struct {
	usage Usage
}

// NewUsage creates a new UsageBuilder.
func NewUsage() *UsageBuilder {
	return &UsageBuilder{}
}

// Input sets the input token count.
func (u *UsageBuilder) Input(tokens int) *UsageBuilder {
	u.usage.Input = tokens
	u.usage.Total = u.usage.Input + u.usage.Output
	return u
}

// Output sets the output token count.
func (u *UsageBuilder) Output(tokens int) *UsageBuilder {
	u.usage.Output = tokens
	u.usage.Total = u.usage.Input + u.usage.Output
	return u
}

// Total sets the total token count explicitly.
// If not set, it's calculated as Input + Output.
func (u *UsageBuilder) Total(tokens int) *UsageBuilder {
	u.usage.Total = tokens
	return u
}

// Unit sets the usage unit (e.g., "TOKENS", "CHARACTERS").
func (u *UsageBuilder) Unit(unit string) *UsageBuilder {
	u.usage.Unit = unit
	return u
}

// InputCost sets the input cost.
func (u *UsageBuilder) InputCost(cost float64) *UsageBuilder {
	u.usage.InputCost = cost
	u.usage.TotalCost = u.usage.InputCost + u.usage.OutputCost
	return u
}

// OutputCost sets the output cost.
func (u *UsageBuilder) OutputCost(cost float64) *UsageBuilder {
	u.usage.OutputCost = cost
	u.usage.TotalCost = u.usage.InputCost + u.usage.OutputCost
	return u
}

// TotalCost sets the total cost explicitly.
func (u *UsageBuilder) TotalCost(cost float64) *UsageBuilder {
	u.usage.TotalCost = cost
	return u
}

// Build returns the constructed Usage.
func (u *UsageBuilder) Build() *Usage {
	return &u.usage
}

// Tokens is a convenience method to set both input and output tokens.
func (u *UsageBuilder) Tokens(input, output int) *UsageBuilder {
	u.usage.Input = input
	u.usage.Output = output
	u.usage.Total = input + output
	return u
}
