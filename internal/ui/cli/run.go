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
	"github.com/eajdias/envctl/internal/usecase"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [subsystem]",
		Short: "Provision and configure the environment",
		Long:  `Executes idempotent provisioning tasks for system packages, shell, skills, and LSPs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "all" {
				if err := runAllProvisioning(); err != nil {
					pterm.Error.Printf("%v\n", err)
					os.Exit(1)
				}
				return nil
			}
			_ = cmd.Help()
			return fmt.Errorf("unknown subsystem '%s' (valid: all, windows, vps, cachyos, providers, winget, apt, pacman, paru, gaming, performance, debloat, bootstrap, volta, pip, shell, skills, lsp, cleanup)", args[0])
		},
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "all",
		Short: "Provision the full profile for this machine (dispatches to windows, vps or cachyos)",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runAllProvisioning(); err != nil {
				pterm.Error.Printf("%v\n", err)
				os.Exit(1)
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "windows",
		Short: "Full Windows 11 workstation profile (tweaks + debloat + packages + shell + skills + LSPs)",
		Run: func(cmd *cobra.Command, args []string) {
			runWindowsProfile()
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "vps",
		Short: "Ubuntu Server 24+ profile (providers + bootstrap + apt + performance + shell + skills + LSPs)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPSProfile()
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "cachyos",
		Short: "Full CachyOS desktop profile (providers + bootstrap + pacman/paru + gaming + performance + shell + skills + LSPs)",
		Run: func(cmd *cobra.Command, args []string) {
			runCachyOSProfile()
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
		Use:   "pacman",
		Short: "Provision Arch/CachyOS pacman packages",
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			runPackagesProvisioning(entity.PackageTypePacman)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "paru",
		Short: "Provision Arch AUR packages via paru",
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			runPackagesProvisioning(entity.PackageTypeParu)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "gaming",
		Short: "Provision opt-in gaming stack (Steam, emulators, MangoHud) on Arch/CachyOS",
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			runGamingProvisioning()
		},
	})

	performanceCmd := &cobra.Command{
		Use:   "performance",
		Short: "Apply the exact Ubuntu Server 24+ or CachyOS performance profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := cmd.Flags()
			opts := usecase.PerformanceOptions{
				DryRun:             mustBool(flags, "dry-run"),
				NoDaemonReexec:     mustBool(flags, "no-daemon-reexec"),
				Timezone:           mustString(flags, "timezone"),
				AllowDebloat:       mustBool(flags, "allow-debloat"),
				DebloatOnly:        mustBool(flags, "debloat-only"),
				ForceRebootPending: mustBool(flags, "force-reboot-pending"),
			}
			if err := opts.Validate(); err != nil {
				return err
			}
			PrintBanner()
			return runPerformanceProvisioning(cmd.Context(), opts)
		},
	}
	performanceCmd.Flags().Bool("dry-run", false, "Show the exact OS profile and changes without applying them")
	performanceCmd.Flags().Bool("no-daemon-reexec", false, "Write the limits drop-ins without re-executing PID 1 (the new defaults then apply on the next reboot)")
	performanceCmd.Flags().String("timezone", "", "Enforce an IANA timezone instead of only verifying the host's (for example America/Sao_Paulo)")
	performanceCmd.Flags().Bool("allow-debloat", false, "Also remove the packages listed in manifests/debloat_linux.yaml (opt-in: this destroys installed packages)")
	performanceCmd.Flags().Bool("debloat-only", false, "Run only the package removal, skipping every other step")
	performanceCmd.Flags().Bool("force-reboot-pending", false, "Proceed even though /var/run/reboot-required exists")
	cmd.AddCommand(performanceCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "debloat",
		Short: "Apply opt-in Windows 11 debloat (telemetry/privacy registry, gaming visuals, Appx removal, services, startup entries)",
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			runDebloatProvisioning()
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "providers",
		Short: "Phase 0 preflight: ensure Volta, Node and the OpenCode/CommandCode CLIs are present and current",
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			runProvidersProvisioning()
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "bootstrap",
		Short: "Provision the Linux toolchain (Volta, Node, OpenCode + CommandCode CLI, gh, delta, yq, uv, ruff, fd)",
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			runBootstrapProvisioning()
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "volta",
		Short: "Provision Volta Node.js toolchains and global ecosystem (pnpm, stylelint, etc.)",
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
		Use:   "tweaks",
		Short: "Provision Windows 11 registry tweaks only (LongPaths, DevMode, Explorer, Themes)",
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

func runAllProvisioning() error {
	PrintBanner()
	platform := entity.DetectedPlatform()
	switch {
	case platform.GOOS == "windows":
		runWindowsProfile()
		return nil
	case entity.PerformanceProfileMatchesOS(entity.PerformanceProfileCachyOS, platform):
		runCachyOSProfile()
		return nil
	default:
		return runVPSProfile()
	}
}

// requireSudoNOPASSWD warns once on Linux when passwordless sudo is missing,
// since package installs and performance tuning fail without it.
func requireSudoNOPASSWD() {
	if err := exec.Command("sudo", "-n", "true").Run(); err != nil {
		user := os.Getenv("USER")
		if user == "" {
			user = "$USER"
		}
		pterm.Warning.Println("sudo NOPASSWD is not configured. Package installs will fail.")
		pterm.Info.Printf("Fix: echo '%s ALL=(ALL) NOPASSWD:ALL' | sudo tee /etc/sudoers.d/%s-nopasswd\n", user, user)
		pterm.Println()
	}
}

func finishProfile(title string) {
	pterm.Println()
	pterm.DefaultBox.WithTitle(pterm.LightGreen("🎉 Provisioning Completed Successfully")).Println(
		title + "\nRun 'envctl doctor' at any time to verify system health.",
	)

	PrintSecretGuidance()
}

func runWindowsProfile() {
	PrintBanner()
	pterm.DefaultHeader.WithFullWidth().Println("Starting Windows 11 Workstation Provisioning")

	if runtime.GOOS != "windows" {
		pterm.Warning.Println("Windows profile requested on a non-Windows host; Windows-only steps will be skipped.")
		pterm.Println()
	}

	total := 7
	section := func(n int, text string) string {
		return fmt.Sprintf("%d/%d %s", n, total, text)
	}

	PrintSection(section(1, "Phase 0: Ensuring providers (Volta, Node, OpenCode & CommandCode CLIs)"))
	runProvidersProvisioning()

	PrintSection(section(2, "Provisioning Windows 11 Registry Tweaks, Features & Fonts"))
	runWindowsProvisioning()

	PrintSection(section(3, "Provisioning Windows 11 Debloat (telemetry/privacy/gaming/Appx/services/startup)"))
	runDebloatProvisioning()

	PrintSection(section(4, "Provisioning System Packages & Toolchains"))
	runPackagesProvisioning("")

	PrintSection(section(5, "Provisioning Shell, Environment Variables & Config Files"))
	runShellProvisioning()

	PrintSection(section(6, "Provisioning Agent Skills (OpenCode + CommandCode)"))
	runSkillsProvisioning()

	PrintSection(section(7, "Provisioning Language Server Protocols (LSP)"))
	runLSPProvisioning()

	finishProfile("All Windows workstation components, debloat, toolchains, skills, and shell configurations have been applied.")
}

func runVPSProfile() error {
	PrintBanner()
	pterm.DefaultHeader.WithFullWidth().Println("Starting Ubuntu/Debian Server (VPS) Provisioning")

	requireSudoNOPASSWD()

	total := 7
	section := func(n int, text string) string {
		return fmt.Sprintf("%d/%d %s", n, total, text)
	}

	PrintSection(section(1, "Phase 0: Ensuring providers (Volta, Node, OpenCode & CommandCode CLIs)"))
	runProvidersProvisioning()

	PrintSection(section(2, "Provisioning Linux Toolchain (Volta, Node, OpenCode CLI & CLI tools)"))
	runBootstrapProvisioning()

	PrintSection(section(3, "Provisioning System Packages & Toolchains"))
	runPackagesProvisioning("")

	// The profile is a hard requirement on this path, not an optional extra:
	// the owner declared Ubuntu Server 24+ as the only server target, so a host
	// outside the manifest's declared minimum must fail loudly instead of
	// silently skipping every performance change.
	PrintSection(section(4, "Applying Ubuntu Server Performance Profile (zram + sysctl)"))
	if err := runPerformanceProvisioning(context.Background(), usecase.PerformanceOptions{}); err != nil {
		return fmt.Errorf("the ubuntu-server performance profile is required by `run vps`: %w", err)
	}

	PrintSection(section(5, "Provisioning Shell, Environment Variables & Config Files"))
	runShellProvisioning()

	PrintSection(section(6, "Provisioning Agent Skills (OpenCode + CommandCode)"))
	runSkillsProvisioning()

	PrintSection(section(7, "Provisioning Language Server Protocols (LSP)"))
	runLSPProvisioning()

	finishProfile("All server components, performance tuning, toolchains, skills, and shell configurations have been applied.")
	return nil
}

func runCachyOSProfile() {
	PrintBanner()
	pterm.DefaultHeader.WithFullWidth().Println("Starting CachyOS Desktop Provisioning")

	requireSudoNOPASSWD()

	total := 8
	section := func(n int, text string) string {
		return fmt.Sprintf("%d/%d %s", n, total, text)
	}

	PrintSection(section(1, "Phase 0: Ensuring providers (Volta, Node, OpenCode & CommandCode CLIs)"))
	runProvidersProvisioning()

	PrintSection(section(2, "Provisioning Linux Toolchain (Volta, Node, OpenCode CLI & CLI tools)"))
	runBootstrapProvisioning()

	PrintSection(section(3, "Provisioning System Packages & Toolchains"))
	runPackagesProvisioning("")

	PrintSection(section(4, "Provisioning Gaming Stack (Steam, Proton, emulators, MangoHud)"))
	runGamingProvisioning()

	PrintSection(section(5, "Applying CachyOS Performance Profile (zram)"))
	if err := runPerformanceProvisioning(context.Background(), usecase.PerformanceOptions{}); err != nil {
		pterm.Warning.Printf("Performance profile skipped: %v\n", err)
	}

	PrintSection(section(6, "Provisioning Shell, Environment Variables & Config Files"))
	runShellProvisioning()

	PrintSection(section(7, "Provisioning Agent Skills (OpenCode + CommandCode)"))
	runSkillsProvisioning()

	PrintSection(section(8, "Provisioning Language Server Protocols (LSP)"))
	runLSPProvisioning()

	finishProfile("All CachyOS desktop components, gaming, performance tuning, toolchains, skills, and shell configurations have been applied.")
}

func runProvidersProvisioning() {
	spinner, _ := pterm.DefaultSpinner.Start("Ensuring providers (Volta, Node, OpenCode & CommandCode CLIs)...")
	ctx := context.Background()

	diags, err := appCtx.ProvisionProvidersUC.Execute(ctx)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed providers preflight: %v", err))
		return
	}

	failed := 0
	for _, d := range diags {
		if d.Category == entity.DiagOK {
			pterm.Success.Printf("  • [Providers] %s: %s\n", d.Target, d.Details)
			continue
		}
		failed++
		pterm.Warning.Printf("  • [Providers] %s: %s\n", d.Target, d.Details)
		if d.FixHint != "" {
			pterm.Info.Printf("      fix: %s\n", d.FixHint)
		}
	}

	if failed > 0 {
		spinner.Warning("Providers preflight finished with warnings")
		return
	}
	spinner.Success("Providers ready")
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

func runGamingProvisioning() {
	spinner, _ := pterm.DefaultSpinner.Start("Inspecting and installing gaming packages...")
	ctx := context.Background()

	pkgs, err := appCtx.ProvisionPkgsUC.ExecuteGaming(ctx, func(pkg entity.Package, status string, err error) {
		if err != nil {
			pterm.Warning.Printf("  • %s: %s (%v)\n", pkg, status, err)
		} else {
			pterm.Success.Printf("  • %s: %s\n", pkg, status)
		}
	})

	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed gaming provisioning: %v", err))
		return
	}

	spinner.Success(fmt.Sprintf("Processed %d gaming packages", len(pkgs)))
}

func runShellProvisioning(categories ...string) {
	spinner, _ := pterm.DefaultSpinner.Start("Configuring shell, environment and configs...")
	ctx := context.Background()

	res, err := appCtx.ProvisionShellUC.Execute(ctx, categories...)
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

// runSkillsForTarget deploys the manifest skills to one target directory and
// returns how many were deployed, how many stale ones were quarantined, and how
// many quarantined entries aged out of the recovery window.
func runSkillsForTarget(label, targetBaseDir string) (deployed int, quarantined int, expired int) {
	ctx := context.Background()

	results, prunedNames, expiredNames, err := appCtx.ProvisionSkillsUC.Execute(ctx, targetBaseDir)
	if err != nil {
		pterm.Error.Printf("  • [%s] skills deployment failed: %v\n", label, err)
		return 0, 0, 0
	}

	for _, r := range results {
		if r.Status == entity.DiagOK {
			deployed++
		} else {
			pterm.Warning.Printf("  • [%s] skill %s failed: %s\n", label, r.SkillName, r.ErrorMessage)
		}
	}
	for _, name := range prunedNames {
		pterm.Info.Printf("  • [%s] quarantined stale skill: %s\n", label, name)
	}
	for _, name := range expiredNames {
		pterm.Info.Printf("  • [%s] expired quarantined skill (past the recovery window): %s\n", label, name)
	}
	return deployed, len(prunedNames), len(expiredNames)
}

func runSkillsProvisioning() {
	pterm.Info.Println("Deploying agent skills to OpenCode & CommandCode...")

	deployedOC, prunedOC, expiredOC := runSkillsForTarget("OpenCode", "")
	deployedCC, prunedCC, expiredCC := runSkillsForTarget("CommandCode", "~/.commandcode/skills")

	pterm.Success.Printf("Deployed %d skills to OpenCode, %d to CommandCode (quarantined %d/%d stale, expired %d/%d)\n",
		deployedOC, deployedCC, prunedOC, prunedCC, expiredOC, expiredCC)
}

// runAgentProvisioning provisions a single agent end to end — its config files,
// agent directories, cleanup items and skill tree — leaving the other agent
// untouched. Machine-level layers (packages, toolchains, LSP binaries, shell/git
// config) stay with `envctl run all`.
func runAgentProvisioning(category, label, skillsTarget string) {
	PrintSection(fmt.Sprintf("Provisioning %s (configs, agents, MCP, skills)", label))
	runShellProvisioning(category)

	PrintSection(fmt.Sprintf("Deploying %s skills", label))
	deployed, pruned, expired := runSkillsForTarget(label, skillsTarget)

	pterm.Println()
	pterm.Success.Printf("%s provisioning complete (%d skills deployed, %d quarantined, %d expired). Run 'envctl doctor' to verify.", label, deployed, pruned, expired)
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

	if res.StoreNote != "" {
		pterm.Info.Printf("  • OpenCode session store: %s\n", res.StoreNote)
	}

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
	runTweakStack(
		"Applying Windows 11 system tweaks, registry settings & fonts...",
		"Failed Windows tweaks provisioning: %v",
		"Processed %d Windows system tweaks and customizations",
		appCtx.ProvisionWindowsUC,
	)
}

func runDebloatProvisioning() {
	runTweakStack(
		"Applying opt-in Windows 11 debloat (run as Administrator for Appx/HKLM/services/startup)...",
		"Failed debloat provisioning: %v",
		"Processed %d debloat tweaks (telemetry/privacy/gaming/apps/services/startup)",
		appCtx.ProvisionDebloatUC,
	)
}

func runTweakStack(spinnerMsg, failMsg, doneMsg string, uc *usecase.ProvisionTweaksUseCase) {
	spinner, _ := pterm.DefaultSpinner.Start(spinnerMsg)
	ctx := context.Background()

	results, err := uc.Execute(ctx, func(tweak entity.WindowsTweak, status, details string) {
		targetName := usecase.TweakDisplayName(tweak)
		switch status {
		case "applied", "skipped":
			pterm.Success.Printf("  • %s: %s\n", targetName, details)
		case "failed":
			pterm.Error.Printf("  • %s: %s\n", targetName, details)
		}
	})

	if err != nil {
		spinner.Fail(fmt.Sprintf(failMsg, err))
		return
	}

	spinner.Success(fmt.Sprintf(doneMsg, len(results)))
}
