package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/codcod/jerry/internal/doc"
	"github.com/codcod/jerry/internal/forge"
	"github.com/codcod/jerry/internal/match"
)

// adoptionLogPath is where the adoption log lives, relative to the repo
// root — a plain committed file, per DESIGN.md's "a file in the repository,
// not a metrics backend" (mirrors index.DefaultPath being a hardcoded
// constant, not a flag default computed elsewhere).
const adoptionLogPath = "jerry-adoption.jsonl"

// adoptionLogEntry is one line appended to the adoption log on every
// successful post, so §9's "is the read side actually used" question is
// answerable from day one (adoption-report, unfiled, reads this log).
type adoptionLogEntry struct {
	Repo      string   `json:"repo"`
	PR        int      `json:"pr"`
	Decisions []string `json:"decisions"`
	Timestamp string   `json:"timestamp"`
}

func commentCmd(g *globals) *cobra.Command {
	var (
		base        string
		adoptionLog string
		dryRun      bool
	)
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Post the decisions governing this merge request's changed files",
		Long: "Runs `related` over the merge request's changed files and, when at least one\n" +
			"decision matches, posts (or updates) a comment listing the governing decisions.\n" +
			"No output at all when nothing matches.\n\n" +
			"A docs tool must never be the reason a merge request cannot merge (DESIGN.md\n" +
			"§7.2): an absent or insufficiently-scoped CI token, or any other runtime\n" +
			"failure, degrades this command to a logged-but-silent no-op, never a failed\n" +
			"pipeline.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{kindKey: kindWrite},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := runComment(cmd, g, base, adoptionLog, dryRun); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "jerry comment: %v\n", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&base, "base", "origin/main", "base ref for changed files")
	cmd.Flags().StringVar(&adoptionLog, "adoption-log", adoptionLogPath,
		"path (relative to the repo root) the adoption log is appended to")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false,
		"render the comment body without posting it or logging adoption")
	return cmd
}

func runComment(cmd *cobra.Command, g *globals, base, adoptionLog string, dryRun bool) error {
	corpus, cfg, err := openCorpus(g)
	if err != nil {
		return err
	}
	changed, err := changedFiles(cfg.Root, base)
	if err != nil {
		return err
	}

	matches := aggregateMatches(corpus, changed)
	if len(matches) == 0 {
		return nil // decision 5(a): nothing matched
	}
	body := renderCommentBody(matches)

	if dryRun {
		cmd.Printf("(dry run) would post:\n\n%s", body)
		return nil
	}

	client, ok := forge.NewGitHubFromEnv()
	if !ok {
		return nil // decision 5(b): no token / not a pull-request CI run
	}
	if err := client.PostOrUpdate(body); err != nil {
		return err
	}

	return appendAdoptionLog(filepath.Join(cfg.Root, adoptionLog), client, matches)
}

// aggregateMatches collects match.Resolve results across every changed
// file, deduped by document and keeping each document's highest-specificity
// match seen (decision 3): a decision matching three changed files is
// reported once, not three times.
func aggregateMatches(corpus *doc.Corpus, changed map[string]bool) []match.Match {
	best := map[string]match.Match{}
	for path := range changed {
		for _, m := range match.Resolve(corpus, path) {
			if existing, ok := best[m.Doc.Path]; !ok || match.LessSpecific(existing, m) {
				best[m.Doc.Path] = m
			}
		}
	}

	matches := make([]match.Match, 0, len(best))
	for _, m := range best {
		matches = append(matches, m)
	}
	sort.SliceStable(matches, func(i, j int) bool {
		return match.LessSpecific(matches[j], matches[i]) // descending: most specific first
	})
	return matches
}

// renderCommentBody is a Markdown bullet list, one line per governing
// decision — title and repo-relative path, no forge URL (decision 4).
func renderCommentBody(matches []match.Match) string {
	var buf strings.Builder
	for _, m := range matches {
		fmt.Fprintf(&buf, "- %s (%s)\n", m.Doc.Front.Title, m.Doc.Path)
	}
	return buf.String()
}

func appendAdoptionLog(path string, client *forge.GitHubClient, matches []match.Match) error {
	decisions := make([]string, len(matches))
	for i, m := range matches {
		decisions[i] = m.Doc.Path
	}
	entry := adoptionLogEntry{
		Repo:      client.Repository(),
		PR:        client.PRNumber(),
		Decisions: decisions,
		Timestamp: now().Format(time.RFC3339),
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}
