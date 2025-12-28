package evaluation

import (
	"testing"
)

func TestRAGTraceBuilder_ValidateInput(t *testing.T) {
	tests := []struct {
		name        string
		input       *RAGInput
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid rag input",
			input: &RAGInput{
				Query:   "What is Go?",
				Context: []string{"Go is a programming language"},
			},
			expectError: false,
		},
		{
			name: "missing query",
			input: &RAGInput{
				Context: []string{"Go is a programming language"},
			},
			expectError: true,
			errorMsg:    "query",
		},
		{
			name: "missing context",
			input: &RAGInput{
				Query: "What is Go?",
			},
			expectError: true,
			errorMsg:    "context",
		},
		{
			name: "empty query",
			input: &RAGInput{
				Query:   "",
				Context: []string{"context"},
			},
			expectError: true,
			errorMsg:    "query",
		},
		{
			name: "nil context",
			input: &RAGInput{
				Query:   "query",
				Context: nil,
			},
			expectError: true,
			errorMsg:    "context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &RAGOutput{Output: "test"}
			err := ValidateFor(tt.input, output, RAGEvaluator)

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

func TestRAGTraceBuilder_FluentAPI(t *testing.T) {
	builder := &RAGTraceBuilder{
		ragInput: &RAGInput{},
	}

	// Test fluent chaining
	result := builder.
		Query("test query").
		Context("context1", "context2").
		GroundTruth("expected answer").
		AdditionalContext(map[string]any{"key": "value"})

	if result != builder {
		t.Error("fluent methods should return the same builder")
	}

	if builder.ragInput.Query != "test query" {
		t.Errorf("Query not set correctly: got %s", builder.ragInput.Query)
	}

	if len(builder.ragInput.Context) != 2 {
		t.Errorf("Context length incorrect: got %d, want 2", len(builder.ragInput.Context))
	}

	if builder.ragInput.GroundTruth != "expected answer" {
		t.Errorf("GroundTruth not set correctly: got %s", builder.ragInput.GroundTruth)
	}

	if builder.ragInput.AdditionalContext["key"] != "value" {
		t.Error("AdditionalContext not set correctly")
	}
}

func TestRAGTraceContext_GetInput(t *testing.T) {
	input := &RAGInput{
		Query:       "test query",
		Context:     []string{"ctx1", "ctx2"},
		GroundTruth: "truth",
	}

	ctx := &RAGTraceContext{
		input: input,
	}

	if ctx.GetInput() != input {
		t.Error("GetInput should return the input")
	}

	if ctx.GetInput().Query != "test query" {
		t.Errorf("GetInput().Query = %s, want 'test query'", ctx.GetInput().Query)
	}
}

func TestRAGTraceContext_GetOutput(t *testing.T) {
	output := &RAGOutput{
		Output:    "test answer",
		Citations: []string{"doc1.txt"},
	}

	ctx := &RAGTraceContext{
		output: output,
	}

	if ctx.GetOutput() != output {
		t.Error("GetOutput should return the output")
	}

	if ctx.GetOutput().Output != "test answer" {
		t.Errorf("GetOutput().Output = %s, want 'test answer'", ctx.GetOutput().Output)
	}
}

func TestRAGTraceContext_ValidateForEvaluation(t *testing.T) {
	tests := []struct {
		name        string
		input       *RAGInput
		output      *RAGOutput
		expectError bool
	}{
		{
			name: "valid for evaluation",
			input: &RAGInput{
				Query:   "test query",
				Context: []string{"context"},
			},
			output: &RAGOutput{
				Output: "test answer",
			},
			expectError: false,
		},
		{
			name: "nil output",
			input: &RAGInput{
				Query:   "test query",
				Context: []string{"context"},
			},
			output:      nil,
			expectError: true,
		},
		{
			name: "empty output",
			input: &RAGInput{
				Query:   "test query",
				Context: []string{"context"},
			},
			output:      &RAGOutput{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RAGTraceContext{
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

func TestRAGInput_Fields(t *testing.T) {
	input := &RAGInput{
		Query:       "What is Go?",
		Context:     []string{"Go is a language", "Created by Google"},
		GroundTruth: "Go is a programming language",
		AdditionalContext: map[string]any{
			"source": "wikipedia",
		},
	}

	if input.Query != "What is Go?" {
		t.Errorf("Query = %s, want 'What is Go?'", input.Query)
	}

	if len(input.Context) != 2 {
		t.Errorf("Context length = %d, want 2", len(input.Context))
	}

	if input.GroundTruth != "Go is a programming language" {
		t.Errorf("GroundTruth = %s", input.GroundTruth)
	}

	if input.AdditionalContext["source"] != "wikipedia" {
		t.Error("AdditionalContext not set correctly")
	}
}

func TestRAGOutput_Fields(t *testing.T) {
	output := &RAGOutput{
		Output:       "Go is a programming language",
		Citations:    []string{"doc1.txt", "doc2.txt"},
		SourceChunks: []int{0, 1},
		Confidence:   0.95,
		Metadata: map[string]any{
			"model": "gpt-4",
		},
	}

	if output.Output != "Go is a programming language" {
		t.Errorf("Output = %s", output.Output)
	}

	if len(output.Citations) != 2 {
		t.Errorf("Citations length = %d, want 2", len(output.Citations))
	}

	if len(output.SourceChunks) != 2 {
		t.Errorf("SourceChunks length = %d, want 2", len(output.SourceChunks))
	}

	if output.Confidence != 0.95 {
		t.Errorf("Confidence = %f, want 0.95", output.Confidence)
	}

	if output.Metadata["model"] != "gpt-4" {
		t.Error("Metadata not set correctly")
	}
}
