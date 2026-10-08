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
