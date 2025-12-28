package evaluation

import (
	"context"
	"testing"

	"github.com/jdziat/langfuse-go/langfusetest"
)

// TestSummarizationTraceBuilder_WrapperMethods tests all wrapper methods on SummarizationTraceBuilder.
func TestSummarizationTraceBuilder_WrapperMethods(t *testing.T) {
	client, _ := langfusetest.NewTestClient(t)

	builder := NewSummarizationTrace(client, "test-summarization")

	// Test all wrapper methods return the builder for chaining
	result := builder.
		ID("trace-id-123").
		UserID("user-456").
		SessionID("session-789").
		Tags([]string{"tag1", "tag2"}).
		Metadata(map[string]any{"key": "value"}).
		Release("v1.0.0").
		Version("1").
		Environment("production").
		Public(true)

	if result != builder {
		t.Error("wrapper methods should return the same builder for chaining")
	}
}

// TestSummarizationTraceBuilder_Validate tests validation logic.
func TestSummarizationTraceBuilder_Validate(t *testing.T) {
	client, _ := langfusetest.NewTestClient(t)

	tests := []struct {
		name        string
		setup       func(*SummarizationTraceBuilder)
		expectError bool
	}{
		{
			name: "valid - has input",
			setup: func(b *SummarizationTraceBuilder) {
				b.Input("Long article to summarize")
			},
			expectError: false,
		},
		{
			name:        "invalid - missing input",
			setup:       func(b *SummarizationTraceBuilder) {},
			expectError: true,
		},
		{
			name: "invalid - empty input",
			setup: func(b *SummarizationTraceBuilder) {
				b.Input("")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewSummarizationTrace(client, "test")
			tt.setup(builder)

			err := builder.Validate()
			if tt.expectError && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

// TestSummarizationTraceBuilder_Create tests the Create method.
func TestSummarizationTraceBuilder_Create(t *testing.T) {
	client, _ := langfusetest.NewTestClient(t)

	t.Run("success", func(t *testing.T) {
		builder := NewSummarizationTrace(client, "test-summarization").
			Input("Long article to summarize").
			GroundTruth("Expected summary").
			MaxLength(100).
			Style("paragraph")

		ctx, err := builder.Create(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ctx == nil {
			t.Fatal("expected non-nil context")
		}
		if ctx.GetInput() == nil {
			t.Fatal("expected input to be set")
		}
		if ctx.GetInput().Input != "Long article to summarize" {
			t.Errorf("got input %q, want %q", ctx.GetInput().Input, "Long article to summarize")
		}
	})

	t.Run("validation failure", func(t *testing.T) {
		builder := NewSummarizationTrace(client, "test-summarization")
		// No input set

		_, err := builder.Create(context.Background())
		if err == nil {
			t.Error("expected error due to missing input")
		}
	})
}

// TestSummarizationTraceContext_UpdateMethods tests UpdateOutput methods.
func TestSummarizationTraceContext_UpdateMethods(t *testing.T) {
	client, _ := langfusetest.NewTestClient(t)

	builder := NewSummarizationTrace(client, "test-summarization").
		Input("Long article to summarize")

	ctx, err := builder.Create(context.Background())
	if err != nil {
		t.Fatalf("failed to create trace: %v", err)
	}

	t.Run("UpdateOutput", func(t *testing.T) {
		err := ctx.UpdateOutput(context.Background(), "Brief summary")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ctx.GetOutput() == nil {
			t.Fatal("expected output to be set")
		}
		if ctx.GetOutput().Output != "Brief summary" {
			t.Errorf("got output %q, want %q", ctx.GetOutput().Output, "Brief summary")
		}
	})

	t.Run("UpdateOutputWithMetadata", func(t *testing.T) {
		output := &SummarizationOutput{
			Output:           "Detailed summary",
			Length:           10,
			CompressionRatio: 0.1,
			Metadata:         map[string]any{"model": "gpt-4"},
		}
		err := ctx.UpdateOutputWithMetadata(context.Background(), output)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ctx.GetOutput().Output != "Detailed summary" {
			t.Errorf("got output %q, want %q", ctx.GetOutput().Output, "Detailed summary")
		}
	})
}

// TestQATraceBuilder_WrapperMethods tests all wrapper methods on QATraceBuilder.
func TestQATraceBuilder_WrapperMethods(t *testing.T) {
	client, _ := langfusetest.NewTestClient(t)

	builder := NewQATrace(client, "test-qa")

	result := builder.
		ID("trace-id-123").
		UserID("user-456").
		SessionID("session-789").
		Tags([]string{"tag1", "tag2"}).
		Metadata(map[string]any{"key": "value"}).
		Release("v1.0.0").
		Version("1").
		Environment("production").
		Public(true)

	if result != builder {
		t.Error("wrapper methods should return the same builder for chaining")
	}
}

// TestQATraceBuilder_Validate tests validation logic.
func TestQATraceBuilder_Validate(t *testing.T) {
	client, _ := langfusetest.NewTestClient(t)

	tests := []struct {
		name        string
		setup       func(*QATraceBuilder)
		expectError bool
	}{
		{
			name: "valid - has query",
			setup: func(b *QATraceBuilder) {
				b.Query("What is the capital of France?")
			},
			expectError: false,
		},
		{
			name:        "invalid - missing query",
			setup:       func(b *QATraceBuilder) {},
			expectError: true,
		},
		{
			name: "invalid - empty query",
			setup: func(b *QATraceBuilder) {
				b.Query("")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewQATrace(client, "test")
			tt.setup(builder)

			err := builder.Validate()
			if tt.expectError && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

// TestQATraceBuilder_Create tests the Create method.
func TestQATraceBuilder_Create(t *testing.T) {
	client, _ := langfusetest.NewTestClient(t)

	t.Run("success", func(t *testing.T) {
		builder := NewQATrace(client, "test-qa").
			Query("What is the capital of France?").
			GroundTruth("Paris").
			Context("France is a country in Europe.")

		ctx, err := builder.Create(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ctx == nil {
			t.Fatal("expected non-nil context")
		}
		if ctx.GetInput() == nil {
			t.Fatal("expected input to be set")
		}
		if ctx.GetInput().Query != "What is the capital of France?" {
			t.Errorf("got query %q, want %q", ctx.GetInput().Query, "What is the capital of France?")
		}
	})

	t.Run("validation failure", func(t *testing.T) {
		builder := NewQATrace(client, "test-qa")
		// No query set

		_, err := builder.Create(context.Background())
		if err == nil {
			t.Error("expected error due to missing query")
		}
	})
}

// TestQATraceContext_UpdateMethods tests UpdateOutput methods.
func TestQATraceContext_UpdateMethods(t *testing.T) {
	client, _ := langfusetest.NewTestClient(t)

	builder := NewQATrace(client, "test-qa").
		Query("What is the capital of France?")

	ctx, err := builder.Create(context.Background())
	if err != nil {
		t.Fatalf("failed to create trace: %v", err)
	}

	t.Run("UpdateOutput", func(t *testing.T) {
		err := ctx.UpdateOutput(context.Background(), "Paris", 0.95)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ctx.GetOutput() == nil {
			t.Fatal("expected output to be set")
		}
		if ctx.GetOutput().Output != "Paris" {
			t.Errorf("got output %q, want %q", ctx.GetOutput().Output, "Paris")
		}
		if ctx.GetOutput().Confidence != 0.95 {
			t.Errorf("got confidence %f, want %f", ctx.GetOutput().Confidence, 0.95)
		}
	})

	t.Run("UpdateOutputWithMetadata", func(t *testing.T) {
		output := &QAOutput{
			Output:     "Paris, the capital city",
			Confidence: 0.98,
			Reasoning:  "Based on geographic knowledge",
			Metadata:   map[string]any{"model": "gpt-4"},
		}
		err := ctx.UpdateOutputWithMetadata(context.Background(), output)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ctx.GetOutput().Output != "Paris, the capital city" {
			t.Errorf("got output %q, want %q", ctx.GetOutput().Output, "Paris, the capital city")
		}
	})
}

// TestQATraceContext_ValidateForEvaluation_Full tests validation for evaluation.
func TestQATraceContext_ValidateForEvaluation_Full(t *testing.T) {
	client, _ := langfusetest.NewTestClient(t)

	builder := NewQATrace(client, "test-qa").
		Query("What is the capital of France?")

	ctx, err := builder.Create(context.Background())
	if err != nil {
		t.Fatalf("failed to create trace: %v", err)
	}

	t.Run("fails without output", func(t *testing.T) {
		err := ctx.ValidateForEvaluation()
		if err == nil {
			t.Error("expected error due to missing output")
		}
	})

	t.Run("succeeds with output", func(t *testing.T) {
		_ = ctx.UpdateOutput(context.Background(), "Paris", 0.95)
		err := ctx.ValidateForEvaluation()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// TestRAGTraceBuilder_WrapperMethods tests all wrapper methods on RAGTraceBuilder.
func TestRAGTraceBuilder_WrapperMethods(t *testing.T) {
	client, _ := langfusetest.NewTestClient(t)

	builder := NewRAGTrace(client, "test-rag")

	result := builder.
		ID("trace-id-123").
		UserID("user-456").
		SessionID("session-789").
		Tags([]string{"tag1", "tag2"}).
		Metadata(map[string]any{"key": "value"}).
		Release("v1.0.0").
		Version("1").
		Environment("production").
		Public(true)

	if result != builder {
		t.Error("wrapper methods should return the same builder for chaining")
	}
}

// TestRAGTraceBuilder_Validate tests validation logic.
func TestRAGTraceBuilder_Validate(t *testing.T) {
	client, _ := langfusetest.NewTestClient(t)

	tests := []struct {
		name        string
		setup       func(*RAGTraceBuilder)
		expectError bool
	}{
		{
			name: "valid - has query and context",
			setup: func(b *RAGTraceBuilder) {
				b.Query("What is the capital of France?")
				b.Context("France is a country in Europe.", "Paris is the capital of France.")
			},
			expectError: false,
		},
		{
			name:        "invalid - missing query",
			setup:       func(b *RAGTraceBuilder) {},
			expectError: true,
		},
		{
			name: "invalid - missing context",
			setup: func(b *RAGTraceBuilder) {
				b.Query("What is the capital of France?")
			},
			expectError: true,
		},
		{
			name: "invalid - empty query",
			setup: func(b *RAGTraceBuilder) {
				b.Query("")
				b.Context("Some context")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewRAGTrace(client, "test")
			tt.setup(builder)

			err := builder.Validate()
			if tt.expectError && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

// TestRAGTraceBuilder_Create tests the Create method.
func TestRAGTraceBuilder_Create(t *testing.T) {
	client, _ := langfusetest.NewTestClient(t)

	t.Run("success", func(t *testing.T) {
		builder := NewRAGTrace(client, "test-rag").
			Query("What is the capital of France?").
			Context("France is a country in Europe.", "Paris is the capital of France.").
			GroundTruth("Paris").
			AdditionalContext(map[string]any{"source": "wikipedia"})

		ctx, err := builder.Create(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ctx == nil {
			t.Fatal("expected non-nil context")
		}
		if ctx.GetInput() == nil {
			t.Fatal("expected input to be set")
		}
		if ctx.GetInput().Query != "What is the capital of France?" {
			t.Errorf("got query %q, want %q", ctx.GetInput().Query, "What is the capital of France?")
		}
		if len(ctx.GetInput().Context) != 2 {
			t.Errorf("got %d context chunks, want 2", len(ctx.GetInput().Context))
		}
	})

	t.Run("validation failure - missing query", func(t *testing.T) {
		builder := NewRAGTrace(client, "test-rag").
			Context("Some context")
		// No query set

		_, err := builder.Create(context.Background())
		if err == nil {
			t.Error("expected error due to missing query")
		}
	})

	t.Run("validation failure - missing context", func(t *testing.T) {
		builder := NewRAGTrace(client, "test-rag").
			Query("What is the capital of France?")
		// No context set

		_, err := builder.Create(context.Background())
		if err == nil {
			t.Error("expected error due to missing context")
		}
	})
}

// TestRAGTraceContext_UpdateMethods tests UpdateOutput methods.
func TestRAGTraceContext_UpdateMethods(t *testing.T) {
	client, _ := langfusetest.NewTestClient(t)

	builder := NewRAGTrace(client, "test-rag").
		Query("What is the capital of France?").
		Context("France is a country in Europe.", "Paris is the capital of France.")

	ctx, err := builder.Create(context.Background())
	if err != nil {
		t.Fatalf("failed to create trace: %v", err)
	}

	t.Run("UpdateOutput", func(t *testing.T) {
		err := ctx.UpdateOutput(context.Background(), "Paris", "source1", "source2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ctx.GetOutput() == nil {
			t.Fatal("expected output to be set")
		}
		if ctx.GetOutput().Output != "Paris" {
			t.Errorf("got output %q, want %q", ctx.GetOutput().Output, "Paris")
		}
		if len(ctx.GetOutput().Citations) != 2 {
			t.Errorf("got %d citations, want 2", len(ctx.GetOutput().Citations))
		}
	})

	t.Run("UpdateOutputWithMetadata", func(t *testing.T) {
		output := &RAGOutput{
			Output:       "Paris, the capital city",
			Citations:    []string{"Wikipedia", "Encyclopedia"},
			SourceChunks: []int{0, 1},
			Confidence:   0.92,
			Metadata:     map[string]any{"model": "gpt-4"},
		}
		err := ctx.UpdateOutputWithMetadata(context.Background(), output)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ctx.GetOutput().Output != "Paris, the capital city" {
			t.Errorf("got output %q, want %q", ctx.GetOutput().Output, "Paris, the capital city")
		}
	})
}

// TestRAGTraceContext_ValidateForEvaluation_Full tests validation for evaluation.
func TestRAGTraceContext_ValidateForEvaluation_Full(t *testing.T) {
	client, _ := langfusetest.NewTestClient(t)

	builder := NewRAGTrace(client, "test-rag").
		Query("What is the capital of France?").
		Context("France is a country in Europe.")

	ctx, err := builder.Create(context.Background())
	if err != nil {
		t.Fatalf("failed to create trace: %v", err)
	}

	t.Run("fails without output", func(t *testing.T) {
		err := ctx.ValidateForEvaluation()
		if err == nil {
			t.Error("expected error due to missing output")
		}
	})

	t.Run("succeeds with output", func(t *testing.T) {
		_ = ctx.UpdateOutput(context.Background(), "Paris")
		err := ctx.ValidateForEvaluation()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// TestClassificationTraceBuilder_WrapperMethods tests all wrapper methods on ClassificationTraceBuilder.
func TestClassificationTraceBuilder_WrapperMethods(t *testing.T) {
	client, _ := langfusetest.NewTestClient(t)

	builder := NewClassificationTrace(client, "test-classification")

	result := builder.
		ID("trace-id-123").
		UserID("user-456").
		SessionID("session-789").
		Tags([]string{"tag1", "tag2"}).
		Metadata(map[string]any{"key": "value"}).
		Release("v1.0.0").
		Version("1").
		Environment("production").
		Public(true)

	if result != builder {
		t.Error("wrapper methods should return the same builder for chaining")
	}
}

// TestClassificationTraceBuilder_Validate tests validation logic.
func TestClassificationTraceBuilder_Validate(t *testing.T) {
	client, _ := langfusetest.NewTestClient(t)

	tests := []struct {
		name        string
		setup       func(*ClassificationTraceBuilder)
		expectError bool
	}{
		{
			name: "valid - has input",
			setup: func(b *ClassificationTraceBuilder) {
				b.Input("This is a spam email")
			},
			expectError: false,
		},
		{
			name:        "invalid - missing input",
			setup:       func(b *ClassificationTraceBuilder) {},
			expectError: true,
		},
		{
			name: "invalid - empty input",
			setup: func(b *ClassificationTraceBuilder) {
				b.Input("")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewClassificationTrace(client, "test")
			tt.setup(builder)

			err := builder.Validate()
			if tt.expectError && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

// TestClassificationTraceBuilder_Create tests the Create method.
func TestClassificationTraceBuilder_Create(t *testing.T) {
	client, _ := langfusetest.NewTestClient(t)

	t.Run("success", func(t *testing.T) {
		builder := NewClassificationTrace(client, "test-classification").
			Input("This is a spam email").
			Classes([]string{"spam", "ham"}).
			GroundTruth("spam")

		ctx, err := builder.Create(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ctx == nil {
			t.Fatal("expected non-nil context")
		}
		if ctx.GetInput() == nil {
			t.Fatal("expected input to be set")
		}
		if ctx.GetInput().Input != "This is a spam email" {
			t.Errorf("got input %q, want %q", ctx.GetInput().Input, "This is a spam email")
		}
	})

	t.Run("validation failure", func(t *testing.T) {
		builder := NewClassificationTrace(client, "test-classification")
		// No input set

		_, err := builder.Create(context.Background())
		if err == nil {
			t.Error("expected error due to missing input")
		}
	})
}

// TestClassificationTraceContext_UpdateMethods tests UpdateOutput methods.
func TestClassificationTraceContext_UpdateMethods(t *testing.T) {
	client, _ := langfusetest.NewTestClient(t)

	builder := NewClassificationTrace(client, "test-classification").
		Input("This is a spam email")

	ctx, err := builder.Create(context.Background())
	if err != nil {
		t.Fatalf("failed to create trace: %v", err)
	}

	t.Run("UpdateOutput", func(t *testing.T) {
		err := ctx.UpdateOutput(context.Background(), "spam", 0.95)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ctx.GetOutput() == nil {
			t.Fatal("expected output to be set")
		}
		if ctx.GetOutput().Output != "spam" {
			t.Errorf("got output %q, want %q", ctx.GetOutput().Output, "spam")
		}
		if ctx.GetOutput().Confidence != 0.95 {
			t.Errorf("got confidence %f, want %f", ctx.GetOutput().Confidence, 0.95)
		}
	})

	t.Run("UpdateOutputWithScores", func(t *testing.T) {
		scores := map[string]float64{"spam": 0.95, "ham": 0.05}
		err := ctx.UpdateOutputWithScores(context.Background(), "spam", scores)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ctx.GetOutput().Confidence != 0.95 {
			t.Errorf("got confidence %f, want %f", ctx.GetOutput().Confidence, 0.95)
		}
	})

	t.Run("UpdateOutputWithMetadata", func(t *testing.T) {
		output := &ClassificationOutput{
			Output:     "spam",
			Confidence: 0.98,
			Scores:     map[string]float64{"spam": 0.98, "ham": 0.02},
			Metadata:   map[string]any{"model": "gpt-4"},
		}
		err := ctx.UpdateOutputWithMetadata(context.Background(), output)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ctx.GetOutput().Output != "spam" {
			t.Errorf("got output %q, want %q", ctx.GetOutput().Output, "spam")
		}
	})
}

// TestClassificationTraceContext_ValidateForEvaluation_Full tests validation for evaluation.
func TestClassificationTraceContext_ValidateForEvaluation_Full(t *testing.T) {
	client, _ := langfusetest.NewTestClient(t)

	builder := NewClassificationTrace(client, "test-classification").
		Input("This is a spam email")

	ctx, err := builder.Create(context.Background())
	if err != nil {
		t.Fatalf("failed to create trace: %v", err)
	}

	t.Run("fails without output", func(t *testing.T) {
		err := ctx.ValidateForEvaluation()
		if err == nil {
			t.Error("expected error due to missing output")
		}
	})

	t.Run("succeeds with output", func(t *testing.T) {
		_ = ctx.UpdateOutput(context.Background(), "spam", 0.95)
		err := ctx.ValidateForEvaluation()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// TestNewTraceBuilders tests the constructor functions.
func TestNewTraceBuilders(t *testing.T) {
	client, _ := langfusetest.NewTestClient(t)

	t.Run("NewSummarizationTrace", func(t *testing.T) {
		builder := NewSummarizationTrace(client, "test-name")
		if builder == nil {
			t.Fatal("expected non-nil builder")
		}
		if builder.sumInput == nil {
			t.Error("expected sumInput to be initialized")
		}
	})

	t.Run("NewQATrace", func(t *testing.T) {
		builder := NewQATrace(client, "test-name")
		if builder == nil {
			t.Fatal("expected non-nil builder")
		}
		if builder.qaInput == nil {
			t.Error("expected qaInput to be initialized")
		}
	})

	t.Run("NewRAGTrace", func(t *testing.T) {
		builder := NewRAGTrace(client, "test-name")
		if builder == nil {
			t.Fatal("expected non-nil builder")
		}
		if builder.ragInput == nil {
			t.Error("expected ragInput to be initialized")
		}
	})

	t.Run("NewClassificationTrace", func(t *testing.T) {
		builder := NewClassificationTrace(client, "test-name")
		if builder == nil {
			t.Fatal("expected non-nil builder")
		}
		if builder.classInput == nil {
			t.Error("expected classInput to be initialized")
		}
	})
}

// TestBuilderSpecificMethods tests domain-specific methods that set input fields.
func TestBuilderSpecificMethods(t *testing.T) {
	client, _ := langfusetest.NewTestClient(t)

	t.Run("SummarizationTraceBuilder", func(t *testing.T) {
		builder := NewSummarizationTrace(client, "test").
			Input("Long article").
			GroundTruth("Expected summary").
			MaxLength(100).
			Style("bullet_points")

		if builder.sumInput.Input != "Long article" {
			t.Errorf("Input = %q, want %q", builder.sumInput.Input, "Long article")
		}
		if builder.sumInput.GroundTruth != "Expected summary" {
			t.Errorf("GroundTruth = %q, want %q", builder.sumInput.GroundTruth, "Expected summary")
		}
		if builder.sumInput.MaxLength != 100 {
			t.Errorf("MaxLength = %d, want %d", builder.sumInput.MaxLength, 100)
		}
		if builder.sumInput.Style != "bullet_points" {
			t.Errorf("Style = %q, want %q", builder.sumInput.Style, "bullet_points")
		}
	})

	t.Run("QATraceBuilder", func(t *testing.T) {
		builder := NewQATrace(client, "test").
			Query("What is X?").
			GroundTruth("Y").
			Context("Background info")

		if builder.qaInput.Query != "What is X?" {
			t.Errorf("Query = %q, want %q", builder.qaInput.Query, "What is X?")
		}
		if builder.qaInput.GroundTruth != "Y" {
			t.Errorf("GroundTruth = %q, want %q", builder.qaInput.GroundTruth, "Y")
		}
		if builder.qaInput.Context != "Background info" {
			t.Errorf("Context = %q, want %q", builder.qaInput.Context, "Background info")
		}
	})

	t.Run("RAGTraceBuilder", func(t *testing.T) {
		builder := NewRAGTrace(client, "test").
			Query("What is X?").
			GroundTruth("Y").
			Context("Chunk1", "Chunk2").
			AdditionalContext(map[string]any{"key": "value"})

		if builder.ragInput.Query != "What is X?" {
			t.Errorf("Query = %q, want %q", builder.ragInput.Query, "What is X?")
		}
		if builder.ragInput.GroundTruth != "Y" {
			t.Errorf("GroundTruth = %q, want %q", builder.ragInput.GroundTruth, "Y")
		}
		if len(builder.ragInput.Context) != 2 {
			t.Errorf("Context length = %d, want %d", len(builder.ragInput.Context), 2)
		}
		if builder.ragInput.AdditionalContext["key"] != "value" {
			t.Error("AdditionalContext not set correctly")
		}
	})

	t.Run("ClassificationTraceBuilder", func(t *testing.T) {
		builder := NewClassificationTrace(client, "test").
			Input("Text to classify").
			Classes([]string{"spam", "ham"}).
			GroundTruth("spam")

		if builder.classInput.Input != "Text to classify" {
			t.Errorf("Input = %q, want %q", builder.classInput.Input, "Text to classify")
		}
		if len(builder.classInput.Classes) != 2 {
			t.Errorf("Classes length = %d, want %d", len(builder.classInput.Classes), 2)
		}
		if builder.classInput.GroundTruth != "spam" {
			t.Errorf("GroundTruth = %q, want %q", builder.classInput.GroundTruth, "spam")
		}
	})
}

// TestTraceContext_GetMethods tests Get methods on trace contexts.
func TestTraceContext_GetMethods(t *testing.T) {
	t.Run("QATraceContext", func(t *testing.T) {
		ctx := &QATraceContext{
			input: &QAInput{Query: "test query"},
		}
		if ctx.GetInput().Query != "test query" {
			t.Errorf("GetInput().Query = %q, want %q", ctx.GetInput().Query, "test query")
		}
		if ctx.GetOutput() != nil {
			t.Error("GetOutput() should be nil initially")
		}
	})

	t.Run("RAGTraceContext", func(t *testing.T) {
		ctx := &RAGTraceContext{
			input: &RAGInput{Query: "test query"},
		}
		if ctx.GetInput().Query != "test query" {
			t.Errorf("GetInput().Query = %q, want %q", ctx.GetInput().Query, "test query")
		}
		if ctx.GetOutput() != nil {
			t.Error("GetOutput() should be nil initially")
		}
	})

	t.Run("ClassificationTraceContext", func(t *testing.T) {
		ctx := &ClassificationTraceContext{
			input: &ClassificationInput{Input: "test input"},
		}
		if ctx.GetInput().Input != "test input" {
			t.Errorf("GetInput().Input = %q, want %q", ctx.GetInput().Input, "test input")
		}
		if ctx.GetOutput() != nil {
			t.Error("GetOutput() should be nil initially")
		}
	})
}
