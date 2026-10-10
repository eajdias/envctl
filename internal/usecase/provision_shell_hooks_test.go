package usecase

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The global hooksPath disarm runs during cleanup. Tests must never touch the
// real user git config, so they isolate it with GIT_CONFIG_GLOBAL.

func TestDisarmGlobalHooksPathUnsetsPointingConfig(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, ".config", "git", "hooks")
	env := []string{"GIT_CONFIG_GLOBAL=" + filepath.Join(home, "gitconfig")}
	expand := func(p string) (string, error) {
		return filepath.Clean(strings.Replace(p, "~", home, 1)), nil
	}
	seed := exec.Command("git", "config", "--global", "core.hooksPath", "~/.config/git/hooks")
	seed.Env = append(os.Environ(), env...)
	if out, err := seed.CombinedOutput(); err != nil {
		t.Fatalf("seed config: %v (%s)", err, out)
	}

	if !disarmGlobalHooksPath(context.Background(), expand, env, target) {
		t.Fatal("a hooksPath pointing at the removed directory must be unset")
	}

	read := exec.Command("git", "config", "--global", "--get", "core.hooksPath")
	read.Env = append(os.Environ(), env...)
	if out, err := read.Output(); err == nil {
		t.Fatalf("core.hooksPath must be gone, still %q", out)
	}
}

func TestDisarmGlobalHooksPathKeepsForeignConfig(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, ".config", "git", "hooks")
	env := []string{"GIT_CONFIG_GLOBAL=" + filepath.Join(home, "gitconfig")}
	expand := func(p string) (string, error) {
		return filepath.Clean(strings.Replace(p, "~", home, 1)), nil
	}
	seed := exec.Command("git", "config", "--global", "core.hooksPath", filepath.ToSlash(filepath.Join(home, "my-own-hooks")))
	seed.Env = append(os.Environ(), env...)
	if out, err := seed.CombinedOutput(); err != nil {
		t.Fatalf("seed config: %v (%s)", err, out)
	}

	if disarmGlobalHooksPath(context.Background(), expand, env, target) {
		t.Fatal("a user-owned hooksPath elsewhere must never be touched")
	}

	read := exec.Command("git", "config", "--global", "--get", "core.hooksPath")
	read.Env = append(os.Environ(), env...)
	out, err := read.Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		t.Fatalf("the foreign hooksPath must remain, got %q (%v)", out, err)
	}
}

func TestDisarmGlobalHooksPathNoConfig(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, ".config", "git", "hooks")
	env := []string{"GIT_CONFIG_GLOBAL=" + filepath.Join(home, "gitconfig")}
	expand := func(p string) (string, error) {
		return filepath.Clean(strings.Replace(p, "~", home, 1)), nil
	}

	if disarmGlobalHooksPath(context.Background(), expand, env, target) {
		t.Fatal("without a hooksPath there is nothing to disarm")
	}
}
