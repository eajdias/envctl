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
		if d.Category == entity.DiagWarning {
			statusStr = pterm.Yellow("⚠ WARN")
		} else if d.Category == entity.DiagError {
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
	_ = pterm.DefaultTable.WithHasHeader().WithBoxed().WithData(tableData).Render()

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

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Audit and verify the health of your Windows 11 / Ubuntu Linux / OpenCode environment",
		Long:  `Performs comprehensive diagnostic checks across packages, configs, git, env vars, skills, and LSPs. Use --fix to automatically remediate any issues.`,
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			pterm.DefaultHeader.WithFullWidth().Println("Running Environment Health Diagnostics (Doctor)")

			spinner, _ := pterm.DefaultSpinner.Start("Auditing environment components...")
			ctx := context.Background()

			report, err := appCtx.DoctorAuditUC.Execute(ctx)
			if err != nil {
				spinner.Fail(fmt.Sprintf("Audit execution failed: %v", err))
				return
			}
			spinner.Success("Audit completed")

			renderReport(report)

			if autoFix && (report.Errors > 0 || report.Warnings > 0) {
				pterm.Println()
				pterm.DefaultHeader.WithFullWidth().Println("Executing Auto-Remediation (--fix)")

				// 1. Windows Tweaks
				PrintSection("1/5 Applying Windows 11 System Tweaks")
				runWindowsProvisioning()

				// 2. Packages
				PrintSection("2/5 Remediating System Packages & Toolchains")
				runPackagesProvisioning("")

				// 3. Shell & Configs
				PrintSection("3/5 Remediating Shell, Environment & Config Files")
				runShellProvisioning()

				// 4. Skills
				PrintSection("4/5 Remediating OpenCode Agent Skills")
				runSkillsProvisioning()

				// 5. LSPs
				PrintSection("5/5 Remediating Language Server Protocols (LSP)")
				runLSPProvisioning()

				pterm.Println()
				pterm.DefaultHeader.WithFullWidth().Println("Post-Fix Verification")

				newReport, err := appCtx.DoctorAuditUC.Execute(ctx)
				if err == nil {
					renderReport(newReport)
				}
			}
		},
	}

	cmd.Flags().BoolVarP(&autoFix, "fix", "f", false, "Automatically apply fixes for any detected warnings or errors")
	return cmd
}
