package usecase

import "testing"

func TestFzfHasWalker(t *testing.T) {
	cases := []struct {
		version string
		want    bool
	}{
		// Ubuntu 24.04 ships 0.44.1: no walker, so fzf still falls back to find.
		{"0.44.1", false},
		{"0.46.1", false},
		// 0.47 replaced the `find` fallback with the built-in walker.
		{"0.47.0", true},
		{"0.74.4", true},
		{"1.0.0", true},
		{"0.47", true},
		{"", false},
		{"not-a-version", false},
		{"0", false},
		// `fzf --version` prints "0.74.4 (sha)"; only major/minor are read, so
		// the suffix is harmless.
		{"0.74.4 (a140afeb)", true},
	}
	for _, tc := range cases {
		if got := fzfHasWalker(tc.version); got != tc.want {
			t.Errorf("fzfHasWalker(%q) = %v, want %v", tc.version, got, tc.want)
		}
	}
}
