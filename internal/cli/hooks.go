package cli

import (
	"github.com/spf13/cobra"

	"github.com/codcod/jerry/internal/config"
	"github.com/codcod/jerry/internal/hooks"
)

// hooksCmd manages the pre-commit hook.
//
// Hooks are UX, not enforcement: --no-verify exists, so CI stays the gate. What
// the hook buys is that a stale index or a malformed document is caught before
// the push rather than after it.
func hooksCmd(g *globals) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage the git pre-commit hook",
	}
	cmd.AddCommand(hooksInstallCmd(g), hooksUninstallCmd(g), hooksStatusCmd(g))
	return cmd
}

func hooksRoot(g *globals) (string, error) {
	cfg, err := config.Load(g.configPath)
	if err != nil {
		return "", err
	}
	return cfg.Root, nil
}

func hooksInstallCmd(g *globals) *cobra.Command {
	var (
		force  bool
		dryRun bool
	)
	cmd := &cobra.Command{
		Use:         "install",
		Short:       "Install the pre-commit hook (validate, reindex, re-stage)",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{kindKey: kindWrite},
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := hooksRoot(g)
			if err != nil {
				return err
			}
			if dryRun {
				path, err := hooks.Path(root)
				if err != nil {
					return err
				}
				cmd.Printf("(dry run) would install %s\n", path)
				return nil
			}
			path, err := hooks.Install(root, force)
			if err != nil {
				return err
			}
			if !g.quiet {
				cmd.Printf("Installed %s\n", path)
				cmd.Println("Hooks are per-clone — re-run this after a fresh clone.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "replace an existing hook jerry did not write")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report what would be installed without writing")
	return cmd
}

func hooksUninstallCmd(g *globals) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:         "uninstall",
		Short:       "Remove the pre-commit hook jerry installed",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{kindKey: kindWrite},
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := hooksRoot(g)
			if err != nil {
				return err
			}
			if dryRun {
				path, err := hooks.Path(root)
				if err != nil {
					return err
				}
				cmd.Printf("(dry run) would remove %s\n", path)
				return nil
			}
			path, err := hooks.Uninstall(root)
			if err != nil {
				return err
			}
			if path == "" {
				cmd.Println("No hook installed.")
				return nil
			}
			cmd.Printf("Removed %s\n", path)
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report what would be removed without removing it")
	return cmd
}

func hooksStatusCmd(g *globals) *cobra.Command {
	return &cobra.Command{
		Use:         "status",
		Short:       "Report whether the pre-commit hook is installed",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{kindKey: kindRead},
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := hooksRoot(g)
			if err != nil {
				return err
			}
			installed, owned, path, err := hooks.Status(root)
			switch {
			case err != nil:
				return err
			case !installed:
				cmd.Printf("not installed (%s)\n", path)
			case owned:
				cmd.Printf("installed by jerry (%s)\n", path)
			default:
				cmd.Printf("a hook exists but jerry did not write it (%s)\n", path)
			}
			return nil
		},
	}
}
