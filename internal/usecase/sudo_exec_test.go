package usecase

import (
	"testing"
)

// TestSudoAvailable covers the lookup path only: real elevation is
// environment-dependent and cannot run in CI. The false path (no sudo in
// PATH) is what the scaffolding must degrade to.
func TestSudoAvailable(t *testing.T) {
	t.Setenv("PATH", "")
	if sudoAvailable() {
		t.Error("expected sudo to be reported unavailable with an empty PATH")
	}
}

func TestRunPrivilegedDegradesWithoutSudo(t *testing.T) {
	t.Setenv("PATH", "")
	if _, err := runPrivileged(t.Context(), "true"); err == nil {
		t.Fatal("expected an error when sudo is not on PATH")
	}
}