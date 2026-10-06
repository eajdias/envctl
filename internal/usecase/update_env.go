package usecase

import (
	"context"
	"errors"
)

// NewRealUpdateEnv wires the use case to the actual machine.
func NewRealUpdateEnv() UpdateEnv { return newRealUpdateEnv() }

// realUpdateEnv runs against the machine. It reuses the providers helpers on
// purpose: installedVersion, npmLatest and runWithToolchain are the only way
// this codebase resolves a version or runs a tool, so a second implementation
// would drift from the phase 0 behaviour that already ships.
type realUpdateEnv struct {
	// resolveBinary mirrors resolveOnToolchainPath, injected so the mapping can
	// be exercised without touching the machine.
	resolveBinary func(name string) (string, error)
	run           func(ctx context.Context, name string, args ...string) (string, error)
	uvToolVersion func(ctx context.Context, tool string) string
	installed     func(binary string) string
	npmLatest     func(pkg string) string
	goLatest      func(module string) string
}

func newRealUpdateEnv() UpdateEnv {
	return &realUpdateEnv{
		resolveBinary: resolveOnToolchainPath,
		run:           runWithToolchain,
		installed:     installedVersionOn,
		npmLatest:     npmLatestOn,
		uvToolVersion: uvToolVersionOf,
		goLatest:      goLatestOf,
	}
}

// installedVersionOn and npmLatestOn adapt the shared package-level helpers to
// the interface, which takes no context so the fakes stay trivial.
func installedVersionOn(binary string) string { return installedVersion(context.Background(), binary) }

func npmLatestOn(pkg string) string { return npmLatest(context.Background(), pkg) }

func (e *realUpdateEnv) installedVersion(binary string) string { return e.installed(binary) }

func (e *realUpdateEnv) latestVersion(group UpdateGroup, target string) string {
	switch group {
	case GroupVolta:
		return e.npmLatest(target)
	case GroupUV:
		return e.uvToolVersion(context.Background(), target)
	case GroupGo:
		return e.goLatest(target)
	default:
		return ""
	}
}

func (e *realUpdateEnv) applyUpdate(ctx context.Context, group UpdateGroup, target string) (string, error) {
	var name string
	var args []string
	switch group {
	case GroupVolta:
		name, args = "volta", []string{"install", target + "@latest"}
	case GroupUV:
		name, args = "uv", []string{"tool", "upgrade", target}
	case GroupGo:
		name, args = "go", []string{"install", target + "@latest"}
	default:
		return "", errUnautomatableGroup
	}
	return e.run(ctx, name, args...)
}

// errUnautomatableGroup guards a programming error rather than a machine
// condition: collect() only ever emits the three groups above.
var errUnautomatableGroup = errors.New("install group is not automatable")
