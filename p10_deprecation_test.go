package langfuse

import (
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io/fs"
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
// (the Validated* builder hierarchy) carries "Deprecated:" doc markers that
// redirect to the canonical fluent builders. This guards acceptance criterion 1:
// the named symbols stay marked so staticcheck steers callers to the canonical
// path. It fails if a marker is dropped.
func TestRedundantSurfaceIsDeprecated(t *testing.T) {
	dpkg := parsePackageDoc(t)

	validatedConstructors := []string{
		"NewValidatedTraceBuilder",
		"NewValidatedSpanBuilder",
		"NewValidatedGenerationBuilder",
		"NewValidatedScoreBuilder",
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
