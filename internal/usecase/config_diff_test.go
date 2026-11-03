package usecase

import (
	"strings"
	"testing"
)

func TestUnifiedConfigDiff_MarksChangedLines(t *testing.T) {
	src := []byte("keep\nold-line\nkeep2\n")
	dst := []byte("keep\nnew-line\nkeep2\n")
	out := unifiedConfigDiff("cfg", src, dst)

	if !strings.Contains(out, "--- provisioned (cfg)") || !strings.Contains(out, "+++ on disk (cfg)") {
		t.Errorf("diff must carry both headers, got:\n%s", out)
	}
	if !strings.Contains(out, "-old-line") {
		t.Errorf("diff must mark the removed line, got:\n%s", out)
	}
	if !strings.Contains(out, "+new-line") {
		t.Errorf("diff must mark the added line, got:\n%s", out)
	}
	for _, unchanged := range []string{"\nkeep\n", "\nkeep2\n"} {
		if strings.Contains(out, unchanged) {
			t.Errorf("unchanged line %q must not be emitted, got:\n%s", unchanged, out)
		}
	}
}

func TestUnifiedConfigDiff_IdenticalIsHeadersOnly(t *testing.T) {
	out := unifiedConfigDiff("cfg", []byte("a\nb\n"), []byte("a\nb\n"))
	body := strings.TrimPrefix(strings.TrimPrefix(out, "--- provisioned (cfg)\n"), "+++ on disk (cfg)\n")
	if strings.TrimSpace(body) != "" {
		t.Errorf("identical inputs must render headers only, got:\n%s", out)
	}
}

func TestUnifiedConfigDiff_CapsLongDiffs(t *testing.T) {
	var src, dst strings.Builder
	for i := 0; i < maxConfigDiffLines+20; i++ {
		src.WriteString("old\n")
		dst.WriteString("new\n")
	}
	out := unifiedConfigDiff("cfg", []byte(src.String()), []byte(dst.String()))
	lines := strings.Count(strings.TrimSuffix(out, "\n"), "\n") + 1
	// 2 header lines + cap, plus the hidden-count trailer.
	if lines > maxConfigDiffLines+3 {
		t.Errorf("diff must be capped near %d lines, got %d", maxConfigDiffLines, lines)
	}
	if !strings.Contains(out, "more changed lines hidden") {
		t.Errorf("capped diff must name the hidden count, got tail:\n%s", out[len(out)-120:])
	}
}
