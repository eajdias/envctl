package embedded

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var changelogHeading = regexp.MustCompile(`^## \[v?(\d+)\.(\d+)\.(\d+)\]`)

// TestChangelogHasSingleUnreleased pins the consolidation the manual release
// flow depends on: the tag commit must move entries from [Unreleased] into
// the version section *before* dispatching, and that only works when there is
// exactly one Unreleased section to move.
func TestChangelogHasSingleUnreleased(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRootForDocs(t), "CHANGELOG.md"))
	if err != nil {
		t.Fatalf("read CHANGELOG.md: %v", err)
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimRight(line, "\r") == "## [Unreleased]" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("want exactly one ## [Unreleased], found %d (move stray blocks into their version section)", count)
	}
}

// TestChangelogVersionsDescend pins the reverse-chronological order the
// release-please compare links assume: every version heading must be
// strictly older than the previous one.
func TestChangelogVersionsDescend(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRootForDocs(t), "CHANGELOG.md"))
	if err != nil {
		t.Fatalf("read CHANGELOG.md: %v", err)
	}
	var prev [3]int
	var prevRaw string
	first := true
	for _, line := range strings.Split(string(data), "\n") {
		m := changelogHeading.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			continue
		}
		cur := [3]int{mustAtoi(t, m[1]), mustAtoi(t, m[2]), mustAtoi(t, m[3])}
		if !first && compareSemver(cur, prev) >= 0 {
			t.Errorf("version %s is not older than previous %s", strings.TrimSpace(line), prevRaw)
		}
		prev, prevRaw, first = cur, strings.TrimSpace(line), false
	}
	if first {
		t.Error("no version headings found")
	}
}

func mustAtoi(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(s)
	if err != nil {
		t.Fatalf("not a number: %q", s)
	}
	return n
}

func compareSemver(a, b [3]int) int {
	for i := 0; i < 3; i++ {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}
