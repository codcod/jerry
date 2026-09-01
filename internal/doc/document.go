package doc

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Kind distinguishes the two document types jerry manages.
type Kind int

const (
	KindADR Kind = iota
	KindSD
)

// String renders a kind for messages.
func (k Kind) String() string {
	if k == KindSD {
		return "solution design"
	}
	return "ADR"
}

// Filename patterns. ADRs are NNNN-slug.md, numbered per folder. Solution
// designs have no sequential id — the date is the identifier — so they are
// YYYY-MM-slug.md and the prefix must agree with the frontmatter date.
var (
	ADRFilename = regexp.MustCompile(`^(\d{4})-[a-z0-9]+(?:-[a-z0-9]+)*\.md$`)
	SDFilename  = regexp.MustCompile(`^(\d{4}-\d{2})-[a-z0-9]+(?:-[a-z0-9]+)*\.md$`)
	ADRID       = regexp.MustCompile(`^ADR-(\d{4})$`)
)

// DateLayout is the only date spelling jerry accepts.
const DateLayout = "2006-01-02"

// CrossCutting is the scope reserved for decisions no single team owns.
const CrossCutting = "cross-cutting"

// Document is one ADR or Solution Design on disk.
type Document struct {
	// Path is repo-relative and slash-separated, so findings and index links
	// read the same on every platform.
	Path  string
	Kind  Kind
	Scope string
	// FileID is the identifying part of the filename: "0001" for an ADR,
	// "2026-05" for a solution design. Empty when the filename is malformed.
	FileID string

	Front   Front
	Mapping *yaml.Node
	Body    string
	Raw     string

	// ParseErr records a frontmatter block that could not be read. Such a
	// document still appears in the corpus — reporting "unreadable" is more
	// useful than silently omitting it — but field rules are skipped.
	ParseErr error
}

// LoadFile reads and parses one document.
func LoadFile(root, relPath string, kind Kind, scope string) *Document {
	document := &Document{Path: relPath, Kind: kind, Scope: scope}

	raw, err := os.ReadFile(path.Join(root, relPath))
	if err != nil {
		document.ParseErr = err
		return document
	}
	document.Raw = string(raw)

	name := path.Base(relPath)
	pattern := ADRFilename
	if kind == KindSD {
		pattern = SDFilename
	}
	if match := pattern.FindStringSubmatch(name); match != nil {
		document.FileID = match[1]
	}

	front, mapping, body, err := Parse(document.Raw)
	document.Front, document.Mapping, document.Body, document.ParseErr = front, mapping, body, err
	return document
}

// Ref is the document's own reference, usable from any scope.
func (d *Document) Ref() Ref { return Ref{Scope: d.Scope, ID: d.FileID} }

// Title falls back to the filename stem so an untitled document still renders
// something a reader can click in the index.
func (d *Document) Title() string {
	if d.Front.Title != "" {
		return d.Front.Title
	}
	return strings.TrimSuffix(path.Base(d.Path), ".md")
}

// ParsedDate returns the frontmatter date, or false when it is missing or not
// a real ISO date.
func (d *Document) ParsedDate() (time.Time, bool) {
	parsed, err := time.Parse(DateLayout, d.Front.Date)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

// ScopeLabel is how the scope appears in the index: a bare team name, or
// cross-cutting annotated with the teams it binds.
func (d *Document) ScopeLabel() string {
	if d.Scope == CrossCutting && len(d.Front.Teams) > 0 {
		return fmt.Sprintf("cross-cutting (%s)", strings.Join(d.Front.Teams, ", "))
	}
	return d.Scope
}

// StatusLabel renders the status with its successor folded in, so the index
// answers "superseded by what?" without a second lookup.
func (d *Document) StatusLabel() string {
	status := d.Front.Status
	if status == "" {
		return "-"
	}
	if status == StatusSuperseded && len(d.Front.SupersededBy) > 0 {
		return "Superseded by " + strings.Join(d.Front.SupersededBy, ", ")
	}
	return status
}

// Section reports whether a markdown heading is present in the body.
func (d *Document) Section(heading string) bool {
	return strings.Contains(d.Body, heading)
}

// SectionBody returns the text under a heading, up to the next heading of the
// same or higher level. It is what the empty-section rule inspects.
func (d *Document) SectionBody(heading string) (string, bool) {
	index := strings.Index(d.Body, heading)
	if index < 0 {
		return "", false
	}
	rest := d.Body[index+len(heading):]
	level := len(heading) - len(strings.TrimLeft(heading, "#"))

	lines := strings.Split(rest, "\n")
	var collected []string
	for _, line := range lines[min(1, len(lines)):] {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			depth := len(trimmed) - len(strings.TrimLeft(trimmed, "#"))
			if depth <= level {
				break
			}
		}
		collected = append(collected, line)
	}
	return strings.TrimSpace(strings.Join(collected, "\n")), true
}

// LineOf returns the 1-indexed body line a substring appears on, expressed in
// whole-file coordinates.
func (d *Document) LineOf(needle string) int {
	index := strings.Index(d.Raw, needle)
	if index < 0 {
		return 1
	}
	return strings.Count(d.Raw[:index], "\n") + 1
}

// Slug turns a title into the hyphenated lowercase form used in filenames.
// Anything that is not a letter or digit becomes a separator, so a title with
// punctuation still produces a filename the validator accepts.
func Slug(title string) string {
	var builder strings.Builder
	lastWasDash := true // suppresses a leading dash
	for _, r := range strings.ToLower(title) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			builder.WriteRune(r)
			lastWasDash = false
		default:
			if !lastWasDash {
				builder.WriteByte('-')
				lastWasDash = true
			}
		}
	}
	return strings.Trim(builder.String(), "-")
}
