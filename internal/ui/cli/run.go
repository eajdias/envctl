package cli

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/eajdias/win11-new/internal/domain/entity"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [subsystem]",
		Short: "Provision and configure the Windows 11 environment",
		Long:  `Executes idempotent provisioning tasks for system packages, MSYS2, shell, skills, and LSPs.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 || args[0] == "all" {
				runAllProvisioning()
			} else {
				_ = cmd.Help()
			}
		},
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "all",
		Short: "Provision everything (Packages, Shell, Configs, Skills, LSPs)",
		Run: func(cmd *cobra.Command, args []string) {
			runAllProvisioning()
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "winget",
		Short: "Provision Winget system packages and CLI tools",
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			runPackagesProvisioning(entity.PackageTypeWinget)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "pacman",
		Short: "Provision MSYS2 Pacman packages",
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			runPackagesProvisioning(entity.PackageTypePacman)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "volta",
		Short: "Provision Volta Node.js toolchains and global ecosystem (pnpm, firecrawl, playwright, etc.)",
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			runPackagesProvisioning(entity.PackageTypeVolta)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "shell",
		Short: "Provision environment variables, restricted directories, and shell configs (.bashrc, etc.)",
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			runShellProvisioning()
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "skills",
		Short: "Provision and deploy OpenCode agent skills",
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			runSkillsProvisioning()
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "lsp",
		Short: "Provision Language Server Protocol tools",
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			runLSPProvisioning()
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "windows",
		Short: "Provision Windows 11 registry tweaks (LongPaths, DevMode, Explorer, Themes) and Nerd Fonts",
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			runWindowsProvisioning()
		},
	})

	return cmd
}

func runAllProvisioning() {
	PrintBanner()
	pterm.DefaultHeader.WithFullWidth().Println("Starting Complete Environment Provisioning")

	// 1. Windows 11 Tweaks & Fonts
	PrintSection("1/5 Provisioning Windows 11 Registry Tweaks, Features & Fonts")
	runWindowsProvisioning()

	// 2. Packages (Winget + Pacman + Volta + Dotnet + Go + Rustup)
	PrintSection("2/5 Provisioning System Packages & Toolchains")
	runPackagesProvisioning("")

	// 3. Shell, Env & Configs
	PrintSection("3/5 Provisioning Shell, Environment Variables & Config Files")
	runShellProvisioning()

	// 4. Skills
	PrintSection("4/5 Provisioning OpenCode Agent Skills")
	runSkillsProvisioning()

	// 5. LSPs
	PrintSection("5/5 Provisioning Language Server Protocols (LSP)")
	runLSPProvisioning()

	pterm.Println()
	pterm.DefaultBox.WithTitle(pterm.LightGreen("🎉 Provisioning Completed Successfully")).Println(
		"All system components, toolchains, skills, and shell configurations have been applied.\n" +
			"Run 'win11-new doctor' at any time to verify system health.",
	)

	PrintSecretGuidance()
}

func runPackagesProvisioning(filterType entity.PackageType) {
	spinner, _ := pterm.DefaultSpinner.Start("Inspecting and installing packages...")
	ctx := context.Background()

	pkgs, err := appCtx.ProvisionPkgsUC.Execute(ctx, filterType, func(pkg entity.Package, status string, err error) {
		if err != nil {
			pterm.Warning.Printf("  • %s: %s (%v)\n", pkg, status, err)
		} else {
			pterm.Success.Printf("  • %s: %s\n", pkg, status)
		}
	})

	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed package provisioning: %v", err))
		return
	}

	spinner.Success(fmt.Sprintf("Processed %d packages", len(pkgs)))
}

func runShellProvisioning() {
	spinner, _ := pterm.DefaultSpinner.Start("Configuring shell, environment and configs...")
	ctx := context.Background()

	res, err := appCtx.ProvisionShellUC.Execute(ctx)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed shell provisioning: %v", err))
		return
	}

	for _, d := range res.EnvDiagnostics {
		if d.Category == entity.DiagOK {
			pterm.Success.Printf("  • [Env] %s: %s\n", d.Target, d.Details)
		} else {
			pterm.Warning.Printf("  • [Env] %s: %s\n", d.Target, d.Details)
		}
	}

	for _, d := range res.GitDiagnostics {
		pterm.Success.Printf("  • [Git] %s: %s\n", d.Target, d.Details)
	}

	for _, d := range res.ConfigDiagnostics {
		if d.Category == entity.DiagOK {
			pterm.Success.Printf("  • [%s] %s: %s\n", d.System, d.Target, d.Details)
		} else {
			pterm.Error.Printf("  • [%s] %s: %s\n", d.System, d.Target, d.Details)
		}
	}

	spinner.Success("Shell and configuration files aligned")
}

func runSkillsProvisioning() {
	spinner, _ := pterm.DefaultSpinner.Start("Deploying OpenCode Agent Skills...")
	ctx := context.Background()

	results, err := appCtx.ProvisionSkillsUC.Execute(ctx, "")
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed skills deployment: %v", err))
		return
	}

	deployedCount := 0
	for _, r := range results {
		if r.Status == entity.DiagOK {
			deployedCount++
		} else {
			pterm.Warning.Printf("  • Skill %s failed: %s\n", r.SkillName, r.ErrorMessage)
		}
	}

	spinner.Success(fmt.Sprintf("Deployed %d agent skills to ~/.config/opencode/skills/", deployedCount))
}

func runLSPProvisioning() {
	spinner, _ := pterm.DefaultSpinner.Start("Verifying and installing Language Servers...")
	ctx := context.Background()

	results, err := appCtx.ProvisionLSPUC.Execute(ctx)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed LSP provisioning: %v", err))
		return
	}

	for _, r := range results {
		if r.Status == entity.DiagOK {
			pterm.Success.Printf("  • LSP %s (%s): %s\n", r.LSP.Language, r.LSP.ServerName, r.Details)
		} else {
			pterm.Warning.Printf("  • LSP %s (%s): %s\n", r.LSP.Language, r.LSP.ServerName, r.ErrorMessage)
		}
	}

	spinner.Success(fmt.Sprintf("Checked %d language servers", len(results)))
}

func runWindowsProvisioning() {
	spinner, _ := pterm.DefaultSpinner.Start("Applying Windows 11 system tweaks, registry settings & fonts...")
	ctx := context.Background()

	results, err := appCtx.ProvisionWindowsUC.Execute(ctx, func(tweak entity.WindowsTweak, status, details string) {
		targetName := fmt.Sprintf("%s\\%s", tweak.Path, tweak.Name)
		if tweak.Path == "" {
			targetName = fmt.Sprintf("[%s] %s", tweak.Type, tweak.Name)
		}
		if status == "applied" {
			pterm.Success.Printf("  • %s: %s\n", targetName, details)
		} else if status == "skipped" {
			pterm.Success.Printf("  • %s: %s\n", targetName, details)
		} else if status == "failed" {
			pterm.Error.Printf("  • %s: %s\n", targetName, details)
		}
	})

	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed Windows tweaks provisioning: %v", err))
		return
	}

	spinner.Success(fmt.Sprintf("Processed %d Windows system tweaks and customizations", len(results)))
}
