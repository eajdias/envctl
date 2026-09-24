package usecase

import (
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func TestPackageOwnershipProbe(t *testing.T) {
	openCode := packageOwnershipProbe(entity.Package{
		ID:           "opencode",
		Type:         entity.PackageTypePacman,
		CheckCommand: "opencode --version",
	})
	if openCode.CheckCommand != "" {
		t.Fatalf("opencode probe retained check command %q", openCode.CheckCommand)
	}

	other := packageOwnershipProbe(entity.Package{
		ID:           "steam",
		Type:         entity.PackageTypePacman,
		CheckCommand: "steam --version",
	})
	if other.CheckCommand != "steam --version" {
		t.Fatalf("unrelated package probe changed check command to %q", other.CheckCommand)
	}
}
