package cli

import (
	"fmt"
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
				changed, err := changedFiles(cfg.Root, base)
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
	cmd.Flags().StringVar(&base, "base", "origin/main", "base ref for --diff")
	return cmd
}

// errFailed carries a non-zero exit without printing a second message.
var errFailed = fmt.Errorf("validation failed")

func changedFiles(root, base string) (map[string]bool, error) {
	output, err := exec.Command("git", "-C", root, "diff", "--name-only", base+"...HEAD").Output()
	if err != nil {
		return nil, fmt.Errorf("listing files changed against %s: %w", base, err)
	}
	changed := map[string]bool{}
	for _, line := range strings.Split(string(output), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			changed[trimmed] = true
		}
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
