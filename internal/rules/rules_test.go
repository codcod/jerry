package rules

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/codcod/jerry/internal/doc"
)

// updateGolden regenerates the committed fixtures. Fixtures are always
// rendered through the production path and byte-compared — one patched by hand
// asserts what someone believed, not what the code does.
var updateGolden = flag.Bool("update", false, "regenerate golden fixtures")

// fixedToday pins the clock so the stale-proposal rule is deterministic. A
// fixture that depends on the wall clock rots silently.
var fixedToday = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

func testOptions() Options {
	options := DefaultOptions()
	options.Today = fixedToday
	return options
}

func loadCorpus(t *testing.T) *doc.Corpus {
	t.Helper()
	corpus, err := doc.Load(filepath.Join("testdata", "corpus"), doc.DefaultLayout())
	if err != nil {
		t.Fatalf("loading corpus: %v", err)
	}
	return corpus
}

func TestCheckGolden(t *testing.T) {
	findings := Check(loadCorpus(t), testOptions())

	rendered, err := json.MarshalIndent(findings, "", "  ")
	if err != nil {
		t.Fatalf("marshalling findings: %v", err)
	}
	rendered = append(rendered, '\n')

	golden := filepath.Join("testdata", "findings.golden.json")
	if *updateGolden {
		if err := os.WriteFile(golden, rendered, 0o644); err != nil {
			t.Fatalf("writing fixture: %v", err)
		}
		return
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("reading fixture (run: go test ./internal/rules -update): %v", err)
	}
	if string(rendered) != string(want) {
		t.Fatalf("committed fixture does not match rendered output.\nRun: go test ./internal/rules -update\n--- got ---\n%s\n--- want ---\n%s", rendered, want)
	}
}

// TestCheckFixtureContract asserts the properties the fixture exists to
// demonstrate, so regenerating it can never quietly drop them.
func TestCheckFixtureContract(t *testing.T) {
	findings := Check(loadCorpus(t), testOptions())

	byRule := map[string]int{}
	for _, finding := range findings {
		byRule[finding.Rule]++
	}

	wanted := []string{
		"id-mismatch",         // frontmatter id disagrees with the filename
		"status",              // successor smuggled into the status field
		"team-mismatch",       // frontmatter team disagrees with the folder
		"date",                // 2026-13-01 is not a month
		"missing-field",       // no deciders
		"missing-section",     // no Decision / Consequences
		"duplicate-id",        // two ADR-0001 in one folder
		"unresolved-ref",      // three flavours of dangling reference
		"empty-section",       // heading with nothing under it
		"placeholder",         // template copied and never filled in
		"cross-cutting-teams", // cross-cutting naming fewer than two teams
		"cross-cutting-team-field",
		"filename-date-mismatch", // SD prefix disagrees with its date
		"stale-proposal",         // the only warning in the fixture
	}
	for _, rule := range wanted {
		if byRule[rule] == 0 {
			t.Errorf("fixture no longer exercises rule %q", rule)
		}
	}

	t.Run("CleanDocumentStaysQuiet", func(t *testing.T) {
		// A title containing a colon is the case the previous hand-rolled
		// parser truncated; it must produce no findings at all.
		for _, finding := range findings {
			if strings.Contains(finding.Path, "0001-use-outbox-for-event-publishing.md") {
				t.Errorf("clean document produced a finding: %s", finding)
			}
		}
	})

	t.Run("StaleProposalIsAWarningNotAnError", func(t *testing.T) {
		for _, finding := range findings {
			if finding.Rule == "stale-proposal" && finding.Severity != SeverityWarning {
				t.Errorf("stale-proposal must never fail CI, got severity %q", finding.Severity)
			}
		}
	})

	t.Run("ResolvableCrossScopeRefIsAccepted", func(t *testing.T) {
		// payments/ADR-0001 exists, so only the two broken references in that
		// same field may be reported.
		for _, finding := range findings {
			if finding.Rule == "unresolved-ref" && strings.Contains(finding.Message, "payments/ADR-0001") {
				t.Errorf("a resolvable scope-qualified reference was reported: %s", finding)
			}
		}
	})

	t.Run("EveryFindingCarriesALine", func(t *testing.T) {
		for _, finding := range findings {
			if finding.Line < 1 {
				t.Errorf("%s: line %d is not addressable", finding.Rule, finding.Line)
			}
		}
	})
}

func TestFindingsSortIsStable(t *testing.T) {
	findings := Findings{
		{Path: "b.md", Line: 1, Rule: "x"},
		{Path: "a.md", Line: 9, Rule: "y"},
		{Path: "a.md", Line: 2, Rule: "z"},
	}
	findings.Sort()

	want := []string{"a.md:2", "a.md:9", "b.md:1"}
	for i, finding := range findings {
		got := finding.Path + ":" + string(rune('0'+finding.Line))
		if got != want[i] {
			t.Errorf("position %d: got %s, want %s", i, got, want[i])
		}
	}
}
