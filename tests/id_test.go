package langfuse_test

import (
	"sync"
	"testing"

	"github.com/jdziat/langfuse-go"
)

func TestIDGenerator_Generate(t *testing.T) {
	gen := langfuse.NewIDGenerator(&langfuse.IDGeneratorConfig{
		Mode: langfuse.IDModeFallback,
	})

	id, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if id == "" {
		t.Error("Generate() returned empty string")
	}

	// Should be a valid UUID format (or fallback format)
	if len(id) != 36 && !langfuse.IsFallbackID(id) {
		t.Errorf("Generate() = %q, unexpected format", id)
	}
}

func TestIDGenerator_GenerateUnique(t *testing.T) {
	gen := langfuse.NewIDGenerator(&langfuse.IDGeneratorConfig{
		Mode: langfuse.IDModeFallback,
	})

	seen := make(map[string]bool)
	const count = 1000

	for i := 0; i < count; i++ {
		id, err := gen.Generate()
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		if seen[id] {
			t.Errorf("Generate() returned duplicate ID: %s", id)
		}
		seen[id] = true
	}
}

func TestIDGenerator_MustGenerate(t *testing.T) {
	gen := langfuse.NewIDGenerator(&langfuse.IDGeneratorConfig{
		Mode: langfuse.IDModeFallback,
	})

	// Should not panic in fallback mode
	id := gen.MustGenerate()
	if id == "" {
		t.Error("MustGenerate() returned empty string")
	}
}

func TestIDGenerator_ConcurrentGenerate(t *testing.T) {
	gen := langfuse.NewIDGenerator(&langfuse.IDGeneratorConfig{
		Mode: langfuse.IDModeFallback,
	})

	var wg sync.WaitGroup
	const goroutines = 10
	const iterations = 100

	ids := make(chan string, goroutines*iterations)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				id, err := gen.Generate()
				if err != nil {
					t.Errorf("Generate() error = %v", err)
					return
				}
				ids <- id
			}
		}()
	}

	wg.Wait()
	close(ids)

	// Check for duplicates
	seen := make(map[string]bool)
	for id := range ids {
		if seen[id] {
			t.Errorf("Generate() returned duplicate ID: %s", id)
		}
		seen[id] = true
	}

	if len(seen) != goroutines*iterations {
		t.Errorf("Generated %d unique IDs, want %d", len(seen), goroutines*iterations)
	}
}

func TestIDGenerator_FallbackIDFormat(t *testing.T) {
	// Test that IsFallbackID works correctly on various formats
	tests := []struct {
		id   string
		want bool
	}{
		{"fb-1234567890abc-00000001-1234", true},
		{"fb-abc-def-123", true},
		{"fb-", false}, // Too short (only "fb-", nothing after)
		{"fb", false},
		{"", false},
		{"123e4567-e89b-12d3-a456-426614174000", false}, // UUID
	}

	for _, tt := range tests {
		if got := langfuse.IsFallbackID(tt.id); got != tt.want {
			t.Errorf("IsFallbackID(%q) = %v, want %v", tt.id, got, tt.want)
		}
	}
}

func TestIDGenerator_FallbackIDUnique(t *testing.T) {
	// Test that generated IDs are unique using the public API
	gen := langfuse.NewIDGenerator(&langfuse.IDGeneratorConfig{
		Mode: langfuse.IDModeFallback,
	})

	seen := make(map[string]bool)
	const count = 1000

	for i := 0; i < count; i++ {
		id, err := gen.Generate()
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		if seen[id] {
			t.Errorf("Generate() returned duplicate ID: %s", id)
		}
		seen[id] = true
	}
}

func TestIDGenerator_Stats(t *testing.T) {
	langfuse.ResetCryptoFailureCount()

	gen := langfuse.NewIDGenerator(&langfuse.IDGeneratorConfig{
		Mode: langfuse.IDModeFallback,
	})

	stats := gen.Stats()
	if stats.Mode != langfuse.IDModeFallback {
		t.Errorf("Stats().Mode = %v, want %v", stats.Mode, langfuse.IDModeFallback)
	}

	// CryptoFailures should be 0 if crypto/rand is working
	if stats.CryptoFailures < 0 {
		t.Errorf("Stats().CryptoFailures = %d, want >= 0", stats.CryptoFailures)
	}
}

func TestIsFallbackID(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"fb-123456789-00000001-1234", true},
		{"fb-", false}, // Too short
		{"", false},
		{"123e4567-e89b-12d3-a456-426614174000", false}, // UUID
		{"not-a-fallback", false},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			if got := langfuse.IsFallbackID(tt.id); got != tt.want {
				t.Errorf("IsFallbackID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestIDGenerationMode_String(t *testing.T) {
	tests := []struct {
		mode langfuse.IDGenerationMode
		want string
	}{
		{langfuse.IDModeFallback, "fallback"},
		{langfuse.IDModeStrict, "strict"},
		{langfuse.IDGenerationMode(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateID_Package(t *testing.T) {
	id, err := langfuse.GenerateID()
	if err != nil {
		t.Fatalf("GenerateID() error = %v", err)
	}

	if id == "" {
		t.Error("GenerateID() returned empty string")
	}
}

func TestMustGenerateID_Package(t *testing.T) {
	id := langfuse.MustGenerateID()
	if id == "" {
		t.Error("MustGenerateID() returned empty string")
	}
}

func TestCryptoFailureCount(t *testing.T) {
	// Reset and check initial state
	langfuse.ResetCryptoFailureCount()

	count := langfuse.CryptoFailureCount()
	if count != 0 {
		t.Errorf("CryptoFailureCount() after reset = %d, want 0", count)
	}
}

func BenchmarkIDGenerator_Generate(b *testing.B) {
	gen := langfuse.NewIDGenerator(&langfuse.IDGeneratorConfig{
		Mode: langfuse.IDModeFallback,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = gen.Generate()
	}
}

func BenchmarkIDGenerator_GenerateFallback(b *testing.B) {
	// Benchmark using the public API in fallback mode
	gen := langfuse.NewIDGenerator(&langfuse.IDGeneratorConfig{
		Mode: langfuse.IDModeFallback,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = gen.Generate()
	}
}

func BenchmarkIDGenerator_GenerateConcurrent(b *testing.B) {
	gen := langfuse.NewIDGenerator(&langfuse.IDGeneratorConfig{
		Mode: langfuse.IDModeFallback,
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = gen.Generate()
		}
	})
}
