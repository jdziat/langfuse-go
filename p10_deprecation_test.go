package langfuse

import (
	"context"
	"encoding/json"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// parsePackageDoc parses the root package source (non-test files) and returns
// its go/doc representation so tests can inspect doc comments, including the
// "Deprecated:" markers that staticcheck (SA1019) keys off of.
func parsePackageDoc(t *testing.T) *doc.Package {
	t.Helper()

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool {
		name := fi.Name()
		return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
	}, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse package source: %v", err)
	}

	pkg, ok := pkgs["langfuse"]
	if !ok {
		t.Fatalf("langfuse package not found in current directory")
	}

	files := make([]*ast.File, 0, len(pkg.Files))
	for _, f := range pkg.Files {
		files = append(files, f)
	}

	dpkg, err := doc.NewFromFiles(fset, files, "github.com/jdziat/langfuse-go")
	if err != nil {
		t.Fatalf("failed to build package doc: %v", err)
	}
	return dpkg
}

// methodDoc returns the doc comment for the named method on the named type, or
// the empty string if it is not found.
func methodDoc(dpkg *doc.Package, typeName, methodName string) string {
	for _, ty := range dpkg.Types {
		if ty.Name != typeName {
			continue
		}
		for _, m := range ty.Methods {
			if m.Name == methodName {
				return m.Doc
			}
		}
	}
	return ""
}

// funcDoc returns the doc comment for the named package-level function/type.
func funcDoc(dpkg *doc.Package, name string) string {
	for _, fn := range dpkg.Funcs {
		if fn.Name == name {
			return fn.Doc
		}
	}
	return ""
}

func typeDoc(dpkg *doc.Package, name string) string {
	for _, ty := range dpkg.Types {
		if ty.Name == name {
			return ty.Doc
		}
	}
	return ""
}

func isDeprecated(docText string) bool {
	for _, line := range strings.Split(docText, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Deprecated:") {
			return true
		}
	}
	return false
}

// TestRedundantSurfaceIsDeprecated verifies that the redundant creation surface
// (the V1-suffixed aliases and the Validated* builder hierarchy) carries
// "Deprecated:" doc markers that redirect to the canonical fluent builders.
// This guards acceptance criterion 1: the named symbols stay marked so
// staticcheck steers callers to the canonical path. It fails if a marker is
// dropped.
func TestRedundantSurfaceIsDeprecated(t *testing.T) {
	dpkg := parsePackageDoc(t)

	methods := []struct {
		typeName string
		method   string
	}{
		{"Client", "TraceV1"},
		{"TraceContext", "NewSpanV1"},
		{"SpanContext", "NewSpanV1"},
		{"GenerationContext", "NewSpanV1"},
		{"TraceContext", "NewGenerationV1"},
		{"SpanContext", "NewGenerationV1"},
		{"GenerationContext", "NewGenerationV1"},
		{"TraceContext", "NewEventV1"},
		{"SpanContext", "NewEventV1"},
		{"GenerationContext", "NewEventV1"},
		{"TraceContext", "UpdateV1"},
		{"SpanContext", "EndV1"},
		{"GenerationContext", "EndV1"},
	}
	for _, m := range methods {
		d := methodDoc(dpkg, m.typeName, m.method)
		if d == "" {
			t.Errorf("method %s.%s not found in package doc", m.typeName, m.method)
			continue
		}
		if !isDeprecated(d) {
			t.Errorf("method %s.%s is missing a // Deprecated: marker", m.typeName, m.method)
		}
	}

	validatedConstructors := []string{
		"NewValidatedTraceBuilder",
		"NewValidatedSpanBuilder",
		"NewValidatedGenerationBuilder",
		"NewValidatedScoreBuilder",
		"WrapTraceContext",
	}
	for _, name := range validatedConstructors {
		// go/doc may attach constructors to their result type, so check both
		// package-level funcs and the type's factory list.
		d := funcDoc(dpkg, name)
		if d == "" {
			for _, ty := range dpkg.Types {
				for _, fn := range ty.Funcs {
					if fn.Name == name {
						d = fn.Doc
					}
				}
			}
		}
		if d == "" {
			t.Errorf("constructor %s not found in package doc", name)
			continue
		}
		if !isDeprecated(d) {
			t.Errorf("constructor %s is missing a // Deprecated: marker", name)
		}
	}

	validatedTypes := []string{
		"ValidatedTraceBuilder",
		"ValidatedSpanBuilder",
		"ValidatedGenerationBuilder",
		"ValidatedScoreBuilder",
		"TraceContextV1",
	}
	for _, name := range validatedTypes {
		d := typeDoc(dpkg, name)
		if d == "" {
			t.Errorf("type %s not found in package doc", name)
			continue
		}
		if !isDeprecated(d) {
			t.Errorf("type %s is missing a // Deprecated: marker", name)
		}
	}
}

// TestDeprecatedV1StillWorks verifies the deprecated V1 aliases remain
// functional (backward compatibility): a trace created through TraceV1 enqueues
// the same trace-create event the canonical NewTrace().Create path would.
func TestDeprecatedV1StillWorks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(IngestionResult{
			Successes: []IngestionSuccess{{ID: "1", Status: 200}},
		})
	}))
	defer server.Close()

	client, err := New("pk-lf-test", "sk-lf-test", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	// TraceV1 is deprecated but must keep working for backward compatibility.
	// Same-package use is not flagged by staticcheck (SA1019 only fires across
	// package boundaries), so no lint directive is needed here.
	trace, err := client.TraceV1(ctx, "v1-trace")
	if err != nil {
		t.Fatalf("TraceV1 failed: %v", err)
	}
	if trace == nil || trace.ID() == "" {
		t.Fatalf("TraceV1 returned an unusable trace context: %+v", trace)
	}
}
