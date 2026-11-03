package cli

import (
	"context"
	"os"

	"github.com/eajdias/envctl/internal/usecase"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func newSelfUpdateCmd() *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "self-update",
		Short: "Install the latest verified envctl release (checksum + backup, no sudo)",
		Long: `Downloads the newest release binary, verifies it against SHA256SUMS.txt and installs it over the running binary, keeping a timestamped .bak next to it for instant rollback.

Without flags a newer release is required: an up-to-date binary reports "already at latest" and writes nothing. A downgrade is never automatic — restore the .bak or fetch bootstrap.sh for a pinned version.`,
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			ctx := context.Background()

			latest, err := appCtx.SelfUpdateUC.Latest(ctx)
			if err != nil {
				pterm.Error.Printf("could not resolve the latest release: %v\n", err)
				return
			}
			if !usecase.VersionLess(appVersion, latest) {
				pterm.Success.Printf("already at latest: envctl %s (release %s)\n", appVersion, latest)
				return
			}
			if check {
				pterm.Info.Printf("update available: %s -> %s\n", appVersion, latest)
				pterm.Println("asset:    " + appCtx.SelfUpdateUC.AssetName())
				pterm.Println("backup:   <install-dir>\\envctl.bak.YYYYMMDD-HHMMSS (created on apply)")
				pterm.Println("run without --check to download, verify and install.")
				return
			}
			exe, err := os.Executable()
			if err != nil {
				pterm.Error.Printf("could not resolve the running binary: %v\n", err)
				return
			}
			res, err := appCtx.SelfUpdateUC.Apply(ctx, exe)
			if err != nil {
				pterm.Error.Printf("self-update failed: %v\n", err)
				return
			}
			pterm.Success.Printf("envctl updated to %s (backup kept at %s)\n", res.To, res.Backup)
			pterm.Info.Println("Restart the shell (or open a new one) so PATH resolves the new binary.")
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "Print current -> latest, the asset and the backup path without writing anything")
	return cmd
}
