package usecase

import (
	"testing"
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
