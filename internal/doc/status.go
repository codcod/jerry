package doc

import "strings"

// ADR statuses. Rejected is first-class on purpose: a decision that was
// considered and turned down is the record that stops the same proposal
// coming back every year (DESIGN.md §4).
const (
	StatusProposed   = "Proposed"
	StatusAccepted   = "Accepted"
	StatusRejected   = "Rejected"
	StatusDeprecated = "Deprecated"
	StatusSuperseded = "Superseded"
)

// Solution Design statuses. Unlike ADRs these are not an append-only log —
// a design perishes, so the lifecycle has somewhere for it to end.
const (
	StatusDraft       = "Draft"
	StatusInReview    = "In Review"
	StatusApproved    = "Approved"
	StatusImplemented = "Implemented"
	StatusArchived    = "Archived"
)

// ADRStatuses and SDStatuses are ordered for display in error messages.
var (
	ADRStatuses = []string{StatusProposed, StatusAccepted, StatusRejected, StatusDeprecated, StatusSuperseded}
	SDStatuses  = []string{StatusDraft, StatusInReview, StatusApproved, StatusImplemented, StatusSuperseded, StatusArchived}
)

// Statuses returns the legal statuses for a kind.
func Statuses(kind Kind) []string {
	if kind == KindSD {
		return SDStatuses
	}
	return ADRStatuses
}

// ValidStatus reports whether a status is legal for a kind.
func ValidStatus(kind Kind, status string) bool {
	for _, candidate := range Statuses(kind) {
		if candidate == status {
			return true
		}
	}
	return false
}

// CanonicalStatus resolves a case-insensitive user-typed status to its
// canonical spelling, so `jerry status X accepted` works without the shift key.
func CanonicalStatus(kind Kind, input string) (string, bool) {
	normalised := strings.Join(strings.Fields(strings.ToLower(input)), " ")
	for _, candidate := range Statuses(kind) {
		if strings.ToLower(candidate) == normalised {
			return candidate, true
		}
	}
	return "", false
}

// transitions is the legal-move table for `jerry status`. Superseded is
// deliberately unreachable here: it requires a successor to point at, so it is
// only reachable through `jerry supersede`, which writes both sides of the
// link. That is why the command refuses it rather than accepting a Superseded
// document with no superseded_by.
var transitions = map[Kind]map[string][]string{
	KindADR: {
		StatusProposed:   {StatusAccepted, StatusRejected},
		StatusAccepted:   {StatusDeprecated},
		StatusRejected:   {},
		StatusDeprecated: {},
		StatusSuperseded: {},
	},
	KindSD: {
		StatusDraft:       {StatusInReview, StatusArchived},
		StatusInReview:    {StatusApproved, StatusDraft, StatusArchived},
		StatusApproved:    {StatusImplemented, StatusArchived},
		StatusImplemented: {StatusArchived},
		StatusArchived:    {},
		StatusSuperseded:  {},
	},
}

// NextStatuses lists the statuses reachable from the current one.
func NextStatuses(kind Kind, from string) []string {
	return transitions[kind][from]
}

// CanTransition reports whether from -> to is a legal move.
func CanTransition(kind Kind, from, to string) bool {
	for _, candidate := range transitions[kind][from] {
		if candidate == to {
			return true
		}
	}
	return false
}
