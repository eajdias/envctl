package toolchain

import (
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
)

// Libraries are probed with `py -m pip show` and install into the system
// environment; everything else is a CLI tool for an isolated `uv tool` env.
func TestIsPipLibrary(t *testing.T) {
	cases := map[string]bool{
		"py -m pip show pyyaml":    true,
		"python3 -m pip show lxml": true,
		"pytest --version":         false,
		"yt-dlp --version":         false,
		"sqlfluff --version":       false,
		"":                         false,
	}
	for check, want := range cases {
		pkg := entity.Package{ID: "x", CheckCommand: check}
		if got := isPipLibrary(pkg); got != want {
			t.Errorf("isPipLibrary(%q) = %v, want %v", check, got, want)
		}
	}
}
