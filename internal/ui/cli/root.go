package cli

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
	"github.com/eajdias/envctl/internal/infra/apt"
	"github.com/eajdias/envctl/internal/infra/embedded"
	"github.com/eajdias/envctl/internal/infra/environment"
	"github.com/eajdias/envctl/internal/infra/filesystem"
	"github.com/eajdias/envctl/internal/infra/git"
	"github.com/eajdias/envctl/internal/infra/logger"
	"github.com/eajdias/envctl/internal/infra/toolchain"
	"github.com/eajdias/envctl/internal/infra/windows"
	"github.com/eajdias/envctl/internal/infra/winget"
	"github.com/eajdias/envctl/internal/usecase"
)

type AppContext struct {
	EmbeddedFS      fs.FS
	ManifestRepo    repository.ManifestRepository
	FSManager       repository.FileSystemManager
	EnvManager      repository.WindowsEnvManager
	GitManager      repository.GitManager
	TweaksManager   repository.WindowsTweaksManager
	Logger          repository.Logger
	PackageManagers map[entity.PackageType]repository.PackageManager

	// UseCases
	ProvisionPkgsUC      *usecase.ProvisionPackagesUseCase
	ProvisionShellUC     *usecase.ProvisionShellUseCase
	ProvisionSkillsUC    *usecase.ProvisionSkillsUseCase
	ProvisionLSPUC       *usecase.ProvisionLSPsUseCase
	ProvisionWindowsUC   *usecase.ProvisionWindowsUseCase
	ProvisionBootstrapUC *usecase.ProvisionBootstrapUseCase
	DoctorAuditUC        *usecase.DoctorAuditUseCase
	SnapshotSyncUC       *usecase.SnapshotSyncUseCase
	TempHygieneUC        *usecase.TempHygieneUseCase
	CleanupOpenCodeUC    *usecase.CleanupOpenCodeUseCase
}

var (
	logDirFlag string

	rootCmd = &cobra.Command{
		Use:   "envctl",
		Short: "envctl: Universal Development Environment Provisioner (Windows 11 PRO & Ubuntu Linux / OpenCode)",
		Long:  `envctl is an idempotent, Clean Architecture CLI tool designed to provision, audit, and synchronize your development environments across Windows 11 PRO workstations and Ubuntu Linux VPS servers.`,
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if appCtx != nil && appCtx.Logger != nil {
				logPath := appCtx.Logger.GetLogFilePath()
				_ = appCtx.Logger.Close()
				pterm.Println()
				pterm.Info.Printf("Detailed execution log saved to: %s\n", logPath)
			}
		},
	}

	appCtx *AppContext
	// appVersion is injected from cmd/envctl via InitApp; it is set at build
	// time through -ldflags "-X main.Version=$VERSION" (Makefile/release.yml).
	appVersion = "dev"
)

func InitApp(embeddedFS fs.FS, version string) {
	if version != "" {
		appVersion = version
	}
	rootCmd.PersistentFlags().StringVar(&logDirFlag, "log-dir", "", "Custom directory for execution logs (defaults to ~/.envctl/logs)")

	fsManager := filesystem.NewFileSystemManager()
	manifestRepo := embedded.NewManifestRepository(embeddedFS, "")
	envManager := environment.NewWindowsEnvManager()
	gitManager := git.NewGitManager()

	fileLogger, err := logger.NewFileLogger(logDirFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize file logger: %v\n", err)
	}

	windowsTweaksMgr := windows.NewWindowsTweaksManager(fileLogger)

	pkgManagers := map[entity.PackageType]repository.PackageManager{
		entity.PackageTypeWinget:     winget.NewWingetManager(),
		entity.PackageTypeApt:        apt.NewAptManager(),
		entity.PackageTypeVolta:      toolchain.NewVoltaManager(),
		entity.PackageTypeDotnetTool: toolchain.NewDotnetToolManager(),
		entity.PackageTypeNpm:        toolchain.NewNpmManager(),
		entity.PackageTypePip:        toolchain.NewPipManager(),
		entity.PackageTypeGo:         toolchain.NewGoManager(),
		entity.PackageTypeRustup:     toolchain.NewRustupManager(),
	}

	appCtx = &AppContext{
		EmbeddedFS:           embeddedFS,
		ManifestRepo:         manifestRepo,
		FSManager:            fsManager,
		EnvManager:           envManager,
		GitManager:           gitManager,
		TweaksManager:        windowsTweaksMgr,
		Logger:               fileLogger,
		PackageManagers:      pkgManagers,
		ProvisionPkgsUC:      usecase.NewProvisionPackagesUseCase(manifestRepo, pkgManagers, fileLogger),
		ProvisionShellUC:     usecase.NewProvisionShellUseCase(manifestRepo, fsManager, envManager, gitManager, embeddedFS, fileLogger),
		ProvisionSkillsUC:    usecase.NewProvisionSkillsUseCase(manifestRepo, fsManager, embeddedFS, fileLogger),
		ProvisionLSPUC:       usecase.NewProvisionLSPsUseCase(manifestRepo, pkgManagers, fileLogger),
		ProvisionWindowsUC:   usecase.NewProvisionWindowsUseCase(manifestRepo, windowsTweaksMgr, fileLogger),
		ProvisionBootstrapUC: usecase.NewProvisionBootstrapUseCase(fsManager, fileLogger),
		DoctorAuditUC:        usecase.NewDoctorAuditUseCase(manifestRepo, fsManager, envManager, gitManager, windowsTweaksMgr, pkgManagers, fileLogger),
		SnapshotSyncUC:       usecase.NewSnapshotSyncUseCase(manifestRepo, fsManager, gitManager, fileLogger),
		CleanupOpenCodeUC:    usecase.NewCleanupOpenCodeUseCase(fsManager, fileLogger),
	}

	registerCommands()
}

func registerCommands() {
	rootCmd.AddCommand(newRunCmd())
	rootCmd.AddCommand(newDoctorCmd())
	rootCmd.AddCommand(newSnapshotCmd())
	rootCmd.AddCommand(newVersionCmd())
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print envctl version",
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			PrintInfo(fmt.Sprintf("envctl %s (Windows 11 PRO & Ubuntu Linux / OpenCode Ecosystem)", appVersion))
		},
	}
}
