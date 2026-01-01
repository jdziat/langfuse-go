package langfuse

import (
	"encoding/json"
	"reflect"
	"testing"
)

// testInput is a sample input struct for testing.
type testInput struct {
	Query       string   `json:"query"`
	Context     []string `json:"context"`
	GroundTruth string   `json:"ground_truth,omitempty"`
	Extra       string   `json:"extra,omitempty"`
}

// testOutput is a sample output struct for testing.
type testOutput struct {
	Output     string  `json:"output"`
	Confidence float64 `json:"confidence,omitempty"`
}

func TestInputFlattenerAuto(t *testing.T) {
	flattener := NewInputFlattener(EvaluationModeAuto)

	input := &testInput{
		Query:       "What is Go?",
		Context:     []string{"Go is a language", "Go has goroutines"},
		GroundTruth: "Go is a programming language",
	}

	result := flattener.Flatten(input)

	if result["query"] != "What is Go?" {
		t.Errorf("query = %v, want 'What is Go?'", result["query"])
	}

	ctx, ok := result["context"].([]string)
	if !ok {
		t.Errorf("context is not []string, got %T", result["context"])
	} else if len(ctx) != 2 {
		t.Errorf("context has %d items, want 2", len(ctx))
	}

	if result["ground_truth"] != "Go is a programming language" {
		t.Errorf("ground_truth = %v, want 'Go is a programming language'", result["ground_truth"])
	}
}

func TestInputFlattenerRAGAS(t *testing.T) {
	flattener := NewInputFlattener(EvaluationModeRAGAS)

	input := map[string]any{
		"query":   "What is Go?",
		"context": []string{"Go is a language"},
		"output":  "Go is a programming language",
	}

	result := flattener.Flatten(input)

	// RAGAS mode should map query -> user_input
	if result["user_input"] != "What is Go?" {
		t.Errorf("user_input = %v, want 'What is Go?'", result["user_input"])
	}

	// RAGAS mode should map context -> retrieved_contexts
	if result["retrieved_contexts"] == nil {
		t.Error("retrieved_contexts should be set")
	}

	// RAGAS mode should map output -> response
	if result["response"] != "Go is a programming language" {
		t.Errorf("response = %v, want 'Go is a programming language'", result["response"])
	}
}

func TestInputFlattenerWithMap(t *testing.T) {
	flattener := NewInputFlattener(EvaluationModeAuto)

	input := map[string]any{
		"query":   "test query",
		"context": []string{"doc1", "doc2"},
		"nested": map[string]any{
			"field1": "value1",
			"field2": 42,
		},
	}

	result := flattener.Flatten(input)

	if result["query"] != "test query" {
		t.Errorf("query = %v, want 'test query'", result["query"])
	}

	// Nested maps should be flattened with prefix
	if result["nested_field1"] != "value1" {
		t.Errorf("nested_field1 = %v, want 'value1'", result["nested_field1"])
	}
	if result["nested_field2"] != 42 {
		t.Errorf("nested_field2 = %v, want 42", result["nested_field2"])
	}
}

func TestInputFlattenerWithEvalInput(t *testing.T) {
	flattener := NewInputFlattener(EvaluationModeAuto)

	input := &StandardEvalInput{
		Query:       "What is Go?",
		Context:     []string{"Go is a language"},
		GroundTruth: "Go is a programming language",
	}

	result := flattener.Flatten(input)

	if result["query"] != "What is Go?" {
		t.Errorf("query = %v, want 'What is Go?'", result["query"])
	}

	// StandardEvalInput also provides "input" alias
	if result["input"] != "What is Go?" {
		t.Errorf("input = %v, want 'What is Go?'", result["input"])
	}
}

func TestInputFlattenerNilInput(t *testing.T) {
	flattener := NewInputFlattener(EvaluationModeAuto)

	result := flattener.Flatten(nil)

	if result != nil {
		t.Errorf("result should be nil for nil input, got %v", result)
	}
}

func TestFlattenedInputMarshalJSON(t *testing.T) {
	fi := FlattenedInput{
		Fields: map[string]any{
			"query":   "test",
			"context": []string{"doc1"},
		},
		EvalType: WorkflowRAG,
	}

	data, err := json.Marshal(fi)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if result["query"] != "test" {
		t.Errorf("query = %v, want 'test'", result["query"])
	}
	if result["_eval_type"] != "rag" {
		t.Errorf("_eval_type = %v, want 'rag'", result["_eval_type"])
	}
}

func TestFlattenedOutputMarshalJSON(t *testing.T) {
	fo := FlattenedOutput{
		Fields: map[string]any{
			"output":     "response text",
			"confidence": 0.95,
		},
	}

	data, err := json.Marshal(fo)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if result["output"] != "response text" {
		t.Errorf("output = %v, want 'response text'", result["output"])
	}
	if result["confidence"] != 0.95 {
		t.Errorf("confidence = %v, want 0.95", result["confidence"])
	}
}

func TestStandardEvalInputEvalFields(t *testing.T) {
	input := &StandardEvalInput{
		Query:        "What is Go?",
		Context:      []string{"doc1", "doc2"},
		GroundTruth:  "answer",
		SystemPrompt: "You are helpful",
	}

	fields := input.EvalFields()

	if fields["query"] != "What is Go?" {
		t.Errorf("query = %v, want 'What is Go?'", fields["query"])
	}
	if fields["input"] != "What is Go?" {
		t.Errorf("input = %v, want 'What is Go?'", fields["input"])
	}
	if !reflect.DeepEqual(fields["context"], []string{"doc1", "doc2"}) {
		t.Errorf("context = %v, want [doc1 doc2]", fields["context"])
	}
	if fields["ground_truth"] != "answer" {
		t.Errorf("ground_truth = %v, want 'answer'", fields["ground_truth"])
	}
	if fields["system_prompt"] != "You are helpful" {
		t.Errorf("system_prompt = %v, want 'You are helpful'", fields["system_prompt"])
	}
}

func TestStandardEvalOutputEvalFields(t *testing.T) {
	output := &StandardEvalOutput{
		Output:     "response",
		Citations:  []string{"source1"},
		Confidence: 0.9,
		Reasoning:  "because...",
	}

	fields := output.EvalFields()

	if fields["output"] != "response" {
		t.Errorf("output = %v, want 'response'", fields["output"])
	}
	if !reflect.DeepEqual(fields["citations"], []string{"source1"}) {
		t.Errorf("citations = %v, want [source1]", fields["citations"])
	}
	if fields["confidence"] != 0.9 {
		t.Errorf("confidence = %v, want 0.9", fields["confidence"])
	}
	if fields["reasoning"] != "because..." {
		t.Errorf("reasoning = %v, want 'because...'", fields["reasoning"])
	}
}

func TestPrepareInputForEval(t *testing.T) {
	config := DefaultEvaluationConfig()

	input := map[string]any{
		"query":   "test",
		"context": []string{"doc1"},
	}

	result := prepareInputForEval(input, config)

	fi, ok := result.(FlattenedInput)
	if !ok {
		t.Fatalf("result is not FlattenedInput, got %T", result)
	}

	if fi.Fields["query"] != "test" {
		t.Errorf("query = %v, want 'test'", fi.Fields["query"])
	}
}

func TestPrepareInputForEvalOff(t *testing.T) {
	config := &EvaluationConfig{Mode: EvaluationModeOff}

	input := map[string]any{"query": "test"}

	result := prepareInputForEval(input, config)

	// Should return input unchanged
	if !reflect.DeepEqual(result, input) {
		t.Errorf("result should equal input when mode is off")
	}
}

func TestPrepareOutputForEvalString(t *testing.T) {
	config := DefaultEvaluationConfig()

	output := "This is the response"

	result := prepareOutputForEval(output, config)

	fo, ok := result.(FlattenedOutput)
	if !ok {
		t.Fatalf("result is not FlattenedOutput, got %T", result)
	}

	if fo.Fields["output"] != "This is the response" {
		t.Errorf("output = %v, want 'This is the response'", fo.Fields["output"])
	}
}

func TestExtractFieldPresence(t *testing.T) {
	tests := []struct {
		name     string
		data     any
		expected map[string]bool
	}{
		{
			name: "map input",
			data: map[string]any{
				"query":   "test",
				"context": []string{"doc1"},
			},
			expected: map[string]bool{"query": true, "context": true},
		},
		{
			name: "struct input",
			data: &testInput{
				Query:   "test",
				Context: []string{"doc1"},
			},
			expected: map[string]bool{"query": true, "context": true},
		},
		{
			name: "EvalInput implementation",
			data: &StandardEvalInput{
				Query:   "test",
				Context: []string{"doc1"},
			},
			expected: map[string]bool{"query": true, "input": true, "context": true},
		},
		{
			name:     "nil input",
			data:     nil,
			expected: map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFieldPresence(tt.data)

			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("result[%s] = %v, want %v", k, result[k], v)
				}
			}
		})
	}
}

func TestMergeFieldPresence(t *testing.T) {
	a := map[string]bool{"query": true, "context": true}
	b := map[string]bool{"output": true, "context": true}

	result := mergeFieldPresence(a, b)

	expected := map[string]bool{"query": true, "context": true, "output": true}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("result = %v, want %v", result, expected)
	}
}
