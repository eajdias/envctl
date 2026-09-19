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

func TestVoltaListContains(t *testing.T) {
	// Project-pinned output: the "(current @ /path/package.json)" line
	// contains a standalone "@" token. A scoped ID must never match it
	// (regression: Split(id, "@")[0] is "" for scoped IDs and matched
	// HasPrefix(token, "@"), so `run volta` skipped installing
	// @playwright/cli while doctor still reported it missing).
	projectPinned := `⚡️ Currently active tools:
    runtime node@24.20.0 (default)
    package axios@project / / node@project npm@project (current @ /home/user/package.json)
    package playwright@1.63.0 / playwright / node@24.20.0 npm@built-in (default)`
	withScoped := projectPinned + `
    package @playwright/cli@0.1.21 / playwright-cli / node@24.20.0 npm@built-in (default)`
	noProjectPin := `⚡️ Currently active tools:
    runtime node@24.20.0 (default)
    package playwright@1.63.0 / playwright / node@24.20.0 npm@built-in (default)`

	tests := []struct {
		name  string
		out   string
		pkgID string
		want  bool
	}{
		{"scoped missing with project pin", projectPinned, "@playwright/cli", false},
		{"scoped installed", withScoped, "@playwright/cli", true},
		{"scoped must not match unscoped sibling", noProjectPin, "@playwright/cli", false},
		{"unscoped sibling present", projectPinned, "playwright", true},
		{"unscoped with version qualifier", projectPinned, "node@24.19.0", true},
		{"unscoped plain", projectPinned, "pnpm", false},
		{"empty list", "", "@playwright/cli", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := voltaListContains(tt.out, tt.pkgID)
			if got != tt.want {
				t.Errorf("voltaListContains(out, %q) = %v, want %v", tt.pkgID, got, tt.want)
			}
		})
	}
}
