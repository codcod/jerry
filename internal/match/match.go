// Package match resolves a changed path against every document's applies_to
// entries, using gitignore-style patterns with a most-specific-pattern-wins
// precedence rule. Semantics are specified in DESIGN.md §4.1 — this package
// is that specification's only implementation, so a behaviour change here is
// a behaviour change to the spec too.
package match

import (
	"path"
	"sort"
	"strings"

	"github.com/codcod/jerry/internal/doc"
)

// Match is one (document, pattern) pair where the document's applies_to
// entry Pattern matched the queried path.
type Match struct {
	Doc     *doc.Document
	Pattern string
}

// Resolve returns every document in the corpus whose applies_to matches
// changedPath, most-specific pattern first (DESIGN.md §4.1). It does not
// collapse to a single winner — that choice belongs to the caller.
func Resolve(corpus *doc.Corpus, changedPath string) []Match {
	var matches []Match
	for _, document := range corpus.Docs {
		for _, pattern := range document.Front.AppliesTo {
			if matchPattern(pattern, changedPath) {
				matches = append(matches, Match{Doc: document, Pattern: pattern})
			}
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		return lessSpecific(matches[j], matches[i]) // descending: most specific first
	})
	return matches
}

// lessSpecific reports whether a is strictly less specific than b, per
// DESIGN.md §4.1's three-step precedence rule.
func lessSpecific(a, b Match) bool {
	aLiteral, aLen := specificity(a.Pattern)
	bLiteral, bLen := specificity(b.Pattern)
	if aLiteral != bLiteral {
		return aLiteral < bLiteral
	}
	if aLen != bLen {
		return aLen < bLen
	}
	return a.Doc.Path > b.Doc.Path // lexicographically later Path sorts as "less specific"
}

// specificity scores a pattern: literalSegments is the count of '/'-split
// segments containing none of '*', '?', '[' (a directory-prefix's implicit
// remainder and a literal "**" segment both count 0); length is the pattern's
// character count, the tie-break beneath it.
func specificity(pattern string) (literalSegments, length int) {
	trimmed := strings.TrimSuffix(pattern, "/")
	for _, segment := range strings.Split(trimmed, "/") {
		if isLiteralSegment(segment) {
			literalSegments++
		}
	}
	return literalSegments, len(pattern)
}

func isLiteralSegment(segment string) bool {
	if segment == "**" {
		return false
	}
	return !strings.ContainsAny(segment, "*?[")
}

// matchPattern reports whether pattern matches candidate, per DESIGN.md
// §4.1's dialect: a trailing '/' is a directory-prefix, matching everything
// nested under it; otherwise the two are matched segment-by-segment, with
// '**' matching zero or more whole segments.
func matchPattern(pattern, candidate string) bool {
	if dir, ok := strings.CutSuffix(pattern, "/"); ok {
		return candidate == dir || strings.HasPrefix(candidate, dir+"/")
	}
	return matchSegments(strings.Split(pattern, "/"), strings.Split(candidate, "/"))
}

// matchSegments walks pattern and candidate in lockstep, backtracking over
// every possible span a "**" could consume. Paths in this corpus are a
// handful of segments deep, so the backtrack is effectively O(1), not a
// scaling concern.
func matchSegments(pattern, candidate []string) bool {
	if len(pattern) == 0 {
		return len(candidate) == 0
	}
	if pattern[0] == "**" {
		for consumed := 0; consumed <= len(candidate); consumed++ {
			if matchSegments(pattern[1:], candidate[consumed:]) {
				return true
			}
		}
		return false
	}
	if len(candidate) == 0 {
		return false
	}
	ok, err := path.Match(pattern[0], candidate[0])
	if err != nil || !ok {
		return false
	}
	return matchSegments(pattern[1:], candidate[1:])
}
