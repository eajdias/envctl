package usecase

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
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

// Markdown section markers. HTML comments on their own line keep them inert to
// every Markdown renderer while still giving the merge a stable, exact anchor.
// The user block is the only part a machine's operator is expected to edit; the
// managed block is replaced wholesale from the template on every run.
const (
	markdownManagedStart = "<!-- envctl:managed:start -->"
	markdownManagedEnd   = "<!-- envctl:managed:end -->"
	markdownUserStart    = "<!-- envctl:user:start -->"
	markdownUserEnd      = "<!-- envctl:user:end -->"
)

// mergeMarkdownSections combines the template with the destination, preserving
// the destination's user block verbatim. The template wins for the managed
// block (so the baseline stays current); the destination wins for the user
// block (so lessons and preferences survive a provisioning run).
//
// The template is the source of truth for both blocks' markers: the merged
// output is always the template's managed block plus whatever user content the
// destination carries. A destination without any marker — a file written before
// this merge mode existed, or a first-run file edited directly — is preserved
// whole as user content, because losing it would be worse than keeping a stale
// managed block.
// splitFrontmatter separates a leading YAML frontmatter block (`---\n...
// ---\n) from the body. Files without one return ("", whole). The doctor's
// frontmatter validators require `---` at byte zero, so merged Markdown must
// keep the template's header verbatim ahead of the managed block.
func splitFrontmatter(content string) (header, rest string) {
	if !strings.HasPrefix(content, "---\n") {
		return "", content
	}
	lines := strings.Split(content[len("---\n"):], "\n")
	for i, line := range lines {
		if strings.TrimRight(line, " \t") == "---" {
			end := len("---\n") + len(strings.Join(lines[:i+1], "\n")) + 1
			return content[:end], content[end:]
		}
	}
	return "", content
}

func mergeMarkdownSections(template, existing []byte) []byte {
	tplHeader, tpl := splitFrontmatter(string(template))
	_, ex := splitFrontmatter(string(existing))

	managed := sectionBetween(tpl, markdownManagedStart, markdownManagedEnd)
	if managed == "" && !strings.Contains(tpl, markdownManagedStart) {
		// The template carries no managed markers: everything is managed.
		managed = tpl
	}

	// The user block seed lives in the template's own user section, so a fresh
	// destination inherits it; the destination's block replaces it afterwards.
	user := sectionBetween(tpl, markdownUserStart, markdownUserEnd)

	if len(ex) > 0 {
		if strings.Contains(ex, markdownUserStart) || strings.Contains(ex, markdownManagedStart) {
			// Already-marked destination: preserve exactly its user block.
			user = sectionBetween(ex, markdownUserStart, markdownUserEnd)
		} else {
			// Legacy destination with no markers at all: keep everything.
			user = strings.TrimRight(ex, "\n")
		}
	}

	managed = strings.TrimRight(managed, "\n")
	user = strings.TrimRight(user, "\n")

	var b strings.Builder
	b.WriteString(tplHeader)
	b.WriteString(markdownManagedStart)
	b.WriteString("\n")
	b.WriteString(managed)
	b.WriteString("\n")
	b.WriteString(markdownManagedEnd)
	b.WriteString("\n")
	b.WriteString(markdownUserStart)
	b.WriteString("\n")
	if user != "" {
		b.WriteString(user)
		b.WriteString("\n")
	}
	b.WriteString(markdownUserEnd)
	b.WriteString("\n")
	return []byte(b.String())
}

// sectionBetween returns the text strictly between the start and end markers,
// or "" when either marker is absent. The markers must sit on their own line:
// the split trims surrounding newlines so callers never depend on incidental
// indentation or trailing whitespace.
func sectionBetween(content, start, end string) string {
	startIdx := strings.Index(content, start)
	if startIdx < 0 {
		return ""
	}
	bodyStart := startIdx + len(start)
	endIdx := strings.Index(content[bodyStart:], end)
	if endIdx < 0 {
		return ""
	}
	return strings.Trim(content[bodyStart:bodyStart+endIdx], "\n")
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

// scriptMarkers are the managed/user delimiters for executable script files
// (PowerShell profile): `#` comments on their own line, so the merged file
// stays runnable. Semantics mirror mergeMarkdownSections exactly — the two
// are pinned to the same behavior table in config_merge_test.go, so a fix
// in one must land in the other.
const (
	scriptManagedStart = "# envctl:managed:start"
	scriptManagedEnd   = "# envctl:managed:end"
	scriptUserStart    = "# envctl:user:start"
	scriptUserEnd      = "# envctl:user:end"
)

// mergeScriptSections combines the template with the destination's script
// file, preserving the destination's user block verbatim. Same contract as
// mergeMarkdownSections, with `#` markers instead of HTML comments.
func mergeScriptSections(template, existing []byte) []byte {
	tpl := string(template)

	managed := sectionBetween(tpl, scriptManagedStart, scriptManagedEnd)
	if managed == "" && !strings.Contains(tpl, scriptManagedStart) {
		// The template carries no managed markers: everything is managed.
		managed = tpl
	}

	// The user block seed lives in the template's own user section, so a fresh
	// destination inherits it; the destination's block replaces it afterwards.
	user := sectionBetween(tpl, scriptUserStart, scriptUserEnd)

	if len(existing) > 0 {
		if strings.Contains(string(existing), scriptUserStart) || strings.Contains(string(existing), scriptManagedStart) {
			// Already-marked destination: preserve exactly its user block.
			user = sectionBetween(string(existing), scriptUserStart, scriptUserEnd)
		} else {
			// Legacy destination with no markers at all: keep everything.
			user = strings.TrimRight(string(existing), "\n")
		}
	}

	managed = strings.TrimRight(managed, "\n")
	user = strings.TrimRight(user, "\n")

	var b strings.Builder
	b.WriteString(scriptManagedStart)
	b.WriteString("\n")
	b.WriteString(managed)
	b.WriteString("\n")
	b.WriteString(scriptManagedEnd)
	b.WriteString("\n")
	b.WriteString(scriptUserStart)
	b.WriteString("\n")
	if user != "" {
		b.WriteString(user)
		b.WriteString("\n")
	}
	b.WriteString(scriptUserEnd)
	b.WriteString("\n")
	return []byte(b.String())
}

// dedupeUnmarkedCopy reports whether an unmarked legacy destination is
// byte-identical to the template's managed section (modulo surrounding
// blank lines). When a template gains markers, converged machines would
// otherwise migrate into managed + a full copy of the old template as
// "user content"; the provisioner treats that case as a fresh deploy
// instead, so the migration is clean and the user block starts empty.
func dedupeUnmarkedCopy(templateManaged, existing string) bool {
	return strings.Trim(existing, "\n") == strings.Trim(templateManaged, "\n")
}

// subtractManagedCopy excises the managed section from a legacy unmarked
// destination when it survives verbatim as one contiguous block (the common
// case: operator content appended or prepended around the template). The
// remainder is the true user content, so the first marked deploy migrates
// clean instead of duplicating. When the managed text was edited in place
// (no contiguous copy left), it reports false and the caller keeps the
// whole file — losing content would be worse than a one-time duplication
// that doctor --diff surfaces for a manual trim.
func subtractManagedCopy(managed, existing string) (string, bool) {
	managed = strings.Trim(managed, "\n")
	existing = strings.Trim(existing, "\n")
	if managed == "" {
		return existing, false
	}
	idx := strings.Index(existing, managed)
	if idx < 0 {
		return existing, false
	}
	rest := strings.Trim(existing[:idx]+existing[idx+len(managed):], "\n")
	return rest, true
}

// mergeJSONDeep recursively merges a JSON template over the live file with
// the destination winning every conflict: objects merge key by key, arrays
// and scalars keep the live value, template-only keys are added. Both sides
// must decode to JSON objects; anything else is an error and the caller
// keeps the live file untouched. Output keys sort alphabetically
// (encoding/json map order), so the file layout differs from the template —
// irrelevant, since merged files skip byte comparison by design.
func mergeJSONDeep(template, existing []byte) ([]byte, error) {
	var tpl, live map[string]any
	if err := json.Unmarshal(template, &tpl); err != nil {
		return nil, fmt.Errorf("template is not a JSON object: %w", err)
	}
	if err := json.Unmarshal(existing, &live); err != nil {
		return nil, fmt.Errorf("live file is not a JSON object: %w", err)
	}
	merged := deepMergeMaps(tpl, live)
	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func deepMergeMaps(tpl, live map[string]any) map[string]any {
	merged := make(map[string]any, len(tpl)+len(live))
	for k, tv := range tpl {
		merged[k] = tv
	}
	for k, lv := range live {
		tv, ok := tpl[k]
		if !ok {
			merged[k] = lv
			continue
		}
		tm, tmOK := tv.(map[string]any)
		lm, lmOK := lv.(map[string]any)
		if tmOK && lmOK {
			merged[k] = deepMergeMaps(tm, lm)
			continue
		}
		merged[k] = lv
	}
	return merged
}

// permissionsShape is the CommandCode settings contract the union merge
// understands. Anything else is not merged — the live file wins untouched.
type permissionsShape struct {
	Permissions struct {
		Allow []string `json:"allow"`
		Ask   []string `json:"ask"`
		Deny  []string `json:"deny"`
	} `json:"permissions"`
}

// mergePermissionsUnion merges the CommandCode settings shape by union:
// template entries and operator/agent-approved entries coexist, template
// order first, extras appended. Either side failing to decode to the shape
// is an error and the caller keeps the live file untouched with a warning.
func mergePermissionsUnion(template, existing []byte) ([]byte, error) {
	// Fail closed on shape: both sides must carry the permissions object,
	// otherwise "merging" would silently mint a new shape around user data.
	for name, data := range map[string][]byte{"template": template, "live file": existing} {
		var shape map[string]json.RawMessage
		if err := json.Unmarshal(data, &shape); err != nil {
			return nil, fmt.Errorf("%s is not a permissions settings file: %w", name, err)
		}
		raw, ok := shape["permissions"]
		if !ok {
			return nil, fmt.Errorf("%s is not a permissions settings file: missing permissions object", name)
		}
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(raw, &obj); err != nil {
			return nil, fmt.Errorf("%s is not a permissions settings file: permissions is not an object", name)
		}
	}
	var tpl, live permissionsShape
	if err := json.Unmarshal(template, &tpl); err != nil {
		return nil, fmt.Errorf("template is not a permissions settings file: %w", err)
	}
	if err := json.Unmarshal(existing, &live); err != nil {
		return nil, fmt.Errorf("live file is not a permissions settings file: %w", err)
	}
	live.Permissions.Allow = unionStrings(tpl.Permissions.Allow, live.Permissions.Allow)
	live.Permissions.Ask = unionStrings(tpl.Permissions.Ask, live.Permissions.Ask)
	live.Permissions.Deny = unionStrings(tpl.Permissions.Deny, live.Permissions.Deny)
	out, err := json.MarshalIndent(map[string]any{"permissions": live.Permissions}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func unionStrings(base, extra []string) []string {
	seen := make(map[string]bool, len(base)+len(extra))
	out := make([]string, 0, len(base)+len(extra))
	for _, s := range base {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	for _, s := range extra {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func mergeMarkersFor(mode entity.MergeMode) (managedStart, managedEnd string, ok bool) {
	switch mode {
	case entity.MergeMarkdownSections:
		return markdownManagedStart, markdownManagedEnd, true
	case entity.MergeScriptSections:
		return scriptManagedStart, scriptManagedEnd, true
	}
	return "", "", false
}

// freshMigrationTarget returns nil when the legacy destination carries no
// markers and is byte-identical to the template's managed section (modulo
// surrounding blank lines): the operator added nothing, so the migration
// starts clean instead of duplicating the whole template into the user
// block. Anything else passes through untouched.
func freshMigrationTarget(template, existing []byte, mode entity.MergeMode) []byte {
	managedStart, managedEnd, ok := mergeMarkersFor(mode)
	if !ok || len(existing) == 0 {
		return existing
	}
	if strings.Contains(string(existing), managedStart) {
		return existing
	}
	// Frontmatter is template-owned (the loader requires it at byte zero),
	// so only the body counts for the identity check.
	_, exBody := splitFrontmatter(string(existing))
	managed := sectionBetween(string(template), managedStart, managedEnd)
	if dedupeUnmarkedCopy(managed, exBody) {
		return nil
	}
	if rest, ok := subtractManagedCopy(managed, exBody); ok {
		return []byte(rest)
	}
	return existing
}

// applyConfigMerge runs the manifest-declared merge mode over the template
// with the live file. Modes that cannot fail (ssh, markdown, script)
// return a nil error; JSON modes fail on undecodable input and the caller
// keeps the live file untouched.
func applyConfigMerge(mode entity.MergeMode, template, existing []byte) ([]byte, error) {
	switch mode {
	case entity.MergeSSHHosts:
		return mergeSSHHosts(template, existing), nil
	case entity.MergeMarkdownSections:
		return mergeMarkdownSections(template, existing), nil
	case entity.MergeScriptSections:
		return mergeScriptSections(template, existing), nil
	case entity.MergeJSONDeps:
		return mergeJSONDeps(template, existing)
	case entity.MergeJSONDeep:
		return mergeJSONDeep(template, existing)
	case entity.MergePermissionsUnion:
		return mergePermissionsUnion(template, existing)
	default:
		return template, nil
	}
}
