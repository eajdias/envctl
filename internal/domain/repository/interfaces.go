package repository

import (
	"context"

	"github.com/eajdias/envctl/internal/domain/entity"
)

// PackageManager defines operations for installing and checking software packages.
type PackageManager interface {
	Type() entity.PackageType
	IsAvailable(ctx context.Context) bool
	IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error)
	Install(ctx context.Context, pkg entity.Package) error
	ListInstalled(ctx context.Context) ([]entity.Package, error)
}

// Logger provides structured and persistent execution logging to disk.
type Logger interface {
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
	Debug(format string, args ...any)
	LogCommand(cmd string, args []string, exitCode int, output string, err error)
	LogIdempotency(system, target string, skipped bool, reason string)
	GetLogFilePath() string
	Close() error
}
