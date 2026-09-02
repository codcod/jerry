---
id: JRY-006
title: Placeholder rule: skip fenced blocks and add a per-document opt-out
project: jerry
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# JRY-006 — Placeholder rule: skip fenced blocks and add a per-document opt-out

## Outcome

An author whose decision legitimately contains a placeholder phrase — because it documents the
id convention, or quotes a template in a fenced block — can silence that one finding on that
one document, instead of deleting the phrase from `jerry.yaml` and switching the check off for
the whole repository.

## Description

`checkPlaceholders` (`internal/rules/rules.go`) is `strings.Contains` over `document.Raw`:
frontmatter, prose and fenced code blocks alike. DESIGN.md §5 calls this the most valuable check
in the catalogue — a copied, half-filled template is the single most common real defect — but
its false-positive path is unmanaged. A decision that discusses the `ADR-NNNN` form, or shows an
example frontmatter block, fails on a phrase it is legitimately talking about, and today the only
escape is `placeholders:` in `jerry.yaml`, which is repository-wide. The cheapest fix for a false
positive is therefore disabling the rule, which is the wrong incentive for the check that catches
the most. Recorded as divergence 6 in DESIGN.md §10.

Two changes, both now specified in DESIGN.md §5:

1. **Exclude fenced code blocks from the scan.** Quoting a template is the common legitimate
   case and never indicates an unfilled document.
2. **Add an inline opt-out**, `<!-- jerry:allow placeholder -->`, scoped to one document (or
   one phrase — settle which in refinement; per-phrase is more precise, per-document is
   simpler and probably enough).

Watch the interaction with the empty-section rule: HTML comments are stripped before that check
so that `jerry new`'s guidance prompts do not count as content (`stripComments`). An opt-out
marker is also an HTML comment, so it must not accidentally become the only content in a
section and it must not make an empty section look filled.

Also confirm the existing test that asserts every default placeholder appears in some shipped
template still holds — it is what stops the list rotting into decoration, and templates are in
`skip-dirs` so they are not themselves scanned.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-02 — created (TO DO). source: pickle ticket new
