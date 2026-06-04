// Package version is the single source of truth for the langfuse-go SDK
// version.
//
// The version string is defined exactly once, in version.txt, and embedded at
// build time. Every other surface that reports a version (the root langfuse
// package's Version, the pkg/client User-Agent, the root VERSION file) derives
// from this package so the value can never drift.
//
// The release tooling (.releaserc.json) writes the new version into
// version.txt (and mirrors it into the root VERSION file for badges and
// external tooling). Do not hard-code version literals anywhere else.
package version

import (
	_ "embed"
	"strings"
)

//go:embed version.txt
var raw string

// Version is the current SDK version, trimmed of surrounding whitespace.
var Version = strings.TrimSpace(raw)
