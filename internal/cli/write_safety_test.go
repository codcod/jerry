package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// leaves walks the command tree from root and returns every command with no
// subcommands — the ones the write-safety rules actually apply to. A new
// command with children is a parent and is exempt by definition; a new leaf is
// exactly the case this file exists to catch.
func leaves(cmd *cobra.Command) []*cobra.Command {
	if len(cmd.Commands()) == 0 {
		return []*cobra.Command{cmd}
	}
	var out []*cobra.Command
	for _, child := range cmd.Commands() {
		out = append(out, leaves(child)...)
	}
	return out
}

// kindOf returns the leaf's jerry/kind annotation, or "" when absent — never
// defaulted, so an unclassified command fails the check below instead of
// silently passing as a read.
func kindOf(cmd *cobra.Command) string { return cmd.Annotations[kindKey] }

// generated names cobra's own built-ins, which jerry does not author and
// therefore does not annotate.
var generated = map[string]bool{"help": true, "completion": true}

func TestWriteSafety(t *testing.T) {
	root, _ := newRoot("test")
	var cmds []*cobra.Command
	for _, cmd := range leaves(root) {
		if !generated[cmd.Name()] && !generated[cmd.Parent().Name()] {
			cmds = append(cmds, cmd)
		}
	}
	if len(cmds) == 0 {
		t.Fatal("no leaf commands found — the walker is broken, not the tree")
	}

	t.Run("EveryLeafIsClassified", func(t *testing.T) {
		for _, cmd := range cmds {
			switch kindOf(cmd) {
			case kindRead, kindWrite, kindOther:
			default:
				t.Errorf("%s: missing or unknown jerry/kind annotation (got %q); every leaf must declare read, write, or other",
					cmd.CommandPath(), kindOf(cmd))
			}
		}
	})

	t.Run("EveryWriteHasDryRun", func(t *testing.T) {
		for _, cmd := range cmds {
			if kindOf(cmd) != kindWrite {
				continue
			}
			// `index` and `fmt` spell it --check as well, but --dry-run must
			// always exist so the rule is one rule and not a list of exceptions.
			if cmd.Flags().Lookup("dry-run") == nil {
				t.Errorf("%s: jerry/kind=write but has no --dry-run flag", cmd.CommandPath())
			}
		}
	})

	t.Run("DestructiveFlagsNameARegisteredConfirm", func(t *testing.T) {
		for _, cmd := range cmds {
			cmd.Flags().VisitAll(func(f *pflag.Flag) {
				confirms, ok := f.Annotations["jerry/destructive"]
				if !ok {
					return
				}
				if len(confirms) != 1 {
					t.Errorf("%s --%s: jerry/destructive must name exactly one confirming flag, got %v",
						cmd.CommandPath(), f.Name, confirms)
					return
				}
				if cmd.Flags().Lookup(confirms[0]) == nil {
					t.Errorf("%s --%s: names confirming flag --%s, which is not registered on this command",
						cmd.CommandPath(), f.Name, confirms[0])
				}
			})
		}
	})

	t.Run("EveryLeafHasAShortDescription", func(t *testing.T) {
		for _, cmd := range cmds {
			if cmd.Short == "" {
				t.Errorf("%s: no Short description, so it is invisible in --help", cmd.CommandPath())
			}
		}
	})
}

// TestWriteSafety_UnclassifiedLeafIsCaught proves the classifier actually fires
// on a command that was never registered, rather than only on the tree as it
// stands today.
func TestWriteSafety_UnclassifiedLeafIsCaught(t *testing.T) {
	forgotten := &cobra.Command{Use: "forgotten", RunE: func(*cobra.Command, []string) error { return nil }}
	root := &cobra.Command{Use: "root"}
	root.AddCommand(forgotten)

	found := leaves(root)
	if len(found) != 1 || found[0] != forgotten {
		t.Fatalf("leaves() = %v, want exactly [forgotten]", found)
	}
	switch kindOf(forgotten) {
	case kindRead, kindWrite, kindOther:
		t.Fatalf("an unannotated command must not resolve to a legal kind, got %q", kindOf(forgotten))
	}
}

// TestRootBuildsIsolatedTrees pins the reason there is no package-level root
// command: two trees in one process must not share flag state.
func TestRootBuildsIsolatedTrees(t *testing.T) {
	first, firstGlobals := newRoot("test")
	second, secondGlobals := newRoot("test")
	if first == second || firstGlobals == secondGlobals {
		t.Fatal("newRoot returned shared state")
	}

	if err := first.PersistentFlags().Set("quiet", "true"); err != nil {
		t.Fatalf("setting flag: %v", err)
	}
	if secondGlobals.quiet {
		t.Error("setting a flag on one tree changed another tree's state")
	}
}
