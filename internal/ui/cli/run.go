package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [subsystem]",
		Short: "Provision and configure the environment",
		Long:  `Executes idempotent provisioning tasks for system packages, shell, skills, and LSPs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "all" {
				runAllProvisioning()
				return nil
			}
			_ = cmd.Help()
			return fmt.Errorf("unknown subsystem '%s' (valid: all, winget, apt, bootstrap, volta, pip, shell, skills, lsp, windows, cleanup)", args[0])
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
		Use:   "apt",
		Short: "Provision Debian/Ubuntu APT packages",
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			runPackagesProvisioning(entity.PackageTypeApt)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "bootstrap",
		Short: "Provision the Linux toolchain (Volta, Node, OpenCode + CommandCode CLI, gh, delta, yq, uv, ruff, oh-my-posh, fd)",
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			runBootstrapProvisioning()
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
		Use:   "pip",
		Short: "Provision global Python packages (pyyaml, requests, etc.)",
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			runPackagesProvisioning(entity.PackageTypePip)
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
		Short: "Provision and deploy agent skills (OpenCode + CommandCode)",
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

	cmd.AddCommand(&cobra.Command{
		Use:   "cleanup",
		Short: "Clean agent storage accumulation (legacy configs, duplicate cache, oversized tool-output, stale scratch)",
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			runCleanup()
		},
	})

	return cmd
}

func runAllProvisioning() {
	PrintBanner()
	pterm.DefaultHeader.WithFullWidth().Println("Starting Complete Environment Provisioning")

	isLinux := runtime.GOOS == "linux"

	// Pre-flight: verify sudo NOPASSWD on Linux so apt packages don't fail silently.
	if isLinux {
		if err := exec.Command("sudo", "-n", "true").Run(); err != nil {
			user := os.Getenv("USER")
			if user == "" {
				user = "$USER"
			}
			pterm.Warning.Println("sudo NOPASSWD is not configured. APT packages will fail to install.")
			pterm.Info.Printf("Fix: echo '%s ALL=(ALL) NOPASSWD:ALL' | sudo tee /etc/sudoers.d/%s-nopasswd\n", user, user)
			pterm.Println()
		}
	}

	total := 6
	section := func(n int, text string) string {
		return fmt.Sprintf("%d/%d %s", n, total, text)
	}

	// 1. Windows 11 Tweaks & Fonts (Windows only)
	if runtime.GOOS == "windows" {
		PrintSection(section(1, "Provisioning Windows 11 Registry Tweaks, Features & Fonts"))
		runWindowsProvisioning()
	} else {
		PrintSection(section(1, "Skipping Windows Tweaks (Linux/POSIX environment)"))
	}

	// 2. Linux Toolchain Bootstrap (Volta, Node, OpenCode, CLI tools) - Linux only.
	// Runs BEFORE packages: the packages manifest includes Volta-managed npm
	// packages (node, pnpm, firecrawl-cli, LSP servers) that require the Volta
	// toolchain — on a fresh VPS `run all` must bootstrap first, otherwise
	// every Volta package is skipped as "manager not available".
	if isLinux {
		PrintSection(section(2, "Provisioning Linux Toolchain (Volta, Node, OpenCode CLI & CLI tools)"))
		runBootstrapProvisioning()
	} else {
		PrintSection(section(2, "Skipping Linux Toolchain Bootstrap (Windows environment)"))
	}

	// 3. Packages (Winget / APT + Volta + Dotnet + Go + Rustup)
	PrintSection(section(3, "Provisioning System Packages & Toolchains"))
	runPackagesProvisioning("")

	// 4. Shell, Env & Configs
	PrintSection(section(4, "Provisioning Shell, Environment Variables & Config Files"))
	runShellProvisioning()

	// 5. Skills
	PrintSection(section(5, "Provisioning Agent Skills (OpenCode + CommandCode)"))
	runSkillsProvisioning()

	// 6. LSPs
	PrintSection(section(6, "Provisioning Language Server Protocols (LSP)"))
	runLSPProvisioning()

	pterm.Println()
	pterm.DefaultBox.WithTitle(pterm.LightGreen("🎉 Provisioning Completed Successfully")).Println(
		"All system components, toolchains, skills, and shell configurations have been applied.\n" +
			"Run 'envctl doctor' at any time to verify system health.",
	)

	PrintSecretGuidance()
}

func runBootstrapProvisioning() {
	spinner, _ := pterm.DefaultSpinner.Start("Bootstrapping Linux toolchain (Volta, Node, OpenCode CLI, tools)...")
	ctx := context.Background()

	res, err := appCtx.ProvisionBootstrapUC.Execute(ctx)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed Linux toolchain bootstrap: %v", err))
		return
	}

	for _, d := range res.Diagnostics {
		if d.Category == entity.DiagOK {
			pterm.Success.Printf("  • [Bootstrap] %s: %s\n", d.Target, d.Details)
		} else {
			pterm.Warning.Printf("  • [Bootstrap] %s: %s\n", d.Target, d.Details)
		}
	}

	spinner.Success("Linux toolchain bootstrap complete")
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
	spinner, _ := pterm.DefaultSpinner.Start("Deploying agent skills to OpenCode & CommandCode...")
	ctx := context.Background()

	// Deploy to OpenCode (default target)
	results, err := appCtx.ProvisionSkillsUC.Execute(ctx, "")
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed skills deployment to OpenCode: %v", err))
		return
	}

	deployedOC := 0
	for _, r := range results {
		if r.Status == entity.DiagOK {
			deployedOC++
		} else {
			pterm.Warning.Printf("  • [OpenCode] Skill %s failed: %s\n", r.SkillName, r.ErrorMessage)
		}
	}

	// Deploy to CommandCode
	resultsCC, err := appCtx.ProvisionSkillsUC.Execute(ctx, "~/.commandcode/skills")
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed skills deployment to CommandCode: %v", err))
		return
	}

	deployedCC := 0
	for _, r := range resultsCC {
		if r.Status == entity.DiagOK {
			deployedCC++
		} else {
			pterm.Warning.Printf("  • [CommandCode] Skill %s failed: %s\n", r.SkillName, r.ErrorMessage)
		}
	}

	spinner.Success(fmt.Sprintf("Deployed %d skills to OpenCode, %d to CommandCode", deployedOC, deployedCC))
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

func runCleanup() {
	spinner, _ := pterm.DefaultSpinner.Start("Cleaning agent storage accumulation (OpenCode + CommandCode)...")
	ctx := context.Background()

	res, err := appCtx.CleanupOpenCodeUC.Execute(ctx)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed cleanup: %v", err))
		return
	}

	removed := res.RemovedFiles
	freed := res.FreedBytes

	if appCtx.TempHygieneUC != nil {
		tempReport, tempErr := appCtx.TempHygieneUC.Cleanup(ctx)
		if tempErr == nil {
			if tempReport.Removed > 0 {
				removed = append(removed, fmt.Sprintf("temp hygiene: %d stale entries (%.1f MB)", tempReport.Removed, float64(tempReport.FreedBytes)/(1024*1024)))
			}
			freed += tempReport.FreedBytes
			removed = append(removed, tempReport.Skipped...)
			removed = append(removed, tempReport.Failed...)
		}
	}

	if appCtx.CleanupCommandCodeUC != nil {
		ccReport, ccErr := appCtx.CleanupCommandCodeUC.Execute(ctx)
		if ccErr == nil && len(ccReport.RemovedFiles) > 0 {
			removed = append(removed, ccReport.RemovedFiles...)
			freed += ccReport.FreedBytes
		}
	}

	if len(removed) == 0 {
		spinner.Success("Nothing to clean — agent storage is already tidy")
		return
	}

	spinner.Success(fmt.Sprintf("Removed %d items, freed %.1f MB", len(removed), float64(freed)/(1024*1024)))
	for _, f := range removed {
		pterm.Success.Printf("  • %s\n", f)
	}
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
