package evaluation

import (
	"testing"
)

func TestQATraceBuilder_ValidateInput(t *testing.T) {
	tests := []struct {
		name        string
		input       *QAInput
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid qa input",
			input: &QAInput{
				Query: "What is the capital of France?",
			},
			expectError: false,
		},
		{
			name:        "missing query",
			input:       &QAInput{},
			expectError: true,
			errorMsg:    "query",
		},
		{
			name: "empty query",
			input: &QAInput{
				Query: "",
			},
			expectError: true,
			errorMsg:    "query",
		},
		{
			name: "valid with ground truth",
			input: &QAInput{
				Query:       "What is 2+2?",
				GroundTruth: "4",
			},
			expectError: false,
		},
		{
			name: "valid with context",
			input: &QAInput{
				Query:   "What is mentioned?",
				Context: "The document mentions Paris.",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &QAOutput{Output: "test"}
			err := ValidateFor(tt.input, output, QAEvaluator)

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

func TestQATraceBuilder_FluentAPI(t *testing.T) {
	builder := &QATraceBuilder{
		qaInput: &QAInput{},
	}

	result := builder.
		Query("What is the capital of France?").
		GroundTruth("Paris").
		Context("France is a country in Europe")

	if result != builder {
		t.Error("fluent methods should return the same builder")
	}

	if builder.qaInput.Query != "What is the capital of France?" {
		t.Errorf("Query not set correctly: got %s", builder.qaInput.Query)
	}

	if builder.qaInput.GroundTruth != "Paris" {
		t.Errorf("GroundTruth not set correctly: got %s", builder.qaInput.GroundTruth)
	}

	if builder.qaInput.Context != "France is a country in Europe" {
		t.Errorf("Context not set correctly: got %s", builder.qaInput.Context)
	}
}

func TestQATraceContext_GetInput(t *testing.T) {
	input := &QAInput{
		Query:       "test query",
		GroundTruth: "truth",
		Context:     "context",
	}

	ctx := &QATraceContext{
		input: input,
	}

	if ctx.GetInput() != input {
		t.Error("GetInput should return the input")
	}

	if ctx.GetInput().Query != "test query" {
		t.Errorf("GetInput().Query = %s, want 'test query'", ctx.GetInput().Query)
	}
}

func TestQATraceContext_GetOutput(t *testing.T) {
	output := &QAOutput{
		Output:     "Paris",
		Confidence: 0.99,
		Reasoning:  "Based on geographic knowledge",
	}

	ctx := &QATraceContext{
		output: output,
	}

	if ctx.GetOutput() != output {
		t.Error("GetOutput should return the output")
	}

	if ctx.GetOutput().Output != "Paris" {
		t.Errorf("GetOutput().Output = %s, want 'Paris'", ctx.GetOutput().Output)
	}

	if ctx.GetOutput().Confidence != 0.99 {
		t.Errorf("GetOutput().Confidence = %f, want 0.99", ctx.GetOutput().Confidence)
	}
}

func TestQATraceContext_ValidateForEvaluation(t *testing.T) {
	tests := []struct {
		name        string
		input       *QAInput
		output      *QAOutput
		expectError bool
	}{
		{
			name: "valid for evaluation",
			input: &QAInput{
				Query: "test query",
			},
			output: &QAOutput{
				Output: "test answer",
			},
			expectError: false,
		},
		{
			name: "nil output",
			input: &QAInput{
				Query: "test query",
			},
			output:      nil,
			expectError: true,
		},
		{
			name: "empty output",
			input: &QAInput{
				Query: "test query",
			},
			output:      &QAOutput{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &QATraceContext{
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

func TestQAInput_Fields(t *testing.T) {
	input := &QAInput{
		Query:       "What is the capital of France?",
		GroundTruth: "Paris",
		Context:     "France is a country in Western Europe",
	}

	if input.Query != "What is the capital of France?" {
		t.Errorf("Query = %s", input.Query)
	}

	if input.GroundTruth != "Paris" {
		t.Errorf("GroundTruth = %s", input.GroundTruth)
	}

	if input.Context != "France is a country in Western Europe" {
		t.Errorf("Context = %s", input.Context)
	}
}

func TestQAOutput_Fields(t *testing.T) {
	output := &QAOutput{
		Output:     "Paris",
		Confidence: 0.98,
		Reasoning:  "Based on world geography",
		Metadata: map[string]any{
			"source": "knowledge_base",
		},
	}

	if output.Output != "Paris" {
		t.Errorf("Output = %s", output.Output)
	}

	if output.Confidence != 0.98 {
		t.Errorf("Confidence = %f", output.Confidence)
	}

	if output.Reasoning != "Based on world geography" {
		t.Errorf("Reasoning = %s", output.Reasoning)
	}

	if output.Metadata["source"] != "knowledge_base" {
		t.Error("Metadata not set correctly")
	}
}
