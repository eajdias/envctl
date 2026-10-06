// Version helpers shared by providers, bootstrap, update and audit.
//
// Every version string this codebase compares, prints or probes flows through
// here: parsing (parseSemver), normalization (normalizeVersion,
// firstVersionToken), liveness probes (installedVersion, toolVersion,
// installSource), latest-resolution (npmLatest, defaultLatestProviderVersion,
// uvToolVersionOf, goLatestOf, latestFromJSON) and policy
// (openCodeWithinMajorUpdateNeeded, versionMajorAtLeast, fzfHasWalker,
// isProviderRuntime, printableVersion).
package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/eajdias/envctl/internal/infra/filesystem"
)

// installedVersion runs `<binary> --version` and reduces the output to the first
// version-looking token (tools report "opencode v2.0.5", "1.55.1", ...).
func installedVersion(ctx context.Context, binary string) string {
	resolved, err := resolveOnToolchainPath(binary)
	if err != nil {
		return ""
	}
	// exec.Cmd.Env does not affect binary resolution (LookPath uses the
	// process PATH), so execute the resolved absolute path.
	out, err := runWithToolchain(ctx, resolved, "--version")
	if err != nil {
		return ""
	}
	return firstVersionToken(out)
}

// installSource classifies where a binary comes from, which decides whether
// envctl may touch it.
func installSource(binary string) string {
	path, err := resolveOnToolchainPath(binary)
	if err != nil {
		return sourceAbsent
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return classifyInstallSource(path, home)
}

// classifyInstallSource names the owner of a binary path: mise's shims,
// envctl's own prefix, or the system. Only the envctl prefix and mise are safe
// for envctl to replace.
func classifyInstallSource(path, home string) string {
	normalized := filepath.ToSlash(path)
	switch {
	case home == "":
		return sourceSystem
	case strings.HasPrefix(normalized, filepath.ToSlash(filepath.Join(home, ".local", "share", "mise"))):
		return sourceMise
	case strings.HasPrefix(normalized, filepath.ToSlash(filepath.Join(home, ".local"))),
		strings.HasPrefix(normalized, filepath.ToSlash(filepath.Join(home, ".opencode"))):
		return sourceEnvctl
	default:
		return sourceSystem
	}
}

func (uc *ProvisionBootstrapUseCase) toolVersion(ctx context.Context, name string) string {
	out, err := uc.runShellStdout(ctx, "command -v "+name+" >/dev/null 2>&1 && "+name+" --version")
	if err != nil {
		return ""
	}
	return firstVersionToken(out)
}

// npmLatest reads the "latest" dist-tag of an npm package.
func npmLatest(ctx context.Context, pkg string) string {
	out, err := runWithToolchain(ctx, "curl", "-fsSL", "https://registry.npmjs.org/"+pkg+"/latest")
	if err != nil {
		return ""
	}
	// Deliberately dependency-free: the registry answers with a JSON object whose
	// "version" field is all this needs.
	return latestFromJSON(out)
}

// defaultLatestProviderVersion resolves the newest version of a standalone
// provider. OpenCode publishes its own update channel, separate from npm (which
// lags the release line); the official installer itself reads this same URL.
func defaultLatestProviderVersion(ctx context.Context, binary string) string {
	switch binary {
	case "opencode":
		out, err := runWithToolchain(ctx, "curl", "-fsSL", "https://opencode.ai/update/api/latest/cli/npm")
		if err != nil {
			return ""
		}
		return latestFromJSON(out)
	default:
		return ""
	}
}

// latestFromJSON extracts the "version" field from a minimal registry-style
// JSON object. Deliberately dependency-free: both npm and the OpenCode update
// channel answer with an object whose version field is all this needs.
func latestFromJSON(body string) string {
	idx := strings.Index(body, `"version":"`)
	if idx < 0 {
		return ""
	}
	rest := body[idx+len(`"version":"`):]
	if end := strings.Index(rest, `"`); end > 0 {
		return rest[:end]
	}
	return ""
}

// uvToolVersionOf asks uv which version of a tool is installed. `uv tool list`
// is the supported inventory command; the per-tool line carries the version.
func uvToolVersionOf(ctx context.Context, tool string) string {
	out, err := runWithToolchain(ctx, "uv", "tool", "list")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != tool {
			continue
		}
		// A pinned requirement (package==1.2.3) reports the pinned version; an
		// unpinned tool (package>=1.2.3) reports the resolved one after the "v".
		rest := strings.TrimPrefix(fields[1], "package")
		rest = strings.TrimLeft(rest, "<>=!~ ")
		rest = strings.TrimPrefix(rest, "v")
		if idx := strings.IndexAny(rest, " \t"); idx > 0 {
			rest = rest[:idx]
		}
		return rest
	}
	return ""
}

// goLatestOf resolves the latest released version of a Go module. `go list -m`
// against the module proxy answers without touching the working tree.
func goLatestOf(module string) string {
	path := module
	if idx := strings.LastIndex(module, "@"); idx >= 0 {
		path = module[:idx]
	}
	out, err := runWithToolchain(context.Background(), "go", "list", "-m", "-f", "{{.Version}}", path+"@latest")
	if err != nil {
		return ""
	}
	return firstVersionToken(out)
}

// parseSemver reads the first three numeric segments of a semver-ish string,
// tolerating a leading "v" and surrounding whitespace. Anything beyond the
// patch (prerelease/build metadata) is ignored: provider updates are decided
// on the numeric release triple only.
func parseSemver(version string) (major, minor, patch int, ok bool) {
	v := strings.TrimPrefix(strings.TrimSpace(version), "v")
	if idx := strings.IndexAny(v, "-+"); idx >= 0 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	if len(parts) < 3 {
		return 0, 0, 0, false
	}
	var errs [3]error
	major, errs[0] = strconv.Atoi(parts[0])
	minor, errs[1] = strconv.Atoi(parts[1])
	patch, errs[2] = strconv.Atoi(parts[2])
	if errs[0] != nil || errs[1] != nil || errs[2] != nil {
		return 0, 0, 0, false
	}
	return major, minor, patch, true
}

// versionsDiffer reports whether two version strings disagree once normalized
// (leading "v" and surrounding whitespace removed).
func versionsDiffer(installed, latest string) bool {
	return normalizeVersion(installed) != normalizeVersion(latest)
}

// normalizeVersion reduces a version to a comparable form: no "v" prefix, no
// leading zeros inside numeric segments, and no build metadata.
//
// The zero-padding case is real, not theoretical: yt-dlp reports
// "2026.08.19" from --version while `uv tool list` reports "v2026.8.19" for the
// same release. Comparing the raw strings would report a difference and run an
// update that changes nothing.
func normalizeVersion(version string) string {
	v := strings.TrimPrefix(strings.TrimSpace(version), "v")
	if idx := strings.IndexAny(v, "+"); idx > 0 {
		v = v[:idx]
	}
	if !strings.Contains(v, ".") {
		return v
	}
	segments := strings.Split(v, ".")
	for i, segment := range segments {
		trimmed := strings.TrimLeft(segment, "0")
		if trimmed == "" && segment != "" {
			// A segment of only zeros is zero, not empty.
			trimmed = "0"
		}
		segments[i] = trimmed
	}
	return strings.Join(segments, ".")
}

// versionMajorAtLeast reports whether version's numeric major is at least
// major. It intentionally rejects malformed versions instead of treating an
// unparseable provider as compatible with the V2-native configuration.
func versionMajorAtLeast(version string, major int) bool {
	normalized := normalizeVersion(version)
	majorToken, _, _ := strings.Cut(normalized, ".")
	parsed, err := strconv.Atoi(majorToken)
	return err == nil && parsed >= major
}

// firstVersionToken extracts the first token that starts with a digit, dropping
// the tool name and any leading "v".
func firstVersionToken(output string) string {
	for _, field := range strings.Fields(output) {
		candidate := strings.TrimPrefix(field, "v")
		if candidate == "" {
			continue
		}
		if candidate[0] >= '0' && candidate[0] <= '9' {
			return strings.TrimRight(candidate, ".,;")
		}
	}
	return ""
}

func printableVersion(version string) string {
	if version == "" {
		return "(missing)"
	}
	return "v" + version
}

// openCodeWithinMajorUpdateNeeded reports whether the installed OpenCode is
// older than latest while still on the required major (2.0.15 -> 2.0.23). A
// version on a higher major than required is never downgraded; an unparseable
// version is never updated (fail closed). Comparison is by major.minor.patch
// with the guard that the two majors are equal and match requiredMajor.
func openCodeWithinMajorUpdateNeeded(installed, latest string, requiredMajor int) bool {
	iMaj, iMin, iPat, okInst := parseSemver(installed)
	lMaj, lMin, lPat, okLatest := parseSemver(latest)
	if !okInst || !okLatest {
		return false
	}
	if iMaj != requiredMajor || lMaj != requiredMajor || iMaj != lMaj {
		return false
	}
	if iMin != lMin {
		return iMin < lMin
	}
	return iPat < lPat
}

// providerRuntimePrefix marks manifest ids that pin a runtime version rather
// than naming a tool. The providers phase owns those (node@24.19.0), and
// "volta install node@24.19.0@latest" is not a thing you can run.
func isProviderRuntime(id string) bool {
	return strings.Contains(id, "@")
}

// fzfSupportsWalker reports whether the installed fzf provides the built-in
// directory walker (0.47+).
func (uc *ProvisionBootstrapUseCase) fzfSupportsWalker(ctx context.Context) bool {
	out, err := uc.runShellStdout(ctx, `command -v fzf >/dev/null 2>&1 && fzf --version 2>/dev/null | awk '{print $1}'`)
	if err != nil {
		return false
	}
	return fzfHasWalker(out)
}

// fzfHasWalker reports whether a "MAJOR.MINOR[.PATCH]" version string is at
// least 0.47, the release that replaced the `find` fallback with fzf's own
// directory walker.
func fzfHasWalker(version string) bool {
	parts := strings.SplitN(strings.TrimSpace(version), ".", 3)
	if len(parts) < 2 {
		return false
	}
	major, majorErr := strconv.Atoi(parts[0])
	minor, minorErr := strconv.Atoi(parts[1])
	if majorErr != nil || minorErr != nil {
		return false
	}
	return major > 0 || minor >= 47
}

// archiveShadowedUserCopies archives every active non-system copy of binary
// while pacman owns it, re-probing version and source after each archive so
// the caller decides on the distro-owned binary. Phase 0 and the Linux
// bootstrap share this loop so both converge a stale user-local copy the
// same way; the surrounding flows keep their own diagnostics and log
// prefixes, which is why the wording arrives as parameters. The returned
// ok is false when archival failed and the caller must stop.
func archiveShadowedUserCopies(
	ctx context.Context,
	log func(format string, args ...any),
	name, binary, locateHint, archiveHint string,
	pacmanOwns bool,
	installed, source string,
	warn func(target, details, fixHint string),
) (string, string, bool) {
	for pacmanOwns && source != sourceSystem {
		path, err := resolveOnToolchainPath(binary)
		if err != nil {
			warn(name, fmt.Sprintf("could not locate the user-local binary to archive: %v", err), locateHint)
			return installed, source, false
		}
		backup, err := filesystem.ArchivePath(path)
		if err != nil {
			warn(name, fmt.Sprintf("could not archive user-local opencode at %s: %v", path, err), archiveHint)
			return installed, source, false
		}
		log("archived user-local %s at %s; pacman remains authoritative", binary, backup)
		installed = installedVersion(ctx, binary)
		source = installSource(binary)
	}
	return installed, source, true
}
