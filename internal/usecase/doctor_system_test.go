package usecase

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/infra/embedded"
	"github.com/eajdias/envctl/internal/infra/filesystem"
)

func TestPathListsDir(t *testing.T) {
	winShim := `C:\Users\u\AppData\Local\mise\shims`
	tests := []struct {
		name     string
		pathVal  string
		sep      byte
		dir      string
		foldCase bool
		want     bool
	}{
		{"windows present", `C:\Windows;` + winShim + `;C:\Go\bin`, ';', winShim, true, true},
		{"windows present different case", `C:\Windows;c:\users\u\appdata\local\mise\shims`, ';', winShim, true, true},
		{"windows absent", `C:\Windows;C:\Go\bin`, ';', winShim, true, false},
		{"windows prefix is not a hit", winShim + `-old;C:\Windows`, ';', winShim, true, false},
		{"posix present", "/usr/bin:/home/u/.local/share/mise/shims:/bin", ':', "/home/u/.local/share/mise/shims", false, true},
		{"posix absent", "/usr/bin:/bin", ':', "/home/u/.local/share/mise/shims", false, false},
		{"empty path", "", ':', "/home/u/.local/share/mise/shims", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pathListsDir(tt.pathVal, tt.sep, tt.dir, tt.foldCase); got != tt.want {
				t.Errorf("pathListsDir(%q, %q, %q, fold=%v) = %v, want %v",
					tt.pathVal, tt.sep, tt.dir, tt.foldCase, got, tt.want)
			}
		})
	}
}

// Fixtures from live `py -0p` (Windows launcher): the `*` marks the default.
func TestPyDefaultIsFreeThreaded(t *testing.T) {
	healthy := " -V:3.14t         C:\\Program Files\\Python314\\python3.14t.exe\n -V:3.14 *        C:\\Program Files\\Python314\\python.exe\n"
	broken := " -V:3.14t *        C:\\Program Files\\Python314\\python3.14t.exe\n -V:3.14          C:\\Program Files\\Python314\\python.exe\n"
	tests := []struct {
		name string
		out  string
		want bool
	}{
		{"regular default", healthy, false},
		{"free-threaded default", broken, true},
		{"no default marker", " -V:3.14  C:\\Python314\\python.exe\n", false},
		{"empty output", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pyDefaultIsFreeThreaded(tt.out); got != tt.want {
				t.Errorf("pyDefaultIsFreeThreaded(%q) = %v, want %v", tt.out, got, tt.want)
			}
		})
	}
}

// stampFixture is the smallest skill source set a deploy stamp can cover,
// returned both as a filesystem (for provisioning) and as a manifest repo
// (for hashing and auditing) so both sides read identical bytes.
func stampFixture() (fstest.MapFS, *embedded.ManifestRepository) {
	fsys := fstest.MapFS{
		"manifests/skills.yaml": {Data: []byte(`skills:
  - name: demo
    description: Demo skill
    source: configs/skills/demo
    target_dir: ""
    enabled: true
`)},
		"configs/skills/demo/SKILL.md": {Data: []byte("# demo\n")},
	}
	return fsys, embedded.NewManifestRepository(fsys, "")
}

func collectDiags(f func(addDiag func(entity.Diagnostic))) []entity.Diagnostic {
	var diags []entity.Diagnostic
	f(func(d entity.Diagnostic) { diags = append(diags, d) })
	return diags
}

func TestAuditSkillsContentParity(t *testing.T) {
	_, repo := stampFixture()
	newUC := func() *DoctorAuditUseCase {
		return &DoctorAuditUseCase{manifestRepo: repo, fsManager: filesystem.NewFileSystemManager()}
	}
	deploy := func(t *testing.T, dir string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(dir, "demo"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "demo", "SKILL.md"), []byte("# demo\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("matching tree passes", func(t *testing.T) {
		dir := t.TempDir()
		deploy(t, dir)
		diags := collectDiags(func(add func(entity.Diagnostic)) {
			newUC().auditSkillsContentParity("Skills", dir, add)
		})
		if len(diags) != 1 || diags[0].Category != entity.DiagOK {
			t.Fatalf("diags = %+v, want one OK", diags)
		}
	})

	t.Run("tampered file warns and names it", func(t *testing.T) {
		dir := t.TempDir()
		deploy(t, dir)
		if err := os.WriteFile(filepath.Join(dir, "demo", "SKILL.md"), []byte("# demo!\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		diags := collectDiags(func(add func(entity.Diagnostic)) {
			newUC().auditSkillsContentParity("Skills", dir, add)
		})
		if len(diags) != 1 || diags[0].Category != entity.DiagWarning {
			t.Fatalf("diags = %+v, want one WARNING", diags)
		}
		if !strings.Contains(diags[0].Details, "demo/SKILL.md") {
			t.Errorf("Details = %q, want the file named", diags[0].Details)
		}
	})

	t.Run("missing file warns", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "demo"), 0o755); err != nil {
			t.Fatal(err)
		}
		diags := collectDiags(func(add func(entity.Diagnostic)) {
			newUC().auditSkillsContentParity("Skills", dir, add)
		})
		if len(diags) != 1 || diags[0].Category != entity.DiagWarning {
			t.Fatalf("diags = %+v, want one WARNING", diags)
		}
	})

	t.Run("missing dir is silent", func(t *testing.T) {
		diags := collectDiags(func(add func(entity.Diagnostic)) {
			newUC().auditSkillsContentParity("Skills", filepath.Join(t.TempDir(), "nope"), add)
		})
		if len(diags) != 0 {
			t.Fatalf("diags = %+v, want silence (tree check owns it)", diags)
		}
	})

	t.Run("unmanaged files are ignored", func(t *testing.T) {
		dir := t.TempDir()
		deploy(t, dir)
		for _, extra := range []string{".envctl-deploy.hash", "SKILL.md.bak.20261008-120000", "notes.txt"} {
			if err := os.WriteFile(filepath.Join(dir, "demo", extra), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		diags := collectDiags(func(add func(entity.Diagnostic)) {
			newUC().auditSkillsContentParity("Skills", dir, add)
		})
		if len(diags) != 1 || diags[0].Category != entity.DiagOK {
			t.Fatalf("diags = %+v, want one OK", diags)
		}
	})
}
