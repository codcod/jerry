package match

import (
	"testing"

	"github.com/codcod/jerry/internal/doc"
)

func docWith(path string, appliesTo ...string) *doc.Document {
	return &doc.Document{Path: path, Front: doc.Front{AppliesTo: doc.List(appliesTo)}}
}

func corpusOf(docs ...*doc.Document) *doc.Corpus {
	return &doc.Corpus{Docs: docs}
}

func TestResolveMatchPattern(t *testing.T) {
	cases := []struct {
		name      string
		pattern   string
		candidate string
		want      bool
	}{
		{"directory-prefix match", "internal/rules/", "internal/rules/rules.go", true},
		{"directory-prefix matches itself minus slash", "internal/rules/", "internal/rules", true},
		{"directory-prefix non-match sibling", "internal/rules/", "internal/index/index.go", false},
		{"directory-prefix non-match prefix collision", "internal/rules/", "internal/rulesx/x.go", false},
		{"single-segment glob match", "internal/*/rules.go", "internal/rules/rules.go", true},
		{"single-segment glob does not cross slash", "internal/*.go", "internal/rules/rules.go", false},
		{"double-star matches zero segments", "internal/**/rules.go", "internal/rules.go", true},
		{"double-star matches one segment", "internal/**/rules.go", "internal/rules/rules.go", true},
		{"double-star matches several segments", "internal/**/rules.go", "internal/a/b/c/rules.go", true},
		{"no match at all", "teams/payments/", "teams/checkout/adr/0001-x.md", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchPattern(tc.pattern, tc.candidate); got != tc.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tc.pattern, tc.candidate, got, tc.want)
			}
		})
	}
}

func TestResolvePrecedence(t *testing.T) {
	t.Run("more specific pattern wins by literal segment count", func(t *testing.T) {
		shallow := docWith("teams/a/adr/0001-shallow.md", "internal/")
		deep := docWith("teams/b/adr/0001-deep.md", "internal/rules/")
		corpus := corpusOf(shallow, deep)

		matches := Resolve(corpus, "internal/rules/rules.go")
		if len(matches) != 2 {
			t.Fatalf("got %d matches, want 2", len(matches))
		}
		if matches[0].Doc != deep {
			t.Errorf("most specific match first: got %s, want %s", matches[0].Doc.Path, deep.Path)
		}
		if matches[1].Doc != shallow {
			t.Errorf("least specific match last: got %s, want %s", matches[1].Doc.Path, shallow.Path)
		}
	})

	t.Run("tied literal segments, longer pattern wins", func(t *testing.T) {
		// Both patterns have 2 literal segments ("internal", "rules.go"); the
		// middle segment is a wildcard in each, but a longer one in "long".
		short := docWith("teams/a/adr/0001-short.md", "internal/*/rules.go")
		long := docWith("teams/b/adr/0001-long.md", "internal/ru??s/rules.go")
		corpus := corpusOf(short, long)

		matches := Resolve(corpus, "internal/rules/rules.go")
		if len(matches) != 2 {
			t.Fatalf("got %d matches, want 2", len(matches))
		}
		if matches[0].Doc != long {
			t.Errorf("longer pattern first: got %s, want %s", matches[0].Doc.Path, long.Path)
		}
	})

	t.Run("no match returns empty, not nil-panicking", func(t *testing.T) {
		corpus := corpusOf(docWith("teams/a/adr/0001-x.md", "teams/payments/"))
		matches := Resolve(corpus, "teams/checkout/adr/0001-y.md")
		if len(matches) != 0 {
			t.Fatalf("got %d matches, want 0", len(matches))
		}
	})
}
