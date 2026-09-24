package entity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// repoRootDir resolves the repository root by searching upward for go.mod.
// Test working dir is the package dir, so CWD assumptions would break.
func repoRootDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd failed: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found searching upward from package dir")
		}
		dir = parent
	}
}

func collectStringValues(node any, key string, out *[]string) {
	switch v := node.(type) {
	case map[string]any:
		for k, val := range v {
			if k == key {
				if s, ok := val.(string); ok {
					*out = append(*out, s)
				}
			}
			collectStringValues(val, key, out)
		}
	case map[any]any:
		for k, val := range v {
			if ks, ok := k.(string); ok && ks == key {
				if s, ok := val.(string); ok {
					*out = append(*out, s)
				}
			}
			collectStringValues(val, key, out)
		}
	case []any:
		for _, item := range v {
			collectStringValues(item, key, out)
		}
	}
}

func splitOSTokens(filter string) []string {
	return strings.FieldsFunc(strings.ToLower(filter), func(r rune) bool {
		return r == ',' || r == ' ' || r == '|'
	})
}

// TestManifestOSLint bans bare `os: linux` and quoted check_command payloads.
func TestManifestOSLint(t *testing.T) {
	root := repoRootDir(t)
	manifests, err := filepath.Glob(filepath.Join(root, "manifests", "*.yaml"))
	if err != nil {
		t.Fatalf("glob manifests failed: %v", err)
	}
	if len(manifests) == 0 {
		t.Fatalf("no manifests found under %s", filepath.Join(root, "manifests"))
	}

	allowed := map[string]bool{
		"windows": true, "linux": true, "darwin": true,
		"arch": true, "archlinux": true, "cachyos": true,
		"debian": true, "ubuntu": true,
	}

	for _, mf := range manifests {
		data, err := os.ReadFile(mf)
		if err != nil {
			t.Fatalf("failed to read %s: %v", mf, err)
		}
		var doc any
		if err := yaml.Unmarshal(data, &doc); err != nil {
			t.Fatalf("failed to parse %s: %v", mf, err)
		}

		var osValues []string
		collectStringValues(doc, "os", &osValues)
		for _, filter := range osValues {
			for _, token := range splitOSTokens(filter) {
				if token == "" {
					continue
				}
				if token == "linux" {
					t.Errorf("%s: bare `os: linux` is banned (use arch,cachyos / debian,ubuntu / explicit 4-list / omit os:); got %q", filepath.Base(mf), filter)
				} else if !allowed[token] {
					t.Errorf("%s: unknown os token %q in %q (closed set: windows, arch, archlinux, cachyos, debian, ubuntu; darwin/linux legacy only)", filepath.Base(mf), token, filter)
				}
			}
		}

		var targetDistro []string
		collectStringValues(doc, "target_distro", &targetDistro)
		for _, target := range targetDistro {
			if target != "ubuntu" && target != "cachyos" {
				t.Errorf("%s: unknown target_distro %q (allowed: ubuntu, cachyos)", filepath.Base(mf), target)
			}
		}

		var minVersions []string
		collectStringValues(doc, "min_distro_version", &minVersions)
		for _, version := range minVersions {
			if version == "" {
				t.Errorf("%s: min_distro_version must not be empty", filepath.Base(mf))
			}
		}

		var checks []string
		collectStringValues(doc, "check_command", &checks)
		for _, c := range checks {
			if strings.Contains(c, `"`) {
				t.Errorf("%s: check_command with quotes is undeployable (strings.Fields splits on whitespace): %q", filepath.Base(mf), c)
			}
		}
	}
}
