package embedded

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/eajdias/envctl"
)

// TestShellManifestDeclaresEmulatorConfigs pins the seed templates in
// shell.yaml to the exact keys the doctor audit table (gamingEmulatorConfigs
// in internal/usecase/doctor_audit.go) verifies. A template whose key drifts
// from the audit would provision a config the doctor then reports as missing
// forever — this test makes that a build-time failure instead.
func TestShellManifestDeclaresEmulatorConfigs(t *testing.T) {
	repo := NewManifestRepository(envctl.EmbeddedFS, ".")
	configFiles, err := repo.LoadConfigFiles()
	if err != nil {
		t.Fatalf("failed to load config files: %v", err)
	}

	// destination -> required key fragment, mirroring gamingEmulatorConfigs.
	want := map[string]string{
		"~/.config/dolphin-emu/Dolphin.ini":      "GFXBackend = Vulkan",
		"~/.config/retroarch/retroarch.cfg":      `video_driver = "vulkan"`,
		"~/.config/ppsspp/PSP/SYSTEM/ppsspp.ini": "GraphicsBackend = 3",
		"~/.config/PCSX2/inis/PCSX2.ini":         "Renderer = 14",
		"~/.config/duckstation/settings.ini":     "Renderer = Vulkan",
		"~/.config/azahar-emu/qt-config.ini":     "graphics_api=2",
		"~/.config/eden/qt-config.ini":           "backend=1",
		"~/.config/Vita3K/config.yml":            "backend-renderer: Vulkan",
		"~/.config/Cemu/settings.xml":            "<api>1</api>",
	}

	seeded := make(map[string]string)
	for _, cf := range configFiles {
		if cf.Category != "gaming" || cf.Source == "configs/gaming.conf" || cf.Source == "configs/MangoHud.conf" {
			continue
		}
		if !cf.SeedIfMissing {
			t.Errorf("emulator config %s must be seed_if_missing (user edits must win)", cf.ID)
		}
		seeded[cf.Destination] = cf.Source
	}

	for dest, key := range want {
		src, ok := seeded[dest]
		if !ok {
			t.Errorf("shell.yaml has no seed for %s (required by the doctor audit)", dest)
			continue
		}
		content, err := fs.ReadFile(envctl.EmbeddedFS, src)
		if err != nil {
			t.Fatalf("cannot read embedded template %s: %v", src, err)
		}
		if !strings.Contains(string(content), key) {
			t.Errorf("template %s (%s) lacks the audited key %q", src, dest, key)
		}
	}
}