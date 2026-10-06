package arch

import (
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func TestPacmanPackageManager_Type(t *testing.T) {
	pm := NewPacmanManager()
	if pm == nil {
		t.Fatal("Expected PacmanManager to not be nil")
	}

	if pm.Type() != entity.PackageTypePacman {
		t.Errorf("Expected package manager type %s, got %s", entity.PackageTypePacman, pm.Type())
	}
}

func TestParuPackageManager_Type(t *testing.T) {
	pm := NewParuManager()
	if pm == nil {
		t.Fatal("Expected ParuManager to not be nil")
	}

	if pm.Type() != entity.PackageTypeParu {
		t.Errorf("Expected package manager type %s, got %s", entity.PackageTypeParu, pm.Type())
	}
}
