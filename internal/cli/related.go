package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/codcod/jerry/internal/match"
)

// RelatedEnvelopeSchema versions the machine-readable `related` output.
// Consumers pin it, so changes are additive only.
const RelatedEnvelopeSchema = "jerry.related/1"

type relatedEnvelope struct {
	Schema  string          `json:"schema"`
	Results []relatedResult `json:"results"`
}

type relatedResult struct {
	Path      string            `json:"path"`
	Decisions []relatedDecision `json:"decisions"`
}

type relatedDecision struct {
	ID      string `json:"id,omitempty"`
	Title   string `json:"title"`
	Path    string `json:"path"`
	Pattern string `json:"pattern"`
}

func relatedCmd(g *globals) *cobra.Command {
	var (
		paths  []string
		format string
	)
	cmd := &cobra.Command{
		Use:   "related",
		Short: "Which decisions govern a set of changed paths",
		Long: "Resolves each --paths entry against every decision's applies_to, offline, from\n" +
			"the terminal. Unlike validate, this is a query, not a gate: it exits non-zero\n" +
			"only on bad input, never because nothing matched — a queried path with no\n" +
			"governing decision is a normal, silent result.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{kindKey: kindRead},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if g.json {
				format = "json"
			}
			if format != "text" && format != "json" {
				return fmt.Errorf("unknown format %q (want one of: text, json)", format)
			}
			if len(paths) == 0 {
				return fmt.Errorf("--paths is required")
			}

			corpus, _, err := openCorpus(g)
			if err != nil {
				return err
			}

			envelope := relatedEnvelope{Schema: RelatedEnvelopeSchema}
			for _, path := range paths {
				result := relatedResult{Path: path, Decisions: []relatedDecision{}}
				for _, m := range match.Resolve(corpus, path) {
					result.Decisions = append(result.Decisions, relatedDecision{
						ID:      m.Doc.Front.ID,
						Title:   m.Doc.Front.Title,
						Path:    m.Doc.Path,
						Pattern: m.Pattern,
					})
				}
				envelope.Results = append(envelope.Results, result)
			}

			return renderRelated(cmd.OutOrStdout(), envelope, format)
		},
	}
	cmd.Flags().StringSliceVar(&paths, "paths", nil, "changed paths to resolve against every decision's applies_to")
	cmd.Flags().StringVarP(&format, "format", "f", "text", "output format: text or json")
	return cmd
}

func renderRelated(w io.Writer, envelope relatedEnvelope, format string) error {
	if format == "json" {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)
		return encoder.Encode(envelope)
	}
	for _, result := range envelope.Results {
		if len(result.Decisions) == 0 {
			continue
		}
		if _, err := fmt.Fprintln(w, result.Path); err != nil {
			return err
		}
		for _, decision := range result.Decisions {
			label := decision.Title
			if decision.ID != "" {
				label = decision.ID + " " + label
			}
			if _, err := fmt.Fprintf(w, "  %s (%s)\n", label, decision.Path); err != nil {
				return err
			}
		}
	}
	return nil
}
