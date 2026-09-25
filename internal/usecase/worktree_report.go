package usecase

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
)

type gitWorktree struct {
	Path           string
	Head           string
	Branch         string
	Detached       bool
	Bare           bool
	Locked         bool
	LockReason     string
	Prunable       bool
	PrunableReason string
}

// parseWorktreeListPorcelain parses the stable record format emitted by
// `git worktree list --porcelain`. It is intentionally pure so doctor behavior
// can be tested without creating or mutating a repository.
func parseWorktreeListPorcelain(output string) ([]gitWorktree, error) {
	var worktrees []gitWorktree
	var current *gitWorktree

	flush := func() {
		if current != nil {
			worktrees = append(worktrees, *current)
			current = nil
		}
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			flush()
			continue
		}

		key, value, _ := strings.Cut(line, " ")
		switch key {
		case "worktree":
			flush()
			if strings.TrimSpace(value) == "" {
				return nil, fmt.Errorf("worktree record has an empty path")
			}
			current = &gitWorktree{Path: value}
		case "HEAD":
			if current == nil {
				return nil, fmt.Errorf("worktree record has HEAD before worktree header")
			}
			current.Head = value
		case "branch":
			if current == nil {
				return nil, fmt.Errorf("worktree record has branch before worktree header")
			}
			current.Branch = value
		case "detached":
			if current == nil {
				return nil, fmt.Errorf("worktree record has detached before worktree header")
			}
			current.Detached = true
		case "bare":
			if current == nil {
				return nil, fmt.Errorf("worktree record has bare before worktree header")
			}
			current.Bare = true
		case "locked":
			if current == nil {
				return nil, fmt.Errorf("worktree record has locked before worktree header")
			}
			current.Locked = true
			current.LockReason = value
		case "prunable":
			if current == nil {
				return nil, fmt.Errorf("worktree record has prunable before worktree header")
			}
			current.Prunable = true
			current.PrunableReason = value
		default:
			// Git may add record keys in future versions. Unknown keys are
			// ignored rather than making the audit fail closed.
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read worktree list: %w", err)
	}
	flush()

	return worktrees, nil
}

func worktreeFindings(worktrees []gitWorktree) []entity.Diagnostic {
	var diagnostics []entity.Diagnostic
	for _, worktree := range worktrees {
		if worktree.Locked {
			reason := worktree.LockReason
			if reason == "" {
				reason = "no reason reported"
			}
			diagnostics = append(diagnostics, entity.Diagnostic{
				Category: entity.DiagInfo,
				System:   "Git",
				Target:   "worktree",
				Details:  fmt.Sprintf("Worktree %q is locked (%s)", worktree.Path, reason),
				FixHint:  "leave it locked when the work is intentional; inspect before unlocking",
			})
		}
		if worktree.Prunable {
			reason := worktree.PrunableReason
			if reason == "" {
				reason = "no reason reported"
			}
			diagnostics = append(diagnostics, entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "Git",
				Target:   "worktree",
				Details:  fmt.Sprintf("Worktree %q is prunable: %s", worktree.Path, reason),
				FixHint:  "inspect the worktree and run 'git worktree prune' only after confirming no uncommitted work",
			})
		}
	}
	return diagnostics
}
