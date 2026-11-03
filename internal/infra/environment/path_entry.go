package environment

import (
	"path/filepath"
	"runtime"
	"strings"
)

// pathLineDirs returns the directories a single shell startup line contributes
// to PATH, or nil when the line does not declare PATH.
//
// Only PATH declarations count. A path mentioned in a comment, assigned to a
// different variable, or sitting in a sibling directory with the same prefix is
// not a PATH entry: treating it as one makes the writer skip a needed line (the
// shell never receives the entry) or makes the audit report a false pass.
//
// home is passed in rather than read from the environment so callers resolve it
// once per file and tests stay deterministic.
func pathLineDirs(line string, fish bool, home string) []string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return nil
	}
	if fish {
		return fishPathLineDirs(trimmed, home)
	}
	return posixPathLineDirs(trimmed, home)
}

// posixPathLineDirs parses `export PATH=...` and the bare `PATH=...` form.
func posixPathLineDirs(line, home string) []string {
	value := strings.TrimSpace(strings.TrimPrefix(line, "export "))
	value, ok := strings.CutPrefix(value, "PATH=")
	if !ok {
		return nil
	}
	// Drop an inline comment (`export PATH="$HOME/bin:$PATH" # note`) so a path
	// named in the trailing comment is not read as part of PATH.
	if idx := strings.Index(value, " #"); idx >= 0 {
		value = value[:idx]
	}
	return normalizePathSegments(strings.Split(value, ":"), home)
}

// fishPathLineDirs parses `set [-flags] PATH value...` and `fish_add_path`,
// the idiomatic way a user declares PATH entries in config.fish. fish keeps
// PATH as a list, so each argument may carry a colon-joined string as well.
func fishPathLineDirs(line, home string) []string {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil
	}
	switch fields[0] {
	case "fish_add_path":
		var args []string
		for _, f := range fields[1:] {
			if strings.HasPrefix(f, "-") {
				continue
			}
			args = append(args, f)
		}
		return normalizePathSegments(args, home)
	case "set":
		i := 1
		for i < len(fields) && strings.HasPrefix(fields[i], "-") {
			i++
		}
		if i >= len(fields) || fields[i] != "PATH" {
			return nil
		}
		return normalizePathSegments(fields[i+1:], home)
	}
	return nil
}

// normalizePathSegments expands the portable spellings ($HOME, ${HOME}, ~),
// drops unresolved expansions such as a literal $PATH, and cleans the result so
// comparison is exact. Non-absolute values are not entries.
func normalizePathSegments(segments []string, home string) []string {
	var dirs []string
	for _, segment := range segments {
		for _, part := range strings.Split(segment, ":") {
			if dir := normalizePathSegment(part, home); dir != "" {
				dirs = append(dirs, dir)
			}
		}
	}
	return dirs
}

func normalizePathSegment(segment, home string) string {
	segment = strings.Trim(strings.TrimSpace(segment), `"'`)
	switch {
	case segment == "~", segment == "$HOME", segment == "${HOME}":
		segment = home
	case strings.HasPrefix(segment, "~/"):
		segment = filepath.Join(home, segment[len("~/"):])
	case strings.HasPrefix(segment, "$HOME/"):
		segment = filepath.Join(home, segment[len("$HOME/"):])
	case strings.HasPrefix(segment, "${HOME}/"):
		segment = filepath.Join(home, segment[len("${HOME}/"):])
	}
	if segment == "" || !filepath.IsAbs(segment) {
		return ""
	}
	return filepath.Clean(segment)
}

// samePath compares two cleaned paths, folding case on Windows where the
// filesystem does.
func samePath(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

// hasPathEntry reports whether any PATH declaration in data contains dir.
// dir goes through the same expansion as the parsed segments so a caller that
// passes the portable `$HOME/...` spelling (the bootstrap steps do) still
// matches an absolute line and vice versa.
func hasPathEntry(data, dir, home string, fish bool) bool {
	target := normalizePathSegment(dir, home)
	if target == "" {
		target = filepath.Clean(dir)
	}
	for _, line := range strings.Split(data, "\n") {
		for _, declared := range pathLineDirs(line, fish, home) {
			if samePath(declared, target) {
				return true
			}
		}
	}
	return false
}
