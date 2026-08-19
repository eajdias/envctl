package cli

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func newSnapshotCmd() *cobra.Command {
	var prFlag bool

	cmd := &cobra.Command{
		Use:     "snapshot",
		Aliases: []string{"export", "sync"},
		Short:   "Capture current environment state and synchronize manifests/configs",
		Long:    `Inspects active configs, skills, and Git settings, and updates the manifests and configs directories. Optionally creates a GitHub Pull Request.`,
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			pterm.DefaultHeader.WithFullWidth().Println("Exporting Environment Snapshot")

			spinner, _ := pterm.DefaultSpinner.Start("Inspecting current system and updating manifests...")
			ctx := context.Background()

			result, err := appCtx.SnapshotSyncUC.Execute(ctx, prFlag)
			if err != nil {
				spinner.Fail(fmt.Sprintf("Snapshot failed: %v", err))
				return
			}

			spinner.Success("Environment snapshot synchronized")

			pterm.Println()
			pterm.DefaultSection.Println("Synchronized Artifacts")
			for _, file := range result.UpdatedFiles {
				pterm.Success.Printf("  • Updated: %s\n", file)
			}
			pterm.Info.Printf("  • Skills cataloged: %d\n", result.DiscoveredSkills)

			if result.PRUrl != "" {
				pterm.Println()
				pterm.DefaultBox.WithTitle(pterm.LightGreen("🚀 GitHub Pull Request Created")).Println(
					fmt.Sprintf("Branch: %s\nPR URL: %s",
						pterm.Cyan(result.BranchName),
						pterm.LightMagenta(result.PRUrl),
					),
				)
			} else {
				pterm.Println()
				pterm.Info.Println("Tip: Pass '--pr' to automatically create a branch and open a GitHub Pull Request.")
			}
		},
	}

	cmd.Flags().BoolVarP(&prFlag, "pr", "p", false, "Create a Git branch and open a GitHub Pull Request via gh CLI")

	return cmd
}
