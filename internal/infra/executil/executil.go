package executil

import (
	"context"
	"os"
	"os/exec"
	"strings"
)

func ProbeCheckCommand(ctx context.Context, check string) (string, bool) {
	parts := strings.Fields(check)
	if len(parts) == 0 {
		return "", false
	}
	// #nosec G204 -- argv elements handed to exec directly (no shell); the
	// check comes from the embedded manifest, never from user input.
	out, err := exec.CommandContext(ctx, parts[0], parts[1:]...).CombinedOutput()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

func IsNonRoot() bool {
	return os.Geteuid() != 0
}
