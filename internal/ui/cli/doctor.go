package cli

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/eajdias/win11-new/internal/domain/entity"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Audit and verify the health of your Windows 11 / MSYS2 / OpenCode environment",
		Long:  `Performs comprehensive diagnostic checks across packages, configs, git, env vars, skills, and LSPs.`,
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
		},
	}
}
