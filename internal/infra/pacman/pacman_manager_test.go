package pacman

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
