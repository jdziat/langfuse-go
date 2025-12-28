package evaluation

import (
	"strings"
	"testing"
)

func TestValidateFor_RAG_Valid(t *testing.T) {
	input := &RAGInput{
		Query:   "What is Go?",
		Context: []string{"Go is a language"},
	}
	output := &RAGOutput{
		Output: "Go is a programming language",
	}

	err := ValidateFor(input, output, RAGEvaluator)
	if err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}
}

func TestValidateFor_RAG_MissingQuery(t *testing.T) {
	input := &RAGInput{
		Context: []string{"Go is a language"},
	}
	output := &RAGOutput{
		Output: "answer",
	}

	err := ValidateFor(input, output, RAGEvaluator)
	if err == nil {
		t.Error("expected error for missing query")
	}

	if !strings.Contains(err.Error(), "query") {
		t.Errorf("error should mention 'query': %v", err)
	}
}

func TestValidateFor_RAG_MissingContext(t *testing.T) {
	input := &RAGInput{
		Query: "What is Go?",
	}
	output := &RAGOutput{
		Output: "answer",
	}

	err := ValidateFor(input, output, RAGEvaluator)
	if err == nil {
		t.Error("expected error for missing context")
	}

	if !strings.Contains(err.Error(), "context") {
		t.Errorf("error should mention 'context': %v", err)
	}
}

func TestValidateFor_RAG_MissingOutput(t *testing.T) {
	input := &RAGInput{
		Query:   "What is Go?",
		Context: []string{"Go is a language"},
	}
	output := &RAGOutput{}

	err := ValidateFor(input, output, RAGEvaluator)
	if err == nil {
		t.Error("expected error for missing output")
	}

	if !strings.Contains(err.Error(), "output") {
		t.Errorf("error should mention 'output': %v", err)
	}
}

func TestValidateFor_QA_Valid(t *testing.T) {
	input := &QAInput{
		Query: "What is the capital of France?",
	}
	output := &QAOutput{
		Output: "Paris",
	}

	err := ValidateFor(input, output, QAEvaluator)
	if err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}
}

func TestValidateFor_QA_MissingQuery(t *testing.T) {
	input := &QAInput{}
	output := &QAOutput{
		Output: "Paris",
	}

	err := ValidateFor(input, output, QAEvaluator)
	if err == nil {
		t.Error("expected error for missing query")
	}
}

func TestValidateFor_Summarization_Valid(t *testing.T) {
	input := &SummarizationInput{
		Input: "Long article text...",
	}
	output := &SummarizationOutput{
		Output: "Summary of article",
	}

	err := ValidateFor(input, output, SummarizationEvaluator)
	if err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}
}

func TestValidateFor_Classification_Valid(t *testing.T) {
	input := &ClassificationInput{
		Input: "I love this product!",
	}
	output := &ClassificationOutput{
		Output: "positive",
	}

	err := ValidateFor(input, output, ClassificationEvaluator)
	if err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}
}

func TestValidateFor_WithMap(t *testing.T) {
	input := map[string]any{
		"query":   "What is Go?",
		"context": []string{"Go is a language"},
	}
	output := map[string]any{
		"output": "answer",
	}

	err := ValidateFor(input, output, RAGEvaluator)
	if err != nil {
		t.Errorf("expected valid with maps, got error: %v", err)
	}
}

func TestValidateFor_NilInput(t *testing.T) {
	output := &RAGOutput{
		Output: "answer",
	}

	err := ValidateFor(nil, output, RAGEvaluator)
	if err == nil {
		t.Error("expected error for nil input")
	}
}

func TestValidateFor_NilOutput(t *testing.T) {
	input := &RAGInput{
		Query:   "What is Go?",
		Context: []string{"Go is a language"},
	}

	err := ValidateFor(input, nil, RAGEvaluator)
	if err == nil {
		t.Error("expected error for nil output")
	}
}

func TestExtractFields_Struct(t *testing.T) {
	input := &RAGInput{
		Query:   "test",
		Context: []string{"ctx"},
	}

	fields := extractFields(input)

	if !containsField(fields, "query") {
		t.Error("expected 'query' field")
	}

	if !containsField(fields, "context") {
		t.Error("expected 'context' field")
	}

	if containsField(fields, "ground_truth") {
		t.Error("expected 'ground_truth' to be absent when empty")
	}
}

func TestExtractFields_Map(t *testing.T) {
	input := map[string]any{
		"query":   "test",
		"context": []string{"ctx"},
		"empty":   nil,
	}

	fields := extractFields(input)

	if !containsField(fields, "query") {
		t.Error("expected 'query' field")
	}

	if !containsField(fields, "context") {
		t.Error("expected 'context' field")
	}

	if containsField(fields, "empty") {
		t.Error("expected 'empty' to be absent when nil")
	}
}

func TestExtractFields_Nil(t *testing.T) {
	fields := extractFields(nil)
	if len(fields) != 0 {
		t.Errorf("expected empty fields for nil, got %v", fields)
	}
}

func TestValidateDetailed(t *testing.T) {
	input := &RAGInput{
		Query:   "What is Go?",
		Context: []string{"Go is a language"},
	}
	output := &RAGOutput{
		Output: "Go is a programming language",
	}

	result := ValidateDetailed(input, output, RAGEvaluator)

	if !result.Valid {
		t.Errorf("expected valid result, got invalid: %v", result.MissingFields)
	}

	if result.EvaluatorName != "RAG" {
		t.Errorf("expected evaluator name 'RAG', got %s", result.EvaluatorName)
	}

	if len(result.PresentFields) < 3 {
		t.Errorf("expected at least 3 present fields, got %d", len(result.PresentFields))
	}
}

func TestValidateDetailed_Invalid(t *testing.T) {
	input := &RAGInput{
		Query: "What is Go?",
	}
	output := &RAGOutput{
		Output: "answer",
	}

	result := ValidateDetailed(input, output, RAGEvaluator)

	if result.Valid {
		t.Error("expected invalid result")
	}

	if len(result.MissingFields) == 0 {
		t.Error("expected missing fields")
	}

	if !containsField(result.MissingFields, "context") {
		t.Error("expected 'context' in missing fields")
	}

	err := result.Error()
	if err == nil {
		t.Error("expected error from result")
	}
}

func TestValidateDetailed_Warnings(t *testing.T) {
	input := &RAGInput{
		Query:   "What is Go?",
		Context: []string{"Go is a language"},
	}
	output := &RAGOutput{
		Output: "answer",
	}

	result := ValidateDetailed(input, output, RAGEvaluator)

	if !result.Valid {
		t.Errorf("expected valid result, got: %v", result.MissingFields)
	}

	if len(result.Warnings) == 0 {
		t.Error("expected warnings about missing optional fields")
	}
}

func TestEvaluatorRequirements(t *testing.T) {
	evaluators := []EvaluatorRequirements{
		RAGEvaluator,
		QAEvaluator,
		SummarizationEvaluator,
		ClassificationEvaluator,
		ToxicityEvaluator,
		HallucinationEvaluator,
		FaithfulnessEvaluator,
		AnswerRelevanceEvaluator,
		ContextRelevanceEvaluator,
		ContextCorrectnessEvaluator,
	}

	for _, eval := range evaluators {
		if eval.Name == "" {
			t.Errorf("evaluator should have a name")
		}

		if len(eval.RequiredFields) == 0 {
			t.Errorf("evaluator %s should have required fields", eval.Name)
		}

		if eval.Description == "" {
			t.Errorf("evaluator %s should have a description", eval.Name)
		}
	}
}

func TestContextRelevanceEvaluator(t *testing.T) {
	input := &RAGInput{
		Query:   "What is Go?",
		Context: []string{"Go is a language"},
	}

	err := ValidateFor(input, nil, ContextRelevanceEvaluator)
	if err != nil {
		t.Errorf("expected valid for context relevance, got: %v", err)
	}
}

func TestContextCorrectnessEvaluator(t *testing.T) {
	input := &RAGInput{
		Query:       "What is Go?",
		Context:     []string{"Go is a language"},
		GroundTruth: "Go is a programming language created by Google",
	}

	err := ValidateFor(input, nil, ContextCorrectnessEvaluator)
	if err != nil {
		t.Errorf("expected valid for context correctness, got: %v", err)
	}
}

func TestContextCorrectnessEvaluator_MissingGroundTruth(t *testing.T) {
	input := &RAGInput{
		Query:   "What is Go?",
		Context: []string{"Go is a language"},
	}

	err := ValidateFor(input, nil, ContextCorrectnessEvaluator)
	if err == nil {
		t.Error("expected error for missing ground_truth")
	}
}
