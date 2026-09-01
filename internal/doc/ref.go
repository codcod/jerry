package doc

import (
	"fmt"
	"regexp"
)

// refPattern matches an ADR reference, optionally scope-qualified.
//
// IDs are only unique within one folder, which is the price of per-folder
// numbering. A bare ADR-0007 therefore means "in this document's own folder";
// anything else must name the scope: payments/ADR-0007, cross-cutting/ADR-0003.
var refPattern = regexp.MustCompile(`^(?:([a-z0-9][a-z0-9-]*)/)?ADR-(\d{4})$`)

// Ref is a parsed ADR reference.
type Ref struct {
	Scope string // empty means "the referring document's own scope"
	ID    string // four digits
}

// ParseRef parses a reference. defaultScope fills in an unqualified one.
func ParseRef(input, defaultScope string) (Ref, error) {
	match := refPattern.FindStringSubmatch(input)
	if match == nil {
		return Ref{}, fmt.Errorf("%q is not a valid ADR reference (expected 'ADR-0007' or 'payments/ADR-0007')", input)
	}
	scope := match[1]
	if scope == "" {
		scope = defaultScope
	}
	return Ref{Scope: scope, ID: match[2]}, nil
}

// String renders a reference in its scope-qualified form.
func (r Ref) String() string { return r.Scope + "/ADR-" + r.ID }

// Short renders a reference as an author would write it from within scope.
func (r Ref) Short(from string) string {
	if r.Scope == from {
		return "ADR-" + r.ID
	}
	return r.String()
}
