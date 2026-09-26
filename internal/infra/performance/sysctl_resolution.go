package performance

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
)

// sysctlDropinDirs are the configuration directories systemd-sysctl reads, in
// the precedence order documented by sysctl.d(5): "Configuration files are read
// from directories in /etc/, /run/, /usr/local/lib/, and /usr/lib/, in order of
// precedence".
var sysctlDropinDirs = []string{
	"/etc/sysctl.d",
	"/run/sysctl.d",
	"/usr/local/lib/sysctl.d",
	"/usr/lib/sysctl.d",
}

// resolveSysctlAssignments reproduces, read-only, what systemd-sysctl will apply
// at the next boot: every *.conf across the four directories, sorted by
// filename in lexicographic order regardless of directory, with a file in /etc
// overriding a file of the same name under /usr.
//
// Why this exists: the fleet ships /etc/sysctl.d/99-swappiness.conf, written by
// the cloud agent, which sorts after the profile's
// /etc/sysctl.d/90-envctl-performance.conf. A run writes vm.swappiness to the
// profile's file and sets it live, and the next boot restores 10 with no
// diagnostic anywhere. Resolving the files is the only way to see that.
//
// managedPath is the drop-in this project writes; assignments coming from it are
// flagged Managed. Pass "" to flag nothing.
func resolveSysctlAssignments(dirs []string, managedPath string) []entity.SysctlAssignment {
	type candidate struct {
		name    string
		dirRank int
		path    string
	}

	var candidates []candidate
	seen := map[string]int{} // basename -> index of the winning directory rank
	for rank, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".conf" {
				continue
			}
			// A name in a higher-precedence directory supersedes the same name
			// below it, so only the first directory offering it contributes.
			if previous, ok := seen[entry.Name()]; ok && previous < rank {
				continue
			}
			seen[entry.Name()] = rank
			candidates = append(candidates, candidate{
				name:    entry.Name(),
				dirRank: rank,
				path:    filepath.Join(dir, entry.Name()),
			})
		}
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].name != candidates[j].name {
			return candidates[i].name < candidates[j].name
		}
		return candidates[i].dirRank < candidates[j].dirRank
	})

	resolved := map[string]entity.SysctlAssignment{}
	for _, item := range candidates {
		data, err := os.ReadFile(item.path)
		if err != nil {
			continue
		}
		for key, value := range parseSysctlAssignments(string(data)) {
			resolved[key] = entity.SysctlAssignment{
				Key:     key,
				File:    item.path,
				Boot:    value,
				Managed: managedPath != "" && item.path == managedPath,
			}
		}
	}

	keys := make([]string, 0, len(resolved))
	for key := range resolved {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	assignments := make([]entity.SysctlAssignment, 0, len(keys))
	for _, key := range keys {
		assignment := resolved[key]
		if live, ok := readSysctlValue(key); ok {
			assignment.Live = live
		}
		assignments = append(assignments, assignment)
	}
	return assignments
}

// parseSysctlAssignments reads `key = value` pairs. Comments, blank lines, the
// optional `-` exclusion prefix and anything without a separator are ignored,
// per sysctl.d(5). Keys are normalised to the dotted form because the manual
// states that "a/b/c" and "a.b.c" name the same key.
func parseSysctlAssignments(data string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "-") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = normaliseSysctlKey(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		out[key] = value
	}
	return out
}

// normaliseSysctlKey collapses a key to its dotted form. Only the first
// separator is special: with a leading slash the remaining slashes are literal,
// and with a leading dot the slashes and dots are interchanged, so replacing
// every separator with a dot after normalising the leading one is equivalent.
func normaliseSysctlKey(key string) string {
	key = strings.TrimSpace(key)
	if strings.HasPrefix(key, "/") {
		return strings.ReplaceAll(key, "/", ".")
	}
	return strings.ReplaceAll(key, "/", ".")
}
