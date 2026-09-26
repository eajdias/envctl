package performance

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/eajdias/envctl/internal/domain/entity"
)

// zramService is the swap unit systemd-zram-generator emits. The generator
// creates it; the tool only starts it when the device is absent.
const zramService = "dev-zram0.swap"

// journaldService is restarted, never stopped: man 8 systemd-journald.
const journaldService = "systemd-journald"

// The 90- prefix follows the same convention systemd documents for
// *.conf.d/ directories: files are sorted lexicographically by filename and the
// last one wins for single-value keys, so a prefix orders the admin drop-in
// after anything a vendor ships.
const (
	defaultSystemLimitsDropin = "/etc/systemd/system.conf.d/90-envctl-limits.conf"
	defaultPAMLimitsDropin    = "/etc/security/limits.d/90-envctl-limits.conf"
)

// defaultJournaldDropin uses the 90- prefix because systemd sorts drop-ins in
// *.conf.d/ directories lexicographically by filename and the last file wins
// for single-value keys.
const defaultJournaldDropin = "/etc/systemd/journald.conf.d/90-envctl-journald.conf"

// nowFunc and needsElevation are the two environment facts every adapter needs.
// They are indirections so the production values are decided in exactly one
// place and tests never depend on the host they run on.
var (
	nowFunc        = time.Now
	needsElevation = func() bool { return os.Geteuid() != 0 }
)

// execCommand is the single place this package shells out. Every call is
// argv-based; no manager ever builds a shell string.
func execCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

func sortSysctlSettings(settings []entity.SysctlSetting) {
	sort.Slice(settings, func(i, j int) bool { return settings[i].Key < settings[j].Key })
}

// dropinWriter installs a single managed configuration file atomically. Every
// privileged write in this package goes through it so the backup, the
// same-directory rename and the dry-run guarantee are implemented exactly once
// instead of once per manager.
type dropinWriter struct {
	destination string
	run         commandRunner
	now         func() time.Time
	elevate     bool
}

func newDropinWriter(destination string, run commandRunner, now func() time.Time, elevate bool) *dropinWriter {
	return &dropinWriter{
		destination: destination,
		run:         run,
		now:         now,
		elevate:     elevate,
	}
}

// Install writes content to the managed path.
//
// The sequence is: back up the existing file with cp -a, install a temporary
// file *in the destination directory*, then rename it over the destination. The
// temporary file must share the destination's directory so the rename stays
// atomic; a temporary in the system temp directory would degrade to a
// cross-device copy and could leave a truncated drop-in behind.
//
// Identical content is a no-op: no command runs and no backup is created, so a
// second provisioning run over a converged host changes nothing.
func (w *dropinWriter) Install(content string, mode os.FileMode, dryRun bool) (changed bool, backup string, err error) {
	existing, readErr := os.ReadFile(w.destination)
	if readErr == nil && string(existing) == content {
		return false, "", nil
	}
	if readErr != nil && !os.IsNotExist(readErr) {
		return false, "", fmt.Errorf("read existing %s: %w", w.destination, readErr)
	}
	if dryRun {
		return false, "", nil
	}

	if readErr == nil {
		backup = w.nextBackupPath()
		if out, err := w.command(context.Background(), "cp", "-a", w.destination, backup); err != nil {
			return false, "", fmt.Errorf("backup %s failed: %v (%s)", w.destination, err, strings.TrimSpace(string(out)))
		}
	}

	// The drop-in directory does not exist on a clean host: Ubuntu ships
	// journald.conf only as a single file, with no journald.conf.d/, and
	// system.conf.d/ is absent unless something created it. The temporary file
	// must share the destination's directory for the rename to be atomic, so
	// the directory has to exist first.
	if err := w.ensureParentDir(context.Background()); err != nil {
		return false, backup, err
	}

	tmp, err := os.CreateTemp(filepath.Dir(w.destination), ".envctl-dropin-")
	if err != nil {
		return false, backup, fmt.Errorf("create temporary drop-in next to %s: %w", w.destination, err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		cleanup()
		return false, backup, fmt.Errorf("write temporary drop-in: %w", err)
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		cleanup()
		return false, backup, fmt.Errorf("set temporary drop-in permissions: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return false, backup, fmt.Errorf("close temporary drop-in: %w", err)
	}

	tempDestination := w.nextTempPath()
	if out, err := w.command(context.Background(), "install", "-m", modeString(mode), tmpName, tempDestination); err != nil {
		cleanup()
		return false, backup, fmt.Errorf("stage drop-in failed: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	cleanup()

	if out, err := w.command(context.Background(), "mv", "-f", tempDestination, w.destination); err != nil {
		_, _ = w.command(context.Background(), "rm", "-f", tempDestination)
		return false, backup, fmt.Errorf("replace %s failed: %v (%s)", w.destination, err, strings.TrimSpace(string(out)))
	}
	return true, backup, nil
}

// ensureParentDir creates the drop-in's directory when it is missing.
func (w *dropinWriter) ensureParentDir(ctx context.Context) error {
	dir := filepath.Dir(w.destination)
	if _, err := os.Stat(dir); err == nil {
		return nil
	}
	if out, err := w.command(ctx, "mkdir", "-p", dir); err != nil {
		return fmt.Errorf("create drop-in directory %s failed: %v (%s)", dir, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (w *dropinWriter) command(ctx context.Context, name string, args ...string) ([]byte, error) {
	if w.elevate {
		return w.run(ctx, "sudo", append([]string{"-n", name}, args...)...)
	}
	return w.run(ctx, name, args...)
}

func (w *dropinWriter) nextBackupPath() string {
	stamp := w.now().Format("20060102-150405")
	return w.disambiguate(fmt.Sprintf("%s.bak.%s", w.destination, stamp))
}

func (w *dropinWriter) nextTempPath() string {
	stamp := w.now().Format("20060102-150405")
	return w.disambiguate(fmt.Sprintf("%s.tmp.%s", w.destination, stamp))
}

func (w *dropinWriter) disambiguate(base string) string {
	if _, err := os.Stat(base); os.IsNotExist(err) {
		return base
	}
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func modeString(mode os.FileMode) string {
	return fmt.Sprintf("%04o", mode.Perm())
}
