package evaluation

import (
	"encoding/json"
	"testing"
)

func TestRAGInputJSON(t *testing.T) {
	input := &RAGInput{
		Query: "What is Go?",
		Context: []string{
			"Go is a programming language.",
			"Go was created by Google.",
		},
		GroundTruth: "Go is a statically typed language",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal RAGInput: %v", err)
	}

	var decoded RAGInput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal RAGInput: %v", err)
	}

	if decoded.Query != input.Query {
		t.Errorf("Query mismatch: got %s, want %s", decoded.Query, input.Query)
	}

	if len(decoded.Context) != len(input.Context) {
		t.Errorf("Context length mismatch: got %d, want %d", len(decoded.Context), len(input.Context))
	}

	if decoded.GroundTruth != input.GroundTruth {
		t.Errorf("GroundTruth mismatch: got %s, want %s", decoded.GroundTruth, input.GroundTruth)
	}
}

func TestRAGInputJSONFields(t *testing.T) {
	input := &RAGInput{
		Query:   "test query",
		Context: []string{"ctx1"},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := m["query"]; !ok {
		t.Error("expected 'query' field in JSON")
	}

	if _, ok := m["context"]; !ok {
		t.Error("expected 'context' field in JSON")
	}

	// Ground truth should be omitted when empty
	if _, ok := m["ground_truth"]; ok {
		t.Error("expected 'ground_truth' to be omitted when empty")
	}
}

func TestRAGOutputJSON(t *testing.T) {
	output := &RAGOutput{
		Output:       "This is the answer",
		Citations:    []string{"doc1.txt", "doc2.txt"},
		SourceChunks: []int{0, 2},
		Confidence:   0.95,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("failed to marshal RAGOutput: %v", err)
	}

	var decoded RAGOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal RAGOutput: %v", err)
	}

	if decoded.Output != output.Output {
		t.Errorf("Output mismatch: got %s, want %s", decoded.Output, output.Output)
	}

	if len(decoded.Citations) != len(output.Citations) {
		t.Errorf("Citations length mismatch: got %d, want %d", len(decoded.Citations), len(output.Citations))
	}

	if decoded.Confidence != output.Confidence {
		t.Errorf("Confidence mismatch: got %f, want %f", decoded.Confidence, output.Confidence)
	}
}

func TestQAInputJSON(t *testing.T) {
	input := &QAInput{
		Query:       "What is the capital of France?",
		GroundTruth: "Paris",
		Context:     "France is a country in Europe.",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal QAInput: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if m["query"] != "What is the capital of France?" {
		t.Errorf("expected query field, got %v", m["query"])
	}

	if m["ground_truth"] != "Paris" {
		t.Errorf("expected ground_truth field, got %v", m["ground_truth"])
	}
}

func TestQAOutputJSON(t *testing.T) {
	output := &QAOutput{
		Output:     "Paris",
		Confidence: 0.99,
		Reasoning:  "Based on geographic knowledge",
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("failed to marshal QAOutput: %v", err)
	}

	var decoded QAOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal QAOutput: %v", err)
	}

	if decoded.Output != output.Output {
		t.Errorf("Output mismatch: got %s, want %s", decoded.Output, output.Output)
	}

	if decoded.Confidence != output.Confidence {
		t.Errorf("Confidence mismatch: got %f, want %f", decoded.Confidence, output.Confidence)
	}
}

func TestSummarizationInputJSON(t *testing.T) {
	input := &SummarizationInput{
		Input:       "This is a long article about programming...",
		MaxLength:   100,
		Style:       "bullet_points",
		GroundTruth: "A summary of programming concepts.",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal SummarizationInput: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if m["input"] != input.Input {
		t.Errorf("expected input field")
	}

	if int(m["max_length"].(float64)) != input.MaxLength {
		t.Errorf("expected max_length field")
	}
}

func TestSummarizationOutputJSON(t *testing.T) {
	output := &SummarizationOutput{
		Output:           "Summary of the article",
		Length:           5,
		CompressionRatio: 0.1,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("failed to marshal SummarizationOutput: %v", err)
	}

	var decoded SummarizationOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal SummarizationOutput: %v", err)
	}

	if decoded.Output != output.Output {
		t.Errorf("Output mismatch")
	}

	if decoded.CompressionRatio != output.CompressionRatio {
		t.Errorf("CompressionRatio mismatch")
	}
}

func TestClassificationInputJSON(t *testing.T) {
	input := &ClassificationInput{
		Input:       "I love this product!",
		Classes:     []string{"positive", "negative", "neutral"},
		GroundTruth: "positive",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal ClassificationInput: %v", err)
	}

	var decoded ClassificationInput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal ClassificationInput: %v", err)
	}

	if decoded.Input != input.Input {
		t.Errorf("Input mismatch")
	}

	if len(decoded.Classes) != len(input.Classes) {
		t.Errorf("Classes length mismatch")
	}

	if decoded.GroundTruth != input.GroundTruth {
		t.Errorf("GroundTruth mismatch")
	}
}

func TestClassificationOutputJSON(t *testing.T) {
	output := &ClassificationOutput{
		Output:     "positive",
		Confidence: 0.92,
		Scores: map[string]float64{
			"positive": 0.92,
			"negative": 0.05,
			"neutral":  0.03,
		},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("failed to marshal ClassificationOutput: %v", err)
	}

	var decoded ClassificationOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal ClassificationOutput: %v", err)
	}

	if decoded.Output != output.Output {
		t.Errorf("Output mismatch")
	}

	if len(decoded.Scores) != len(output.Scores) {
		t.Errorf("Scores length mismatch")
	}
}

func TestEvaluationTypeConstants(t *testing.T) {
	tests := []struct {
		evalType EvaluationType
		expected string
	}{
		{EvaluationTypeRAG, "rag"},
		{EvaluationTypeQA, "qa"},
		{EvaluationTypeSummarization, "summarization"},
		{EvaluationTypeClassification, "classification"},
	}

	for _, tt := range tests {
		if string(tt.evalType) != tt.expected {
			t.Errorf("EvaluationType mismatch: got %s, want %s", tt.evalType, tt.expected)
		}
	}
}

func TestOmitEmptyFields(t *testing.T) {
	input := &RAGInput{
		Query:   "test",
		Context: []string{"ctx"},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := m["ground_truth"]; ok {
		t.Error("expected ground_truth to be omitted when empty")
	}

	if _, ok := m["additional_context"]; ok {
		t.Error("expected additional_context to be omitted when nil")
	}
}
