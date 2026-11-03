package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/infra/windows"
)

// specsProbeTimeout caps the whole specs section. WMI/CIM is cold-start
// sensitive; a stuck probe must stall neither the doctor run nor the table.
const specsProbeTimeout = 60 * time.Second

// auditSystemSpecs reports the Windows hardware inventory as DiagInfo: the
// collector is an inventory "without judgment" — absence is context, never
// drift, so every row is Info and a probe failure stays informational too.
// Non-Windows hosts are silently skipped (the collector is Windows-only for
// now; Linux parity is a future task, not a per-run INFO).
func (uc *DoctorAuditUseCase) auditSystemSpecs(ctx context.Context, addDiag func(entity.Diagnostic)) {
	platform := entity.DetectedPlatform()
	if uc.platform != nil {
		platform = uc.platform()
	}
	if platform.GOOS != "windows" {
		return
	}

	probe := uc.specsProbe
	if probe == nil {
		probe = windows.SpecsProbe
	}

	probeCtx, cancel := context.WithTimeout(ctx, specsProbeTimeout)
	defer cancel()
	rows, err := probe(probeCtx)
	if err != nil {
		addDiag(entity.Info(
			"Hardware",
			"specs-probe",
			fmt.Sprintf("specs probe failed: %v; the inventory is unavailable this run", err),
			"run 'envctl doctor' again; the log has the raw PowerShell output",
		))
		return
	}

	for _, field := range windows.SpecsFields {
		details := strings.ReplaceAll(rows[field], ";", "; ")
		if details == "" {
			details = "not detected"
		}
		addDiag(entity.Info("Hardware", field, details))
	}
}
