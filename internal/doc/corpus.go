package doc

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Layout names the directories a corpus is assembled from. It comes from
// jerry.yaml so an organisation can rename the folders without forking jerry.
type Layout struct {
	ADRDir string
	SDDir  string
	Skip   []string
}

// DefaultLayout matches what `jerry init` scaffolds.
func DefaultLayout() Layout {
	return Layout{
		ADRDir: "adr",
		SDDir:  "solution-designs",
		Skip:   []string{"templates", "index", ".git", "node_modules", "dist"},
	}
}

// Corpus is every document in one repository, plus the id registry that makes
// references resolvable.
type Corpus struct {
	Root   string
	Layout Layout
	Docs   []*Document

	// registry is scope -> set of ADR ids present, which is what reference
	// resolution consults. It is built from filenames, not frontmatter, so a
	// document with broken frontmatter can still be the target of a link.
	registry map[string]map[string]bool
}

// Load walks a repository and reads every ADR and Solution Design in it.
func Load(root string, layout Layout) (*Corpus, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	corpus := &Corpus{Root: absolute, Layout: layout, registry: map[string]map[string]bool{}}

	skip := make(map[string]bool, len(layout.Skip))
	for _, name := range layout.Skip {
		skip[name] = true
	}

	err = filepath.WalkDir(absolute, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			return nil
		}
		if skip[entry.Name()] {
			return fs.SkipDir
		}

		relative, err := filepath.Rel(absolute, current)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)

		var kind Kind
		switch entry.Name() {
		case layout.ADRDir:
			kind = KindADR
		case layout.SDDir:
			kind = KindSD
		default:
			return nil
		}

		scope := ScopeOf(relative)
		if scope == "" {
			return nil
		}
		return corpus.loadDir(relative, kind, scope)
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(corpus.Docs, func(i, j int) bool { return corpus.Docs[i].Path < corpus.Docs[j].Path })
	return corpus, nil
}

func (c *Corpus) loadDir(relDir string, kind Kind, scope string) error {
	entries, err := os.ReadDir(path.Join(c.Root, relDir))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".md") || name == "README.md" {
			continue
		}
		document := LoadFile(c.Root, path.Join(relDir, name), kind, scope)
		c.Docs = append(c.Docs, document)
		if kind == KindADR && document.FileID != "" {
			if c.registry[scope] == nil {
				c.registry[scope] = map[string]bool{}
			}
			c.registry[scope][document.FileID] = true
		}
	}
	return nil
}

// ScopeOf derives a document's scope from the directory holding it:
// "payments" for teams/payments/adr, "cross-cutting" for cross-cutting/adr.
func ScopeOf(relDir string) string {
	parts := strings.Split(relDir, "/")
	if len(parts) == 0 {
		return ""
	}
	if parts[0] == "teams" {
		if len(parts) < 2 {
			return ""
		}
		return parts[1]
	}
	return parts[0]
}

// Scopes lists every scope that holds at least one ADR, sorted.
func (c *Corpus) Scopes() []string {
	scopes := make([]string, 0, len(c.registry))
	for scope := range c.registry {
		scopes = append(scopes, scope)
	}
	sort.Strings(scopes)
	return scopes
}

// Resolve reports why a reference does not resolve, or nil when it does.
// Errors name the scope explicitly, because the most common mistake is a bare
// id that silently means "this folder" when the author meant another one.
func (c *Corpus) Resolve(reference, defaultScope string) error {
	parsed, err := ParseRef(reference, defaultScope)
	if err != nil {
		return err
	}
	ids, known := c.registry[parsed.Scope]
	if !known {
		return fmt.Errorf("%q points at unknown scope %q", reference, parsed.Scope)
	}
	if !ids[parsed.ID] {
		return fmt.Errorf("%q does not resolve — no ADR-%s in %q", reference, parsed.ID, parsed.Scope)
	}
	return nil
}

// Find returns the document a reference names.
func (c *Corpus) Find(reference, defaultScope string) (*Document, error) {
	parsed, err := ParseRef(reference, defaultScope)
	if err != nil {
		return nil, err
	}
	for _, document := range c.Docs {
		if document.Kind == KindADR && document.Scope == parsed.Scope && document.FileID == parsed.ID {
			return document, nil
		}
	}
	return nil, fmt.Errorf("%s not found", parsed)
}

// FindAny resolves a reference for the ADR case and a path for anything else,
// which is what the CLI needs: `jerry status` takes either.
func (c *Corpus) FindAny(target string) (*Document, error) {
	if _, err := ParseRef(target, "x"); err == nil {
		return c.Find(target, "")
	}
	wanted := filepath.ToSlash(target)
	for _, document := range c.Docs {
		if document.Path == wanted || strings.HasSuffix(document.Path, "/"+wanted) {
			return document, nil
		}
	}
	return nil, fmt.Errorf("%q matches no document (give a reference like payments/ADR-0007 or a path)", target)
}

// NextID returns the next free four-digit ADR id in a scope. Gaps are not
// reused: a number that was once taken stays spent, so an id in a commit
// message or a chat thread never comes to mean a second document.
func (c *Corpus) NextID(scope string) string {
	highest := 0
	for id := range c.registry[scope] {
		value := 0
		if _, err := fmt.Sscanf(id, "%04d", &value); err == nil && value > highest {
			highest = value
		}
	}
	return fmt.Sprintf("%04d", highest+1)
}

// DirFor returns the repo-relative directory a new document of this kind and
// scope belongs in.
func (c *Corpus) DirFor(kind Kind, scope string) string {
	leaf := c.Layout.ADRDir
	if kind == KindSD {
		leaf = c.Layout.SDDir
	}
	if scope == CrossCutting {
		return path.Join(CrossCutting, leaf)
	}
	return path.Join("teams", scope, leaf)
}

// ByKind filters the corpus.
func (c *Corpus) ByKind(kind Kind) []*Document {
	var out []*Document
	for _, document := range c.Docs {
		if document.Kind == kind {
			out = append(out, document)
		}
	}
	return out
}
