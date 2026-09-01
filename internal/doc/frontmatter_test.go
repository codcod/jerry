package doc

import (
	"strings"
	"testing"
)

// TestParseHandlesRealWorldFrontmatter covers the shapes a hand-rolled line
// splitter gets wrong. Each case is a document someone actually writes.
func TestParseHandlesRealWorldFrontmatter(t *testing.T) {
	cases := []struct {
		name  string
		input string
		check func(*testing.T, Front)
	}{
		{
			name:  "ColonInTitleIsNotTruncated",
			input: "---\ntitle: \"Use Kafka: rationale for the ledger\"\n---\n",
			check: func(t *testing.T, front Front) {
				if front.Title != "Use Kafka: rationale for the ledger" {
					t.Errorf("title = %q", front.Title)
				}
			},
		},
		{
			name:  "UnquotedColonInTitleIsNotTruncated",
			input: "---\ntitle: 'Outbox: at-least-once'\n---\n",
			check: func(t *testing.T, front Front) {
				if front.Title != "Outbox: at-least-once" {
					t.Errorf("title = %q", front.Title)
				}
			},
		},
		{
			name:  "BlockSequence",
			input: "---\nteams:\n  - payments\n  - platform\n---\n",
			check: func(t *testing.T, front Front) {
				if len(front.Teams) != 2 || front.Teams[0] != "payments" {
					t.Errorf("teams = %v", front.Teams)
				}
			},
		},
		{
			name:  "FlowSequence",
			input: "---\ndeciders: [ada, grace]\n---\n",
			check: func(t *testing.T, front Front) {
				if len(front.Deciders) != 2 {
					t.Errorf("deciders = %v", front.Deciders)
				}
			},
		},
		{
			name:  "ScalarWhereAListIsAllowed",
			input: "---\ndeciders: ada\n---\n",
			check: func(t *testing.T, front Front) {
				if len(front.Deciders) != 1 || front.Deciders[0] != "ada" {
					t.Errorf("deciders = %v", front.Deciders)
				}
			},
		},
		{
			name:  "EmptyValueIsNotAOneElementList",
			input: "---\ndeciders:\n---\n",
			check: func(t *testing.T, front Front) {
				if len(front.Deciders) != 0 {
					t.Errorf("deciders = %v, want empty", front.Deciders)
				}
			},
		},
		{
			name:  "CommentsAndBlankLines",
			input: "---\n# a comment\n\nstatus: Accepted   # trailing comment\n---\n",
			check: func(t *testing.T, front Front) {
				if front.Status != "Accepted" {
					t.Errorf("status = %q", front.Status)
				}
			},
		},
		{
			name:  "UnquotedDateStaysAString",
			input: "---\ndate: 2026-01-15\n---\n",
			check: func(t *testing.T, front Front) {
				if front.Date != "2026-01-15" {
					t.Errorf("date = %q", front.Date)
				}
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			front, _, _, err := Parse(testCase.input)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			testCase.check(t, front)
		})
	}
}

func TestParseRejects(t *testing.T) {
	cases := map[string]string{
		"NoFrontmatter": "# just a heading\n",
		"Unterminated":  "---\ntitle: x\n",
		"NotAMapping":   "---\n- a\n- b\n---\n",
		"MalformedYAML": "---\ntitle: [unclosed\n---\n",
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, _, _, err := Parse(input); err == nil {
				t.Error("expected an error, got none")
			}
		})
	}
}

// TestFieldLineIsWholeFileCoordinates pins the off-by-one that makes every
// reported line useless: yaml numbers the block from 1, but the block starts on
// line 2 of the file.
func TestFieldLineIsWholeFileCoordinates(t *testing.T) {
	input := "---\nid: ADR-0001\ntitle: x\nstatus: Accepted\n---\n\nbody\n"
	_, mapping, _, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for key, want := range map[string]int{"id": 2, "title": 3, "status": 4} {
		if got := FieldLine(mapping, key); got != want {
			t.Errorf("FieldLine(%q) = %d, want %d", key, got, want)
		}
	}
	if got := FieldLine(mapping, "absent"); got != 1 {
		t.Errorf("an absent key must still be addressable, got line %d", got)
	}
}

// TestCanonicalPreservesUnknownKeys is what makes `jerry fmt` safe to run on a
// repo that carries organisation-specific fields jerry knows nothing about.
func TestCanonicalPreservesUnknownKeys(t *testing.T) {
	input := "---\ncost-centre: CC-42\nstatus: Accepted\nid: ADR-0001\njira: PROJ-7\n---\n"
	_, mapping, _, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	canonical, err := Canonical(mapping)
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}

	if !strings.Contains(canonical, "cost-centre: CC-42") || !strings.Contains(canonical, "jira: PROJ-7") {
		t.Fatalf("unknown keys were dropped:\n%s", canonical)
	}
	idAt := strings.Index(canonical, "id:")
	statusAt := strings.Index(canonical, "status:")
	costAt := strings.Index(canonical, "cost-centre:")
	jiraAt := strings.Index(canonical, "jira:")
	if !(idAt < statusAt && statusAt < costAt) {
		t.Errorf("known keys are not in canonical order:\n%s", canonical)
	}
	if costAt > jiraAt {
		t.Errorf("unknown keys lost their authored relative order:\n%s", canonical)
	}
}

func TestCanonicalIsIdempotent(t *testing.T) {
	input := "---\nstatus: Accepted\nid: ADR-0001\ntitle: x\n---\n\nbody\n"
	_, mapping, body, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	first, err := Canonical(mapping)
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}
	rendered := Render(first, body)

	_, mapping2, body2, err := Parse(rendered)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	second, err := Canonical(mapping2)
	if err != nil {
		t.Fatalf("Canonical (second): %v", err)
	}
	if again := Render(second, body2); again != rendered {
		t.Errorf("formatting is not a fixed point:\n--- first ---\n%s\n--- second ---\n%s", rendered, again)
	}
}

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"Use Kafka: rationale for the ledger": "use-kafka-rationale-for-the-ledger",
		"  Leading and trailing  ":            "leading-and-trailing",
		"C++ / Rust interop (v2)":             "c-rust-interop-v2",
		"ADR 0007 — supersede":                "adr-0007-supersede",
	}
	for input, want := range cases {
		if got := Slug(input); got != want {
			t.Errorf("Slug(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestStatusTransitions(t *testing.T) {
	t.Run("SupersededIsUnreachableByTransition", func(t *testing.T) {
		// It needs a successor to point at, so only `jerry supersede` may set
		// it. Reachability here would allow a Superseded ADR with no successor.
		for _, from := range ADRStatuses {
			if CanTransition(KindADR, from, StatusSuperseded) {
				t.Errorf("%s -> Superseded must not be a plain transition", from)
			}
		}
	})
	t.Run("ProposedResolvesEitherWay", func(t *testing.T) {
		for _, to := range []string{StatusAccepted, StatusRejected} {
			if !CanTransition(KindADR, StatusProposed, to) {
				t.Errorf("Proposed -> %s must be legal", to)
			}
		}
	})
	t.Run("RejectedIsTerminal", func(t *testing.T) {
		if len(NextStatuses(KindADR, StatusRejected)) != 0 {
			t.Error("Rejected must be terminal — reopening is a new ADR")
		}
	})
	t.Run("CanonicalStatusIsCaseInsensitive", func(t *testing.T) {
		if got, ok := CanonicalStatus(KindSD, "in review"); !ok || got != StatusInReview {
			t.Errorf("CanonicalStatus(sd, %q) = %q, %v", "in review", got, ok)
		}
	})
}

func TestParseRef(t *testing.T) {
	t.Run("BareRefUsesDefaultScope", func(t *testing.T) {
		ref, err := ParseRef("ADR-0007", "payments")
		if err != nil || ref.Scope != "payments" || ref.ID != "0007" {
			t.Fatalf("ParseRef = %+v, %v", ref, err)
		}
		if ref.Short("payments") != "ADR-0007" {
			t.Errorf("Short within its own scope must stay bare, got %q", ref.Short("payments"))
		}
		if ref.Short("platform") != "payments/ADR-0007" {
			t.Errorf("Short from another scope must qualify, got %q", ref.Short("platform"))
		}
	})
	t.Run("Rejects", func(t *testing.T) {
		for _, input := range []string{"ADR-7", "adr-0007", "payments/ADR-7", "0007", "Payments/ADR-0007"} {
			if _, err := ParseRef(input, "x"); err == nil {
				t.Errorf("ParseRef(%q) should have failed", input)
			}
		}
	})
}
