package cli

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/eajdias/win11-new/internal/domain/entity"
	"github.com/eajdias/win11-new/internal/domain/repository"
	"github.com/eajdias/win11-new/internal/infra/embedded"
	"github.com/eajdias/win11-new/internal/infra/environment"
	"github.com/eajdias/win11-new/internal/infra/filesystem"
	"github.com/eajdias/win11-new/internal/infra/git"
	"github.com/eajdias/win11-new/internal/infra/logger"
	"github.com/eajdias/win11-new/internal/infra/msys2"
	"github.com/eajdias/win11-new/internal/infra/toolchain"
	"github.com/eajdias/win11-new/internal/infra/winget"
	"github.com/eajdias/win11-new/internal/usecase"
)

type AppContext struct {
	EmbeddedFS      fs.FS
	ManifestRepo    repository.ManifestRepository
	FSManager       repository.FileSystemManager
	EnvManager      repository.WindowsEnvManager
	GitManager      repository.GitManager
	Logger          repository.Logger
	PackageManagers map[entity.PackageType]repository.PackageManager

	// UseCases
	ProvisionPkgsUC   *usecase.ProvisionPackagesUseCase
	ProvisionShellUC  *usecase.ProvisionShellUseCase
	ProvisionSkillsUC *usecase.ProvisionSkillsUseCase
	ProvisionLSPUC    *usecase.ProvisionLSPsUseCase
	DoctorAuditUC     *usecase.DoctorAuditUseCase
	SnapshotSyncUC    *usecase.SnapshotSyncUseCase
}

var (
	logDirFlag string

	rootCmd = &cobra.Command{
		Use:   "win11-new",
		Short: "win11-new: Automated Windows 11 PRO / MSYS2 / OpenCode Environment Provisioner",
		Long:  `win11-new is an idempotent, Clean Architecture CLI tool designed to provision, audit, and synchronize your Windows 11 development environment.`,
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
)

func InitApp(embeddedFS fs.FS) {
	rootCmd.PersistentFlags().StringVar(&logDirFlag, "log-dir", "", "Custom directory for execution logs (defaults to ~/.win11-new/logs)")

	fsManager := filesystem.NewFileSystemManager()
	manifestRepo := embedded.NewManifestRepository(embeddedFS, "")
	envManager := environment.NewWindowsEnvManager()
	gitManager := git.NewGitManager()

	fileLogger, err := logger.NewFileLogger(logDirFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize file logger: %v\n", err)
	}

	pkgManagers := map[entity.PackageType]repository.PackageManager{
		entity.PackageTypeWinget:     winget.NewWingetManager(),
		entity.PackageTypePacman:     msys2.NewPacmanManager(),
		entity.PackageTypeVolta:      toolchain.NewVoltaManager(),
		entity.PackageTypeDotnetTool: toolchain.NewDotnetToolManager(),
		entity.PackageTypeNpm:        toolchain.NewNpmManager(),
		entity.PackageTypePip:        toolchain.NewPipManager(),
	}

	appCtx = &AppContext{
		EmbeddedFS:        embeddedFS,
		ManifestRepo:      manifestRepo,
		FSManager:         fsManager,
		EnvManager:        envManager,
		GitManager:        gitManager,
		Logger:            fileLogger,
		PackageManagers:   pkgManagers,
		ProvisionPkgsUC:   usecase.NewProvisionPackagesUseCase(manifestRepo, pkgManagers, fileLogger),
		ProvisionShellUC:  usecase.NewProvisionShellUseCase(manifestRepo, fsManager, envManager, gitManager, embeddedFS, fileLogger),
		ProvisionSkillsUC: usecase.NewProvisionSkillsUseCase(manifestRepo, fsManager, embeddedFS, fileLogger),
		ProvisionLSPUC:    usecase.NewProvisionLSPsUseCase(manifestRepo, pkgManagers, fileLogger),
		DoctorAuditUC:     usecase.NewDoctorAuditUseCase(manifestRepo, fsManager, envManager, gitManager, pkgManagers, fileLogger),
		SnapshotSyncUC:    usecase.NewSnapshotSyncUseCase(manifestRepo, fsManager, gitManager, fileLogger),
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
		Short: "Print win11-new version",
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			PrintInfo("win11-new v1.0.0 (Windows 11 PRO / MSYS2 / OpenCode)")
		},
	}
}
