package cli

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func newSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "snapshot",
		Aliases: []string{"export", "sync"},
		Short:   "Capture current environment state and synchronize manifests/configs",
		Long:    `Inspects active configs, skills, and Git settings, and updates the manifests and configs directories locally.`,
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			pterm.DefaultHeader.WithFullWidth().Println("Exporting Environment Snapshot")

			spinner, _ := pterm.DefaultSpinner.Start("Inspecting current system and updating manifests...")
			ctx := context.Background()

			result, err := appCtx.SnapshotSyncUC.Execute(ctx)
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
			pterm.Println()
			pterm.Info.Println("Snapshot completed locally. Review changes and commit manually.")
		},
	}

	return cmd
}
