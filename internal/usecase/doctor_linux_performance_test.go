package usecase

import (
	"context"
	"runtime"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
)

type performanceInspectorStub struct {
	snapshot entity.PerformanceSnapshot
}

func (m performanceInspectorStub) Snapshot(context.Context) entity.PerformanceSnapshot {
	return m.snapshot
}

func TestAuditLinuxPerformanceDoesNotWarnOnOptionalState(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux performance audit is Linux-only")
	}

	uc := &DoctorAuditUseCase{
		performanceInspector: performanceInspectorStub{snapshot: entity.PerformanceSnapshot{
			Swap:            nil,
			ZRAM:            entity.ZRAMState{},
			CPUGovernors:    []string{"schedutil"},
			BlockSchedulers: []entity.BlockScheduler{{Device: "nvme0n1", Selected: "kyber"}},
		}},
	}
	var diagnostics []entity.Diagnostic
	uc.auditLinuxPerformance(context.Background(), func(diagnostic entity.Diagnostic) {
		diagnostics = append(diagnostics, diagnostic)
	})

	if len(diagnostics) == 0 {
		t.Fatal("expected performance diagnostics")
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Category == entity.DiagWarning || diagnostic.Category == entity.DiagError {
			t.Fatalf("optional performance state produced %s: %s", diagnostic.Category, diagnostic.Details)
		}
	}
}

func TestAuditLinuxPerformanceReportsActiveZRAM(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux performance audit is Linux-only")
	}

	uc := &DoctorAuditUseCase{
		performanceInspector: performanceInspectorStub{snapshot: entity.PerformanceSnapshot{
			ZRAM: entity.ZRAMState{Present: true, Name: "/dev/zram0", Algorithm: "zstd", SizeBytes: 22_400_000_000},
		}},
	}
	var diagnostics []entity.Diagnostic
	uc.auditLinuxPerformance(context.Background(), func(diagnostic entity.Diagnostic) {
		diagnostics = append(diagnostics, diagnostic)
	})

	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Target == "zram" && diagnostic.Category == entity.DiagOK {
			found = true
		}
	}
	if !found {
		t.Fatalf("active zram was not reported as OK: %#v", diagnostics)
	}
}
