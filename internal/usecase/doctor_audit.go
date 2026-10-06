package usecase

import (
	"context"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
	"github.com/eajdias/envctl/internal/infra/embedded"
	"github.com/eajdias/envctl/internal/infra/environment"
	"github.com/eajdias/envctl/internal/infra/filesystem"
	"github.com/eajdias/envctl/internal/infra/git"
	"github.com/eajdias/envctl/internal/infra/performance"
	"github.com/eajdias/envctl/internal/infra/windows"
)

type DoctorAuditUseCase struct {
	manifestRepo         *embedded.ManifestRepository
	fsManager            *filesystem.FileSystemManager
	envManager           *environment.WindowsEnvManager
	gitManager           *git.GitManager
	tweaksManager        *windows.TweaksManager
	managers             map[entity.PackageType]repository.PackageManager
	performanceInspector *performance.PerformanceInspector
	// platform is a seam so the performance audit resolves the same profile a
	// run would without reading the host's /etc/os-release. Tests set it;
	// production leaves it nil and gets entity.DetectedPlatform.
	platform func() entity.PlatformInfo
	logger   repository.Logger
	// envctlVersion is the version baked into the running binary (main.Version,
	// injected via -ldflags). Empty or "dev" means an untagged build, which the
	// freshness audit flags when a repo checkout is available to compare.
	envctlVersion string
	// gitDescribe runs `git describe --tags --always` in the repo checkout and
	// returns its trimmed output. A function so the freshness audit is testable
	// without a real checkout; production wires the git-backed default.
	gitDescribe func(ctx context.Context, repoDir string) (string, error)
}

func NewDoctorAuditUseCase(
	manifestRepo *embedded.ManifestRepository,
	fsManager *filesystem.FileSystemManager,
	envManager *environment.WindowsEnvManager,
	gitManager *git.GitManager,
	tweaksManager *windows.TweaksManager,
	managers map[entity.PackageType]repository.PackageManager,
	logger repository.Logger,
	performanceInspectors ...*performance.PerformanceInspector,
) *DoctorAuditUseCase {
	var performanceInspector *performance.PerformanceInspector
	if len(performanceInspectors) > 0 {
		performanceInspector = performanceInspectors[0]
	}
	return &DoctorAuditUseCase{
		manifestRepo:         manifestRepo,
		fsManager:            fsManager,
		envManager:           envManager,
		gitManager:           gitManager,
		tweaksManager:        tweaksManager,
		managers:             managers,
		performanceInspector: performanceInspector,
		logger:               logger,
		gitDescribe:          defaultGitDescribe,
	}
}

// SetEnvctlVersion records the version baked into the running binary, so the
// freshness audit can compare it against the repo checkout. Production wires it
// from main.Version via InitApp; tests call it directly.
func (uc *DoctorAuditUseCase) SetEnvctlVersion(version string) {
	uc.envctlVersion = version
}

// defaultGitDescribe reports the repo's describe output from repoDir, or an
// error when the directory is not a git checkout (or git is missing).
func defaultGitDescribe(ctx context.Context, repoDir string) (string, error) {
	out, err := runWithToolchain(ctx, "git", "-C", repoDir, "describe", "--tags", "--always")
	return out, err
}

type AuditReport struct {
	Diagnostics []entity.Diagnostic
	TotalChecks int
	Passed      int
	Warnings    int
	Errors      int
}

func (uc *DoctorAuditUseCase) Execute(ctx context.Context) (*AuditReport, error) {
	uc.logger.Info("Starting system audit and diagnostic verification")

	report := &AuditReport{}

	addDiag := func(diag entity.Diagnostic) {
		report.Diagnostics = append(report.Diagnostics, diag)
		report.TotalChecks++
		switch diag.Category {
		case entity.DiagOK:
			report.Passed++
			uc.logger.Info("[AUDIT-PASS] [%s] %s: %s", diag.System, diag.Target, diag.Details)
		case entity.DiagWarning:
			report.Warnings++
			uc.logger.Warn("[AUDIT-WARN] [%s] %s: %s (Fix: %s)", diag.System, diag.Target, diag.Details, diag.FixHint)
		case entity.DiagError:
			report.Errors++
			uc.logger.Error("[AUDIT-FAIL] [%s] %s: %s (Fix: %s)", diag.System, diag.Target, diag.Details, diag.FixHint)
		default:
			// DiagInfo and future informational categories count as passed:
			// they carry context, not problems.
			report.Passed++
			uc.logger.Info("[AUDIT-INFO] [%s] %s: %s", diag.System, diag.Target, diag.Details)
		}
	}

	// 1. Audit Environment Variables
	uc.auditEnvVars(addDiag)
	// 1.5. Audit ~/.local/bin on PATH (provisioned helpers like `pw` live here).
	uc.auditLocalBinPATH(addDiag)
	// 2. Audit Git Global Configurations
	uc.auditGitConfigs(ctx, addDiag)
	// 3. Audit Config Files
	uc.auditConfigFiles(addDiag)
	// 4. Audit OpenCode Global Rules (AGENTS.md)
	uc.auditAGENTSRules(addDiag)
	// 4.5. Audit OpenCode MCP file references: opencode fails hard at
	uc.auditOpenCodeMCPRefs(ctx, addDiag)
	// 5. Audit Packages
	uc.auditEnvPackages(ctx, addDiag)
	// 5.5. Audit Gaming stack (Arch/CachyOS only, opt-in: silent unless Steam
	// is installed, so a non-gaming Arch box stays at zero warnings).
	uc.auditGamingStack(ctx, addDiag)

	// 5.6. Audit read-only OS performance state. Optional differences are
	// informational; this audit never applies a performance tweak.
	if uc.performanceInspector != nil {
		uc.auditLinuxPerformance(uc.performanceInspector.Snapshot(ctx), func(diagnostic entity.Diagnostic) {
			addDiag(diagnostic)
		})
	}

	// 6. Audit Skills (only the ones that belong on this OS and are enabled).
	uc.auditSkills(addDiag)
	// 7. Audit LSPs
	uc.auditLSPPresence(addDiag)
	// 7.5. Audit LSP stdio handshakes (presence in PATH is not proof the
	// server speaks LSP: exit codes lie, so health is proven by the
	// absence of a stdio connection error, not by the exit status).
	uc.auditLSPHandshake(ctx, addDiag)

	// 8. Audit Windows 11 Registry Tweaks, Features & Fonts (Windows only)
	uc.auditWindowsTweaks(ctx, addDiag)
	// 9. Audit Browser & Playwright
	uc.auditBrowserStack(addDiag)
	// 10. Audit Git Worktree Support
	uc.auditWorktreeSupport(ctx, addDiag)
	// 10.5 Audit the local verification wiring: the same gates run by the
	uc.auditVerifyWiring(addDiag)
	// 11. Audit Linux Toolchain Bootstrap (Linux only)
	uc.auditLinuxToolchain(ctx, addDiag)
	// 11.5. Audit WSL Ubuntu secondary shell (Windows only)
	uc.auditWSLSecondary(ctx, addDiag)
	// 11.6. Audit Windows console code page (UTF-8 required for Unicode glyph rendering)
	uc.auditConsoleCodePage(ctx, addDiag)
	// 12. Audit OpenCode storage accumulation & standardized temp folder
	uc.auditTempAndScratch(addDiag)
	// 13. Audit CommandCode Agent Health
	uc.auditCommandCodeHealth(addDiag)
	return report, nil
}
