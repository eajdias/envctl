package usecase

import (
	"strings"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func TestParseWorktreeListPorcelain(t *testing.T) {
	output := `worktree /repo
HEAD 1111111111111111111111111111111111111111
branch refs/heads/main

worktree /repo/.worktrees/feat-a
HEAD 2222222222222222222222222222222222222222
branch refs/heads/feat/a
locked reason: active task
prunable gitdir file points to non-existent location

worktree /repo/.worktrees/detached
HEAD 3333333333333333333333333333333333333333
detached
`

	worktrees, err := parseWorktreeListPorcelain(output)
	if err != nil {
		t.Fatalf("parseWorktreeListPorcelain() error = %v", err)
	}
	if len(worktrees) != 3 {
		t.Fatalf("got %d worktrees, want 3", len(worktrees))
	}

	if worktrees[0].Path != "/repo" || worktrees[0].Branch != "refs/heads/main" {
		t.Errorf("root worktree = %+v", worktrees[0])
	}
	if !worktrees[1].Locked || worktrees[1].LockReason != "reason: active task" {
		t.Errorf("locked worktree = %+v", worktrees[1])
	}
	if !worktrees[1].Prunable || worktrees[1].PrunableReason != "gitdir file points to non-existent location" {
		t.Errorf("prunable worktree = %+v", worktrees[1])
	}
	if !worktrees[2].Detached {
		t.Errorf("detached worktree = %+v", worktrees[2])
	}
}

func TestParseWorktreeListPorcelainRejectsRecordWithoutHeader(t *testing.T) {
	_, err := parseWorktreeListPorcelain("HEAD 1111111111111111111111111111111111111111\n")
	if err == nil {
		t.Fatal("expected malformed record to return an error")
	}
	if !strings.Contains(err.Error(), "worktree") {
		t.Errorf("error = %q, want worktree context", err)
	}
}

func TestParseWorktreeListPorcelainAllowsEmptyOutput(t *testing.T) {
	worktrees, err := parseWorktreeListPorcelain("")
	if err != nil {
		t.Fatalf("empty output error = %v", err)
	}
	if len(worktrees) != 0 {
		t.Fatalf("got %d worktrees, want 0", len(worktrees))
	}
}

func TestWorktreeFindingsSeverity(t *testing.T) {
	diagnostics := worktreeFindings([]gitWorktree{
		{Path: "/repo/.worktrees/locked", Locked: true, LockReason: "active task"},
		{Path: "/repo/.worktrees/dead", Prunable: true, PrunableReason: "missing gitdir"},
		{Path: "/repo/.worktrees/healthy"},
	})

	if len(diagnostics) != 2 {
		t.Fatalf("got %d diagnostics, want 2", len(diagnostics))
	}
	if diagnostics[0].Category != entity.DiagInfo {
		t.Errorf("locked category = %v, want INFO", diagnostics[0].Category)
	}
	if diagnostics[1].Category != entity.DiagWarning {
		t.Errorf("prunable category = %v, want WARNING", diagnostics[1].Category)
	}
	if !strings.Contains(diagnostics[1].Details, "missing gitdir") {
		t.Errorf("prunable details = %q", diagnostics[1].Details)
	}
	if diagnostics[1].FixHint == "" {
		t.Error("prunable diagnostic must include a fix hint")
	}
}
