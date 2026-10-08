package embedded

import (
	"testing"
	"testing/fstest"
)

const walkTestSkillsYAML = `skills:
  - name: demo
    description: Demo skill
    source: configs/skills/demo
    target_dir: ""
    enabled: true
  - name: off
    description: Disabled skill
    source: configs/skills/off
    target_dir: ""
    enabled: false
  - name: winonly
    description: Windows-only skill
    source: configs/skills/winonly
    target_dir: ""
    enabled: true
    os: windows
`

func walkTestFS() fstest.MapFS {
	return fstest.MapFS{
		"manifests/skills.yaml":               {Data: []byte(walkTestSkillsYAML)},
		"configs/skills/demo/SKILL.md":        {Data: []byte("# demo\n")},
		"configs/skills/demo/references/r.md": {Data: []byte("ref\n")},
		"configs/skills/off/SKILL.md":         {Data: []byte("# off\n")},
		"configs/skills/winonly/SKILL.md":     {Data: []byte("# win\n")},
	}
}

func collectWalk(t *testing.T, repo *ManifestRepository, goos string) []string {
	t.Helper()
	var got []string
	err := repo.WalkSkillSources(goos, func(skill, rel string, data []byte) error {
		got = append(got, skill+"/"+rel)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkSkillSources: %v", err)
	}
	return got
}

func TestWalkSkillSourcesIsStableAndSorted(t *testing.T) {
	repo := NewManifestRepository(walkTestFS(), "")
	first := collectWalk(t, repo, "windows")
	if len(first) != 3 {
		t.Fatalf("walk = %v, want 3 files (demo x2 + winonly)", first)
	}
	second := collectWalk(t, repo, "windows")
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("unstable order: %v vs %v", first, second)
		}
	}
}

func TestWalkSkillSourcesSeesScope(t *testing.T) {
	repo := NewManifestRepository(walkTestFS(), "")
	// Disabled skills never walk; winonly ships on windows, not on linux.
	if got := collectWalk(t, repo, "windows"); len(got) != 3 {
		t.Errorf("windows walk = %v, want 3 files", got)
	}
	if got := collectWalk(t, repo, "linux"); len(got) != 2 {
		t.Errorf("linux walk = %v, want demo x2 only", got)
	}
	for _, f := range collectWalk(t, repo, "linux") {
		if len(f) >= 4 && f[:4] == "off/" {
			t.Errorf("disabled skill leaked into walk: %v", f)
		}
	}
}
