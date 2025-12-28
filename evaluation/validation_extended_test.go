package evaluation

import (
	"testing"
)

func TestExtractJSONName(t *testing.T) {
	tests := []struct {
		tag      string
		expected string
	}{
		{"query", "query"},
		{"query,omitempty", "query"},
		{"query,omitempty,string", "query"},
		{",omitempty", ""},
		{"", ""},
		{"-", "-"},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			result := extractJSONName(tt.tag)
			if result != tt.expected {
				t.Errorf("extractJSONName(%q) = %q, want %q", tt.tag, result, tt.expected)
			}
		})
	}
}

func TestContainsField(t *testing.T) {
	slice := []string{"query", "context", "output"}

	tests := []struct {
		item     string
		expected bool
	}{
		{"query", true},
		{"context", true},
		{"output", true},
		{"missing", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.item, func(t *testing.T) {
			result := containsField(slice, tt.item)
			if result != tt.expected {
				t.Errorf("containsField() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestContainsField_EmptySlice(t *testing.T) {
	var slice []string
	if containsField(slice, "anything") {
		t.Error("containsField(nil, ...) should return false")
	}
}

func TestMergeFields(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected int
	}{
		{
			name:     "no overlap",
			a:        []string{"a", "b"},
			b:        []string{"c", "d"},
			expected: 4,
		},
		{
			name:     "full overlap",
			a:        []string{"a", "b"},
			b:        []string{"a", "b"},
			expected: 2,
		},
		{
			name:     "partial overlap",
			a:        []string{"a", "b", "c"},
			b:        []string{"b", "c", "d"},
			expected: 4,
		},
		{
			name:     "empty a",
			a:        []string{},
			b:        []string{"a", "b"},
			expected: 2,
		},
		{
			name:     "empty b",
			a:        []string{"a", "b"},
			b:        []string{},
			expected: 2,
		},
		{
			name:     "both empty",
			a:        []string{},
			b:        []string{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeFields(tt.a, tt.b)
			if len(result) != tt.expected {
				t.Errorf("mergeFields() length = %d, want %d", len(result), tt.expected)
			}
		})
	}
}

func TestFilterInputFields(t *testing.T) {
	tests := []struct {
		name     string
		fields   []string
		expected int // number of fields after filtering
	}{
		{
			name:     "all input fields",
			fields:   []string{"query", "context", "input"},
			expected: 3,
		},
		{
			name:     "mixed fields",
			fields:   []string{"query", "output", "context", "confidence"},
			expected: 2, // query, context
		},
		{
			name:     "all output fields",
			fields:   []string{"output", "citations", "scores"},
			expected: 0,
		},
		{
			name:     "empty",
			fields:   []string{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterInputFields(tt.fields)
			if len(result) != tt.expected {
				t.Errorf("filterInputFields() length = %d, want %d (got: %v)", len(result), tt.expected, result)
			}
		})
	}
}

func TestIsZeroValue(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		{"nil", nil, true},
		{"empty string", "", true},
		{"non-empty string", "hello", false},
		{"zero int", 0, true},
		{"non-zero int", 42, false},
		{"zero float", 0.0, true},
		{"non-zero float", 3.14, false},
		{"nil slice", []string(nil), true},
		{"empty slice", []string{}, false}, // empty slice is not nil
		{"nil map", map[string]int(nil), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isZeroValue(tt.value)
			if result != tt.expected {
				t.Errorf("isZeroValue(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestExtractFields_NonStruct(t *testing.T) {
	// Test with primitive types - should return empty
	result := extractFields(42)
	if len(result) != 0 {
		t.Errorf("extractFields(int) should return empty, got %v", result)
	}

	result = extractFields("string")
	if len(result) != 0 {
		t.Errorf("extractFields(string) should return empty, got %v", result)
	}
}

func TestExtractFields_NilPointer(t *testing.T) {
	var input *RAGInput
	result := extractFields(input)
	if len(result) != 0 {
		t.Errorf("extractFields(nil pointer) should return empty, got %v", result)
	}
}

func TestValidateInput(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		reqs        EvaluatorRequirements
		expectError bool
	}{
		{
			name: "valid rag input",
			input: &RAGInput{
				Query:   "test",
				Context: []string{"ctx"},
			},
			reqs:        RAGEvaluator,
			expectError: false,
		},
		{
			name: "missing query in rag input",
			input: &RAGInput{
				Context: []string{"ctx"},
			},
			reqs:        RAGEvaluator,
			expectError: true,
		},
		{
			name: "valid qa input",
			input: &QAInput{
				Query: "test",
			},
			reqs:        QAEvaluator,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInput(tt.input, tt.reqs)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestValidateOutput(t *testing.T) {
	tests := []struct {
		name        string
		output      any
		reqs        EvaluatorRequirements
		expectError bool
	}{
		{
			name:        "valid rag output",
			output:      &RAGOutput{Output: "answer"},
			reqs:        RAGEvaluator,
			expectError: false,
		},
		{
			name:        "empty rag output",
			output:      &RAGOutput{},
			reqs:        RAGEvaluator,
			expectError: true,
		},
		{
			name:        "valid toxicity output",
			output:      &ToxicityOutput{Output: "safe text"},
			reqs:        ToxicityEvaluator,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOutput(tt.output, tt.reqs)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestValidationResult_Error(t *testing.T) {
	// Valid result
	validResult := &ValidationResult{
		Valid:         true,
		EvaluatorName: "Test",
	}
	if validResult.Error() != nil {
		t.Error("valid result should return nil error")
	}

	// Invalid result
	invalidResult := &ValidationResult{
		Valid:         false,
		EvaluatorName: "Test",
		MissingFields: []string{"query", "context"},
	}
	err := invalidResult.Error()
	if err == nil {
		t.Error("invalid result should return error")
	}
	if err != nil && invalidResult.MissingFields == nil {
		t.Error("error should mention missing fields")
	}
}

func TestValidateDetailed_AllOptionalFieldsMissing(t *testing.T) {
	input := &RAGInput{
		Query:   "test",
		Context: []string{"ctx"},
	}
	output := &RAGOutput{
		Output: "answer",
	}

	result := ValidateDetailed(input, output, RAGEvaluator)

	if !result.Valid {
		t.Error("should be valid")
	}

	// Should have warnings for missing optional fields
	if len(result.Warnings) == 0 {
		t.Error("should have warnings for missing optional fields")
	}
}

func TestValidateFor_Hallucination(t *testing.T) {
	input := &HallucinationInput{
		Query:   "What is the capital?",
		Context: []string{"France has Paris as capital"},
	}
	output := &HallucinationOutput{
		Output: "Paris is the capital",
	}

	err := ValidateFor(input, output, HallucinationEvaluator)
	if err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestValidateFor_Faithfulness(t *testing.T) {
	input := map[string]any{
		"context": []string{"Go is a language"},
	}
	output := map[string]any{
		"output": "Go is a programming language",
	}

	err := ValidateFor(input, output, FaithfulnessEvaluator)
	if err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestValidateFor_AnswerRelevance(t *testing.T) {
	input := map[string]any{
		"query": "What is Go?",
	}
	output := map[string]any{
		"output": "Go is a programming language",
	}

	err := ValidateFor(input, output, AnswerRelevanceEvaluator)
	if err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestValidateFor_Toxicity(t *testing.T) {
	output := &ToxicityOutput{
		Output: "This is safe content",
	}

	err := ValidateFor(nil, output, ToxicityEvaluator)
	if err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestAllEvaluatorRequirements(t *testing.T) {
	evaluators := map[string]EvaluatorRequirements{
		"RAG":                RAGEvaluator,
		"ContextRelevance":   ContextRelevanceEvaluator,
		"ContextCorrectness": ContextCorrectnessEvaluator,
		"QA":                 QAEvaluator,
		"Summarization":      SummarizationEvaluator,
		"Classification":     ClassificationEvaluator,
		"Toxicity":           ToxicityEvaluator,
		"Hallucination":      HallucinationEvaluator,
		"Faithfulness":       FaithfulnessEvaluator,
		"AnswerRelevance":    AnswerRelevanceEvaluator,
	}

	for name, eval := range evaluators {
		t.Run(name, func(t *testing.T) {
			if eval.Name == "" {
				t.Error("Name should not be empty")
			}
			if eval.Description == "" {
				t.Error("Description should not be empty")
			}
			if len(eval.RequiredFields) == 0 {
				t.Error("RequiredFields should not be empty")
			}
		})
	}
}
