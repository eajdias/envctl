package performance

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

type swapfileManager struct {
	run      commandRunner
	fstab    *dropinWriter
	exists   func(absolute string) bool
	readFile func(absolute string) ([]byte, error)
	hasBtrfs func() bool
	elevate  bool
}

// NewSwapfileManager creates the production swapfile adapter.
func NewSwapfileManager() repository.SwapManager {
	return newSwapfileManager(
		execCommand,
		newDropinWriter("/etc/fstab", execCommand, nowFunc, needsElevation()),
		func(absolute string) bool {
			_, err := os.Stat(absolute)
			return err == nil
		},
		func(absolute string) ([]byte, error) { return os.ReadFile(absolute) },
		func() bool { _, err := execCommand(context.Background(), "btrfs", "--version"); return err == nil },
		needsElevation(),
	)
}

func newSwapfileManager(
	run commandRunner,
	fstab *dropinWriter,
	exists func(string) bool,
	readFile func(string) ([]byte, error),
	hasBtrfs func() bool,
	elevate bool,
) *swapfileManager {
	return &swapfileManager{
		run: run, fstab: fstab, exists: exists, readFile: readFile, hasBtrfs: hasBtrfs, elevate: elevate,
	}
}

// Ensure adopts an existing swap device or creates one.
//
// Adoption is the common case on this fleet: both Oracle hosts already ship a
// hand-created 8 GB swapfile, and the AWS host is the only one that needs a new
// one. An adopted device is reported and left completely alone — no resize, no
// re-mkswap, no fstab rewrite — because it is state the tool did not create.
func (m *swapfileManager) Ensure(
	ctx context.Context,
	spec entity.SwapSpec,
	hw entity.HardwareState,
	dryRun bool,
) ([]entity.Diagnostic, error) {
	if !spec.Enabled() {
		return []entity.Diagnostic{{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   "swap",
			Details:  "the swap policy is disabled; the host's swap is left exactly as it is",
		}}, nil
	}
	if err := entity.ValidateSwapSpec(spec); err != nil {
		return swapError(err.Error()), err
	}

	// The priority conflict is checked before adoption. A disk device that
	// already outranks the compressed tier means zram will never be used, and
	// silently adopting that state would report success for a host whose swap
	// policy is inverted.
	if conflict, detail := entity.DetectZRAMPriorityConflict(hw, entity.ZRAMPriority); conflict {
		return swapError(detail), fmt.Errorf("%s", detail)
	}

	if adopted, name, size := hw.AdoptedDiskSwap(); adopted {
		return []entity.Diagnostic{{
			Category: entity.DiagOK,
			System:   "Performance",
			Target:   "swap",
			Details: fmt.Sprintf(
				"adopting the existing disk swap %s (%s, priority %d); it was not created by envctl, so it is left untouched",
				name, entity.FormatBytes(size), hw.DiskSwapTopPri,
			),
		}}, nil
	}

	switch entity.ResolveSwapFilesystem(spec, hw.RootFSType) {
	case entity.SwapFilesystemRefuse:
		return []entity.Diagnostic{{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   "swap",
			Details: fmt.Sprintf(
				"not creating a swapfile: the root filesystem is %s and the policy only allows %v (btrfs: %s)",
				orUnknown(hw.RootFSType), spec.FSAllow, orUnknown(spec.FSBtrfs),
			),
		}}, nil
	}

	size, err := entity.ResolveSwapSizeBytes(spec, hw)
	if err != nil {
		return swapError(err.Error()), err
	}

	// A file at the declared path that is not an active swap is a refusal, not
	// something to hand to mkswap: it may be the operator's data.
	if m.exists != nil && m.exists(spec.File) {
		detail := fmt.Sprintf("refusing to touch %s: a file already exists there and no active swap uses it", spec.File)
		return swapError(detail), fmt.Errorf("%s", detail)
	}

	if dryRun {
		return []entity.Diagnostic{{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   "swap",
			Details: fmt.Sprintf("would create a %s swapfile at %s with priority %d and register it in fstab",
				entity.FormatBytes(size), spec.File, spec.Priority),
		}}, nil
	}

	btrfs := entity.ResolveSwapFilesystem(spec, hw.RootFSType) == entity.SwapFilesystemCreateBtrfs
	if err := m.allocate(spec.File, size, btrfs); err != nil {
		return swapError(err.Error()), err
	}
	if out, err := m.command(ctx, "mkswap", spec.File); err != nil {
		return swapError(fmt.Sprintf("mkswap %s failed: %v (%s)", spec.File, err, strings.TrimSpace(string(out)))), err
	}
	if out, err := m.command(ctx, "swapon", spec.File); err != nil {
		return swapError(fmt.Sprintf("swapon %s failed: %v (%s)", spec.File, err, strings.TrimSpace(string(out)))), err
	}
	if err := m.registerFstab(spec); err != nil {
		return swapError(err.Error()), err
	}

	return []entity.Diagnostic{{
		Category: entity.DiagOK,
		System:   "Performance",
		Target:   "swap",
		Details: fmt.Sprintf("created a %s swapfile at %s with priority %d",
			entity.FormatBytes(size), spec.File, spec.Priority),
	}}, nil
}

// allocate reserves the file's blocks. btrfs needs a different sequence: a swap
// file must be NODATACOW, fully allocated and hole-free, which is exactly what
// truncate + chattr + fallocate produces.
func (m *swapfileManager) allocate(path string, size uint64, btrfs bool) error {
	if btrfs {
		if m.hasBtrfs != nil && m.hasBtrfs() {
			if out, err := m.command(context.Background(), "btrfs", "filesystem", "mkswapfile",
				"--size", strconv.FormatUint(size, 10), path); err == nil {
				_ = out
				if _, err := m.command(context.Background(), "chmod", "600", path); err != nil {
					return fmt.Errorf("chmod 600 %s failed: %w", path, err)
				}
				return nil
			}
		}
		if out, err := m.command(context.Background(), "truncate", "-s", "0", path); err != nil {
			return fmt.Errorf("truncate %s failed: %v (%s)", path, err, strings.TrimSpace(string(out)))
		}
		if out, err := m.command(context.Background(), "chattr", "+C", path); err != nil {
			return fmt.Errorf("chattr +C %s failed: %v (%s)", path, err, strings.TrimSpace(string(out)))
		}
	}
	if out, err := m.command(context.Background(), "fallocate",
		"-l", strconv.FormatUint(size, 10), path); err != nil {
		return fmt.Errorf("fallocate %s failed: %v (%s)", path, err, strings.TrimSpace(string(out)))
	}
	if out, err := m.command(context.Background(), "chmod", "600", path); err != nil {
		return fmt.Errorf("chmod 600 %s failed: %v (%s)", path, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (m *swapfileManager) registerFstab(spec entity.SwapSpec) error {
	if m.readFile != nil {
		if data, err := m.readFile("/etc/fstab"); err == nil {
			if fstabHasSwapEntry(string(data), spec.File) {
				return nil
			}
		}
	}
	existing := ""
	if m.readFile != nil {
		if data, err := m.readFile("/etc/fstab"); err == nil {
			existing = string(data)
		}
	}
	content := existing
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += renderFstabEntry(spec.File, spec.Priority)
	changed, backup, err := m.fstab.Install(content, 0o644, false)
	if err != nil {
		return err
	}
	_ = changed
	_ = backup
	return nil
}

func (m *swapfileManager) command(ctx context.Context, name string, args ...string) ([]byte, error) {
	if m.elevate {
		return m.run(ctx, "sudo", append([]string{"-n", name}, args...)...)
	}
	return m.run(ctx, name, args...)
}

func swapError(detail string) []entity.Diagnostic {
	return []entity.Diagnostic{{
		Category: entity.DiagError,
		System:   "Performance",
		Target:   "swap",
		Details:  detail,
	}}
}

// renderFstabEntry emits one swap line. The priority is explicit rather than
// left to the tool default, so the ordering against the zram tier is stated in
// the file an operator will read.
func renderFstabEntry(path string, priority int) string {
	return fmt.Sprintf("%s none swap sw,pri=%d 0 0\n", path, priority)
}

func fstabHasSwapEntry(content, path string) bool {
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == path && fields[2] == "swap" {
			return true
		}
	}
	return false
}
