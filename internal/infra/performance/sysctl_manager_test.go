package performance

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func TestRenderSysctlConfigRejectsNewlines(t *testing.T) {
	_, err := renderSysctlConfig([]entity.SysctlSetting{{Key: "vm.swappiness", Value: "10\nnet.ipv4.ip_forward=1"}})
	if err == nil {
		t.Fatal("expected newline injection to be rejected")
	}
}

func TestSysctlManagerDryRunDoesNotWrite(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "90-envctl-performance.conf")
	called := false
	manager := newSysctlManager(destination, func(context.Context, string, ...string) ([]byte, error) {
		called = true
		return nil, nil
	}, func() time.Time { return time.Unix(1, 0) }, false)

	diags, err := manager.Apply(context.Background(), []entity.SysctlSetting{{Key: "vm.swappiness", Value: "10"}}, true)
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}
	if called {
		t.Fatal("dry-run invoked a command")
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagInfo {
		t.Fatalf("diagnostics = %#v, want one info diagnostic", diags)
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatal("dry-run created the destination file")
	}
}

func TestSysctlManagerBacksUpAndApplies(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "90-envctl-performance.conf")
	oldContent := []byte("vm.swappiness = 10\n")
	if err := os.WriteFile(destination, oldContent, 0644); err != nil {
		t.Fatal(err)
	}

	var commands [][]string
	manager := newSysctlManager(destination, func(_ context.Context, name string, args ...string) ([]byte, error) {
		command := append([]string{name}, args...)
		commands = append(commands, command)
		if name == "cp" && len(args) == 3 {
			content, err := os.ReadFile(args[1])
			if err != nil {
				return nil, err
			}
			if err := os.WriteFile(args[2], content, 0644); err != nil {
				return nil, err
			}
		}
		if name == "install" && len(args) >= 3 {
			content, err := os.ReadFile(args[len(args)-2])
			if err != nil {
				return nil, err
			}
			if err := os.WriteFile(args[len(args)-1], content, 0644); err != nil {
				return nil, err
			}
		}
		if name == "mv" && len(args) >= 3 {
			content, err := os.ReadFile(args[len(args)-2])
			if err != nil {
				return nil, err
			}
			if err := os.WriteFile(args[len(args)-1], content, 0644); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}, func() time.Time { return time.Unix(1, 0) }, false)

	diags, err := manager.Apply(context.Background(), []entity.SysctlSetting{{Key: "vm.swappiness", Value: "100"}}, false)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagOK || !strings.Contains(diags[0].Details, "backup") {
		t.Fatalf("diagnostics = %#v, want successful backup diagnostic", diags)
	}
	if len(commands) != 4 || commands[0][0] != "cp" || commands[1][0] != "install" || commands[2][0] != "mv" || commands[3][0] != "sysctl" {
		t.Fatalf("commands = %#v, want cp/install/mv/sysctl", commands)
	}
	content, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "vm.swappiness = 100") {
		t.Fatalf("applied content = %q", content)
	}
}

func TestSysctlManagerUsesNoninteractiveSudo(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "90-envctl-performance.conf")
	var commands [][]string
	manager := newSysctlManager(destination, func(_ context.Context, name string, args ...string) ([]byte, error) {
		commands = append(commands, append([]string{name}, args...))
		if name == "install" {
			content, err := os.ReadFile(args[len(args)-2])
			if err != nil {
				return nil, err
			}
			if err := os.WriteFile(args[len(args)-1], content, 0644); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}, func() time.Time { return time.Unix(1, 0) }, true)

	_, err := manager.Apply(context.Background(), []entity.SysctlSetting{{Key: "vm.swappiness", Value: "10"}}, false)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if len(commands) != 3 {
		t.Fatalf("commands = %#v, want install, mv, and sysctl", commands)
	}
	if commands[0][0] != "sudo" || commands[0][1] != "-n" || commands[0][2] != "install" {
		t.Fatalf("first command = %#v, want sudo -n install", commands[0])
	}
	if commands[1][0] != "sudo" || commands[1][1] != "-n" || commands[1][2] != "mv" {
		t.Fatalf("second command = %#v, want sudo -n mv", commands[1])
	}
	if commands[2][0] != "sudo" || commands[2][1] != "-n" || commands[2][2] != "sysctl" {
		t.Fatalf("third command = %#v, want sudo -n sysctl", commands[2])
	}
}

func TestSysctlManagerReportsApplyFailureAfterBackup(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "90-envctl-performance.conf")
	if err := os.WriteFile(destination, []byte("vm.swappiness = 10\n"), 0644); err != nil {
		t.Fatal(err)
	}
	manager := newSysctlManager(destination, func(_ context.Context, name string, args ...string) ([]byte, error) {
		switch name {
		case "cp":
			content, err := os.ReadFile(args[1])
			if err != nil {
				return nil, err
			}
			return nil, os.WriteFile(args[2], content, 0644)
		case "install":
			content, err := os.ReadFile(args[len(args)-2])
			if err != nil {
				return nil, err
			}
			return nil, os.WriteFile(args[len(args)-1], content, 0644)
		case "mv":
			content, err := os.ReadFile(args[len(args)-2])
			if err != nil {
				return nil, err
			}
			return nil, os.WriteFile(args[len(args)-1], content, 0644)
		case "sysctl":
			return nil, errors.New("synthetic sysctl failure")
		default:
			return nil, nil
		}
	}, func() time.Time { return time.Unix(1, 0) }, false)

	diags, err := manager.Apply(context.Background(), []entity.SysctlSetting{{Key: "vm.swappiness", Value: "100"}}, false)
	if err == nil {
		t.Fatal("expected sysctl apply failure")
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagError {
		t.Fatalf("diagnostics = %#v, want error", diags)
	}
	backups, globErr := filepath.Glob(destination + ".bak.*")
	if globErr != nil || len(backups) != 1 {
		t.Fatalf("backups = %#v, err=%v; want one recoverable backup", backups, globErr)
	}
}

func TestSysctlManagerIdenticalContentIsNoop(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "90-envctl-performance.conf")
	content := []byte("# Managed by envctl; review before editing.\nvm.swappiness = 10\n")
	if err := os.WriteFile(destination, content, 0644); err != nil {
		t.Fatal(err)
	}
	called := false
	manager := newSysctlManager(destination, func(context.Context, string, ...string) ([]byte, error) {
		called = true
		return nil, nil
	}, func() time.Time { return time.Unix(1, 0) }, false)

	diags, err := manager.Apply(context.Background(), []entity.SysctlSetting{{Key: "vm.swappiness", Value: "10"}}, false)
	if err != nil {
		t.Fatalf("no-op failed: %v", err)
	}
	if called {
		t.Fatal("identical content invoked a command")
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagOK || !strings.Contains(diags[0].Details, "already up to date") {
		t.Fatalf("diagnostics = %#v, want no-op diagnostic", diags)
	}
}
