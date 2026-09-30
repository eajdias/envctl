package usecase

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// sudoAvailable reports whether sudo exists and the current user can attempt
// privilege elevation. It does not confirm a valid credential — that is
// `sudo -v`'s job at the start of a privileged run.
func sudoAvailable() bool {
	_, err := exec.LookPath("sudo")
	return err == nil
}

// sudoPreflight refreshes the sudo credential timestamp so subsequent
// `sudo -n` calls in the same run do not prompt mid-way. It is meant to be
// called once, interactively, at the start of the gаming provision run.
// The returned error is descriptive enough to print as instruction.
func sudoPreflight() error {
	if !sudoAvailable() {
		return fmt.Errorf("sudo is not available on this system")
	}
	cmd := exec.Command("sudo", "-v")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo -v failed: %w — run 'sudo envctl run gaming' or validate your sudo session", err)
	}
	return nil
}

// runPrivileged executes args through `sudo -n` (non-interactive, uses the
// timestamp refreshed by sudoPreflight). It never embeds a password.
func runPrivileged(ctx context.Context, args ...string) (string, error) {
	if !sudoAvailable() {
		return "", fmt.Errorf("sudo is not available on this system")
	}
	cmdArgs := append([]string{"-n"}, args...)
	cmd := exec.CommandContext(ctx, "sudo", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out)), fmt.Errorf("sudo -n %s failed: %v (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}