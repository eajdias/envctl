package performance

import (
	"context"
	"errors"
	"testing"
)

func TestZRAMManagerStartsGeneratorWithNoninteractiveSudo(t *testing.T) {
	deviceChecks := 0
	var commands [][]string
	manager := newZRAMManager(
		func() bool {
			deviceChecks++
			return deviceChecks > 1
		},
		func(_ context.Context, name string, args ...string) ([]byte, error) {
			commands = append(commands, append([]string{name}, args...))
			return nil, nil
		},
		true,
		func(context.Context, func() bool) bool { return true },
	)

	diags, err := manager.Ensure(context.Background(), false)
	if err != nil {
		t.Fatalf("zram ensure failed: %v", err)
	}
	if len(diags) != 1 || diags[0].Category != "OK" {
		t.Fatalf("diagnostics = %#v, want OK", diags)
	}
	if len(commands) != 3 || commands[0][0] != "sudo" || commands[0][1] != "-n" || commands[0][2] != "modprobe" || commands[0][3] != "zram" {
		t.Fatalf("module commands = %#v, want sudo -n modprobe zram", commands)
	}
	if commands[1][0] != "sudo" || commands[1][1] != "-n" || commands[2][0] != "sudo" || commands[2][1] != "-n" {
		t.Fatalf("systemd commands = %#v, want sudo -n systemctl", commands)
	}
	if commands[1][2] != "systemctl" || commands[1][3] != "daemon-reload" {
		t.Fatalf("reload command = %#v, want sudo -n systemctl daemon-reload", commands[1])
	}
	if commands[2][2] != "systemctl" || commands[2][3] != "start" || commands[2][4] != zramService {
		t.Fatalf("service command = %#v, want sudo -n systemctl start %s", commands[2], zramService)
	}
}

func TestProcSwapsContainsZRAM0(t *testing.T) {
	if !procSwapsContainsZRAM0("Filename Type Size Used Priority\n/dev/zram0 partition 1024 0 100\n") {
		t.Fatal("expected /dev/zram0 in proc swaps to be active")
	}
	if procSwapsContainsZRAM0("Filename Type Size Used Priority\n/dev/zram1 partition 1024 0 100\n") {
		t.Fatal("did not expect /dev/zram1 to satisfy the zram0 check")
	}
}

func TestZRAMManagerDoesNotTrustDeviceNodeWithoutSwap(t *testing.T) {
	called := false
	manager := newZRAMManager(
		func() bool { return false },
		func(context.Context, string, ...string) ([]byte, error) {
			called = true
			return nil, nil
		},
		false,
		func(context.Context, func() bool) bool { return true },
		func() []string { return []string{"/dev/zram0", "/sys/block/zram0"} },
	)

	if _, err := manager.Ensure(context.Background(), false); err != nil {
		t.Fatalf("zram ensure failed: %v", err)
	}
	if !called {
		t.Fatal("device node without a /proc/swaps entry was treated as active")
	}
}

func TestZRAMManagerReturnsModprobeFailure(t *testing.T) {
	called := false
	manager := newZRAMManager(
		func() bool { return false },
		func(_ context.Context, name string, _ ...string) ([]byte, error) {
			called = true
			if name == "modprobe" {
				return nil, errors.New("module unavailable")
			}
			return nil, nil
		},
		false,
		func(context.Context, func() bool) bool { return false },
	)

	_, err := manager.Ensure(context.Background(), false)
	if err == nil {
		t.Fatal("expected modprobe failure")
	}
	if !called {
		t.Fatal("manager did not attempt modprobe")
	}
}

func TestZRAMManagerReturnsServiceFailure(t *testing.T) {
	manager := newZRAMManager(
		func() bool { return false },
		func(_ context.Context, name string, _ ...string) ([]byte, error) {
			if name == "systemctl" {
				return nil, errors.New("service failed")
			}
			return nil, nil
		},
		false,
		func(context.Context, func() bool) bool { return true },
	)

	_, err := manager.Ensure(context.Background(), false)
	if err == nil {
		t.Fatal("expected service failure")
	}
}

func TestZRAMManagerRefusesSecondDevice(t *testing.T) {
	called := false
	manager := newZRAMManager(
		func() bool { return false },
		func(context.Context, string, ...string) ([]byte, error) {
			called = true
			return nil, nil
		},
		false,
		func(context.Context, func() bool) bool { return false },
		func() []string { return []string{"/dev/zram1", "/sys/block/zram1"} },
	)

	diags, err := manager.Ensure(context.Background(), false)
	if err != nil {
		t.Fatalf("existing zram1 should not be an error: %v", err)
	}
	if called {
		t.Fatal("manager attempted to create a second zram device")
	}
	if len(diags) != 1 || diags[0].Category != "INFO" {
		t.Fatalf("diagnostics = %#v, want informational refusal", diags)
	}
}

func TestZRAMManagerDryRunDoesNotStart(t *testing.T) {
	called := false
	manager := newZRAMManager(
		func() bool { return false },
		func(context.Context, string, ...string) ([]byte, error) {
			called = true
			return nil, nil
		},
		false,
		func(context.Context, func() bool) bool { return false },
	)

	diags, err := manager.Ensure(context.Background(), true)
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}
	if called {
		t.Fatal("dry-run invoked systemctl")
	}
	if len(diags) != 1 || diags[0].Category != "INFO" {
		t.Fatalf("diagnostics = %#v, want INFO", diags)
	}
}
