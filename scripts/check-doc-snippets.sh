#!/usr/bin/env bash
#
# check-doc-snippets.sh — compile the Go code examples embedded in the docs.
#
# This gate guards against "doc rot": documentation snippets that drift out of
# sync with the real public API (wrong method names, removed options, fictional
# call chains, missing context arguments, ...). It extracts every Go code block
# from the documentation sources, assembles each into a compilable unit, and
# runs `go build` against a throwaway module that points at the local copy of
# this module via a `replace` directive. If a snippet fails to compile the
# script exits non-zero and names the offending doc + block.
#
# Sources scanned:
#   - README.md
#   - doc.go            (the tab-indented code blocks inside its // comment)
#   - **/doc.go         (subpackage package docs: evaluation/, langfusetest/,
#                        pkg/*/ — same tab-indented comment-block extraction)
#   - content/docs/**.md
#
# Convention (so authors know what is checked and how to opt out):
#
#   ```go
#   package main
#   ...
#   ```
#       A block whose first non-blank line begins with "package " is treated as
#       a COMPLETE, self-contained program/file. It is compiled verbatim. These
#       are the load-bearing examples (e.g. the Quick Start) and a compile
#       failure here FAILS the gate.
#
#   ```go
#   trace, err := client.NewTrace().Name("x").Create(ctx)
#   ...
#   ```
#       A block that does NOT declare its own package is treated as an
#       illustrative FRAGMENT — an excerpt that references variables (client,
#       trace, ctx, generation, span, ...) defined in the surrounding prose.
#       The gate synthesises a compilable unit around it: a `package main`
#       clause, a documented import block, and a wrapper function that supplies
#       the commonly-referenced identifiers as pre-typed scaffold parameters
#       (so the body can read/assign them) with the body nested in its own
#       block (so `:=` redeclarations shadow the scaffolds cleanly). The
#       fragment is then BUILT. A fragment that does not compile FAILS the gate
#       just like a full program — this is the whole point: a renamed method or
#       a dropped option in an excerpt is a real doc bug.
#
#       Scaffold identifiers available to every fragment (assign or shadow them
#       freely):
#         ctx        context.Context
#         client     *langfuse.Client
#         trace      *langfuse.TraceContext
#         generation *langfuse.GenerationContext
#         span       *langfuse.SpanContext
#         meta       langfuse.Metadata
#         usage      *langfuse.Usage
#         publicKey  string
#         secretKey  string
#         t          *testing.T
#         err        error
#
#       Fragments may also reference the SDK subpackages by their natural import
#       names: langfusetest, builders, id, and pkgerrors (the pkg/errors helper
#       package, aliased to avoid colliding with the stdlib errors package).
#       stdErrors is the standard library errors package; errors is too, kept
#       for the historical fragments that call errors.Is/errors.As.
#
#   ```go-nocompile
#   ...
#   ```
#       The explicit, intentional opt-out. Use it ONLY for blocks that are not
#       meant to compile: cheat-sheets that list bare method/constant names
#       (e.g. `meta.Set(key, value)` or `langfuse.RegionEU` with no surrounding
#       statement), pseudo-code, or deliberately-broken "don't do this"
#       examples. Tagged blocks are skipped and never fail the gate. Keep this
#       set as small as possible: prefer turning an excerpt into something that
#       compiles over hiding it behind `go-nocompile`. The info strings
#       `go-nocompile` and `go,nocompile` are both recognised.
#
# Tooling: only `go` and standard POSIX/bash utilities (awk, mktemp, ...). No
# network access beyond the local module is required.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

MODULE_PATH="$(go list -m 2>/dev/null || echo "github.com/jdziat/langfuse-go")"

# Documentation sources to scan. Globs are expanded below; missing files are
# tolerated so the script keeps working as docs are added/removed.
DOC_GLOBS=(
	"README.md"
	"doc.go"
	"content/docs"/*.md
)
# Subpackage doc.go files carry godoc examples too (evaluation/doc.go,
# langfusetest/doc.go, pkg/*/doc.go). The gate historically scanned only the
# root doc.go, which let those subpackage examples rot undetected. Discover
# them here (everything ending in /doc.go, i.e. NOT the already-listed root
# doc.go) and route them through the same comment-block extractor below.
while IFS= read -r f; do
	DOC_GLOBS+=("$f")
done < <(find . -type f -name doc.go -not -path './doc.go' | sed 's|^\./||' | sort)

WORKDIR="$(mktemp -d 2>/dev/null || mktemp -d -t langfuse-doc-snippets)"
# shellcheck disable=SC2329  # invoked indirectly via the EXIT trap below.
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# Scratch module that requires the local module via a replace directive so the
# snippets compile against the working tree (not a published release).
SNIPPET_MODULE="langfuse-go-doc-snippets"
{
	echo "module $SNIPPET_MODULE"
	echo ""
	echo "go 1.23"
	echo ""
	echo "require $MODULE_PATH v0.0.0"
	echo ""
	echo "replace $MODULE_PATH => $REPO_ROOT"
} >"$WORKDIR/go.mod"

# Resolve the dependency graph for the scratch module once.
( cd "$WORKDIR" && go mod tidy >/dev/null 2>&1 ) || true

# extract_markdown_blocks <file> writes the code blocks found in <file> to
# numbered files under $WORKDIR, one file per (source, block). It records, per
# block, the source location, the 1-based block index, the fence kind
# (go|go-nocompile) and whether the block looks self-contained (has a package
# clause). Metadata lines are appended to $WORKDIR/index.
extract_markdown_blocks() {
	local src="$1"
	awk -v src="$src" -v workdir="$WORKDIR" '
		function flush() {
			if (kind == "") return
			n++
			path = workdir "/block_" recid "_" n ".txt"
			printf "%s", buf > path
			close(path)
			full = (firstcode ~ /^package /) ? 1 : 0
			print src "\t" n "\t" kind "\t" full "\t" path >> (workdir "/index")
			kind = ""; buf = ""; firstcode = ""
		}
		BEGIN { n = 0; kind = "" }
		# Opening fence for a go (or go-nocompile) block.
		/^[[:space:]]*```go([,-]nocompile)?[[:space:]]*$/ {
			# Determine fence kind from the info string.
			line = $0
			sub(/^[[:space:]]*```/, "", line)
			if (line ~ /nocompile/) kind = "go-nocompile"; else kind = "go"
			inblk = 1; buf = ""; firstcode = ""
			next
		}
		/^[[:space:]]*```/ {
			if (inblk) { inblk = 0; flush() }
			next
		}
		inblk {
			if (firstcode == "" && $0 ~ /[^[:space:]]/) firstcode = $0
			buf = buf $0 "\n"
		}
	' recid="$(basename "$src" | tr -c 'a-zA-Z0-9' '_')" "$src"
}

# extract_docgo_blocks reads doc.go and turns each tab-indented code block that
# appears inside the leading // comment into a block file. godoc renders an
# indented run of comment lines as a code block, so we treat a maximal run of
# "//\t..." lines (allowing blank "//" separators inside) as one block.
extract_docgo_blocks() {
	local src="${1:-doc.go}"
	[ -f "$src" ] || return 0
	# Stable, filesystem-safe record id derived from the source path so blocks
	# from different doc.go files never collide (e.g. evaluation_doc_go).
	local recid
	recid="$(printf '%s' "$src" | tr -c 'a-zA-Z0-9' '_')"
	awk -v src="$src" -v workdir="$WORKDIR" -v recid="$recid" '
		function flush() {
			if (buf == "") return
			n++
			path = workdir "/block_" recid "_" n ".txt"
			printf "%s", buf > path
			close(path)
			gsub(/^[[:space:]]+/, "", firstcode)
			full = (firstcode ~ /^package /) ? 1 : 0
			print src "\t" n "\tgo\t" full "\t" path >> (workdir "/index")
			buf = ""; firstcode = ""; incode = 0
		}
		BEGIN { n = 0; incode = 0 }
		# Stop at the package clause; comments after it are not the doc comment.
		/^package / { flush(); exit }
		{
			line = $0
			# Only consider // comment lines.
			if (line !~ /^\/\//) { flush(); next }
			# Strip the leading "//".
			sub(/^\/\//, "", line)
			# In a gofmt-formatted doc comment, a code block is indented with a
			# TAB ("//\t..."); space-indented runs ("//   - ..." / "//   1. ...")
			# are markdown-style bullet/number lists, i.e. prose, NOT code. Only
			# tab-indented lines start/continue a code block, so the bullet
			# lists are left to the prose branch below.
			if (line ~ /^\t/) {
				if (firstcode == "") firstcode = line
				sub(/^\t/, "", line)
				buf = buf line "\n"
				incode = 1
				next
			}
			# A blank comment line keeps an in-progress code block together.
			if (line ~ /^[[:space:]]*$/ && incode) {
				buf = buf "\n"
				next
			}
			# Any other prose line terminates the current code block.
			flush()
		}
		END { flush() }
	' "$src"
}

: >"$WORKDIR/index"
for g in "${DOC_GLOBS[@]}"; do
	[ -e "$g" ] || continue
	if [ "$g" = "doc.go" ] || [ "${g%/doc.go}" != "$g" ]; then
		extract_docgo_blocks "$g"
	else
		extract_markdown_blocks "$g"
	fi
done

# wrap_fragment composes a buildable Go file for a fragment block. It adds a
# package clause, the documented import block (the SDK aliased "langfuse",
# matching the docs, plus the SDK subpackages the examples reference —
# evaluation, langfusetest, builders, id, pkg/errors aliased "pkgerrors" — and
# the handful of stdlib packages the examples lean on), and a wrapper function
# that exposes the common scaffold identifiers as PARAMETERS. Parameters are never flagged
# "declared and not used", and nesting the fragment body in its own block lets
# `ctx := ...` / `trace, err := ...` shadow the scaffolds without tripping the
# "no new variables on left side of :=" rule. Unused imports are tolerated via
# a blank reference block; the Go compiler still rejects genuinely broken code
# (unknown methods, removed options, type errors, bad syntax) — which is the
# point of the gate.
#
# Excerpts also routinely DECLARE a variable and leave it for "later" prose
# (e.g. `client, err := langfuse.New(...)` with no following use). Go rejects
# unused locals, so the harness scans the body for the names introduced by
# top-level `:=` declarations and emits a trailing `_ = name` for each, marking
# them used. Only declarations at the fragment's own base indent are consumed
# (names bound inside a nested `if`/`for`/`{}` block stay in their own scope and
# are referenced — or not — by the body itself).

# fragment_decl_names <body_file> prints, one per line, the identifiers bound by
# top-level short-variable declarations in the fragment body.
fragment_decl_names() {
	awk '
		BEGIN { base = -1 }
		{
			line = $0
			t = line; sub(/^[[:space:]]+/, "", t)
			if (t == "" || t ~ /^\/\//) next
			# Indent of this line (tabs count as one column, like spaces).
			ind = 0
			for (i = 1; i <= length(line); i++) {
				c = substr(line, i, 1)
				if (c == " " || c == "\t") ind++; else break
			}
			if (base < 0) base = ind
			# Only top-level statements of the fragment declare reusable names.
			if (ind != base) next
			# Control-flow / block forms bind into their own scope; skip them.
			if (t ~ /^(if|for|switch|select|go|defer|return|case|default|}|{)/) next
			# Match a leading "a, b, c :=" short declaration.
			if (match(t, /^[a-zA-Z_][a-zA-Z0-9_]*([[:space:]]*,[[:space:]]*[a-zA-Z_][a-zA-Z0-9_]*)*[[:space:]]*:=/)) {
				lhs = substr(t, 1, RLENGTH)
				sub(/:=$/, "", lhs); gsub(/[[:space:]]/, "", lhs)
				n = split(lhs, arr, ",")
				for (i = 1; i <= n; i++) if (arr[i] != "" && arr[i] != "_") print arr[i]
			}
		}
	' "$1" | sort -u
}

wrap_fragment() {
	local body_file="$1" out_file="$2"
	{
		echo "package main"
		echo ""
		echo "import ("
		echo "	\"context\""
		echo "	\"errors\""
		echo "	stdErrors \"errors\""
		echo "	\"fmt\""
		echo "	\"log\""
		echo "	\"net/http\""
		echo "	\"os\""
		echo "	\"testing\""
		echo "	\"time\""
		echo ""
		echo "	langfuse \"$MODULE_PATH\""
		echo "	\"$MODULE_PATH/evaluation\""
		echo "	\"$MODULE_PATH/langfusetest\""
		echo "	\"$MODULE_PATH/pkg/builders\""
		echo "	pkgerrors \"$MODULE_PATH/pkg/errors\""
		echo "	\"$MODULE_PATH/pkg/id\""
		echo ")"
		echo ""
		echo "// Reference imports so unused ones do not break the build; the"
		echo "// snippet body below is what is actually being type-checked."
		echo "var ("
		echo "	_ = context.Background"
		echo "	_ = errors.Is"
		echo "	_ = stdErrors.Is"
		echo "	_ = fmt.Sprint"
		echo "	_ = log.Print"
		echo "	_ = http.DefaultClient"
		echo "	_ = os.Getenv"
		echo "	_ = time.Second"
		echo "	_ = langfuse.New"
		echo "	_ = evaluation.NewQATrace"
		echo "	_ = langfusetest.NewMockServer"
		echo "	_ = builders.BuildMetadata"
		echo "	_ = pkgerrors.AsAPIError"
		echo "	_ = id.GenerateID"
		echo ")"
		echo ""
		echo "func main() {}"
		echo ""
		echo "// docSnippet hosts a single documentation fragment. The common"
		echo "// identifiers referenced by excerpts are supplied as scaffold"
		echo "// parameters; the fragment body runs in a nested block so its own"
		echo "// short variable declarations shadow the scaffolds cleanly."
		echo "func docSnippet("
		echo "	ctx context.Context,"
		echo "	client *langfuse.Client,"
		echo "	trace *langfuse.TraceContext,"
		echo "	generation *langfuse.GenerationContext,"
		echo "	span *langfuse.SpanContext,"
		echo "	meta langfuse.Metadata,"
		echo "	usage *langfuse.Usage,"
		echo "	publicKey string,"
		echo "	secretKey string,"
		echo "	t *testing.T,"
		echo "	err error,"
		echo ") {"
		echo "	_ = ctx"
		echo "	_ = client"
		echo "	_ = trace"
		echo "	_ = generation"
		echo "	_ = span"
		echo "	_ = meta"
		echo "	_ = usage"
		echo "	_ = publicKey"
		echo "	_ = secretKey"
		echo "	_ = t"
		echo "	_ = err"
		echo "	{"
		cat "$body_file"
		# Mark every top-level-declared name as used so an excerpt that only
		# binds a variable (leaving its use to the surrounding prose) still
		# compiles. A genuinely broken right-hand side still fails to type-check.
		local name
		fragment_decl_names "$body_file" | while IFS= read -r name; do
			[ -n "$name" ] || continue
			echo "		_ = $name"
		done
		echo "	}"
		echo "}"
	} >"$out_file"
}

total=0
compiled=0
skipped=0
failures=0
fail_list=""

while IFS=$'\t' read -r src idx kind full path; do
	[ -n "${src:-}" ] || continue
	total=$((total + 1))
	label="$src block #$idx"

	if [ "$kind" = "go-nocompile" ]; then
		skipped=$((skipped + 1))
		echo "SKIP (nocompile): $label"
		continue
	fi

	blockdir="$WORKDIR/build_${src//[^a-zA-Z0-9]/_}_$idx"
	mkdir -p "$blockdir"

	if [ "$full" = "1" ]; then
		# Self-contained program: compile verbatim. Hard failure on error.
		cp "$path" "$blockdir/main.go"
		if ( cd "$blockdir" && GOFLAGS=-mod=mod go build ./... ) >"$blockdir/build.log" 2>&1; then
			compiled=$((compiled + 1))
			echo "OK   (program):  $label"
		else
			failures=$((failures + 1))
			fail_list="$fail_list\n  - $label (program)"
			echo "FAIL (program):  $label"
			sed 's/^/      /' "$blockdir/build.log"
		fi
	else
		# Illustrative fragment: wrap in the scaffold harness + build. A
		# fragment that does not compile is a hard failure (a real doc bug);
		# intentionally-partial cheat-sheets must opt out via `go-nocompile`.
		wrap_fragment "$path" "$blockdir/main.go"
		if ( cd "$blockdir" && GOFLAGS=-mod=mod go build ./... ) >"$blockdir/build.log" 2>&1; then
			compiled=$((compiled + 1))
			echo "OK   (fragment): $label"
		else
			failures=$((failures + 1))
			fail_list="$fail_list\n  - $label (fragment)"
			echo "FAIL (fragment): $label"
			sed 's/^/      /' "$blockdir/build.log"
		fi
	fi
done <"$WORKDIR/index"

echo ""
echo "doc snippet build gate: $total block(s) — $compiled compiled, $skipped skipped (go-nocompile), $failures failed"

if [ "$failures" -ne 0 ]; then
	echo ""
	echo "The following doc snippets failed to compile:"
	# shellcheck disable=SC2059
	printf "$fail_list\n"
	echo ""
	echo "Fix the snippet so it compiles against the current public API. If the"
	echo "block is an intentionally-partial cheat-sheet or pseudo-code (e.g. a"
	echo "list of bare method/constant names), tag its fence as \`\`\`go-nocompile."
	exit 1
fi

exit 0
