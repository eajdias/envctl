package usecase

import (
	"fmt"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// maxConfigDiffLines caps a single file's --diff output so one huge config
// cannot flood the doctor table. Changed lines beyond the cap are counted,
// not shown.
const maxConfigDiffLines = 60

// FileDiff carries a preview of template-vs-live for one divergent config
// file. It is populated only when the doctor runs with diff preview enabled.
type FileDiff struct {
	Target  string
	Unified string
}

// unifiedConfigDiff renders a compact -/+ diff of template (src) vs live
// (dst), line-oriented. Only changed lines are emitted (no context), each
// prefixed with '-' (provisioned) or '+' (on disk).
func unifiedConfigDiff(target string, src, dst []byte) string {
	dmp := diffmatchpatch.New()
	a, b, lineArray := dmp.DiffLinesToChars(string(src), string(dst))
	diffs := dmp.DiffMain(a, b, false)
	diffs = dmp.DiffCharsToLines(diffs, lineArray)
	dmp.DiffCleanupSemantic(diffs)

	var sb strings.Builder
	fmt.Fprintf(&sb, "--- provisioned (%s)\n+++ on disk (%s)\n", target, target)
	shown := 0
	hidden := 0
	emit := func(prefix, text string) {
		for _, line := range strings.Split(text, "\n") {
			if shown >= maxConfigDiffLines {
				hidden++
				continue
			}
			sb.WriteString(prefix)
			sb.WriteString(line)
			sb.WriteString("\n")
			shown++
		}
	}
	for _, d := range diffs {
		switch d.Type {
		case diffmatchpatch.DiffDelete:
			emit("-", d.Text)
		case diffmatchpatch.DiffInsert:
			emit("+", d.Text)
		}
	}
	if hidden > 0 {
		fmt.Fprintf(&sb, "... (%d more changed lines hidden)\n", hidden)
	}
	return sb.String()
}
