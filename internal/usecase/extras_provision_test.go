package usecase

import (
	"context"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

// TestExecuteExtras_RoutesByPlatform proves the extras provisioner installs
// winget entries on Windows and pacman entries on Arch/CachyOS through the
// same per-type managers the core packages use — the manifest, not the code,
// decides what each OS gets.
func TestExecuteExtras_RoutesByPlatform(t *testing.T) {
	extras := []entity.Package{
		{ID: "Brave.Brave", Type: entity.PackageTypeWinget, Category: "extras"},
		{ID: "brave-origin-bin", Type: entity.PackageTypePacman, Category: "extras"},
	}
	managers := map[entity.PackageType]repository.PackageManager{
		entity.PackageTypeWinget: &mockManager{available: true, installed: map[string]bool{}},
		entity.PackageTypePacman: &mockManager{available: true, installed: map[string]bool{}},
	}
	uc := NewProvisionPackagesUseCase(&mockManifestRepo{extrasPkgs: extras}, managers, &mockLogger{})

	var processed []string
	_, err := uc.ExecuteExtras(context.Background(), func(pkg entity.Package, status string, err error) {
		processed = append(processed, string(pkg.Type))
	})
	if err != nil {
		t.Fatalf("ExecuteExtras failed: %v", err)
	}
	got := map[string]bool{}
	for _, p := range processed {
		got[p] = true
	}
	if !got["winget"] {
		t.Errorf("expected a winget extra to be processed, got %v", processed)
	}
	if !got["pacman"] {
		t.Errorf("expected a pacman extra to be processed, got %v", processed)
	}
}

// mockManager is a minimal PackageManager that reports everything installed
// so the test exercises routing without launching real package managers.
type mockManager struct {
	available bool
	installed map[string]bool
}

func (m *mockManager) Type() entity.PackageType             { return "" }
func (m *mockManager) IsAvailable(ctx context.Context) bool { return m.available }
func (m *mockManager) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
	return m.installed[pkg.ID], "mock", nil
}
func (m *mockManager) Install(ctx context.Context, pkg entity.Package) error { return nil }
func (m *mockManager) ListInstalled(ctx context.Context) ([]entity.Package, error) {
	return nil, nil
}
