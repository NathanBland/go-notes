package main

import (
	"strings"
	"testing"
)

func TestVersionStringIncludesBuildMetadata(t *testing.T) {
	originalVersion := version
	originalCommit := commit
	originalDate := date
	t.Cleanup(func() {
		version = originalVersion
		commit = originalCommit
		date = originalDate
	})

	version = "1.2.3"
	commit = "abc123"
	date = "2026-04-05T00:00:00Z"

	output := versionString()
	for _, expected := range []string{
		"go-notes-mcp",
		"version=1.2.3",
		"commit=abc123",
		"date=2026-04-05T00:00:00Z",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected %q to contain %q", output, expected)
		}
	}
}
