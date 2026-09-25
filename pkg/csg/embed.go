package csg

import (
	"embed"
)

// embeddedDocs contains all Code Style Guide documentation for the crypto-bundle project.
// The folder structure within is preserved — access paths start with "docs/".
//
//go:embed docs
//go:embed AGENTS.md
var embeddedDocs embed.FS

// embeddedAgents contains instructions for AI-agents from AGENTS.md.
//
//go:embed AGENTS.md
var embeddedAgents embed.FS
