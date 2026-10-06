package executil

import (
	"context"
	"os"
	"strings"
)

// PSQuote escapes single quotes for safe embedding in a PowerShell string
// literal. Single home for the helper both the environment and the tweaks
// managers need; quoting must never drift between the two PowerShell
// emitters.
func PSQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func ProbeCheckCommand(ctx context.Context, check string) (string, bool) {
	parts := strings.Fields(check)
	if len(parts) == 0 {
		return "", false
	}
	// Resolved through ExecTool (not bare exec): a CheckCommand naming a
	// toolchain shim (volta-managed node, ~/.local/bin helper) must probe
	// the same binary the install step would run. See ToolchainDirs.
	// #nosec G204 -- argv elements handed to exec directly (no shell); the
	// check comes from the embedded manifest, never from user input.
	out, err := ExecTool(ctx, parts[0], parts[1:]...).CombinedOutput()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

func IsNonRoot() bool {
	return os.Geteuid() != 0
}
