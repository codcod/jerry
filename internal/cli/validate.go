package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/codcod/jerry/internal/rules"
)

func validateCmd(g *globals) *cobra.Command {
	var (
		format   string
		diffOnly bool
		base     string
	)
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Check every document against the rules",
		Long: "Reports every problem it finds rather than stopping at the first: a validator\n" +
			"that reports one error per run turns a five-minute fix into five CI round-trips.\n\n" +
			"Exits non-zero when there is at least one error. Warnings are printed but never\n" +
			"fail — an ADR left Proposed for months needs a human, and failing CI on the\n" +
			"calendar would only teach people to lie about dates.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{kindKey: kindRead},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if g.json {
				format = "json"
			}
			if err := validateFormat(format); err != nil {
				return err
			}

			corpus, cfg, err := openCorpus(g)
			if err != nil {
				return err
			}
			findings := rules.Check(corpus, cfg.RuleOptions(now()))

			if diffOnly {
				changed, err := changedFiles(cfg.Root, resolveDiffBase(cmd, base))
				if err != nil {
					return err
				}
				findings = onlyIn(findings, changed)
			}

			if err := renderFindings(cmd.OutOrStdout(), findings, format); err != nil {
				return err
			}

			if format == "text" && !g.quiet {
				if errorCount := findings.Errors(); errorCount > 0 {
					cmd.Printf("\n%d problem(s) in %d document(s).\n", errorCount, len(corpus.Docs))
				} else {
					cmd.Printf("%d document(s) validated, no problems found.\n", len(corpus.Docs))
				}
			}
			if findings.Errors() > 0 {
				// The findings are the message; a second error line would just
				// repeat them, so fail silently with a non-zero status.
				cmd.SilenceErrors = true
				return errFailed
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&format, "format", "f", "text", "output format: text, json, sarif or junit")
	cmd.Flags().BoolVar(&diffOnly, "diff", false, "only report findings in files changed against --base")
	cmd.Flags().StringVar(&base, "base", "origin/main",
		"base ref for --diff (autodetected from GITHUB_BASE_REF when not set explicitly)")
	return cmd
}

// errFailed carries a non-zero exit without printing a second message.
var errFailed = fmt.Errorf("validation failed")

// resolveDiffBase returns the base ref to diff against: the caller's
// explicit --base when given, otherwise origin/$GITHUB_BASE_REF (GitHub
// Actions' pull_request env var) when set, otherwise the flag's own
// default — never silently guessing when a human already said what they want.
func resolveDiffBase(cmd *cobra.Command, base string) string {
	if cmd.Flags().Changed("base") {
		return base
	}
	if prBase := os.Getenv("GITHUB_BASE_REF"); prBase != "" {
		return "origin/" + prBase
	}
	return base
}

// changedFiles lists the files changed against base, as paths relative to
// root. git always reports diff --name-only paths relative to its own top
// level regardless of -C, so when root sits below that (a jerry.yaml not at
// the git root) the raw output has to be rewritten onto root's own offset —
// resolved via `rev-parse --show-prefix` rather than manual filepath.Rel
// arithmetic, since only git's own view of the tree is guaranteed to agree
// with git's own path output (a symlinked temp dir, e.g. macOS's /tmp ->
// /private/tmp, can make the two disagree).
func changedFiles(root, base string) (map[string]bool, error) {
	prefixOut, err := exec.Command("git", "-C", root, "rev-parse", "--show-prefix").Output()
	if err != nil {
		return nil, fmt.Errorf("resolving %s's offset from the git root: %w", root, err)
	}
	prefix := strings.TrimSpace(string(prefixOut))

	output, err := exec.Command("git", "-C", root, "diff", "--name-only", base+"...HEAD").Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("listing files changed against %s: %s", base, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("listing files changed against %s: %w", base, err)
	}

	changed := map[string]bool{}
	for _, line := range strings.Split(string(output), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if prefix == "" {
			changed[trimmed] = true
			continue
		}
		if rel, ok := strings.CutPrefix(trimmed, prefix); ok {
			changed[rel] = true
		}
		// else: changed outside the corpus root, can never match a finding
	}
	return changed, nil
}

func onlyIn(findings rules.Findings, changed map[string]bool) rules.Findings {
	filtered := make(rules.Findings, 0, len(findings))
	for _, finding := range findings {
		if changed[finding.Path] {
			filtered = append(filtered, finding)
		}
	}
	return filtered
}
