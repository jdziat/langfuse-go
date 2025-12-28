package evaluation

import (
	"testing"
)

func TestSummarizationTraceBuilder_ValidateInput(t *testing.T) {
	tests := []struct {
		name        string
		input       *SummarizationInput
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid summarization input",
			input: &SummarizationInput{
				Input: "This is a long article that needs to be summarized...",
			},
			expectError: false,
		},
		{
			name:        "missing input",
			input:       &SummarizationInput{},
			expectError: true,
			errorMsg:    "input",
		},
		{
			name: "empty input",
			input: &SummarizationInput{
				Input: "",
			},
			expectError: true,
			errorMsg:    "input",
		},
		{
			name: "valid with all options",
			input: &SummarizationInput{
				Input:       "Long article text",
				MaxLength:   100,
				Style:       "bullet_points",
				GroundTruth: "Expected summary",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &SummarizationOutput{Output: "test"}
			err := ValidateFor(tt.input, output, SummarizationEvaluator)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestSummarizationTraceBuilder_FluentAPI(t *testing.T) {
	builder := &SummarizationTraceBuilder{
		sumInput: &SummarizationInput{},
	}

	result := builder.
		Input("Long article about programming").
		GroundTruth("A concise summary about programming").
		MaxLength(50).
		Style("paragraph")

	if result != builder {
		t.Error("fluent methods should return the same builder")
	}

	if builder.sumInput.Input != "Long article about programming" {
		t.Errorf("Input not set correctly: got %s", builder.sumInput.Input)
	}

	if builder.sumInput.GroundTruth != "A concise summary about programming" {
		t.Errorf("GroundTruth not set correctly: got %s", builder.sumInput.GroundTruth)
	}

	if builder.sumInput.MaxLength != 50 {
		t.Errorf("MaxLength not set correctly: got %d, want 50", builder.sumInput.MaxLength)
	}

	if builder.sumInput.Style != "paragraph" {
		t.Errorf("Style not set correctly: got %s, want paragraph", builder.sumInput.Style)
	}
}

func TestSummarizationTraceContext_GetInput(t *testing.T) {
	input := &SummarizationInput{
		Input:       "Long article text",
		GroundTruth: "Expected summary",
		MaxLength:   100,
		Style:       "bullet_points",
	}

	ctx := &SummarizationTraceContext{
		input: input,
	}

	if ctx.GetInput() != input {
		t.Error("GetInput should return the input")
	}

	if ctx.GetInput().Input != "Long article text" {
		t.Errorf("GetInput().Input = %s", ctx.GetInput().Input)
	}

	if ctx.GetInput().MaxLength != 100 {
		t.Errorf("GetInput().MaxLength = %d, want 100", ctx.GetInput().MaxLength)
	}
}

func TestSummarizationTraceContext_GetOutput(t *testing.T) {
	output := &SummarizationOutput{
		Output:           "Summary of the article",
		Length:           5,
		CompressionRatio: 0.1,
	}

	ctx := &SummarizationTraceContext{
		output: output,
	}

	if ctx.GetOutput() != output {
		t.Error("GetOutput should return the output")
	}

	if ctx.GetOutput().Output != "Summary of the article" {
		t.Errorf("GetOutput().Output = %s", ctx.GetOutput().Output)
	}

	if ctx.GetOutput().Length != 5 {
		t.Errorf("GetOutput().Length = %d, want 5", ctx.GetOutput().Length)
	}
}

func TestSummarizationTraceContext_ValidateForEvaluation(t *testing.T) {
	tests := []struct {
		name        string
		input       *SummarizationInput
		output      *SummarizationOutput
		expectError bool
	}{
		{
			name: "valid for evaluation",
			input: &SummarizationInput{
				Input: "Long article text",
			},
			output: &SummarizationOutput{
				Output: "Summary",
			},
			expectError: false,
		},
		{
			name: "nil output",
			input: &SummarizationInput{
				Input: "Long article text",
			},
			output:      nil,
			expectError: true,
		},
		{
			name: "empty output",
			input: &SummarizationInput{
				Input: "Long article text",
			},
			output:      &SummarizationOutput{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &SummarizationTraceContext{
				input:  tt.input,
				output: tt.output,
			}

			err := ctx.ValidateForEvaluation()

			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestSummarizationInput_Fields(t *testing.T) {
	input := &SummarizationInput{
		Input:       "This is a very long article about programming...",
		GroundTruth: "Programming article summary",
		MaxLength:   100,
		Style:       "bullet_points",
	}

	if input.Input == "" {
		t.Error("Input should not be empty")
	}

	if input.GroundTruth != "Programming article summary" {
		t.Errorf("GroundTruth = %s", input.GroundTruth)
	}

	if input.MaxLength != 100 {
		t.Errorf("MaxLength = %d, want 100", input.MaxLength)
	}

	if input.Style != "bullet_points" {
		t.Errorf("Style = %s, want bullet_points", input.Style)
	}
}

func TestSummarizationOutput_Fields(t *testing.T) {
	output := &SummarizationOutput{
		Output:           "Summary of the article",
		Length:           5,
		CompressionRatio: 0.05,
		Metadata: map[string]any{
			"model": "gpt-4",
		},
	}

	if output.Output != "Summary of the article" {
		t.Errorf("Output = %s", output.Output)
	}

	if output.Length != 5 {
		t.Errorf("Length = %d, want 5", output.Length)
	}

	if output.CompressionRatio != 0.05 {
		t.Errorf("CompressionRatio = %f, want 0.05", output.CompressionRatio)
	}

	if output.Metadata["model"] != "gpt-4" {
		t.Error("Metadata not set correctly")
	}
}
