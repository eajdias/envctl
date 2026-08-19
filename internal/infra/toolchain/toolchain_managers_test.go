package toolchain

import (
	"context"
	"testing"

	"github.com/eajdias/win11-new/internal/domain/entity"
)

func TestGoManager(t *testing.T) {
	mgr := NewGoManager()
	if mgr.Type() != entity.PackageTypeGo {
		t.Errorf("expected type %s, got %s", entity.PackageTypeGo, mgr.Type())
	}

	ctx := context.Background()
	if !mgr.IsAvailable(ctx) {
		t.Log("Go is not installed on this machine, skipping live check")
		return
	}

	pkg := entity.Package{
		ID:           "golang.org/x/tools/gopls@latest",
		Name:         "gopls",
		Type:         entity.PackageTypeGo,
		CheckCommand: "gopls version",
	}

	installed, info, err := mgr.IsInstalled(ctx, pkg)
	if err != nil {
		t.Errorf("unexpected error checking gopls: %v", err)
	}
	t.Logf("gopls installed: %v (info: %s)", installed, info)
}

func TestRustupManager(t *testing.T) {
	mgr := NewRustupManager()
	if mgr.Type() != entity.PackageTypeRustup {
		t.Errorf("expected type %s, got %s", entity.PackageTypeRustup, mgr.Type())
	}

	ctx := context.Background()
	if !mgr.IsAvailable(ctx) {
		t.Log("Rustup is not installed on this machine, skipping live check")
		return
	}

	pkg := entity.Package{
		ID:   "rust-analyzer",
		Type: entity.PackageTypeRustup,
	}

	installed, info, err := mgr.IsInstalled(ctx, pkg)
	if err != nil {
		t.Errorf("unexpected error checking rust-analyzer: %v", err)
	}
	t.Logf("rust-analyzer installed: %v (info: %s)", installed, info)
}
