// Package rules holds every check jerry makes about a document corpus.
//
// A rule is a function over the corpus that appends Findings; nothing stops at
// the first problem, because a validator that reports one error per run turns
// a five-minute fix into five CI round-trips. Severity decides the exit code,
// not whether the finding is printed.
package rules

import (
	"fmt"
	"sort"
)

// Severity distinguishes a failure from a nudge.
type Severity string

const (
	// SeverityError fails validation.
	SeverityError Severity = "error"
	// SeverityWarning is reported but does not fail. Used for judgements that
	// need a human — an ADR left Proposed for months is a loose end, not a
	// defect, and blocking CI on the calendar would only teach people to lie
	// about dates.
	SeverityWarning Severity = "warning"
)

// Finding is one problem with one document.
type Finding struct {
	Path     string   `json:"path"`
	Line     int      `json:"line"`
	Severity Severity `json:"severity"`
	Rule     string   `json:"rule"`
	Message  string   `json:"message"`
}

// String renders a finding as one grep-friendly line.
func (f Finding) String() string {
	return fmt.Sprintf("%s:%d: %s: %s (%s)", f.Path, f.Line, f.Severity, f.Message, f.Rule)
}

// Findings is an ordered collection with the accumulate-everything helpers the
// rules use.
type Findings []Finding

func (f *Findings) errorf(path string, line int, rule, format string, args ...any) {
	*f = append(*f, Finding{Path: path, Line: line, Severity: SeverityError, Rule: rule, Message: fmt.Sprintf(format, args...)})
}

func (f *Findings) warnf(path string, line int, rule, format string, args ...any) {
	*f = append(*f, Finding{Path: path, Line: line, Severity: SeverityWarning, Rule: rule, Message: fmt.Sprintf(format, args...)})
}

// Errors counts the findings that fail validation.
func (f Findings) Errors() int {
	count := 0
	for _, finding := range f {
		if finding.Severity == SeverityError {
			count++
		}
	}
	return count
}

// Warnings counts the non-blocking findings.
func (f Findings) Warnings() int { return len(f) - f.Errors() }

// Sort orders findings by path then line, so output is stable across runs and
// diffable between them.
func (f Findings) Sort() {
	sort.SliceStable(f, func(i, j int) bool {
		if f[i].Path != f[j].Path {
			return f[i].Path < f[j].Path
		}
		return f[i].Line < f[j].Line
	})
}
