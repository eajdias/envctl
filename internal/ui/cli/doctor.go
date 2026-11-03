package cli

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/usecase"
)

func renderReport(report *usecase.AuditReport) {
	tableData := pterm.TableData{
		{"Status", "System", "Component", "Details", "Fix Action"},
	}

	for _, d := range report.Diagnostics {
		statusStr := pterm.Green("✔ OK")
		switch d.Category {
		case entity.DiagWarning:
			statusStr = pterm.Yellow("⚠ WARN")
		case entity.DiagError:
			statusStr = pterm.Red("✘ FAIL")
		}

		fix := d.FixHint
		if fix == "" && d.Category == entity.DiagOK {
			fix = "-"
		}

		tableData = append(tableData, []string{
			statusStr,
			d.System,
			d.Target,
			d.Details,
			fix,
		})
	}

	pterm.Println()
	if err := pterm.DefaultTable.WithHasHeader().WithBoxed().WithData(tableData).Render(); err != nil {
		pterm.Error.Printf("failed to render report table: %v\n", err)
	}

	// Summary Box
	healthColor := pterm.FgLightGreen
	healthStatus := "EXCELLENT (100% Ready)"
	if report.Errors > 0 {
		healthColor = pterm.FgLightRed
		healthStatus = fmt.Sprintf("DEGRADED (%d Errors)", report.Errors)
	} else if report.Warnings > 0 {
		healthColor = pterm.FgLightYellow
		healthStatus = fmt.Sprintf("SUB-OPTIMAL (%d Warnings)", report.Warnings)
	}

	pterm.Println()
	pterm.DefaultBox.WithTitle(pterm.NewStyle(healthColor, pterm.Bold).Sprintf("Health Summary: %s", healthStatus)).Println(
		fmt.Sprintf("Total Checks: %d | Passed: %s | Warnings: %s | Errors: %s",
			report.TotalChecks,
			pterm.Green(fmt.Sprintf("%d", report.Passed)),
			pterm.Yellow(fmt.Sprintf("%d", report.Warnings)),
			pterm.Red(fmt.Sprintf("%d", report.Errors)),
		),
	)
}

func newDoctorCmd() *cobra.Command {
	var autoFix bool
	var showDiff bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Audit and verify the health of your Windows 11 / Ubuntu Linux / OpenCode + CommandCode environment",
		Long:  `Performs comprehensive diagnostic checks across packages, configs, git, env vars, and skills. Use --fix to automatically remediate any issues.`,
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			pterm.DefaultHeader.WithFullWidth().Println("Running Environment Health Diagnostics (Doctor)")

			spinner, _ := pterm.DefaultSpinner.Start("Auditing environment components...")
			ctx := context.Background()

			appCtx.DoctorAuditUC.SetShowDiff(showDiff)
			report, err := appCtx.DoctorAuditUC.Execute(ctx)
			if err != nil {
				spinner.Fail(fmt.Sprintf("Audit execution failed: %v", err))
				return
			}
			spinner.Success("Audit completed")

			renderReport(report)
			renderDiffs(report)

			if autoFix && (report.Errors > 0 || report.Warnings > 0) {
				pterm.Println()
				pterm.DefaultHeader.WithFullWidth().Println("Executing Auto-Remediation (--fix)")

				// 1. Windows Tweaks
				PrintSection("1/4 Applying Windows 11 System Tweaks")
				runWindowsProvisioning()

				// 2. Packages
				PrintSection("2/4 Remediating System Packages & Toolchains")
				runPackagesProvisioning("")

				// 3. Shell & Configs
				PrintSection("3/4 Remediating Shell, Environment & Config Files")
				runShellProvisioning()

				// 4. Skills
				PrintSection("4/4 Remediating Agent Skills (OpenCode + CommandCode)")
				runSkillsProvisioning()

				pterm.Println()
				pterm.DefaultHeader.WithFullWidth().Println("Post-Fix Verification")

				newReport, err := appCtx.DoctorAuditUC.Execute(ctx)
				if err == nil {
					renderReport(newReport)
					renderDiffs(newReport)
				}
			}
		},
	}

	cmd.Flags().BoolVarP(&autoFix, "fix", "f", false, "Automatically apply fixes for any detected warnings or errors")
	cmd.Flags().BoolVar(&showDiff, "diff", false, "Show template-vs-live previews for divergent overwrite config files (read-only)")
	return cmd
}

// renderDiffs prints template-vs-live previews collected by the audit when
// the --diff flag is on. Pure display: nothing here writes or fixes.
func renderDiffs(report *usecase.AuditReport) {
	if report == nil || len(report.Diffs) == 0 {
		return
	}
	pterm.Println()
	PrintSection("Config Diffs (provisioned vs on disk — read-only preview)")
	for _, d := range report.Diffs {
		pterm.Println()
		pterm.Info.Printf("%s\n", d.Target)
		pterm.Println(d.Unified)
	}
}
