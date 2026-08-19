package apt

import (
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func TestAptPackageManager_Type(t *testing.T) {
	am := NewAptManager()
	if am == nil {
		t.Fatal("Expected AptManager to not be nil")
	}

	if am.Type() != entity.PackageTypeApt {
		t.Errorf("Expected package manager type %s, got %s", entity.PackageTypeApt, am.Type())
	}
}
