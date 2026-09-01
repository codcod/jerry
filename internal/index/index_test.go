package index

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/jerry/internal/doc"
)

var updateGolden = flag.Bool("update", false, "regenerate golden fixtures")

// goldenCorpus builds the fixture input in code rather than on disk: the
// fixture's job is to pin the rendering, and an inline corpus keeps what is
// being rendered visible next to what it renders to.
func goldenCorpus(t *testing.T) *doc.Corpus {
	t.Helper()
	root := t.TempDir()

	write := func(relPath, content string) {
		full := filepath.Join(root, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("teams/payments/adr/0001-use-outbox.md", `---
id: ADR-0001
title: "Use the outbox pattern: at-least-once publishing"
status: Superseded
superseded_by: [ADR-0002]
team: payments
date: 2026-01-05
deciders: [ada]
---

# body
`)
	write("teams/payments/adr/0002-use-cdc.md", `---
id: ADR-0002
title: Use change data capture
status: Accepted
team: payments
date: 2026-04-01
deciders: [ada]
---

# body
`)
	write("teams/payments/solution-designs/2026-05-ledger-rewrite.md", `---
title: Ledger rewrite
status: Implemented
team: payments
date: 2026-05-02
authors: [grace]
---

# body
`)
	write("cross-cutting/adr/0001-otel.md", `---
id: ADR-0001
title: Standardise on OpenTelemetry
status: Accepted
teams: [payments, platform]
date: 2026-02-10
deciders: [ada, linus]
---

# body
`)
	// A title containing a pipe would otherwise split its own table row.
	write("teams/platform/adr/0001-pipes.md", `---
id: ADR-0001
title: Support a | b routing
status: Proposed
team: platform
date: 2026-06-01
deciders: [linus]
---

# body
`)

	corpus, err := doc.Load(root, doc.DefaultLayout())
	if err != nil {
		t.Fatalf("loading corpus: %v", err)
	}
	return corpus
}

func TestRenderMarkdownGolden(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderMarkdown(&buf, Build(goldenCorpus(t), DefaultPath)); err != nil {
		t.Fatalf("RenderMarkdown: %v", err)
	}
	compare(t, "index.golden.md", buf.Bytes())
}

func TestRenderJSONGolden(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderJSON(&buf, Build(goldenCorpus(t), DefaultPath)); err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}
	compare(t, "index.golden.json", buf.Bytes())
}

func compare(t *testing.T, name string, got []byte) {
	t.Helper()
	golden := filepath.Join("testdata", name)
	if *updateGolden {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatalf("writing fixture: %v", err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("reading fixture (run: go test ./internal/index -update): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("committed fixture does not match rendered output.\nRun: go test ./internal/index -update\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// TestGoldenFixtureContract asserts the properties the fixtures exist to
// demonstrate, so regenerating them can never quietly drop one.
func TestGoldenFixtureContract(t *testing.T) {
	envelope := Build(goldenCorpus(t), DefaultPath)
	rendered := Markdown(envelope)

	t.Run("LinksClimbOutOfTheIndexDirectory", func(t *testing.T) {
		// The bug this pins: root-relative links in a file at index/index.md
		// resolve to index/teams/... and every link in the index is dead.
		for _, entry := range append(append([]Entry{}, envelope.ADRs...), envelope.SDs...) {
			if !strings.HasPrefix(entry.Link, "../") {
				t.Errorf("%s: link %q is not relative to index/", entry.Path, entry.Link)
			}
		}
	})

	t.Run("SupersededRowNamesItsSuccessor", func(t *testing.T) {
		if !strings.Contains(rendered, "Superseded by ADR-0002") {
			t.Error("a superseded ADR must answer 'superseded by what?' without a second lookup")
		}
	})

	t.Run("CrossCuttingRowNamesItsTeams", func(t *testing.T) {
		if !strings.Contains(rendered, "cross-cutting (payments, platform)") {
			t.Error("a cross-cutting row must name the teams it binds")
		}
	})

	t.Run("ColonInTitleSurvives", func(t *testing.T) {
		if !strings.Contains(rendered, "Use the outbox pattern: at-least-once publishing") {
			t.Error("a title containing a colon was truncated")
		}
	})

	t.Run("PipeInTitleIsEscaped", func(t *testing.T) {
		if !strings.Contains(rendered, `Support a \| b routing`) {
			t.Error("a title containing a pipe must not split its own table row")
		}
	})

	t.Run("SolutionDesignsHaveNoIDColumn", func(t *testing.T) {
		// They have no sequential id, so an ID column would be all dashes.
		section := rendered[strings.Index(rendered, "## Solution Designs"):]
		if strings.Contains(section, "| ID |") {
			t.Error("the solution design table still has an ID column")
		}
	})

	t.Run("EnvelopeIsVersioned", func(t *testing.T) {
		if envelope.Schema != EnvelopeSchema {
			t.Errorf("schema = %q, want %q", envelope.Schema, EnvelopeSchema)
		}
	})
}

func TestBuildIsDeterministic(t *testing.T) {
	first := Markdown(Build(goldenCorpus(t), DefaultPath))
	for range 5 {
		if again := Markdown(Build(goldenCorpus(t), DefaultPath)); again != first {
			t.Fatal("index rendering is not deterministic, so --check would flap")
		}
	}
}

func TestLinkFromHandlesRootLevelIndex(t *testing.T) {
	if got := linkFrom(".", "teams/a/adr/0001-x.md"); got != "teams/a/adr/0001-x.md" {
		t.Errorf("an index at the repo root needs no ../ prefix, got %q", got)
	}
	if got := linkFrom("docs/generated", "teams/a.md"); got != "../../teams/a.md" {
		t.Errorf("a nested index needs one ../ per level, got %q", got)
	}
}
