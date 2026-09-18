package usecase

import (
	"encoding/json"
	"strings"
)

// This file implements the non-destructive merge modes declared by
// entity.ConfigFile.Merge. Provisioning replaces managed files wholesale, which
// is correct for files envctl owns but destroys user-owned content in files
// like ~/.ssh/config (per-host stanzas) or ~/package.json (extra deps).

// sshHostBlock is a raw `Host ...` stanza extracted from an ssh client config,
// including the comment lines the user attached directly above it.
type sshHostBlock struct {
	patterns []string
	lines    []string
}

// mergeSSHHosts combines the template with the destination's ssh config,
// preserving every host stanza the template does not define. Preserved stanzas
// are inserted before the first template `Host` entry so specific hosts keep
// precedence over a template `Host *` block (OpenSSH uses the first value it
// obtains for each parameter).
func mergeSSHHosts(template, existing []byte) []byte {
	templatePatterns := sshHostPatterns(string(template))
	var preserved []sshHostBlock
	seen := make(map[string]bool)
	for _, block := range splitSSHHostBlocks(string(existing)) {
		if block.definedIn(templatePatterns) {
			continue
		}
		key := strings.Join(block.patterns, " ")
		if seen[key] {
			continue
		}
		seen[key] = true
		preserved = append(preserved, block)
	}
	if len(preserved) == 0 {
		return template
	}

	lines := strings.Split(strings.TrimRight(string(template), "\n"), "\n")
	insertAt := -1
	for i, line := range lines {
		if sshHostLinePatterns(line) != nil || isSSHMatchLine(line) {
			insertAt = i
			break
		}
	}

	var payload []string
	for _, block := range preserved {
		payload = append(payload, "")
		payload = append(payload, block.lines...)
	}

	if insertAt < 0 {
		lines = append(lines, payload...)
	} else {
		merged := make([]string, 0, len(lines)+len(payload))
		merged = append(merged, lines[:insertAt]...)
		merged = append(merged, payload...)
		merged = append(merged, lines[insertAt:]...)
		lines = merged
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

// sshHostPatterns collects every host pattern defined in an ssh config.
func sshHostPatterns(content string) map[string]bool {
	patterns := make(map[string]bool)
	for _, line := range strings.Split(content, "\n") {
		for _, pattern := range sshHostLinePatterns(line) {
			patterns[pattern] = true
		}
	}
	return patterns
}

// sshHostLinePatterns returns the lowercased host patterns of a `Host` line,
// or nil when the line is not a Host directive.
func sshHostLinePatterns(line string) []string {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 2 || !strings.EqualFold(fields[0], "host") {
		return nil
	}
	patterns := make([]string, 0, len(fields)-1)
	for _, field := range fields[1:] {
		patterns = append(patterns, strings.ToLower(field))
	}
	return patterns
}

// isSSHMatchLine reports whether the line opens a `Match` block, which also
// terminates the preceding Host stanza.
func isSSHMatchLine(line string) bool {
	fields := strings.Fields(strings.TrimSpace(line))
	return len(fields) > 0 && strings.EqualFold(fields[0], "match")
}

// splitSSHHostBlocks extracts each `Host` stanza from an ssh config, attaching
// the contiguous comment lines directly above it so user annotations survive.
func splitSSHHostBlocks(content string) []sshHostBlock {
	lines := strings.Split(content, "\n")
	claimed := make([]bool, len(lines))
	var blocks []sshHostBlock

	for i := 0; i < len(lines); {
		if sshHostLinePatterns(lines[i]) == nil {
			i++
			continue
		}
		start := i
		for start > 0 && !claimed[start-1] && isCommentLine(lines[start-1]) {
			start--
		}
		end := i + 1
		for end < len(lines) && sshHostLinePatterns(lines[end]) == nil && !isSSHMatchLine(lines[end]) {
			end++
		}
		body := end
		for body > i+1 && strings.TrimSpace(lines[body-1]) == "" {
			body--
		}

		blockLines := make([]string, 0, body-start)
		blockLines = append(blockLines, lines[start:body]...)
		blocks = append(blocks, sshHostBlock{patterns: sshHostLinePatterns(lines[i]), lines: blockLines})

		for j := start; j < end; j++ {
			claimed[j] = true
		}
		i = end
	}
	return blocks
}

// isCommentLine reports whether a line is a comment. Only comments directly
// above a `Host` line are attached to its stanza — allowing blank lines would
// let a stanza absorb the template's own section headers, which would then be
// duplicated on the next run and break idempotency.
func isCommentLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "#")
}

// definedIn reports whether any of the block's patterns is already declared by
// the template. A stanza naming at least one known host is considered template
// content and is not preserved.
func (b sshHostBlock) definedIn(templatePatterns map[string]bool) bool {
	for _, pattern := range b.patterns {
		if templatePatterns[pattern] {
			return true
		}
	}
	return false
}

// mergeJSONDeps merges the template JSON over the destination, keeping every
// key the template does not define and unioning the dependency maps. Template
// values win for keys both sides define, so managed baselines stay current
// while user additions are never dropped. It returns an error (and the caller
// keeps the user file untouched) when the destination is not valid JSON.
func mergeJSONDeps(template, existing []byte) ([]byte, error) {
	tpl, err := decodeJSONObject(template)
	if err != nil {
		return nil, err
	}
	current, err := decodeJSONObject(existing)
	if err != nil {
		return nil, err
	}

	merged := make(map[string]json.RawMessage, len(tpl)+len(current))
	for key, value := range current {
		merged[key] = value
	}
	for key, value := range tpl {
		merged[key] = value
	}

	for _, section := range []string{"dependencies", "devDependencies"} {
		deps, err := unionJSONStringMaps(current[section], tpl[section])
		if err != nil {
			return nil, err
		}
		if len(deps) == 0 {
			continue
		}
		encoded, err := json.Marshal(deps)
		if err != nil {
			return nil, err
		}
		merged[section] = encoded
	}

	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func decodeJSONObject(data []byte) (map[string]json.RawMessage, error) {
	object := make(map[string]json.RawMessage)
	if len(strings.TrimSpace(string(data))) == 0 {
		return object, nil
	}
	if err := json.Unmarshal(data, &object); err != nil {
		return nil, err
	}
	return object, nil
}

// unionJSONStringMaps merges two JSON objects of string values, with the
// overlay (template) winning on conflicts.
func unionJSONStringMaps(base, overlay json.RawMessage) (map[string]string, error) {
	merged := make(map[string]string)
	for _, raw := range []json.RawMessage{base, overlay} {
		if len(raw) == 0 || string(raw) == "null" {
			continue
		}
		var section map[string]string
		if err := json.Unmarshal(raw, &section); err != nil {
			return nil, err
		}
		for key, value := range section {
			merged[key] = value
		}
	}
	return merged, nil
}
