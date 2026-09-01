package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/codcod/jerry/internal/index"
)

func indexCmd(g *globals) *cobra.Command {
	var (
		check     bool
		format    string
		output    string
		printPath bool
	)
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Regenerate the index of every decision",
		Long: "Writes index/index.md (or --output). With --check it writes nothing and exits\n" +
			"non-zero when the committed index disagrees with the documents.\n\n" +
			"The index is verified rather than pushed by a bot: a CI job that regenerated it\n" +
			"and pushed to main would need a protected-branch bypass, would add a commit per\n" +
			"merge, and would fail on a non-fast-forward whenever two merges land together.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{kindKey: kindWrite},
		RunE: func(cmd *cobra.Command, _ []string) error {
			corpus, cfg, err := openCorpus(g)
			if err != nil {
				return err
			}

			target := cfg.IndexPath
			if output != "" {
				target = output
			}
			if printPath {
				// Stdout, not cmd.Println: cobra's Print* helpers write to
				// stderr, and the git hook captures this with $(...).
				fmt.Fprintln(cmd.OutOrStdout(), target)
				return nil
			}

			envelope := index.Build(corpus, target)
			if format == "json" || g.json {
				return index.RenderJSON(cmd.OutOrStdout(), envelope)
			}

			rendered := index.Markdown(envelope)
			absolute := filepath.Join(cfg.Root, filepath.FromSlash(target))

			if check {
				existing, err := os.ReadFile(absolute)
				if err != nil || string(existing) != rendered {
					cmd.Printf("%s is out of date. Run `jerry index` and commit the result.\n", target)
					cmd.SilenceErrors = true
					return errFailed
				}
				if !g.quiet {
					cmd.Printf("%s is up to date.\n", target)
				}
				return nil
			}

			if err := writeRendered(absolute, rendered); err != nil {
				return err
			}
			if !g.quiet {
				cmd.Printf("Wrote %s (%d ADRs, %d solution designs)\n", target, len(envelope.ADRs), len(envelope.SDs))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "verify the committed index instead of writing it")
	cmd.Flags().StringVarP(&format, "format", "f", "md", "output format: md or json")
	cmd.Flags().StringVarP(&output, "output", "o", "", "write to this path instead of the configured one")
	cmd.Flags().BoolVar(&printPath, "print-path", false, "print the index path and exit (used by the git hook)")
	// index writes a generated file rather than authored content, so --dry-run
	// is spelled --check: it answers the same question CI asks.
	cmd.Flags().BoolVar(&check, "dry-run", false, "alias for --check")
	return cmd
}

func writeRendered(absolute, rendered string) error {
	if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
		return err
	}
	return os.WriteFile(absolute, []byte(rendered), 0o644)
}
