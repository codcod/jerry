package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/codcod/jerry/internal/doc"
)

// schemaCmd emits JSON Schema for the frontmatter. This is deliberately cheap
// leverage: any editor with a YAML language server gets completion and inline
// validation from it, which is better value than writing an editor plugin.
func schemaCmd(g *globals) *cobra.Command {
	var kind string
	cmd := &cobra.Command{
		Use:         "schema",
		Short:       "Emit JSON Schema for document frontmatter",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{kindKey: kindRead},
		RunE: func(cmd *cobra.Command, _ []string) error {
			var document any
			switch kind {
			case "adr":
				document = adrSchema()
			case "sd":
				document = sdSchema()
			default:
				document = map[string]any{
					"$schema": "https://json-schema.org/draft/2020-12/schema",
					"title":   "jerry documents",
					"oneOf":   []any{adrSchema(), sdSchema()},
				}
			}
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			// Without this, a description mentioning <team> is emitted as
			// \u003cteam\u003e, which is valid JSON and unreadable.
			encoder.SetEscapeHTML(false)
			return encoder.Encode(document)
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "", "restrict to one kind: adr or sd")
	return cmd
}

func stringList(description string) map[string]any {
	return map[string]any{
		"description": description,
		"oneOf": []any{
			map[string]any{"type": "string"},
			map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
	}
}

const refPattern = `^([a-z0-9][a-z0-9-]*/)?ADR-\d{4}$`

func refList(description string) map[string]any {
	return map[string]any{
		"description": description,
		"type":        "array",
		"items":       map[string]any{"type": "string", "pattern": refPattern},
	}
}

func adrSchema() map[string]any {
	return map[string]any{
		"$schema":     "https://json-schema.org/draft/2020-12/schema",
		"title":       "Architecture Decision Record frontmatter",
		"type":        "object",
		"required":    []string{"id", "title", "status", "date", "deciders"},
		"description": "Frontmatter of a file under teams/<team>/adr/ or cross-cutting/adr/.",
		"properties": map[string]any{
			"schema_version": map[string]any{"type": "integer", "const": 1},
			"id":             map[string]any{"type": "string", "pattern": `^ADR-\d{4}$`},
			"title":          map[string]any{"type": "string", "minLength": 1},
			"status":         map[string]any{"type": "string", "enum": doc.ADRStatuses},
			"superseded_by":  refList("Required when status is Superseded."),
			"supersedes":     refList("The reverse pointer, written by jerry supersede."),
			"team":           map[string]any{"type": "string", "description": "Must match the folder under teams/."},
			"teams":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "minItems": 2, "description": "Cross-cutting ADRs use this instead of team."},
			"date":           map[string]any{"type": "string", "format": "date"},
			"deciders":       stringList("Who made the decision."),
			"applies_to":     stringList("Paths this decision governs."),
		},
	}
}

func sdSchema() map[string]any {
	return map[string]any{
		"$schema":     "https://json-schema.org/draft/2020-12/schema",
		"title":       "Solution Design frontmatter",
		"type":        "object",
		"required":    []string{"title", "status", "date", "authors"},
		"description": "Frontmatter of a file under teams/<team>/solution-designs/.",
		"properties": map[string]any{
			"schema_version": map[string]any{"type": "integer", "const": 1},
			"title":          map[string]any{"type": "string", "minLength": 1},
			"status":         map[string]any{"type": "string", "enum": doc.SDStatuses},
			"team":           map[string]any{"type": "string"},
			"date":           map[string]any{"type": "string", "format": "date"},
			"authors":        stringList("Who wrote the design."),
			"related_adrs":   refList("ADRs this design produced or depends on."),
			"applies_to":     stringList("Paths this design covers."),
		},
	}
}
