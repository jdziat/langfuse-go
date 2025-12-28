package evaluation

import (
	"testing"
)

func TestClassificationTraceBuilder_ValidateInput(t *testing.T) {
	tests := []struct {
		name        string
		input       *ClassificationInput
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid classification input",
			input: &ClassificationInput{
				Input: "I love this product!",
			},
			expectError: false,
		},
		{
			name:        "missing input",
			input:       &ClassificationInput{},
			expectError: true,
			errorMsg:    "input",
		},
		{
			name: "empty input",
			input: &ClassificationInput{
				Input: "",
			},
			expectError: true,
			errorMsg:    "input",
		},
		{
			name: "valid with classes",
			input: &ClassificationInput{
				Input:   "I love this!",
				Classes: []string{"positive", "negative", "neutral"},
			},
			expectError: false,
		},
		{
			name: "valid with ground truth",
			input: &ClassificationInput{
				Input:       "I love this!",
				GroundTruth: "positive",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &ClassificationOutput{Output: "positive"}
			err := ValidateFor(tt.input, output, ClassificationEvaluator)

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

func TestClassificationTraceBuilder_FluentAPI(t *testing.T) {
	builder := &ClassificationTraceBuilder{
		classInput: &ClassificationInput{},
	}

	result := builder.
		Input("I love this product!").
		Classes([]string{"positive", "negative", "neutral"}).
		GroundTruth("positive")

	if result != builder {
		t.Error("fluent methods should return the same builder")
	}

	if builder.classInput.Input != "I love this product!" {
		t.Errorf("Input not set correctly: got %s", builder.classInput.Input)
	}

	if len(builder.classInput.Classes) != 3 {
		t.Errorf("Classes length not correct: got %d, want 3", len(builder.classInput.Classes))
	}

	if builder.classInput.GroundTruth != "positive" {
		t.Errorf("GroundTruth not set correctly: got %s", builder.classInput.GroundTruth)
	}
}

func TestClassificationTraceContext_GetInput(t *testing.T) {
	input := &ClassificationInput{
		Input:       "I love this product!",
		Classes:     []string{"positive", "negative", "neutral"},
		GroundTruth: "positive",
	}

	ctx := &ClassificationTraceContext{
		input: input,
	}

	if ctx.GetInput() != input {
		t.Error("GetInput should return the input")
	}

	if ctx.GetInput().Input != "I love this product!" {
		t.Errorf("GetInput().Input = %s", ctx.GetInput().Input)
	}

	if len(ctx.GetInput().Classes) != 3 {
		t.Errorf("GetInput().Classes length = %d, want 3", len(ctx.GetInput().Classes))
	}
}

func TestClassificationTraceContext_GetOutput(t *testing.T) {
	output := &ClassificationOutput{
		Output:     "positive",
		Confidence: 0.92,
		Scores: map[string]float64{
			"positive": 0.92,
			"negative": 0.05,
			"neutral":  0.03,
		},
	}

	ctx := &ClassificationTraceContext{
		output: output,
	}

	if ctx.GetOutput() != output {
		t.Error("GetOutput should return the output")
	}

	if ctx.GetOutput().Output != "positive" {
		t.Errorf("GetOutput().Output = %s", ctx.GetOutput().Output)
	}

	if ctx.GetOutput().Confidence != 0.92 {
		t.Errorf("GetOutput().Confidence = %f, want 0.92", ctx.GetOutput().Confidence)
	}
}

func TestClassificationTraceContext_ValidateForEvaluation(t *testing.T) {
	tests := []struct {
		name        string
		input       *ClassificationInput
		output      *ClassificationOutput
		expectError bool
	}{
		{
			name: "valid for evaluation",
			input: &ClassificationInput{
				Input: "I love this!",
			},
			output: &ClassificationOutput{
				Output: "positive",
			},
			expectError: false,
		},
		{
			name: "nil output",
			input: &ClassificationInput{
				Input: "I love this!",
			},
			output:      nil,
			expectError: true,
		},
		{
			name: "empty output",
			input: &ClassificationInput{
				Input: "I love this!",
			},
			output:      &ClassificationOutput{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &ClassificationTraceContext{
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

func TestClassificationInput_Fields(t *testing.T) {
	input := &ClassificationInput{
		Input:       "I love this product!",
		Classes:     []string{"positive", "negative", "neutral"},
		GroundTruth: "positive",
	}

	if input.Input != "I love this product!" {
		t.Errorf("Input = %s", input.Input)
	}

	if len(input.Classes) != 3 {
		t.Errorf("Classes length = %d, want 3", len(input.Classes))
	}

	if input.GroundTruth != "positive" {
		t.Errorf("GroundTruth = %s", input.GroundTruth)
	}
}

func TestClassificationOutput_Fields(t *testing.T) {
	output := &ClassificationOutput{
		Output:     "positive",
		Confidence: 0.92,
		Scores: map[string]float64{
			"positive": 0.92,
			"negative": 0.05,
			"neutral":  0.03,
		},
		Metadata: map[string]any{
			"model": "sentiment-classifier",
		},
	}

	if output.Output != "positive" {
		t.Errorf("Output = %s", output.Output)
	}

	if output.Confidence != 0.92 {
		t.Errorf("Confidence = %f, want 0.92", output.Confidence)
	}

	if len(output.Scores) != 3 {
		t.Errorf("Scores length = %d, want 3", len(output.Scores))
	}

	if output.Scores["positive"] != 0.92 {
		t.Errorf("Scores[positive] = %f, want 0.92", output.Scores["positive"])
	}

	if output.Metadata["model"] != "sentiment-classifier" {
		t.Error("Metadata not set correctly")
	}
}

func TestToxicityInput_Fields(t *testing.T) {
	input := &ToxicityInput{
		Input: "This is some text to evaluate",
	}

	if input.Input != "This is some text to evaluate" {
		t.Errorf("Input = %s", input.Input)
	}
}

func TestToxicityOutput_Fields(t *testing.T) {
	output := &ToxicityOutput{
		Output:        "This is fine",
		ToxicityScore: 0.05,
		Categories:    []string{"safe"},
		Metadata: map[string]any{
			"checked": true,
		},
	}

	if output.Output != "This is fine" {
		t.Errorf("Output = %s", output.Output)
	}

	if output.ToxicityScore != 0.05 {
		t.Errorf("ToxicityScore = %f, want 0.05", output.ToxicityScore)
	}

	if len(output.Categories) != 1 {
		t.Errorf("Categories length = %d, want 1", len(output.Categories))
	}
}

func TestHallucinationInput_Fields(t *testing.T) {
	input := &HallucinationInput{
		Query:   "What is the capital of France?",
		Context: []string{"France is a country in Europe", "Paris is the capital"},
	}

	if input.Query != "What is the capital of France?" {
		t.Errorf("Query = %s", input.Query)
	}

	if len(input.Context) != 2 {
		t.Errorf("Context length = %d, want 2", len(input.Context))
	}
}

func TestHallucinationOutput_Fields(t *testing.T) {
	output := &HallucinationOutput{
		Output:             "Paris is the capital of France",
		HallucinationScore: 0.1,
		Metadata: map[string]any{
			"verified": true,
		},
	}

	if output.Output != "Paris is the capital of France" {
		t.Errorf("Output = %s", output.Output)
	}

	if output.HallucinationScore != 0.1 {
		t.Errorf("HallucinationScore = %f, want 0.1", output.HallucinationScore)
	}

	if output.Metadata["verified"] != true {
		t.Error("Metadata not set correctly")
	}
}
