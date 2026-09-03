package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/codcod/jerry/internal/doc"
)

// Body prompts are HTML comments on purpose: the empty-section rule ignores
// them, so a document created and never filled in still fails validation
// rather than passing as a hollow record.
const adrBody = `# %s: %s

## Context

<!-- What is the issue that motivates this decision? Include the constraints:
     technical, organizational, timeline. If this supersedes an earlier ADR,
     reference it here and say what changed. -->

## Decision

<!-- What is the change we're making? One or two plain sentences, then
     elaborate if needed. -->

## Consequences

<!-- What becomes easier or harder? Include the negative consequences and
     trade-offs honestly — this is the most valuable part of the record for
     whoever reads it in two years. -->

## Alternatives Considered

<!-- What else was on the table, and why was it not chosen? An ADR with no
     alternatives is a decision announcement, not a decision record. -->

## Related

<!-- Related ADRs, Solution Designs, or external docs. Reference ADRs in other
     folders with their scope: payments/ADR-0007, cross-cutting/ADR-0003. -->
`

const sdBody = `# %s

## Problem Statement

<!-- What problem are we solving? Who is affected, and why now? -->

## Goals / Non-Goals

<!-- Goals: what this design commits to achieving.
     Non-Goals: explicitly out of scope, to prevent scope creep. -->

## Proposed Design

<!-- Describe the design. Diagrams welcome (link or embed). -->

## Alternatives Considered

<!-- What else was considered, and why was it not chosen? -->

## Risks & Trade-offs

<!-- Be honest about what could go wrong and what is being traded away. -->

## Rollout / Migration Plan

<!-- Migration steps, feature flags, phased rollout. -->

## Related ADRs

<!-- Keep related_adrs in the frontmatter in sync; every reference is checked. -->
`

func newCmd(g *globals) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new ADR or Solution Design",
	}
	cmd.AddCommand(newADRCmd(g), newSDCmd(g))
	return cmd
}

func newADRCmd(g *globals) *cobra.Command {
	var (
		team         string
		crossCutting bool
		teams        []string
		status       string
		deciders     []string
		dryRun       bool
	)
	cmd := &cobra.Command{
		Use:   `adr "<title>"`,
		Short: "Create an ADR, allocating the next free id in its folder",
		Long: "Allocates the next free four-digit id in the target folder, slugs the title\n" +
			"into a filename, stamps today's date and fills deciders from your git identity.\n\n" +
			"Gaps are never reused: a number that was once taken stays spent, so an id\n" +
			"quoted in a commit message never comes to mean a second document.",
		Args:        cobra.ExactArgs(1),
		Annotations: map[string]string{kindKey: kindWrite},
		RunE: func(cmd *cobra.Command, args []string) error {
			title := strings.TrimSpace(args[0])
			if title == "" {
				return fmt.Errorf("a title is required")
			}

			corpus, cfg, err := openCorpus(g)
			if err != nil {
				return err
			}

			scope, err := resolveScope(team, crossCutting, teams)
			if err != nil {
				return err
			}
			canonicalStatus, ok := doc.CanonicalStatus(doc.KindADR, status)
			if !ok {
				return fmt.Errorf("status %q is not one of: %s", status, strings.Join(doc.ADRStatuses, ", "))
			}
			if canonicalStatus == doc.StatusSuperseded {
				return fmt.Errorf("a new ADR cannot start Superseded — use `jerry supersede` on the ADR being replaced")
			}

			id := corpus.NextID(scope)
			front := doc.Front{
				SchemaVersion: doc.CurrentSchemaVersion,
				ID:            "ADR-" + id,
				Title:         title,
				Status:        canonicalStatus,
				Date:          now().Format(doc.DateLayout),
				Deciders:      resolveDeciders(deciders, cfg.Root),
			}
			if crossCutting {
				front.Teams = teams
			} else {
				front.Team = scope
			}

			relPath := filepath.ToSlash(filepath.Join(corpus.DirFor(doc.KindADR, scope), id+"-"+doc.Slug(title)+".md"))
			body := fmt.Sprintf(adrBody, front.ID, title)
			return writeDocument(cmd, g, cfg.Root, relPath, front, body, dryRun)
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "team folder to create the ADR in")
	cmd.Flags().BoolVar(&crossCutting, "cross-cutting", false, "create under cross-cutting/ instead of a team folder")
	cmd.Flags().StringSliceVar(&teams, "teams", nil, "affected teams (required with --cross-cutting, two or more)")
	cmd.Flags().StringVar(&status, "status", doc.StatusProposed, "initial status")
	cmd.Flags().StringSliceVar(&deciders, "deciders", nil, "who made the decision (default: your git identity)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the path and content without writing")
	return cmd
}

func newSDCmd(g *globals) *cobra.Command {
	var (
		team         string
		crossCutting bool
		status       string
		authors      []string
		dryRun       bool
	)
	cmd := &cobra.Command{
		Use:   `sd "<title>"`,
		Short: "Create a Solution Design",
		Long: "Solution designs have no sequential id — the date is the identifier — so the\n" +
			"filename is YYYY-MM-<slug>.md and its prefix must match the frontmatter date.",
		Args:        cobra.ExactArgs(1),
		Annotations: map[string]string{kindKey: kindWrite},
		RunE: func(cmd *cobra.Command, args []string) error {
			title := strings.TrimSpace(args[0])
			if title == "" {
				return fmt.Errorf("a title is required")
			}

			corpus, cfg, err := openCorpus(g)
			if err != nil {
				return err
			}
			scope, err := resolveScope(team, crossCutting, nil)
			if err != nil {
				return err
			}
			canonicalStatus, ok := doc.CanonicalStatus(doc.KindSD, status)
			if !ok {
				return fmt.Errorf("status %q is not one of: %s", status, strings.Join(doc.SDStatuses, ", "))
			}

			today := now()
			front := doc.Front{
				SchemaVersion: doc.CurrentSchemaVersion,
				Title:         title,
				Status:        canonicalStatus,
				Team:          scope,
				Date:          today.Format(doc.DateLayout),
				Authors:       resolveDeciders(authors, cfg.Root),
				RelatedADRs:   doc.List{},
			}

			name := today.Format("2006-01") + "-" + doc.Slug(title) + ".md"
			relPath := filepath.ToSlash(filepath.Join(corpus.DirFor(doc.KindSD, scope), name))
			return writeDocument(cmd, g, cfg.Root, relPath, front, fmt.Sprintf(sdBody, title), dryRun)
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "team folder to create the design in")
	cmd.Flags().BoolVar(&crossCutting, "cross-cutting", false, "create under cross-cutting/ instead of a team folder")
	cmd.Flags().StringVar(&status, "status", doc.StatusDraft, "initial status")
	cmd.Flags().StringSliceVar(&authors, "authors", nil, "who wrote the design (default: your git identity)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the path and content without writing")
	return cmd
}

func resolveScope(team string, crossCutting bool, teams []string) (string, error) {
	switch {
	case crossCutting && team != "":
		return "", fmt.Errorf("--cross-cutting and --team are mutually exclusive")
	case crossCutting:
		if len(teams) < 2 {
			return "", fmt.Errorf("--cross-cutting needs --teams naming two or more affected teams; " +
				"if only one team is affected it belongs in that team's folder")
		}
		return doc.CrossCutting, nil
	case team == "":
		return "", fmt.Errorf("--team is required (or --cross-cutting --teams a,b)")
	default:
		return team, nil
	}
}

// resolveDeciders falls back to the git identity, so the common case needs no
// flag and the field is never left empty by accident.
func resolveDeciders(explicit []string, root string) doc.List {
	if len(explicit) > 0 {
		return explicit
	}
	output, err := exec.Command("git", "-C", root, "config", "user.name").Output()
	if name := strings.TrimSpace(string(output)); err == nil && name != "" {
		return doc.List{name}
	}
	return nil
}

func writeDocument(cmd *cobra.Command, g *globals, root, relPath string, front doc.Front, body string, dryRun bool) error {
	frontYAML, err := doc.RenderFront(front)
	if err != nil {
		return err
	}
	content := doc.Render(frontYAML, body)

	if dryRun {
		cmd.Printf("(dry run) would write %s\n\n%s", relPath, content)
		return nil
	}

	absolute := filepath.Join(root, filepath.FromSlash(relPath))
	if _, err := os.Stat(absolute); err == nil {
		return fmt.Errorf("%s already exists", relPath)
	}
	if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(absolute, []byte(content), 0o644); err != nil {
		return err
	}
	if !g.quiet {
		cmd.Printf("Created %s\n", relPath)
		cmd.Println("Fill in the sections — an empty section fails `jerry validate`.")
	}
	return nil
}
