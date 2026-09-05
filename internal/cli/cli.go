// Package cli wires the jerry command tree onto the doc, rules, index and
// scaffold packages.
//
// Persistent flags live in a globals struct owned by Execute and passed to each
// command constructor, so building a command tree has no effect on any other
// tree in the same process (tests build their own). There is no package-level
// root command and no init() registration for the same reason.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Kind annotations classify every leaf command. write_safety_test.go enforces
// that each leaf declares one and that every write has --dry-run, so the rule
// is checked by the suite rather than remembered at review time.
const (
	kindKey   = "jerry/kind"
	kindRead  = "read"
	kindWrite = "write"
	kindOther = "other"
)

// globals holds the persistent flags shared by every command.
type globals struct {
	configPath string
	json       bool
	quiet      bool
}

// Execute runs the root command. version is injected from main.
func Execute(version string) error {
	root, _ := newRoot(version)
	return root.Execute()
}

// newRoot builds the command tree and returns it alongside its flag state, so
// tests can drive a fresh, isolated tree.
func newRoot(version string) (*cobra.Command, *globals) {
	g := &globals{}
	root := &cobra.Command{
		Use:   "jerry",
		Short: "Scaffold and govern a repository of ADRs and Solution Designs",
		Long: "jerry scaffolds an architecture-docs repository and then owns every rule that\n" +
			"governs it: validation, indexing, id allocation and status transitions.\n" +
			"Scaffolded repositories carry no scripts of their own — the binary is the\n" +
			"single source of truth for the rules, so they cannot drift between repos.",
		SilenceUsage:  true,
		SilenceErrors: false,
		Version:       version,
	}
	root.PersistentFlags().StringVarP(&g.configPath, "config", "c", "",
		"path to jerry.yaml (default: nearest one, searching upwards)")
	root.PersistentFlags().BoolVar(&g.json, "json", false,
		"emit machine-readable JSON instead of human output")
	root.PersistentFlags().BoolVarP(&g.quiet, "quiet", "q", false,
		"suppress progress; errors are still shown")

	root.AddCommand(
		initCmd(g),
		newCmd(g),
		validateCmd(g),
		relatedCmd(g),
		commentCmd(g),
		fmtCmd(g),
		indexCmd(g),
		supersedeCmd(g),
		statusCmd(g),
		schemaCmd(g),
		hooksCmd(g),
		versionCmd(version),
	)
	return root, g
}

func versionCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:         "version",
		Short:       "Print the version",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{kindKey: kindOther},
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Stdout, not cmd.Println: cobra's Print* helpers write to stderr,
			// and `jerry version` is read by scripts.
			fmt.Fprintln(cmd.OutOrStdout(), version)
			return nil
		},
	}
}
