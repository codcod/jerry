package cli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/codcod/jerry/internal/doc"
)

// fmtCmd canonicalises frontmatter and trailing whitespace.
//
// It deliberately does not reflow prose. Rewrapping markdown means deciding
// what is a paragraph and what is a list item, a table row, or a fenced block,
// and getting that wrong corrupts authored content — a formatter that damages
// text is worse than no formatter. Frontmatter is structured data with an
// unambiguous canonical form, so that is what this normalises. Keys jerry has
// never heard of are preserved, in their authored order, after the known ones.
func fmtCmd(g *globals) *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "fmt",
		Short: "Canonicalise frontmatter key order and whitespace",
		Long: "Normalises frontmatter key order and trailing whitespace so reviews argue about\n" +
			"decisions rather than field order. Prose is never reflowed: deciding what is a\n" +
			"paragraph and what is a list, table or fenced block is exactly where a markdown\n" +
			"formatter corrupts authored text.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{kindKey: kindWrite},
		RunE: func(cmd *cobra.Command, _ []string) error {
			corpus, cfg, err := openCorpus(g)
			if err != nil {
				return err
			}

			var changed []string
			for _, document := range corpus.Docs {
				if document.ParseErr != nil {
					continue
				}
				frontYAML, err := doc.Canonical(document.Mapping)
				if err != nil {
					return err
				}
				formatted := doc.Render(frontYAML, document.Body)
				if formatted == document.Raw {
					continue
				}
				changed = append(changed, document.Path)
				if check {
					continue
				}
				target := filepath.Join(cfg.Root, filepath.FromSlash(document.Path))
				if err := os.WriteFile(target, []byte(formatted), 0o644); err != nil {
					return err
				}
			}

			for _, path := range changed {
				cmd.Println(path)
			}
			if check && len(changed) > 0 {
				cmd.Printf("\n%d file(s) need formatting. Run `jerry fmt`.\n", len(changed))
				cmd.SilenceErrors = true
				return errFailed
			}
			if !g.quiet && len(changed) == 0 {
				cmd.Println("All documents are already canonical.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "list files that need formatting and exit non-zero")
	cmd.Flags().BoolVar(&check, "dry-run", false, "alias for --check")
	return cmd
}
