package embedded

import (
	"strings"
	"testing"

	"github.com/eajdias/envctl"
)

// TestLoadExtrasManifest pins the optional apps manifest: exactly the owner's
// confirmed set (15 winget + 12 pacman), unique IDs, correct types, and zero
// personal/business apps (the repo is public).
func TestLoadExtrasManifest(t *testing.T) {
	repo := NewManifestRepository(envctl.EmbeddedFS, ".")
	pkgs, err := repo.LoadExtrasPackages()
	if err != nil {
		t.Fatalf("failed to load extras manifest: %v", err)
	}

	want := map[string]string{
		// Windows (winget)
		"Brave.Brave":                             "winget",
		"Obsidian.Obsidian":                       "winget",
		"Valve.Steam":                             "winget",
		"Tailscale.Tailscale":                     "winget",
		"VideoLAN.VLC":                            "winget",
		"ONLYOFFICE.DesktopEditors":               "winget",
		"BillStewart.SyncthingWindowsSetup":       "winget",
		"MoonlightGameStreamingProject.Moonlight": "winget",
		"WinSCP.WinSCP":                           "winget",
		"WiresharkFoundation.Wireshark":           "winget",
		"Insecure.Nmap":                           "winget",
		"Termius.Termius":                         "winget",
		"JAMSoftware.TreeSize.Free":               "winget",
		"Klocman.BulkCrapUninstaller":             "winget",
		"CrystalRich.LockHunter":                  "winget",
		// Arch/CachyOS (pacman)
		"brave-origin-bin": "pacman",
		"obsidian":         "pacman",
		"onlyoffice-bin":   "pacman",
		"vlc":              "pacman",
		"transmission-qt":  "pacman",
		"picard":           "pacman",
		"rustdesk-bin":     "pacman",
		"anydesk-bin":      "pacman",
		"tailscale":        "pacman",
		"boosteroid":       "pacman",
		"alacritty":        "pacman",
		"mpv":              "pacman",
	}
	if len(pkgs) != len(want) {
		t.Fatalf("expected %d extras packages, got %d", len(want), len(pkgs))
	}

	seen := map[string]bool{}
	for _, pkg := range pkgs {
		if seen[pkg.ID] {
			t.Errorf("duplicate extras package id %q", pkg.ID)
		}
		seen[pkg.ID] = true
		wantType, ok := want[pkg.ID]
		if !ok {
			t.Errorf("unexpected extras package %q (not in the owner's confirmed set)", pkg.ID)
			continue
		}
		if string(pkg.Type) != wantType {
			t.Errorf("extras package %q: type %q, want %q", pkg.ID, pkg.Type, wantType)
		}
		if pkg.Category != "extras" {
			t.Errorf("extras package %q: category %q, want %q", pkg.ID, pkg.Category, "extras")
		}
		if pkg.Name == "" {
			t.Errorf("extras package %q must carry a descriptive name", pkg.ID)
		}
	}

	// Public-repo hygiene: no business/personal apps may sneak in. RustDesk is
	// legitimate on CachyOS (pacman); only the Windows winget entry is absent
	// by design (no winget package), documented in the manifest header.
	forbidden := []string{"zscan", "whatsapp", "discord", "spotify", "path of exile", "npcap"}
	for _, pkg := range pkgs {
		lower := strings.ToLower(pkg.ID + " " + pkg.Name)
		for _, f := range forbidden {
			if strings.Contains(lower, f) {
				t.Errorf("extras package %q mentions forbidden app %q (public repo, zero PII)", pkg.ID, f)
			}
		}
	}
}
