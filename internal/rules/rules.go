package rules

import (
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/codcod/jerry/internal/doc"
)

// Options tunes the rules an organisation can reasonably disagree about.
type Options struct {
	RequiredADRSections []string
	RequiredSDSections  []string
	ProposedStaleDays   int
	// Today is injected so the staleness rule is deterministic under test.
	Today time.Time
	// Placeholders are template phrases that must not survive into a real
	// document. A half-filled template is the single most common defect: it
	// passes every structural check while saying nothing.
	Placeholders []string
}

// DefaultOptions matches what `jerry init` scaffolds.
func DefaultOptions() Options {
	return Options{
		RequiredADRSections: []string{"## Context", "## Decision", "## Consequences"},
		RequiredSDSections:  []string{"## Problem Statement", "## Proposed Design", "## Risks & Trade-offs"},
		ProposedStaleDays:   90,
		Today:               time.Now(),
		Placeholders: []string{
			"YYYY-MM-DD",
			"ADR-NNNN",
			"Short title of the decision",
			"Short title of the design",
			"your-team",
			"**Option A** — why it was rejected",
			"[alice, bob]",
		},
	}
}

// Check runs every rule over a corpus and returns the findings, sorted.
func Check(corpus *doc.Corpus, options Options) Findings {
	var findings Findings

	for _, document := range corpus.Docs {
		switch document.Kind {
		case doc.KindADR:
			checkADR(&findings, corpus, document, options)
		case doc.KindSD:
			checkSD(&findings, corpus, document, options)
		}
		checkPlaceholders(&findings, document, options)
		checkAppliesTo(&findings, document)
		checkUnknownKeys(&findings, document)
		checkSchemaVersion(&findings, document)
	}
	checkDuplicateIDs(&findings, corpus)

	findings.Sort()
	return findings
}

// checkDuplicateIDs is the hazard per-folder numbering creates: two branches
// cut around the same time both pick "the next number", and neither notices
// until they are both on main.
func checkDuplicateIDs(findings *Findings, corpus *doc.Corpus) {
	byDir := map[string]map[string][]string{}
	for _, document := range corpus.Docs {
		if document.Kind != doc.KindADR || document.FileID == "" {
			continue
		}
		dir := path.Dir(document.Path)
		if byDir[dir] == nil {
			byDir[dir] = map[string][]string{}
		}
		byDir[dir][document.FileID] = append(byDir[dir][document.FileID], path.Base(document.Path))
	}

	dirs := make([]string, 0, len(byDir))
	for dir := range byDir {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	for _, dir := range dirs {
		ids := make([]string, 0, len(byDir[dir]))
		for id := range byDir[dir] {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			names := byDir[dir][id]
			if len(names) < 2 {
				continue
			}
			sort.Strings(names)
			findings.errorf(path.Join(dir, names[0]), 1, "duplicate-id",
				"duplicate ADR id %s in %s: %s — renumber one to the next free number in this folder",
				id, dir, strings.Join(names, ", "))
		}
	}
}

func checkADR(findings *Findings, corpus *doc.Corpus, document *doc.Document, options Options) {
	if document.FileID == "" {
		findings.errorf(document.Path, 1, "filename",
			"filename must be NNNN-short-title.md (lowercase, hyphenated)")
	}
	if document.ParseErr != nil {
		findings.errorf(document.Path, 1, "frontmatter", "%s", document.ParseErr)
		return
	}

	front := document.Front
	line := func(key string) int { return doc.FieldLine(document.Mapping, key) }

	match := doc.ADRID.FindStringSubmatch(front.ID)
	switch {
	case match == nil:
		findings.errorf(document.Path, line("id"), "id-format",
			"frontmatter id must be ADR-NNNN, got %q", front.ID)
	case document.FileID != "" && match[1] != document.FileID:
		findings.errorf(document.Path, line("id"), "id-mismatch",
			"frontmatter id %s does not match filename number %s", front.ID, document.FileID)
	}

	if front.Title == "" {
		findings.errorf(document.Path, 1, "missing-field", "frontmatter is missing `title`")
	}

	if !doc.ValidStatus(doc.KindADR, front.Status) {
		findings.errorf(document.Path, line("status"), "status",
			"status %q is not one of: %s. Use `status: Superseded` plus `superseded_by:` rather than "+
				"putting the successor in the status field", front.Status, strings.Join(doc.ADRStatuses, ", "))
	}

	// The successor is a field rather than prose in the status, which is the
	// whole reason this link can be machine-checked at all.
	if front.Status == doc.StatusSuperseded && len(front.SupersededBy) == 0 {
		findings.errorf(document.Path, line("status"), "superseded-by-missing",
			"status is Superseded but `superseded_by` is missing")
	}
	if len(front.SupersededBy) > 0 && front.Status != doc.StatusSuperseded {
		findings.errorf(document.Path, line("superseded_by"), "superseded-by-orphan",
			"`superseded_by` is set but status is %q", front.Status)
	}
	checkRefs(findings, corpus, document, "superseded_by", front.SupersededBy)
	checkRefs(findings, corpus, document, "supersedes", front.Supersedes)

	if document.Scope == doc.CrossCutting {
		if len(front.Teams) < 2 {
			findings.errorf(document.Path, line("teams"), "cross-cutting-teams",
				"cross-cutting ADRs need a `teams:` list naming every affected team (two or more). "+
					"If only one team is affected, it belongs under teams/<team>/adr/")
		}
		if front.Team != "" {
			findings.errorf(document.Path, line("team"), "cross-cutting-team-field",
				"cross-cutting ADRs use `teams:`, not `team:`")
		}
	} else if front.Team != document.Scope {
		findings.errorf(document.Path, line("team"), "team-mismatch",
			"frontmatter team %q does not match folder %q", front.Team, document.Scope)
	}

	checkDate(findings, document, options)

	if len(front.Deciders) == 0 {
		findings.errorf(document.Path, 1, "missing-field", "frontmatter is missing `deciders`")
	}

	checkSections(findings, document, options.RequiredADRSections)
}

func checkSD(findings *Findings, corpus *doc.Corpus, document *doc.Document, options Options) {
	if document.FileID == "" {
		findings.errorf(document.Path, 1, "filename",
			"filename must be YYYY-MM-short-title.md (lowercase, hyphenated)")
	}
	if document.ParseErr != nil {
		findings.errorf(document.Path, 1, "frontmatter", "%s", document.ParseErr)
		return
	}

	front := document.Front
	line := func(key string) int { return doc.FieldLine(document.Mapping, key) }

	if front.Title == "" {
		findings.errorf(document.Path, 1, "missing-field", "frontmatter is missing `title`")
	}
	if !doc.ValidStatus(doc.KindSD, front.Status) {
		findings.errorf(document.Path, line("status"), "status",
			"status %q is not one of: %s", front.Status, strings.Join(doc.SDStatuses, ", "))
	}

	if parsed, ok := checkDate(findings, document, options); ok && document.FileID != "" {
		if prefix := parsed.Format("2006-01"); prefix != document.FileID {
			findings.errorf(document.Path, line("date"), "filename-date-mismatch",
				"filename prefix %s does not match date %s", document.FileID, front.Date)
		}
	}

	if document.Scope != doc.CrossCutting && front.Team != document.Scope {
		findings.errorf(document.Path, line("team"), "team-mismatch",
			"frontmatter team %q does not match folder %q", front.Team, document.Scope)
	}
	if len(front.Authors) == 0 {
		findings.errorf(document.Path, 1, "missing-field", "frontmatter is missing `authors`")
	}

	checkRefs(findings, corpus, document, "related_adrs", front.RelatedADRs)
	checkSections(findings, document, options.RequiredSDSections)
}

func checkRefs(findings *Findings, corpus *doc.Corpus, document *doc.Document, field string, refs doc.List) {
	for _, reference := range refs {
		if err := corpus.Resolve(reference, document.Scope); err != nil {
			findings.errorf(document.Path, doc.FieldLine(document.Mapping, field), "unresolved-ref", "%s", err)
		}
	}
}

func checkDate(findings *Findings, document *doc.Document, options Options) (time.Time, bool) {
	parsed, ok := document.ParsedDate()
	if !ok {
		findings.errorf(document.Path, doc.FieldLine(document.Mapping, "date"), "date",
			"date %q is not a valid ISO date (YYYY-MM-DD)", document.Front.Date)
		return time.Time{}, false
	}
	if document.Kind == doc.KindADR && document.Front.Status == doc.StatusProposed {
		if age := int(options.Today.Sub(parsed).Hours() / 24); age > options.ProposedStaleDays {
			findings.warnf(document.Path, doc.FieldLine(document.Mapping, "status"), "stale-proposal",
				"still Proposed after %d days — accept it, reject it, or close it out", age)
		}
	}
	return parsed, true
}

func checkSections(findings *Findings, document *doc.Document, required []string) {
	var missing []string
	for _, heading := range required {
		if !document.Section(heading) {
			missing = append(missing, heading)
			continue
		}
		// A heading with nothing under it is worse than a missing one: it
		// looks like the question was answered.
		if body, ok := document.SectionBody(heading); ok && stripComments(body) == "" {
			findings.errorf(document.Path, document.LineOf(heading), "empty-section",
				"section %q is empty — a heading with nothing under it reads as though the question was answered", heading)
		}
	}
	if len(missing) > 0 {
		findings.errorf(document.Path, 1, "missing-section",
			"missing required section(s): %s", strings.Join(missing, ", "))
	}
}

// placeholderAllowMarker silences checkPlaceholders for the whole document it
// appears in — the escape for a decision that legitimately discusses a
// placeholder phrase instead of leaving it unfilled.
const placeholderAllowMarker = "<!-- jerry:allow placeholder -->"

// fencedBlockPattern matches a fenced code block. Quoting a template is the
// common legitimate reason a real placeholder phrase appears in a document,
// and never indicates an unfilled one.
var fencedBlockPattern = regexp.MustCompile("(?s)```.*?```")

// checkPlaceholders catches a template that was copied and half-filled — the
// most common real defect, and one every structural check passes.
func checkPlaceholders(findings *Findings, document *doc.Document, options Options) {
	if strings.Contains(document.Raw, placeholderAllowMarker) {
		return
	}
	scanText := fencedBlockPattern.ReplaceAllString(document.Raw, "")
	for _, placeholder := range options.Placeholders {
		if strings.Contains(scanText, placeholder) {
			findings.errorf(document.Path, document.LineOf(placeholder), "placeholder",
				"template placeholder %q was never filled in", placeholder)
		}
	}
}

// checkAppliesTo rejects an applies_to entry that cannot be a path. Matching
// semantics (globs, existence) is separate, larger work — this only guards
// against the field silently governing nothing.
func checkAppliesTo(findings *Findings, document *doc.Document) {
	line := doc.FieldLine(document.Mapping, "applies_to")
	for _, entry := range document.Front.AppliesTo {
		if reason, bad := badAppliesToPath(entry); bad {
			findings.errorf(document.Path, line, "applies-to",
				"applies_to entry %q is not a valid path: %s", entry, reason)
		}
	}
}

func badAppliesToPath(entry string) (string, bool) {
	if strings.TrimSpace(entry) == "" {
		return "empty or whitespace-only", true
	}
	if path.IsAbs(entry) {
		return "must be relative", true
	}
	for _, segment := range strings.Split(entry, "/") {
		if segment == ".." {
			return "must not contain `..`", true
		}
	}
	return "", false
}

// knownKeys is doc.KeyOrder as a set, so checkUnknownKeys can test membership
// without a linear scan per key.
var knownKeys = func() map[string]bool {
	set := make(map[string]bool, len(doc.KeyOrder))
	for _, key := range doc.KeyOrder {
		set[key] = true
	}
	return set
}()

// checkUnknownKeys warns on a frontmatter key jerry does not recognise. A
// warning, never an error: a repository may legitimately carry fields jerry
// has no opinion about. `jerry fmt` still preserves the key untouched — this
// only makes its presence visible.
func checkUnknownKeys(findings *Findings, document *doc.Document) {
	mapping := document.Mapping
	if mapping == nil {
		return
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key := mapping.Content[i].Value
		if !knownKeys[key] {
			findings.warnf(document.Path, mapping.Content[i].Line, "unknown-key",
				"frontmatter key %q is not one jerry knows — check for a typo (preserved as-is by `jerry fmt`)", key)
		}
	}
}

// checkSchemaVersion warns when a document claims a schema_version newer than
// this binary knows, and never blocks: DESIGN.md §3.6 makes "the binary is
// replaceable" a charter item, so an old jerry must not turn a newer
// repository red merely for existing.
func checkSchemaVersion(findings *Findings, document *doc.Document) {
	if document.Front.SchemaVersion <= doc.CurrentSchemaVersion {
		return
	}
	findings.warnf(document.Path, doc.FieldLine(document.Mapping, "schema_version"), "schema-version-ahead",
		"schema_version %d is newer than this jerry binary knows (%d) — checked with %[2]d's rules; upgrade jerry to read %[1]d's own rules",
		document.Front.SchemaVersion, doc.CurrentSchemaVersion)
}

// commentPattern matches an HTML comment. The guidance prompts `jerry new`
// writes are comments, so they do not count as content: a document created
// this morning and never filled in must still fail the empty-section rule.
var commentPattern = regexp.MustCompile(`(?s)<!--.*?-->`)

func stripComments(body string) string {
	return strings.TrimSpace(commentPattern.ReplaceAllString(body, ""))
}
