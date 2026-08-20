package usecase

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

const (
	tempHygieneWarnThresholdBytes = 300 * 1024 * 1024
	tempReclaimableWarnBytes      = 100 * 1024 * 1024
	tempScratchDirMaxAge          = 6 * time.Hour
	tempRandomDirMaxAge           = 24 * time.Hour
)

// TempHygieneUseCase audits and prunes stale temporary/scratch artifacts that
// accumulate in the MSYS2/OS temp directories (e.g. C:\msys64\tmp on Windows).
type TempHygieneUseCase struct {
	logger repository.Logger
}

func NewTempHygieneUseCase(logger repository.Logger) *TempHygieneUseCase {
	return &TempHygieneUseCase{logger: logger}
}

// TempCleanupReport summarizes a temp hygiene cleanup pass.
type TempCleanupReport struct {
	CheckedDirs int
	Removed     int
	FreedBytes  int64
	Remaining   int64
	Skipped     []string
	Failed      []string
}

// tempMatch is the classification result for a single temp entry.
type tempMatch struct {
	remove bool
	reason string
}

// classifyTempEntry decides whether a top-level temp entry is a stale artifact
// safe to remove. age is the time elapsed since the entry was last modified.
func classifyTempEntry(name string, isDir bool, age time.Duration) tempMatch {
	lower := strings.ToLower(name)
	always := func(reason string) tempMatch { return tempMatch{true, reason} }
	never := tempMatch{false, ""}

	// Bun runtime native-module extractions (one set per opencode/bun execution).
	if (strings.HasPrefix(lower, ".bdef") && strings.HasSuffix(lower, ".dll")) ||
		(strings.HasPrefix(lower, ".feef") && strings.HasSuffix(lower, ".node")) {
		return always("bun native module extraction")
	}
	// Node/V8 and tsx runtime caches.
	if strings.HasPrefix(lower, "node-compile-cache") || strings.HasPrefix(lower, "tsx-") {
		return always("node/tsx runtime cache")
	}
	// Agent scratch and tool downloads from past sessions.
	if strings.HasPrefix(lower, "zscan-") || lower == "meslo.zip" {
		return always("agent scratch/tool download")
	}
	if strings.HasPrefix(lower, "playwright_chromiumdev_profile-") ||
		strings.HasPrefix(lower, "system-commandline-sentinel-files") {
		return always("playwright/opencode session scratch")
	}
	// Native package-manager / editor caches (regenerable on demand).
	if lower == "winget" || lower == "nugetscratch" || lower == "chocolatey" ||
		lower == "vscode-stable-user-x64" {
		return always("package manager / editor cache")
	}
	// Stray native installer/tool logs.
	if strings.HasPrefix(lower, "dd_vcredist_") ||
		strings.HasPrefix(lower, "microsoft.net.workload_") ||
		strings.HasPrefix(lower, "vscode-inno-updater-") ||
		strings.HasPrefix(lower, "gbak_repair") ||
		strings.HasPrefix(lower, "gfix_") {
		return always("installer/tool log")
	}
	// Known agent scratch files by exact name.
	switch lower {
	case "del", "doctor_out.txt", "jrdfiles.txt", "tree.json", "validation.cpp",
		"defined.txt", "defined_classes.txt", "encrypt.js", "diagnostics.html",
		"diag-ipv6.png", "before-save-auto.png", "after-save-auto.png", "after-save-auto2.png",
		"doctor_vps.txt", "ods.h", "ods25.h", "backup25.cpp", "backup25.epp",
		"cch25.cpp", "cch_proto.h", "dpm25.cpp", "dpm25.epp", "gds.cpp",
		"dump_pages.py", "analyze_pages.py", "check_mermaid.py":
		return always("agent scratch file")
	}
	if strings.HasPrefix(lower, "del") && strings.HasSuffix(lower, ".tmp") {
		return always("stray temp file")
	}
	if strings.HasSuffix(lower, ".fdb") || strings.HasSuffix(lower, ".epp") {
		return always("database/source scratch copy")
	}
	// Agent scratch directory from a previous session (protect active ones).
	if lower == "opencode" && isDir {
		if age > tempScratchDirMaxAge {
			return always("opencode session scratch (stale)")
		}
		return never // may belong to the active session
	}
	// Random hash / 3-char random temp dirs (NuGet, MSBuild, .NET tooling).
	if isDir && age > tempRandomDirMaxAge && isRandomTempDir(lower) {
		return always("tooling temp dir (stale)")
	}
	return never
}

func isRandomTempDir(lower string) bool {
	if len(lower) == 32 {
		allHex := true
		for _, r := range lower {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
				allHex = false
				break
			}
		}
		if allHex {
			return true
		}
	}
	parts := strings.Split(lower, ".")
	if len(parts) != 2 {
		return false
	}
	base, ext := parts[0], parts[1]
	if len(base) < 3 || len(base) > 12 {
		return false
	}
	switch ext {
	case "yvw", "vte", "pyr", "tmp":
		return true
	}
	return false
}

// tempRoots returns the unique, existing temp directories relevant for hygiene.
func tempRoots() []string {
	seen := make(map[string]bool)
	var out []string
	add := func(p string) {
		if p == "" {
			return
		}
		p = filepath.Clean(p)
		if seen[p] {
			return
		}
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			seen[p] = true
			out = append(out, p)
		}
	}
	for _, k := range []string{"TMP", "TEMP", "TMPDIR"} {
		add(os.Getenv(k))
	}
	add(os.TempDir())
	if runtime.GOOS == "windows" {
		add(`C:\msys64\tmp`)
	} else {
		add("/tmp")
	}
	return out
}

func dirSize(path string) (int64, error) {
	var size int64
	_ = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.Type().IsRegular() {
			if info, err := d.Info(); err == nil {
				size += info.Size()
			}
		}
		return nil
	})
	return size, nil
}

// Audit produces health diagnostics about temp directory usage.
func (uc *TempHygieneUseCase) Audit(ctx context.Context) []entity.Diagnostic {
	var diags []entity.Diagnostic
	roots := tempRoots()
	if len(roots) == 0 {
		return []entity.Diagnostic{{
			Category: entity.DiagWarning,
			System:   "TempHygiene",
			Target:   "temp",
			Details:  "No temp directory could be detected for hygiene audit",
			FixHint:  "ensure TMP/TEMP/TMPDIR points to an existing directory",
		}}
	}

	var total, reclaimable int64
	var reclaimDetails []string
	now := time.Now()
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			info, err := e.Info()
			if err != nil {
				continue
			}
			size, _ := dirSize(filepath.Join(root, e.Name()))
			total += size
			if classifyTempEntry(e.Name(), e.IsDir(), now.Sub(info.ModTime())).remove {
				reclaimable += size
				reclaimDetails = append(reclaimDetails,
					fmt.Sprintf("%s (%.1fMB)", e.Name(), float64(size)/(1024*1024)))
			}
		}
	}

	target := strings.Join(roots, ", ")
	if reclaimable > tempReclaimableWarnBytes {
		diags = append(diags, entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   "TempHygiene",
			Target:   target,
			Details:  fmt.Sprintf("Stale temp artifacts: %.1f MB across %d entries — e.g. %s", float64(reclaimable)/(1024*1024), len(reclaimDetails), joinPreview(reclaimDetails, 3)),
			FixHint:  "run 'envctl doctor --fix' (TempHygiene step) to prune them",
		})
	} else if total > tempHygieneWarnThresholdBytes {
		diags = append(diags, entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   "TempHygiene",
			Target:   target,
			Details:  fmt.Sprintf("Temp usage %.1f MB, but only %.1f MB is reclaimable — review manually if disk is tight", float64(total)/(1024*1024), float64(reclaimable)/(1024*1024)),
			FixHint:  "review temp directory contents manually",
		})
	} else {
		diags = append(diags, entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "TempHygiene",
			Target:   target,
			Details:  fmt.Sprintf("Temp usage %.1f MB, reclaimable %.1f MB", float64(total)/(1024*1024), float64(reclaimable)/(1024*1024)),
		})
	}
	return diags
}

// Cleanup prunes stale temp artifacts, skipping entries locked by running processes.
func (uc *TempHygieneUseCase) Cleanup(ctx context.Context) (*TempCleanupReport, error) {
	report := &TempCleanupReport{}
	roots := tempRoots()
	report.CheckedDirs = len(roots)
	now := time.Now()

	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			report.Failed = append(report.Failed, fmt.Sprintf("%s: %v", root, err))
			continue
		}
		for _, e := range entries {
			if ctx.Err() != nil {
				report.Failed = append(report.Failed, "context cancelled before full pass")
				return report, ctx.Err()
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			name := e.Name()
			if !classifyTempEntry(name, e.IsDir(), now.Sub(info.ModTime())).remove {
				size, _ := dirSize(filepath.Join(root, name))
				report.Remaining += size
				continue
			}
			path := filepath.Join(root, name)
			size, _ := dirSize(path)
			if err := os.RemoveAll(path); err != nil {
				report.Skipped = append(report.Skipped, fmt.Sprintf("%s (in use: %v)", path, err))
				report.Remaining += size
				continue
			}
			report.Removed++
			report.FreedBytes += size
			if uc.logger != nil {
				uc.logger.Info("[TEMP-CLEAN] removed %s (%d bytes)", path, size)
			}
		}
	}
	return report, nil
}

func joinPreview(items []string, max int) string {
	if len(items) == 0 {
		return "none"
	}
	if len(items) > max {
		return fmt.Sprintf("%s, ... (%d total)", strings.Join(items[:max], ", "), len(items))
	}
	return strings.Join(items, ", ")
}
