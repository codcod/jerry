package scaffold

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/codcod/jerry/internal/doc"
	"github.com/codcod/jerry/internal/index"
	"github.com/codcod/jerry/internal/rules"
)

func emit(t *testing.T, forge Forge) string {
	t.Helper()
	root := t.TempDir()
	if _, err := Run(Options{Root: root, Forge: forge, Version: "0.1.0"}); err != nil {
		t.Fatalf("scaffolding %s: %v", forge, err)
	}
	return root
}

// TestScaffoldValidatesClean is the test that matters most: a repository jerry
// wrote must pass jerry's own rules on the day it is written. Without it, init
// can ship a repo that fails its first pipeline — which it did, until the
// generated index was added.
func TestScaffoldValidatesClean(t *testing.T) {
	for _, forge := range []Forge{ForgeGitLab, ForgeGitHub} {
		t.Run(string(forge), func(t *testing.T) {
			root := emit(t, forge)

			corpus, err := doc.Load(root, doc.DefaultLayout())
			if err != nil {
				t.Fatalf("loading corpus: %v", err)
			}

			options := rules.DefaultOptions()
			// Pin the clock far enough ahead that the example ADR would trip
			// the stale-proposal warning if it were ever left Proposed.
			options.Today = time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
			for _, finding := range rules.Check(corpus, options) {
				t.Errorf("freshly scaffolded repo is not clean: %s", finding)
			}

			t.Run("IndexIsUpToDate", func(t *testing.T) {
				// init generates the index after scaffolding; this asserts the
				// generated file matches what `jerry index --check` recomputes.
				rendered := index.Markdown(index.Build(corpus, "index/index.md"))
				written, err := os.ReadFile(filepath.Join(root, "index", "index.md"))
				if err != nil {
					t.Fatalf("scaffold did not write an index: %v", err)
				}
				if string(written) != rendered {
					t.Errorf("scaffolded index does not match a fresh render")
				}
			})
		})
	}
}

// TestIndexLinksResolve pins the bug that shipped in the shell-script original:
// an index at index/index.md linking to "teams/..." resolves to
// "index/teams/..." and every link in it is dead.
func TestIndexLinksResolve(t *testing.T) {
	root := emit(t, ForgeGitHub)
	corpus, err := doc.Load(root, doc.DefaultLayout())
	if err != nil {
		t.Fatalf("loading corpus: %v", err)
	}

	envelope := index.Build(corpus, "index/index.md")
	if len(envelope.ADRs) == 0 {
		t.Fatal("scaffold no longer ships an example ADR, so links are untested")
	}
	for _, entry := range envelope.ADRs {
		if !strings.HasPrefix(entry.Link, "../") {
			t.Errorf("link %q is not relative to the index's own directory", entry.Link)
		}
		target := filepath.Join(root, "index", filepath.FromSlash(entry.Link))
		if _, err := os.Stat(target); err != nil {
			t.Errorf("link %q does not resolve from index/: %v", entry.Link, err)
		}
	}
}

// TestPlaceholdersMatchShippedTemplates keeps the placeholder rule honest: a
// phrase in the default list that no template contains catches nothing, and a
// template phrase absent from the list is a copied-and-unfilled document that
// validates clean.
func TestPlaceholdersMatchShippedTemplates(t *testing.T) {
	root := emit(t, ForgeGitHub)

	var templateText strings.Builder
	for _, name := range []string{"adr-template.md", "solution-design-template.md"} {
		content, err := os.ReadFile(filepath.Join(root, "templates", name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		templateText.Write(content)
	}

	for _, placeholder := range rules.DefaultOptions().Placeholders {
		if !strings.Contains(templateText.String(), placeholder) {
			t.Errorf("placeholder %q appears in no shipped template, so it can never fire", placeholder)
		}
	}
}

func TestVersionPinning(t *testing.T) {
	cases := []struct {
		name    string
		version string
		want    string
	}{
		{"ReleasedVersionIsPinned", "0.4.2", "@v0.4.2"},
		{"AlreadyPrefixedIsNotDoubled", "v0.4.2", "@v0.4.2"},
		{"DevBuildFallsBackToLatest", "dev", "@latest"},
		{"DirtyBuildFallsBackToLatest", "v0.4.2-3-gabc-dirty", "@latest"},
		{"NonDirtyIntermediateBuildFallsBackToLatest", "v0.1.1-3-g3f336b9", "@latest"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			root := t.TempDir()
			if _, err := Run(Options{Root: root, Forge: ForgeGitHub, Version: testCase.version}); err != nil {
				t.Fatalf("scaffolding: %v", err)
			}
			content, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "docs.yml"))
			if err != nil {
				t.Fatalf("reading workflow: %v", err)
			}
			if !strings.Contains(string(content), "github.com/codcod/jerry/cmd/jerry"+testCase.want) {
				t.Errorf("workflow does not pin %s", testCase.want)
			}
			if strings.Contains(string(content), versionToken) {
				t.Errorf("version token was left unreplaced")
			}
		})
	}
}

func TestNoScriptsDirectoryIsEmitted(t *testing.T) {
	// The whole point of the binary owning the rules is that consuming repos
	// carry no copy of them to drift.
	root := emit(t, ForgeGitLab)
	if _, err := os.Stat(filepath.Join(root, "scripts")); !os.IsNotExist(err) {
		t.Errorf("scaffold emitted a scripts/ directory")
	}
}

func TestRunIsIdempotent(t *testing.T) {
	root := t.TempDir()
	first, err := Run(Options{Root: root, Forge: ForgeGitHub, Version: "0.1.0"})
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	second, err := Run(Options{Root: root, Forge: ForgeGitHub, Version: "0.1.0"})
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if len(second.Created) != 0 {
		t.Errorf("re-running init created %v", second.Created)
	}
	// The index is regenerated on every run by design — it describes the
	// documents, not the scaffold — so it is reported separately from files
	// that are copied once.
	if len(second.Generated) != 1 {
		t.Errorf("re-run regenerated %v, want just the index", second.Generated)
	}
	sort.Strings(first.Created)
	sort.Strings(second.Skipped)
	if len(second.Skipped) != len(first.Created) {
		t.Errorf("re-run skipped %d files, first run created %d", len(second.Skipped), len(first.Created))
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	root := t.TempDir()
	result, err := Run(Options{Root: root, Forge: ForgeGitHub, Version: "0.1.0", DryRun: true})
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if len(result.Created) == 0 {
		t.Fatal("dry run reported nothing would be written")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("reading root: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("dry run wrote %d entries", len(entries))
	}
}

func TestParseForgeRejectsUnknown(t *testing.T) {
	if _, err := ParseForge("bitbucket"); err == nil {
		t.Error("an unknown forge must be rejected, not silently defaulted")
	}
}
