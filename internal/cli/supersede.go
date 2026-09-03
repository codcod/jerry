package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/codcod/jerry/internal/doc"
)

// supersedeCmd writes both halves of a supersession in one step.
//
// Doing it by hand is how one side of the link ends up missing: the author
// remembers to mark the old ADR superseded and forgets the reverse pointer, or
// writes the successor and never touches the predecessor. Neither omission is
// visible until someone follows a dead link a year later.
func supersedeCmd(g *globals) *cobra.Command {
	var (
		with     string
		status   string
		deciders []string
		dryRun   bool
	)
	cmd := &cobra.Command{
		Use:         "supersede <ref>",
		Short:       "Supersede an ADR with a new one, writing both pointers",
		Args:        cobra.ExactArgs(1),
		Annotations: map[string]string{kindKey: kindWrite},
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(with) == "" {
				return fmt.Errorf("--with \"<title>\" is required: superseding needs a successor to point at")
			}

			corpus, cfg, err := openCorpus(g)
			if err != nil {
				return err
			}

			old, err := corpus.Find(args[0], "")
			if err != nil {
				return err
			}
			if old.ParseErr != nil {
				return fmt.Errorf("%s: %w", old.Path, old.ParseErr)
			}
			if old.Front.Status == doc.StatusSuperseded {
				return fmt.Errorf("%s is already superseded by %s",
					old.Ref(), strings.Join(old.Front.SupersededBy, ", "))
			}

			canonicalStatus, ok := doc.CanonicalStatus(doc.KindADR, status)
			if !ok {
				return fmt.Errorf("status %q is not one of: %s", status, strings.Join(doc.ADRStatuses, ", "))
			}

			// The successor is created in the same folder as its predecessor:
			// a decision that replaces another belongs to the same owner.
			scope := old.Scope
			id := corpus.NextID(scope)
			successorRef := doc.Ref{Scope: scope, ID: id}

			front := doc.Front{
				SchemaVersion: doc.CurrentSchemaVersion,
				ID:            "ADR-" + id,
				Title:         with,
				Status:        canonicalStatus,
				Supersedes:    doc.List{old.Ref().Short(scope)},
				Date:          now().Format(doc.DateLayout),
				Deciders:      resolveDeciders(deciders, cfg.Root),
			}
			if scope == doc.CrossCutting {
				front.Teams = old.Front.Teams
			} else {
				front.Team = scope
			}

			relPath := filepath.ToSlash(filepath.Join(corpus.DirFor(doc.KindADR, scope), id+"-"+doc.Slug(with)+".md"))
			body := fmt.Sprintf(adrBody, front.ID, with)
			body = strings.Replace(body, "## Context\n\n<!--",
				fmt.Sprintf("## Context\n\nSupersedes %s.\n\n<!--", old.Ref().Short(scope)), 1)

			updated, err := markSuperseded(old, successorRef.Short(scope))
			if err != nil {
				return err
			}

			if dryRun {
				cmd.Printf("(dry run) would create %s and mark %s superseded\n", relPath, old.Path)
				return nil
			}

			if err := writeDocument(cmd, g, cfg.Root, relPath, front, body, false); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(cfg.Root, filepath.FromSlash(old.Path)), []byte(updated), 0o644); err != nil {
				return err
			}
			if !g.quiet {
				cmd.Printf("Marked %s superseded by %s\n", old.Path, successorRef.Short(scope))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&with, "with", "", "title of the superseding ADR")
	cmd.Flags().StringVar(&status, "status", doc.StatusAccepted, "status of the new ADR")
	cmd.Flags().StringSliceVar(&deciders, "deciders", nil, "who made the decision (default: your git identity)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report what would change without writing")
	return cmd
}

// markSuperseded rewrites the predecessor: status, the forward pointer, and a
// line under Consequences so a reader of the prose alone still learns of it.
func markSuperseded(old *doc.Document, successor string) (string, error) {
	front := old.Front
	front.Status = doc.StatusSuperseded
	front.SupersededBy = append(front.SupersededBy, successor)

	frontYAML, err := doc.RenderFront(front)
	if err != nil {
		return "", err
	}

	note := fmt.Sprintf("Superseded by %s.", successor)
	body := old.Body
	if heading := "## Consequences"; strings.Contains(body, heading) {
		index := strings.Index(body, heading) + len(heading)
		body = body[:index] + "\n\n" + note + body[index:]
	} else {
		body = strings.TrimRight(body, "\n") + "\n\n## Consequences\n\n" + note + "\n"
	}
	return doc.Render(frontYAML, body), nil
}
