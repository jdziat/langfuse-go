#!/usr/bin/env bash
#
# check-doc-snippets.sh — compile the Go code examples embedded in the docs.
#
# This gate guards against "doc rot": documentation snippets that drift out of
# sync with the real public API (wrong method names, removed options, fictional
# call chains, missing context arguments, ...). It extracts every Go code block
# from the documentation sources, assembles each into a compilable unit, and
# runs `go build` against a throwaway module that points at the local copy of
# this module via a `replace` directive. If a self-contained example fails to
# compile the script exits non-zero and names the offending doc + block.
#
# Sources scanned:
#   - README.md
#   - doc.go            (the tab-indented code blocks inside its // comment)
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
#       trace, ctx, ...) defined in the surrounding prose, or a cheat-sheet of
#       call signatures. Fragments are wrapped in a minimal `func main` and a
#       build is attempted on a best-effort basis: a fragment that compiles is
#       reported as OK, but a fragment that does not (because it is, by design,
#       incomplete) is reported as SKIPPED and does NOT fail the gate.
#
#   ```go-nocompile
#   ...
#   ```
#       Any block tagged `go-nocompile` (info string `go-nocompile`, or
#       `go,nocompile`) is skipped entirely. Use this for blocks that are
#       intentionally not valid Go.
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

# extract_blocks <file> writes the code blocks found in <file> to numbered
# files under $WORKDIR/blocks, one directory per (file, block). It records, per
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
	local src="doc.go"
	[ -f "$src" ] || return 0
	awk -v src="$src" -v workdir="$WORKDIR" -v recid="doc_go" '
		function flush() {
			if (buf == "") return
			n++
			path = workdir "/block_" recid "_" n ".txt"
			printf "%s", buf > path
			close(path)
			gsub(/^[[:space:]]+/, "", firstcode)
			full = (firstcode ~ /^package /) ? 1 : 0
			# doc.go comment blocks are always illustrative fragments.
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
			# An indented comment line ("//\t..." or "//    ...") is code.
			if (line ~ /^\t/ || line ~ /^    /) {
				if (firstcode == "") firstcode = line
				# Normalise a single leading tab (godocs typical indent).
				sub(/^\t/, "", line)
				sub(/^    /, "", line)
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
	if [ "$g" = "doc.go" ]; then
		extract_docgo_blocks
	else
		extract_markdown_blocks "$g"
	fi
done

# Compose a buildable Go file for a fragment block: add a package clause, an
# import of the SDK (aliased "langfuse", matching the docs) plus a handful of
# commonly used stdlib packages, and wrap the body in func main. Unused imports
# are tolerated via a blank reference; the Go compiler still rejects genuinely
# broken code (unknown methods, type errors, bad syntax).
wrap_fragment() {
	local body_file="$1" out_file="$2"
	{
		echo "package main"
		echo ""
		echo "import ("
		echo "	\"context\""
		echo "	\"fmt\""
		echo "	\"log\""
		echo "	\"net/http\""
		echo "	\"os\""
		echo "	\"time\""
		echo ""
		echo "	langfuse \"$MODULE_PATH\""
		echo "	\"$MODULE_PATH/evaluation\""
		echo ")"
		echo ""
		echo "// Reference imports so unused ones do not break the build; the"
		echo "// snippet body below is what is actually being type-checked."
		echo "var ("
		echo "	_ = context.Background"
		echo "	_ = fmt.Sprint"
		echo "	_ = log.Print"
		echo "	_ = http.DefaultClient"
		echo "	_ = os.Getenv"
		echo "	_ = time.Second"
		echo "	_ = langfuse.New"
		echo "	_ = evaluation.NewQATrace"
		echo ")"
		echo ""
		echo "func main() {"
		cat "$body_file"
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
			fail_list="$fail_list\n  - $label"
			echo "FAIL (program):  $label"
			sed 's/^/      /' "$blockdir/build.log"
		fi
	else
		# Illustrative fragment: best-effort wrap + build. Never a hard failure.
		wrap_fragment "$path" "$blockdir/main.go"
		if ( cd "$blockdir" && GOFLAGS=-mod=mod go build ./... ) >"$blockdir/build.log" 2>&1; then
			compiled=$((compiled + 1))
			echo "OK   (fragment): $label"
		else
			skipped=$((skipped + 1))
			echo "SKIP (fragment): $label (does not compile standalone; treated as a partial excerpt)"
		fi
	fi
done <"$WORKDIR/index"

echo ""
echo "doc snippet build gate: $total block(s) — $compiled compiled, $skipped skipped, $failures failed"

if [ "$failures" -ne 0 ]; then
	echo ""
	echo "The following self-contained doc examples failed to compile:"
	# shellcheck disable=SC2059
	printf "$fail_list\n"
	echo ""
	echo "Fix the snippet, or tag the fence as \`\`\`go-nocompile if it is"
	echo "intentionally not compilable."
	exit 1
fi

exit 0
