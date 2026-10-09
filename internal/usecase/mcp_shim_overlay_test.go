package usecase

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eajdias/envctl"
	"github.com/eajdias/envctl/internal/infra/executil"
)

// winSep is two literal backslashes (the JSON encoding of one Windows path
// separator), built without typing a backslash pair so no transport layer
// can mangle the count. bs is one literal backslash for negative assertions.
var winSep = string([]byte{92, 92})
var bs = string([]byte{92})

// Fixtures mirror the real template fragments (configs/opencode.json and
// configs/commandcode/mcp.json). They are intentionally minimal: the overlay
// reasons about exact fragments, never about full-document shape.

const shimOverlayOpenCodeDoc = `{
  "mcp": {
    "servers": {
      "brave": {
        "type": "local",
        "command": [
          "bunx",
          "@brave/brave-search-mcp-server@2.1.4",
          "--transport",
          "stdio"
        ]
      },
      "chrome-devtools": {
        "type": "local",
        "command": [
          "bunx",
          "chrome-devtools-mcp@1.8.0",
          "--no-usage-statistics"
        ]
      },
      "ssh-manager": {
        "type": "local",
        "command": [
          "mcp-ssh-manager"
        ]
      }
    }
  }
}`

const shimOverlayCommandCodeDoc = `{
  "mcpServers": {
    "brave": {
      "command": "npx",
      "args": ["-y", "@brave/brave-search-mcp-server@2.1.4", "--transport", "stdio"]
    },
    "chrome-devtools": {
      "command": "bunx",
      "args": ["chrome-devtools-mcp@1.8.0", "--no-usage-statistics"]
    },
    "ssh-manager": {
      "command": "mcp-ssh-manager"
    }
  }
}`

func TestMCPShimOverlayOpenCode(t *testing.T) {
	patches := []mcpShimPatch{
		{
			old: `"bunx",` + "\n" + `          "@brave/brave-search-mcp-server@2.1.4",`,
			new: `"C:` + winSep + `shims` + winSep + `brave-search-mcp-server.exe",`,
		},
		{
			old: `"bunx",` + "\n" + `          "chrome-devtools-mcp@1.8.0",`,
			new: `"C:` + winSep + `shims` + winSep + `chrome-devtools-mcp.exe",`,
		},
		{
			old: `"mcp-ssh-manager"`,
			new: `"C:` + winSep + `shims` + winSep + `mcp-ssh-manager.exe"`,
		},
	}
	got := withMCPShimOverlay([]byte(shimOverlayOpenCodeDoc), patches)
	if !json.Valid(got) {
		t.Fatalf("overlay produced invalid JSON:\n%s", got)
	}
	for _, want := range []string{
		`"C:` + winSep + `shims` + winSep + `brave-search-mcp-server.exe",`,
		`"C:` + winSep + `shims` + winSep + `chrome-devtools-mcp.exe",`,
		`"C:` + winSep + `shims` + winSep + `mcp-ssh-manager.exe"`,
	} {
		if !strings.Contains(string(got), want) {
			t.Errorf("overlay output misses %q:\n%s", want, got)
		}
	}
	for _, gone := range []string{`"bunx"`, `"mcp-ssh-manager"`, "@brave/brave-search-mcp-server@2.1.4", "chrome-devtools-mcp@1.8.0"} {
		if strings.Contains(string(got), gone) {
			t.Errorf("overlay output still contains bare fragment %q:\n%s", gone, got)
		}
	}
}

func TestMCPShimOverlayCommandCode(t *testing.T) {
	patches := []mcpShimPatch{
		{
			old: `"command": "npx",` + "\n" + `      "args": ["-y", "@brave/brave-search-mcp-server@2.1.4", "--transport", "stdio"]`,
			new: `"command": "/home/u/.local/share/mise/shims/brave-search-mcp-server",` + "\n" + `      "args": ["--transport", "stdio"]`,
		},
		{
			old: `"command": "bunx",` + "\n" + `      "args": ["chrome-devtools-mcp@1.8.0", "--no-usage-statistics"]`,
			new: `"command": "/home/u/.local/share/mise/shims/chrome-devtools-mcp",` + "\n" + `      "args": ["--no-usage-statistics"]`,
		},
		{
			old: `"command": "mcp-ssh-manager"`,
			new: `"command": "/home/u/.local/share/mise/shims/mcp-ssh-manager"`,
		},
	}
	got := withMCPShimOverlay([]byte(shimOverlayCommandCodeDoc), patches)
	if !json.Valid(got) {
		t.Fatalf("overlay produced invalid JSON:\n%s", got)
	}
	for _, want := range []string{
		`"command": "/home/u/.local/share/mise/shims/brave-search-mcp-server"`,
		`"args": ["--transport", "stdio"]`,
		`"command": "/home/u/.local/share/mise/shims/chrome-devtools-mcp"`,
		`"args": ["--no-usage-statistics"]`,
		`"command": "/home/u/.local/share/mise/shims/mcp-ssh-manager"`,
	} {
		if !strings.Contains(string(got), want) {
			t.Errorf("overlay output misses %q:\n%s", want, got)
		}
	}
	if strings.Contains(string(got), `"command": "npx"`) || strings.Contains(string(got), `"command": "bunx"`) {
		t.Errorf("overlay output still contains a bare launcher:\n%s", got)
	}
}

func TestMCPShimOverlayIdempotent(t *testing.T) {
	patches := []mcpShimPatch{{old: `"mcp-ssh-manager"`, new: `"C:` + winSep + `shims` + winSep + `mcp-ssh-manager.exe"`}}
	once := withMCPShimOverlay([]byte(shimOverlayOpenCodeDoc), patches)
	twice := withMCPShimOverlay(once, patches)
	if string(once) != string(twice) {
		t.Errorf("overlay is not idempotent:\n--- once ---\n%s\n--- twice ---\n%s", once, twice)
	}
}

func TestMCPShimOverlayPassThrough(t *testing.T) {
	patches := []mcpShimPatch{{old: `"mcp-ssh-manager"`, new: `"C:` + winSep + `shims` + winSep + `mcp-ssh-manager.exe"`}}

	// No patches: byte-identical.
	if got := withMCPShimOverlay([]byte(shimOverlayOpenCodeDoc), nil); string(got) != shimOverlayOpenCodeDoc {
		t.Errorf("nil patches must pass through untouched:\n%s", got)
	}
	// Unknown fragment: byte-identical.
	unknown := []mcpShimPatch{{old: `"no-such-server"`, new: `"x"`}}
	if got := withMCPShimOverlay([]byte(shimOverlayOpenCodeDoc), unknown); string(got) != shimOverlayOpenCodeDoc {
		t.Errorf("unknown fragment must pass through untouched:\n%s", got)
	}
	// Invalid JSON: byte-identical.
	broken := "{ not json"
	if got := withMCPShimOverlay([]byte(broken), patches); string(got) != broken {
		t.Errorf("invalid JSON must pass through untouched:\n%s", got)
	}
	// Fragment occurring twice: that patch is skipped, document otherwise intact.
	// A single backslash can never appear in correct output (separators are
	// always JSON-escaped pairs), so its absence proves the skip.
	doubled := strings.Replace(shimOverlayOpenCodeDoc, `"mcp-ssh-manager"`, `"mcp-ssh-manager" "mcp-ssh-manager"`, 1)
	if got := withMCPShimOverlay([]byte(doubled), patches); strings.Contains(string(got), `"C:`+bs+`shims`) {
		t.Errorf("ambiguous (double) fragment must not be patched:\n%s", got)
	}
}

func TestMCPShimPatchesTable(t *testing.T) {
	// Fake shim dir with all three binaries present: full table, both flavors.
	dir := t.TempDir()
	for _, bin := range []string{"brave-search-mcp-server", "chrome-devtools-mcp", "mcp-ssh-manager"} {
		if err := os.WriteFile(filepath.Join(dir, bin), []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		configID string
		wantOld  []string
	}{
		{"opencode_config", []string{`"bunx",`, "chrome-devtools-mcp@1.8.0", `"mcp-ssh-manager"`}},
		{"opencode_config_linux", []string{`"bunx",`, "chrome-devtools-mcp@1.8.0", `"mcp-ssh-manager"`}},
		{"commandcode_mcp", []string{`"command": "npx"`, `"command": "bunx"`, `"command": "mcp-ssh-manager"`}},
		{"unrelated_id", nil},
	} {
		patches := mcpShimPatches(dir, "", tc.configID)
		if len(patches) != len(tc.wantOld) {
			t.Errorf("%s: got %d patches, want %d", tc.configID, len(patches), len(tc.wantOld))
			continue
		}
		for i, want := range tc.wantOld {
			if !strings.Contains(patches[i].old, want) {
				t.Errorf("%s patch %d: old fragment misses %q", tc.configID, i, want)
			}
			if strings.ContainsAny(patches[i].new, "\\") {
				t.Errorf("%s patch %d: replacement carries backslashes (must be forward-slash): %q", tc.configID, i, patches[i].new)
			}
		}
	}
	// Missing shim: that server contributes no patch, the rest still do.
	if err := os.Remove(filepath.Join(dir, "mcp-ssh-manager")); err != nil {
		t.Fatal(err)
	}
	if got := len(mcpShimPatches(dir, "", "opencode_config")); got != 2 {
		t.Errorf("with ssh shim absent: got %d patches, want 2", got)
	}
	// Windows flavor: .exe suffix resolves and deploys.
	wdir := t.TempDir()
	if err := os.WriteFile(filepath.Join(wdir, "mcp-ssh-manager.exe"), []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	wp := mcpShimPatches(wdir, ".exe", "commandcode_mcp")
	if len(wp) != 1 || !strings.Contains(wp[0].new, "mcp-ssh-manager.exe") {
		t.Errorf("windows flavor: got %+v, want single ssh patch with .exe", wp)
	}
}

func TestMCPShimOverlayOnShippedTemplates(t *testing.T) {
	// End-to-end against the real embedded templates (not hand-copied
	// fixtures): every bare launcher fragment the table knows must actually
	// occur in its template, and the deployed document must stay valid with
	// a native V2 shape. A template edit that renames a fragment (e.g. a
	// version bump) fails here until the table follows — by design.
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	dir := t.TempDir()
	for _, bin := range []string{"brave-search-mcp-server", "chrome-devtools-mcp", "mcp-ssh-manager"} {
		if err := os.WriteFile(filepath.Join(dir, bin+suffix), []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		template string
		configID string
		bare     []string
	}{
		{"configs/opencode.json", "opencode_config", []string{`"bunx"`, "chrome-devtools-mcp@1.8.0", `"mcp-ssh-manager"`}},
		{"configs/commandcode/mcp.json", "commandcode_mcp", []string{"@brave/brave-search-mcp-server@2.1.4", "chrome-devtools-mcp@1.8.0", `"command": "mcp-ssh-manager"`}},
	} {
		base, err := envctl.EmbeddedFS.ReadFile(tc.template)
		if err != nil {
			t.Fatalf("read %s: %v", tc.template, err)
		}
		deployed := withMCPShimOverlay(base, mcpShimPatches(dir, suffix, tc.configID))
		if !json.Valid(deployed) {
			t.Fatalf("%s: deployed config is not valid JSON", tc.template)
		}
		if string(deployed) == string(base) {
			t.Fatalf("%s: overlay changed nothing — table fragments drifted from the template", tc.template)
		}
		if !strings.Contains(string(deployed), filepath.ToSlash(dir)+"/") {
			t.Errorf("%s: deployed config carries no absolute shim path", tc.template)
		}
		for _, bare := range tc.bare {
			if strings.Contains(string(deployed), bare) {
				t.Errorf("%s: deployed config still contains bare fragment %q", tc.template, bare)
			}
		}
		if tc.configID == "opencode_config" {
			if problems := validateOpenCodeConfigShape(deployed); len(problems) > 0 {
				t.Errorf("deployed opencode config has shape problems: %s", strings.Join(problems, "; "))
			}
		}
		if again := withMCPShimOverlay(deployed, mcpShimPatches(dir, suffix, tc.configID)); string(again) != string(deployed) {
			t.Errorf("%s: overlay is not idempotent", tc.template)
		}
	}
}

func TestMCPShimOverlayNormalizesCRLFCheckout(t *testing.T) {
	// A Windows checkout carries CRLF (go:embed captures checkout bytes);
	// the overlay normalizes to LF like withWindowsShellOverlay/withSSHOSOverlay.
	crlf := strings.ReplaceAll(shimOverlayOpenCodeDoc, "\n", "\r\n")
	patches := []mcpShimPatch{{old: `"mcp-ssh-manager"`, new: `"C:` + winSep + `shims` + winSep + `mcp-ssh-manager.exe"`}}
	got := withMCPShimOverlay([]byte(crlf), patches)
	if strings.Contains(string(got), "\r") {
		t.Errorf("overlay output still carries CR")
	}
	if !json.Valid(got) {
		t.Fatalf("overlay on CRLF input produced invalid JSON:\n%s", got)
	}
	if !strings.Contains(string(got), `"C:`+winSep+`shims`+winSep+`mcp-ssh-manager.exe"`) {
		t.Errorf("overlay on CRLF input did not patch:\n%s", got)
	}
}

// TestDeployedMCPConfigParity locks the shared helper both call sites
// (provisioning and the doctor drift comparison) must use: end-to-end over
// the shipped templates with a fake HOME, so a one-sided edit is a test
// failure instead of a permanent drift WARN.
func TestDeployedMCPConfigParity(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("LOCALAPPDATA", home)
	shimDir := executil.MiseShimDir(home)
	if err := os.MkdirAll(shimDir, 0o755); err != nil {
		t.Fatal(err)
	}
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	for _, bin := range []string{"brave-search-mcp-server", "chrome-devtools-mcp", "mcp-ssh-manager"} {
		if err := os.WriteFile(filepath.Join(shimDir, bin+suffix), []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		template string
		configID string
	}{
		{"configs/opencode.json", "opencode_config"},
		{"configs/opencode.json", "opencode_config_linux"},
		{"configs/commandcode/mcp.json", "commandcode_mcp"},
	} {
		base, err := envctl.EmbeddedFS.ReadFile(tc.template)
		if err != nil {
			t.Fatalf("read %s: %v", tc.template, err)
		}
		got := deployedMCPConfig(base, tc.configID)
		if !json.Valid(got) {
			t.Fatalf("%s/%s: helper output is not valid JSON", tc.template, tc.configID)
		}
		if string(got) == string(base) {
			t.Fatalf("%s/%s: helper changed nothing", tc.template, tc.configID)
		}
		if !strings.Contains(string(got), filepath.ToSlash(shimDir)+"/") {
			t.Errorf("%s/%s: helper output carries no shim path from the fake HOME", tc.template, tc.configID)
		}
		if again := deployedMCPConfig(got, tc.configID); string(again) != string(got) {
			t.Errorf("%s/%s: helper is not idempotent", tc.template, tc.configID)
		}
	}
}
