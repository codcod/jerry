package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/codcod/jerry/internal/scaffold"
)

func initCmd(g *globals) *cobra.Command {
	var (
		forge  string
		force  bool
		dryRun bool
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Scaffold an architecture-docs repository in the current directory",
		Long: "Writes a complete architecture-docs repository: templates, an example ADR,\n" +
			"CODEOWNERS, CI for the chosen forge, and jerry.yaml. Existing files are left\n" +
			"alone unless --force is given, so init is safe to re-run.\n\n" +
			"The emitted CI pins the version of jerry that wrote it, so the rules a repo is\n" +
			"checked against are the rules it was created with.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{kindKey: kindWrite},
		RunE: func(cmd *cobra.Command, _ []string) error {
			selected, err := scaffold.ParseForge(forge)
			if err != nil {
				return err
			}
			root, err := os.Getwd()
			if err != nil {
				return err
			}

			result, err := scaffold.Run(scaffold.Options{
				Root:    root,
				Forge:   selected,
				Version: cmd.Root().Version,
				Force:   force,
				DryRun:  dryRun,
			})
			if err != nil {
				return err
			}

			if !g.quiet {
				scaffold.Print(cmd.OutOrStdout(), result, dryRun)
			}
			if !dryRun && !g.quiet {
				cmd.Println()
				cmd.Println("Next: jerry hooks install, then edit CODEOWNERS for your teams.")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&forge, "forge", string(scaffold.ForgeGitHub), "CI and CODEOWNERS variant: gitlab or github")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite files that already exist")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "list what would be written without writing it")
	return cmd
}
