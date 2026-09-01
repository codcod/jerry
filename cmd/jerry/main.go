// Command jerry scaffolds and governs a repository of Architecture Decision
// Records and Solution Designs: it writes the repository, validates every
// document in it, allocates ids, moves statuses, and regenerates the index.
//
// Scaffolded repositories carry no scripts of their own — the binary is the
// single source of truth for the rules, which is what stops them drifting
// between the repositories that share them. See README.md and DESIGN.md.
package main

import (
	"os"

	"github.com/codcod/jerry/internal/cli"
)

// Version is injected at build time via -ldflags "-X main.Version=...".
var Version = "dev"

func main() {
	if err := cli.Execute(Version); err != nil {
		os.Exit(1)
	}
}
