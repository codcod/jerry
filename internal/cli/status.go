package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/codcod/jerry/internal/doc"
)

func statusCmd(g *globals) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "status <ref|path> <status>",
		Short: "Move a document to a new status, checking the transition is legal",
		Long: "Superseded is deliberately unreachable here: it needs a successor to point at,\n" +
			"so it is only reachable through `jerry supersede`, which writes both sides of\n" +
			"the link. Accepting it here would allow a Superseded document with no successor.",
		Args:        cobra.ExactArgs(2),
		Annotations: map[string]string{kindKey: kindWrite},
		RunE: func(cmd *cobra.Command, args []string) error {
			corpus, cfg, err := openCorpus(g)
			if err != nil {
				return err
			}

			document, err := corpus.FindAny(args[0])
			if err != nil {
				return err
			}
			if document.ParseErr != nil {
				return fmt.Errorf("%s: %w", document.Path, document.ParseErr)
			}

			target, ok := doc.CanonicalStatus(document.Kind, args[1])
			if !ok {
				return fmt.Errorf("status %q is not one of: %s",
					args[1], strings.Join(doc.Statuses(document.Kind), ", "))
			}
			if target == doc.StatusSuperseded {
				return fmt.Errorf("refusing to set Superseded directly — use `jerry supersede %s --with \"<title>\"`, "+
					"which also writes the successor and the reverse pointer", document.Ref().Short(""))
			}

			current := document.Front.Status
			if current == target {
				return fmt.Errorf("%s is already %s", document.Path, target)
			}
			if !doc.CanTransition(document.Kind, current, target) {
				reachable := doc.NextStatuses(document.Kind, current)
				if len(reachable) == 0 {
					return fmt.Errorf("%s is %s, which is terminal — nothing follows it", document.Path, current)
				}
				return fmt.Errorf("%s -> %s is not a legal transition for this %s; from %s you can go to: %s",
					current, target, document.Kind, current, strings.Join(reachable, ", "))
			}

			front := document.Front
			front.Status = target
			frontYAML, err := doc.RenderFront(front)
			if err != nil {
				return err
			}
			content := doc.Render(frontYAML, document.Body)

			if dryRun {
				cmd.Printf("(dry run) would move %s from %s to %s\n", document.Path, current, target)
				return nil
			}
			if err := os.WriteFile(filepath.Join(cfg.Root, filepath.FromSlash(document.Path)), []byte(content), 0o644); err != nil {
				return err
			}
			if !g.quiet {
				cmd.Printf("%s: %s -> %s\n", document.Path, current, target)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report the transition without writing")
	return cmd
}
