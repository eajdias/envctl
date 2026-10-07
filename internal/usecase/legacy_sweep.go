package usecase

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
	"github.com/eajdias/envctl/internal/infra/executil"
	"github.com/eajdias/envctl/internal/infra/filesystem"
)

// legacyShimDirs are the pre-mise locations whose copies shadow mise shims on
// PATH: the npm `--prefix ~/.local` bin dir and the Volta shim dir. Volta is
// still installed on migrated machines, so its shims stay shadowing until
// archived file by file.
func legacyShimDirs(home string) []string {
	dirs := []string{filepath.Join(home, ".local", "bin")}
	if volta := executil.VoltaBinDir(home); volta != "" {
		dirs = append(dirs, volta)
	}
	return dirs
}

// toolBinName takes the binary out of a manifest check command
// ("prettier --version" -> "prettier").
func toolBinName(checkCommand string) string {
	fields := strings.Fields(checkCommand)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// shimFileExists reports whether the mise shim for bin exists.
func shimFileExists(shimDir, bin string) bool {
	candidates := []string{bin}
	if runtime.GOOS == "windows" {
		candidates = []string{bin + ".exe", bin}
	}
	for _, c := range candidates {
		if st, err := os.Stat(filepath.Join(shimDir, c)); err == nil && !st.IsDir() {
			return true
		}
	}
	return false
}

// legacyCandidates lists the stale filenames a pre-mise installer may have
// left for bin (bare name on POSIX; npm .cmd/.ps1 and Volta extensionless
// pairs on Windows).
func legacyCandidates(dir, bin string) []string {
	paths := []string{filepath.Join(dir, bin)}
	if runtime.GOOS == "windows" {
		for _, ext := range []string{".cmd", ".ps1", ".exe", ".bat"} {
			paths = append(paths, filepath.Join(dir, bin+ext))
		}
	}
	return paths
}

// sweepMigratedBin archives pre-mise copies of one tool when it migrated to
// the mise npm backend (id "npm:<pkg>"). Call it before the installed-check
// (a legacy shim would otherwise answer the probe and skip the migration)
// and after a successful install (the shim only exists then). No-ops for
// every other type.
func sweepMigratedBin(logger repository.Logger, pkgType entity.PackageType, id, bin string) {
	if pkgType != entity.PackageTypeMise || !strings.HasPrefix(id, "npm:") || bin == "" {
		return
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return
	}
	bins := []string{bin}
	if short := npmShortName(id); short != "" && short != bin {
		// Volta links alias shims under the package name too
		// ("command-code" beside "cmdc"); the shim-exists guard inside
		// keeps this safe — a missing shim skips, never deletes.
		bins = append(bins, short)
	}
	sweepLegacyToolShims(logger, home, bins)
}

// npmShortName takes the install name out of a backend id
// ("npm:command-code" -> "command-code", "npm:@playwright/cli" -> "cli").
func npmShortName(id string) string {
	rest := id
	if i := strings.Index(rest, ":"); i >= 0 {
		rest = rest[i+1:]
	} else {
		return ""
	}
	if i := strings.LastIndex(rest, "/"); i >= 0 {
		rest = rest[i+1:]
	}
	return rest
}

// sweepLegacyToolShims archives stale pre-mise copies of bins once the mise
// shim for the same binary exists, so the new install actually wins on PATH.
// It never removes a binary without a working replacement: a missing shim
// leaves every legacy copy untouched. Returns the archived paths.
func sweepLegacyToolShims(logger repository.Logger, home string, bins []string) []string {
	shimDir := executil.MiseShimDir(home)
	var swept []string
	for _, bin := range bins {
		if bin == "" {
			continue
		}
		if !shimFileExists(shimDir, bin) {
			continue
		}
		for _, dir := range legacyShimDirs(home) {
			for _, legacy := range legacyCandidates(dir, bin) {
				st, err := os.Stat(legacy)
				if err != nil || st.IsDir() {
					continue
				}
				backup, err := filesystem.ArchivePath(legacy)
				if err != nil {
					logger.Warn("Legacy sweep: could not archive stale %s: %v", legacy, err)
					continue
				}
				logger.Info("Legacy sweep: archived stale %s (mise shim %s wins)", legacy, bin)
				swept = append(swept, backup)
			}
		}
	}
	return swept
}
