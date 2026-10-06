package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestRunErrorListsRegisteredSubsystems locks the unknown-subsystem contract:
// every name in the (valid: ...) list must exist as a registered subcommand
// and every registered subcommand must appear in the list, so adding a
// subsystem without registering it (or vice versa) breaks the build.
func TestRunErrorListsRegisteredSubsystems(t *testing.T) {
	// Dispatch through a parent like production does: cobra only falls back
	// to the run RunE for nested dispatch, a bare newRunCmd().Execute()
	// rejects the arg as "unknown command" before RunE runs.
	root := &cobra.Command{Use: "envctl"}
	run := newRunCmd()
	run.SilenceErrors = true
	run.SilenceUsage = true
	root.AddCommand(run)
	root.SetArgs([]string{"run", "subsystem-that-does-not-exist"})
	root.SilenceErrors = true
	root.SilenceUsage = true

	err := root.Execute()
	if err == nil {
		t.Fatal("expected an unknown-subsystem error")
	}
	const prefix = "(valid: "
	msg := err.Error()
	start := strings.Index(msg, prefix)
	if start < 0 {
		t.Fatalf("error %q carries no (valid: ...) list", msg)
	}
	listed := map[string]bool{}
	for _, name := range strings.Split(strings.TrimSuffix(msg[start+len(prefix):], ")"), ", ") {
		listed[strings.TrimSpace(name)] = true
	}

	registered := map[string]bool{}
	for _, sub := range run.Commands() {
		registered[sub.Name()] = true
	}
	for name := range listed {
		if !registered[name] {
			t.Errorf("error lists %q, which is not a registered run subcommand", name)
		}
	}
	for name := range registered {
		if !listed[name] {
			t.Errorf("registered run subcommand %q is missing from the error list", name)
		}
	}
}

// TestRunTargetsCoverHelpEntries guards the table itself: every runTargets
// row must surface in the help text with its Short description.
func TestRunTargetsCoverHelpEntries(t *testing.T) {
	cmd := newRunCmd()
	var sb strings.Builder
	cmd.SetOut(&sb)
	cmd.SetErr(&sb)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("run --help: %v", err)
	}
	help := sb.String()
	for _, target := range runTargets {
		if !strings.Contains(help, target.name) || !strings.Contains(help, target.short) {
			t.Errorf("runTargets row %q missing from help output", target.name)
		}
	}
}
