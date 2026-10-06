package executil

import (
	"context"
	"fmt"
	"os/exec"
)

// SudoAvailable reports whether sudo exists and the current user can attempt
// privilege elevation. It does not confirm a valid credential — that is
// `sudo -v`'s job at the start of a privileged run.
func SudoAvailable() bool {
	_, err := exec.LookPath("sudo")
	return err == nil
}

// SudoPreflight refreshes the sudo credential timestamp so subsequent
// `sudo -n` calls in the same run do not prompt mid-way. It is meant to be
// called once, interactively, at the start of a run that touches privileged
// files. The returned error is descriptive enough to print as an instruction
// to the user.
func SudoPreflight() error {
	if !SudoAvailable() {
		return fmt.Errorf("sudo is not available on this system")
	}
	cmd := exec.Command("sudo", "-v")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo -v failed: %w — run 'sudo envctl run gaming' or validate your sudo session", err)
	}
	return nil
}

// RunPrivileged executes args through `sudo -n` (non-interactive, uses the
// timestamp refreshed by SudoPreflight). It never embeds a password. The
// combined output is folded into the error so a failing step stays
// diagnosable without leaking anything to stdout.
func RunPrivileged(ctx context.Context, args ...string) error {
	if !SudoAvailable() {
		return fmt.Errorf("sudo is not available on this system")
	}
	cmdArgs := append([]string{"-n"}, args...)
	cmd := exec.CommandContext(ctx, "sudo", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := string(out)
		if len(msg) > 300 {
			msg = msg[:300] + "..."
		}
		return fmt.Errorf("sudo -n %s failed: %v (%s)", args[0], err, msg)
	}
	return nil
}
