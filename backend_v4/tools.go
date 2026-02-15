//go:build tools

package tools

// This file tracks versions of CLI tool dependencies.
// It is not compiled into the binary.
//
// Tool dependencies are managed via 'tool' directive in go.mod (Go 1.24+).
// Install tools: go install tool
// Run tools:     go tool <name>
//
// Tools will be added as they are needed:
// - github.com/matryer/moq (Phase 3)
// - github.com/99designs/gqlgen (Phase 9)

import (
	_ "github.com/99designs/gqlgen"
)
