package paru

import (
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func TestParuPackageManager_Type(t *testing.T) {
	pm := NewParuManager()
	if pm == nil {
		t.Fatal("Expected ParuManager to not be nil")
	}

	if pm.Type() != entity.PackageTypeParu {
		t.Errorf("Expected package manager type %s, got %s", entity.PackageTypeParu, pm.Type())
	}
}
