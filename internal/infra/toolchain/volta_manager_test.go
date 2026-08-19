package toolchain

import (
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func TestVoltaPackageManager_Type(t *testing.T) {
	vm := NewVoltaManager()
	if vm == nil {
		t.Fatal("Expected VoltaManager to not be nil")
	}

	if vm.Type() != entity.PackageTypeVolta {
		t.Errorf("Expected package manager type %s, got %s", entity.PackageTypeVolta, vm.Type())
	}
}
