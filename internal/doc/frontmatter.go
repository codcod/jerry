// Package doc models the two document kinds jerry manages — Architecture
// Decision Records and Solution Designs — and the corpus they form.
//
// Every document is a markdown file opening with a YAML frontmatter block.
// The package keeps two views of that block: a typed Front for the fields
// jerry has rules about, and the raw yaml.Node it was decoded from. The node
// is what makes `jerry fmt` safe — it carries source line numbers for precise
// findings, and it preserves keys jerry knows nothing about instead of
// dropping them on the next write.
package doc

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Delim is the frontmatter fence.
const Delim = "---"

// ErrNoFrontmatter reports a file that does not open with a frontmatter block.
var ErrNoFrontmatter = errors.New("file does not start with a '---' frontmatter block")

// ErrUnterminated reports a frontmatter block that is never closed.
var ErrUnterminated = errors.New("frontmatter block is never closed with '---'")

// List is a frontmatter field that authors write either as a bare scalar or as
// a sequence. `team: payments` and `teams: [a, b]` both land here, so rules can
// treat them uniformly without caring which spelling was used.
type List []string

// UnmarshalYAML accepts a scalar, a flow sequence, or a block sequence.
func (l *List) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		if node.Value == "" {
			*l = nil
			return nil
		}
		*l = List{node.Value}
		return nil
	case yaml.SequenceNode:
		out := make(List, 0, len(node.Content))
		for _, item := range node.Content {
			if item.Value != "" {
				out = append(out, item.Value)
			}
		}
		*l = out
		return nil
	default:
		return fmt.Errorf("line %d: expected a scalar or a list", node.Line)
	}
}

// MarshalYAML always emits a flow sequence, so a canonicalised file has one
// spelling per field regardless of how it was authored.
func (l List) MarshalYAML() (any, error) {
	return []string(l), nil
}

// CurrentSchemaVersion is the newest schema_version this binary understands.
// jerry new and jerry supersede write it into every document; jerry schema
// publishes it as a floor, not an equality (DESIGN.md §3.6): a document above
// it must warn, never error, so upgrading jerry is never a merge prerequisite.
const CurrentSchemaVersion = 1

// Front is the typed view of the fields jerry has rules about. Fields absent
// from a document stay at their zero value; rules distinguish "absent" from
// "invalid" themselves, so nothing here is defaulted.
// Field order here is KeyOrder, so marshalling a Front is canonical by
// construction and `jerry new` cannot emit a document `jerry fmt` would
// immediately reorder.
type Front struct {
	SchemaVersion int    `yaml:"schema_version,omitempty"`
	ID            string `yaml:"id,omitempty"`
	Title         string `yaml:"title,omitempty"`
	Status        string `yaml:"status,omitempty"`
	SupersededBy  List   `yaml:"superseded_by,omitempty"`
	Supersedes    List   `yaml:"supersedes,omitempty"`
	Team          string `yaml:"team,omitempty"`
	Teams         List   `yaml:"teams,omitempty"`
	Date          string `yaml:"date,omitempty"`
	Deciders      List   `yaml:"deciders,omitempty"`
	Authors       List   `yaml:"authors,omitempty"`
	RelatedADRs   List   `yaml:"related_adrs,omitempty"`
	AppliesTo     List   `yaml:"applies_to,omitempty"`
}

// RenderFront marshals a Front in canonical key order.
func RenderFront(front Front) (string, error) {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(front); err != nil {
		return "", err
	}
	if err := encoder.Close(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// KeyOrder is the canonical frontmatter key order used by `jerry fmt` and by
// every template jerry writes. Keys not listed here are unknown to jerry and
// are preserved, in their original relative order, after these.
var KeyOrder = []string{
	"schema_version",
	"id",
	"title",
	"status",
	"superseded_by",
	"supersedes",
	"team",
	"teams",
	"date",
	"deciders",
	"authors",
	"related_adrs",
	"applies_to",
}

// Split separates the frontmatter block from the body. The returned front is
// the YAML between the fences; body is everything after the closing fence.
func Split(text string) (front, body string, err error) {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != Delim {
		return "", "", ErrNoFrontmatter
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == Delim {
			return strings.Join(lines[1:i], "\n"), strings.Join(lines[i+1:], "\n"), nil
		}
	}
	return "", "", ErrUnterminated
}

// Parse splits and decodes a document's frontmatter, returning both views. The
// node is nil when the block is empty.
func Parse(text string) (Front, *yaml.Node, string, error) {
	raw, body, err := Split(text)
	if err != nil {
		return Front{}, nil, "", err
	}

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(raw), &root); err != nil {
		return Front{}, nil, body, fmt.Errorf("frontmatter is not valid YAML: %w", err)
	}
	if len(root.Content) == 0 {
		return Front{}, nil, body, nil
	}
	mapping := root.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return Front{}, nil, body, fmt.Errorf("frontmatter must be a mapping of key: value pairs")
	}

	var front Front
	if err := mapping.Decode(&front); err != nil {
		return Front{}, mapping, body, fmt.Errorf("frontmatter: %w", err)
	}
	// The frontmatter block starts on line 2 of the file, but yaml.v3 numbers
	// it from 1 — shift every node so reported lines match the real file.
	shift(mapping, 1)
	return front, mapping, body, nil
}

func shift(node *yaml.Node, by int) {
	if node == nil {
		return
	}
	node.Line += by
	for _, child := range node.Content {
		shift(child, by)
	}
}

// FieldLine returns the source line of a frontmatter key, or 1 when the key is
// absent — findings always carry a line a reader can jump to.
func FieldLine(mapping *yaml.Node, key string) int {
	if mapping == nil {
		return 1
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i].Line
		}
	}
	return 1
}

// HasField reports whether a key is present at all, which is not the same as
// its decoded value being non-zero: `title:` with no value is present but empty.
func HasField(mapping *yaml.Node, key string) bool {
	if mapping == nil {
		return false
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return true
		}
	}
	return false
}

// Canonical re-emits a frontmatter mapping with keys in KeyOrder, preserving
// unknown keys (in their original relative order) after the known ones. It
// works on the node rather than on Front so that a key jerry has never heard
// of survives a format pass untouched.
func Canonical(mapping *yaml.Node) (string, error) {
	if mapping == nil {
		return "", nil
	}

	rank := make(map[string]int, len(KeyOrder))
	for i, key := range KeyOrder {
		rank[key] = i
	}

	type pair struct {
		key   *yaml.Node
		value *yaml.Node
		rank  int
		order int
	}
	pairs := make([]pair, 0, len(mapping.Content)/2)
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key := mapping.Content[i]
		position, known := rank[key.Value]
		if !known {
			position = len(KeyOrder)
		}
		pairs = append(pairs, pair{key: key, value: mapping.Content[i+1], rank: position, order: i})
	}
	// A stable sort by rank keeps unknown keys in their authored order.
	for i := 1; i < len(pairs); i++ {
		for j := i; j > 0 && pairs[j-1].rank > pairs[j].rank; j-- {
			pairs[j-1], pairs[j] = pairs[j], pairs[j-1]
		}
	}

	out := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, p := range pairs {
		out.Content = append(out.Content, p.key, p.value)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(out); err != nil {
		return "", err
	}
	if err := encoder.Close(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Render assembles a whole document from a canonical frontmatter block and a
// body, normalising trailing whitespace to exactly one final newline.
func Render(front, body string) string {
	var buf strings.Builder
	buf.WriteString(Delim + "\n")
	buf.WriteString(strings.TrimRight(front, "\n") + "\n")
	buf.WriteString(Delim + "\n")
	trimmed := strings.TrimLeft(body, "\n")
	if trimmed != "" {
		buf.WriteString("\n" + strings.TrimRight(trimmed, "\n") + "\n")
	}
	return buf.String()
}
