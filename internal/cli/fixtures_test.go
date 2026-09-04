package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/codcod/jerry/internal/config"
	"github.com/codcod/jerry/internal/doc"
	"github.com/codcod/jerry/internal/match"
	"github.com/codcod/jerry/internal/rules"
	"github.com/codcod/jerry/internal/scaffold"
)

// goldenClock is the pinned "today" every golden case runs under (decision 4,
// JRY-007): validate, new and supersede all read the package clock, and an
// unpinned one would churn dates in the golden files across runs. The example
// ADR ships Accepted (never subject to the staleness rule), so any fixed date
// keeps the clean fixture clean.
var goldenClock = time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)

// cleanFixture builds a freshly scaffolded repository — the same production
// path internal/scaffold/scaffold_test.go's emit() uses — with no findings at
// all, matching TestScaffoldValidatesClean. git init gives hooks install/
// uninstall a real .git to act on.
func cleanFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, err := scaffold.Run(scaffold.Options{Root: root, Forge: scaffold.ForgeGitHub, Version: "test"}); err != nil {
		t.Fatalf("scaffolding clean fixture: %v", err)
	}
	gitInit(t, root)
	return root
}

// exampleADRPath is where the scaffold writes its one example ADR — the
// document dirtyFixture perturbs.
const exampleADRPath = "teams/example-team/adr/0001-example-use-postgres-for-primary-store.md"

// dirtyFixture builds the same scaffolded repository as cleanFixture, then
// introduces exactly one rules.SeverityError finding and exactly one
// rules.SeverityWarning finding by splicing two lines into the example ADR's
// frontmatter — not by hand-authoring a document. doc.Front has no field for
// an unrecognised key (that is what makes it unrecognised), so a byte-level
// insertion into the frontmatter block is the only way to produce checkDate's
// unknown-key case; the malformed applies_to entry is the same class of
// deliberate, minimal perturbation. TestDirtyFixtureContract pins the exact
// count so a future edit here cannot silently drift it.
func dirtyFixture(t *testing.T) string {
	t.Helper()
	root := cleanFixture(t)

	path := filepath.Join(root, filepath.FromSlash(exampleADRPath))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading example ADR: %v", err)
	}
	const marker = "\ndeciders: [example-author]\n---\n"
	replacement := "\ndeciders: [example-author]\n" +
		`applies_to: ["../escape"]` + "\n" +
		`jerry_test_unknown_field: true` + "\n" +
		"---\n"
	if !strings.Contains(string(raw), marker) {
		t.Fatalf("example ADR frontmatter did not match the expected shape:\n%s", raw)
	}
	perturbed := strings.Replace(string(raw), marker, replacement, 1)
	if err := os.WriteFile(path, []byte(perturbed), 0o644); err != nil {
		t.Fatalf("writing perturbed example ADR: %v", err)
	}
	return root
}

// relatedFixture builds the same scaffolded repository as cleanFixture, then
// gives the example ADR a valid directory-prefix applies_to entry (JRY-011
// decision 2) — a real, matchable value, unlike dirtyFixture's deliberately
// invalid one — so `jerry related` has something to resolve.
func relatedFixture(t *testing.T) string {
	t.Helper()
	root := cleanFixture(t)

	path := filepath.Join(root, filepath.FromSlash(exampleADRPath))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading example ADR: %v", err)
	}
	const marker = "\ndeciders: [example-author]\n---\n"
	replacement := "\ndeciders: [example-author]\n" +
		`applies_to: ["teams/example-team/"]` + "\n" +
		"---\n"
	if !strings.Contains(string(raw), marker) {
		t.Fatalf("example ADR frontmatter did not match the expected shape:\n%s", raw)
	}
	perturbed := strings.Replace(string(raw), marker, replacement, 1)
	if err := os.WriteFile(path, []byte(perturbed), 0o644); err != nil {
		t.Fatalf("writing perturbed example ADR: %v", err)
	}
	return root
}

func gitInit(t *testing.T, root string) {
	t.Helper()
	cmd := exec.Command("git", "init", "--quiet", root)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init %s: %v\n%s", root, err, out)
	}
}

// loadFixtureFindings runs the real rules.Check over a fixture root the same
// way `jerry validate` does, under the pinned clock.
func loadFixtureFindings(t *testing.T, root string) rules.Findings {
	t.Helper()
	cfg, err := config.Load(filepath.Join(root, config.DefaultFile))
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	corpus, err := doc.Load(cfg.Root, cfg.Layout())
	if err != nil {
		t.Fatalf("loading corpus: %v", err)
	}
	return rules.Check(corpus, cfg.RuleOptions(goldenClock))
}

// TestCleanFixtureContract pins the property TestGolden's clean cases rely
// on: a freshly scaffolded repo validates with zero findings.
func TestCleanFixtureContract(t *testing.T) {
	findings := loadFixtureFindings(t, cleanFixture(t))
	if len(findings) != 0 {
		t.Fatalf("clean fixture is not clean: %v", findings)
	}
}

// TestDirtyFixtureContract pins the property TestGolden's dirty cases rely
// on: exactly one error and one warning, so regenerating the fixture can
// never silently drop the case it exists to demonstrate.
func TestDirtyFixtureContract(t *testing.T) {
	findings := loadFixtureFindings(t, dirtyFixture(t))

	var errors, warnings int
	for _, finding := range findings {
		switch finding.Severity {
		case rules.SeverityError:
			errors++
		case rules.SeverityWarning:
			warnings++
		}
	}
	if errors != 1 || warnings != 1 {
		t.Fatalf("dirty fixture must carry exactly one error and one warning, got %d error(s) and %d warning(s): %v",
			errors, warnings, findings)
	}
}

// TestRelatedFixtureContract pins the property the "related-match" golden
// case relies on: a path under the fixture's applies_to prefix matches the
// example ADR, and a path outside it matches nothing.
func TestRelatedFixtureContract(t *testing.T) {
	root := relatedFixture(t)
	cfg, err := config.Load(filepath.Join(root, config.DefaultFile))
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	corpus, err := doc.Load(cfg.Root, cfg.Layout())
	if err != nil {
		t.Fatalf("loading corpus: %v", err)
	}

	if matches := match.Resolve(corpus, "teams/example-team/src/db.go"); len(matches) != 1 {
		t.Fatalf("expected exactly one match under the applies_to prefix, got %d: %v", len(matches), matches)
	}
	if matches := match.Resolve(corpus, "docs/readme.md"); len(matches) != 0 {
		t.Fatalf("expected no match outside the applies_to prefix, got %d: %v", len(matches), matches)
	}
}
