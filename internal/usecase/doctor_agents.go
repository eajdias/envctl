package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
)

// 4. Audit OpenCode Global Rules (AGENTS.md)
func (uc *DoctorAuditUseCase) auditAGENTSRules(addDiag func(entity.Diagnostic)) {
	globalAgentsPath := filepath.Join("~/.config/opencode/AGENTS.md")
	if !uc.fsManager.Exists(globalAgentsPath) {
		addDiag(entity.Warn(
			"OpenCode",
			"AGENTS.md (global rules)",
			fmt.Sprintf("Global rules file missing (%s) - opencode loads rules from this path, not ~/AGENTS.md", globalAgentsPath),
			"run 'envctl run shell'",
		))
	} else {
		addDiag(entity.OK(
			"OpenCode",
			"AGENTS.md (global rules)",
			"Global rules present at ~/.config/opencode/AGENTS.md",
		))
	}
}

// 4.5. Audit OpenCode MCP file references: opencode fails hard at
// startup when a `{file:...}` reference points to a missing file
// (e.g. a secrets key that was never created on this machine), so a
// "present on disk" opencode.json is not enough — every referenced
// file must exist too.
func (uc *DoctorAuditUseCase) auditOpenCodeMCPRefs(ctx context.Context, addDiag func(entity.Diagnostic)) {
	//nolint:errcheck // audit continues with an empty list when the manifest fails to load.
	configFiles, _ := uc.manifestRepo.LoadConfigFiles()
	uc.auditOpenCodeFileRefs(addDiag)
	uc.auditRemovedMCPEntries(addDiag)
	uc.auditAgentsIdentityCoverage(addDiag, configFiles)
	uc.auditOpenCodeConfigShape(addDiag)
	uc.auditOpenCodeVersionSkew(ctx, addDiag)
	uc.auditEnvctlFreshness(ctx, addDiag)
}

// 6. Audit Skills (only the ones that belong on this OS and are enabled).
// Both agents share the validator: a skill whose frontmatter the loader
// rejects is silently ignored at runtime, so an existence-only check would
// report a healthy tree while the agent sees nothing.
func (uc *DoctorAuditUseCase) auditSkills(addDiag func(entity.Diagnostic)) {
	if skillsDir, expandErr := uc.fsManager.ExpandUserPath("~/.config/opencode/skills"); expandErr == nil {
		uc.auditSkillTree("Skills", skillsDir, addDiag)
		uc.auditSkillsContentParity("Skills", skillsDir, addDiag)
		if catalogSkills, err := uc.manifestRepo.LoadSkills(); err == nil {
			uc.auditSkillCatalogBudget(catalogSkills, addDiag)
		}
	}
}

// 13. Audit CommandCode Agent Health
func (uc *DoctorAuditUseCase) auditCommandCodeHealth(addDiag func(entity.Diagnostic)) {
	if runtime.GOOS == "linux" || runtime.GOOS == "windows" {
		var cmdPath string
		var cmdErr error
		if runtime.GOOS == "windows" {
			// On Windows bare `cmd` resolves to System32\cmd.exe even when
			// CommandCode is absent, so only cmdc/command-code prove install.
			cmdPath, cmdErr = exec.LookPath("cmdc")
			if cmdErr != nil {
				cmdPath, cmdErr = exec.LookPath("command-code")
			}
			if cmdErr != nil {
				cmdPath, cmdErr = exec.LookPath("commandcode")
			}
		} else {
			cmdPath, cmdErr = exec.LookPath("cmd")
			if cmdErr != nil {
				cmdPath, cmdErr = exec.LookPath("cmdc")
			}
		}
		if cmdErr != nil {
			fixHint := "Run 'envctl run bootstrap' or 'npm install -g command-code'"
			if runtime.GOOS == "windows" {
				fixHint = "Run 'envctl run packages' or 'npm install -g command-code'"
			}
			addDiag(entity.Warn(
				"CommandCode",
				"CommandCode CLI",
				"CommandCode CLI not found in PATH",
				fixHint,
			))
		} else {
			addDiag(entity.OK(
				"CommandCode",
				"CommandCode CLI",
				fmt.Sprintf("Found at %s", cmdPath),
			))
		}

		//nolint:errcheck // an unresolvable home yields an empty path and the check below is skipped.
		ccConfigDir, _ := uc.fsManager.ExpandUserPath("~/.commandcode")
		if uc.fsManager.Exists(ccConfigDir) {
			addDiag(entity.OK(
				"CommandCode",
				"~/.commandcode/",
				"Config directory exists",
			))

			mcpPath := filepath.Join(ccConfigDir, "mcp.json")
			if uc.fsManager.Exists(mcpPath) {
				data, readErr := os.ReadFile(mcpPath)
				switch {
				case readErr != nil:
					addDiag(entity.Warn(
						"CommandCode",
						"MCP config",
						fmt.Sprintf("mcp.json unreadable: %v", readErr),
						"Run 'envctl run shell' to re-provision CommandCode configs",
					))
				case !json.Valid(data):
					addDiag(entity.Warn(
						"CommandCode",
						"MCP config",
						"mcp.json is not valid JSON — CommandCode cannot load the servers",
						"Run 'envctl run shell' to re-provision CommandCode configs",
					))
				default:
					addDiag(entity.OK(
						"CommandCode",
						"MCP config",
						"mcp.json present and valid JSON",
					))
				}
			} else {
				addDiag(entity.Warn(
					"CommandCode",
					"MCP config",
					"mcp.json not found",
					"Run 'envctl run shell' to provision CommandCode configs",
				))
			}

			settingsPath := filepath.Join(ccConfigDir, "settings.json")
			if uc.fsManager.Exists(settingsPath) {
				data, readErr := os.ReadFile(settingsPath)
				switch {
				case readErr != nil:
					addDiag(entity.Warn(
						"CommandCode",
						"Settings",
						fmt.Sprintf("settings.json unreadable: %v", readErr),
						"Run 'envctl run shell' to re-provision CommandCode configs",
					))
				case !json.Valid(data):
					addDiag(entity.Warn(
						"CommandCode",
						"Settings",
						"settings.json is not valid JSON — CommandCode refuses an invalid settings file",
						"Run 'envctl run shell' to re-provision CommandCode configs",
					))
				default:
					addDiag(entity.OK(
						"CommandCode",
						"Settings",
						"settings.json present and valid JSON",
					))
				}
			}

			uc.auditCommandCodeAgents(ccConfigDir, addDiag)

			uc.auditSkillTree("CommandCode", filepath.Join(ccConfigDir, "skills"), addDiag)
			uc.auditSkillsContentParity("CommandCode", filepath.Join(ccConfigDir, "skills"), addDiag)
		} else {
			addDiag(entity.Info(
				"CommandCode",
				"~/.commandcode/",
				"Config directory not yet created (CommandCode not provisioned)",
			))
		}
	}
}

var openCodeFileRefPattern = regexp.MustCompile(`\{file:([^}]+)\}`)

// auditOpenCodeFileRefs resolves every `{file:...}` reference found in the
// deployed opencode.json and reports the ones pointing at missing files.
// opencode aborts with "Configuration is invalid: bad file reference" when
// any of them is absent, which breaks EVERY command (even `opencode models`),
// while a plain "file present" check on opencode.json stays green — exactly
// the blind spot that left a VPS with a dead opencode and a passing doctor.

func (uc *DoctorAuditUseCase) auditOpenCodeFileRefs(addDiag func(entity.Diagnostic)) {
	rawPath := "~/.config/opencode/opencode.json"
	expanded, err := uc.fsManager.ExpandUserPath(rawPath)
	if err != nil {
		return
	}
	data, err := os.ReadFile(expanded)
	if err != nil {
		return
	}
	matches := openCodeFileRefPattern.FindAllSubmatch(data, -1)
	if len(matches) == 0 {
		return
	}
	seen := map[string]bool{}
	var missing []string
	for _, m := range matches {
		ref := strings.TrimSpace(string(m[1]))
		if ref == "" || seen[ref] {
			continue
		}
		seen[ref] = true
		resolved, err := uc.fsManager.ExpandUserPath(ref)
		if err != nil {
			resolved = ref
		}
		//nolint:gosec // G703: resolved derives from ExpandUserPath of a manifest-controlled ref.
		if _, statErr := os.Stat(resolved); statErr != nil {
			missing = append(missing, ref)
		}
	}
	if len(missing) > 0 {
		addDiag(entity.Error(
			"OpenCode",
			"Config file references",
			fmt.Sprintf("opencode.json references %d missing file(s): %s — opencode refuses to start until they exist", len(missing), strings.Join(missing, "; ")),
			"create the missing file(s) or run 'envctl run shell' to re-provision opencode.json",
		))
		return
	}
	addDiag(entity.OK(
		"OpenCode",
		"Config file references",
		fmt.Sprintf("%d {file:...} reference(s) resolve on disk", len(seen)),
	))
}

// auditOpenCodeVersionSkew warns when the installed opencode CLI predates the
// V2-native config the repo deploys (review the 2026-09-22 migration): a v1
// binary discards unknown keys (strict()), inverts MCP flags and fails V2
// plugins with LoadError, while doctor would otherwise stay green.

func (uc *DoctorAuditUseCase) auditOpenCodeVersionSkew(_ context.Context, addDiag func(entity.Diagnostic)) {
	resolved, err := resolveOnToolchainPath("opencode")
	if err != nil {
		return
	}
	out, err := runWithToolchain(context.Background(), resolved, "--version")
	if err != nil {
		return
	}
	version := firstVersionToken(strings.TrimSpace(out))
	if version == "" {
		return
	}
	if !versionMajorAtLeast(version, 2) {
		addDiag(entity.Warn(
			"OpenCode",
			"Version skew",
			fmt.Sprintf("opencode v%s + V2 config (agents/permissions/plugins/skills/mcp.servers): v1 discards keys, inverts MCP flags and fails V2 plugins", version),
			"upgrade opencode to 2.x (envctl run providers), then 'opencode debug config'",
		))
	}
}

// auditEnvctlFreshness warns when the running binary predates the repo checkout
// it would provision from. The templates are embedded at build time, so a stale
// binary provisions configs from as many releases ago as it is old — silently.
// The audit only runs when a repo checkout is reachable; a binary outside a
// checkout cannot be compared and is left alone (not an error, just no source).

var removedMCPEntries = []string{"zscan"}

// auditRemovedMCPEntries reports deployed agent configs that still declare
// MCP servers envctl removed (see removedMCPEntries). The generic ConfigFile
// drift check would only say "content diverges"; naming the stale entry tells
// the user exactly what to delete.

func (uc *DoctorAuditUseCase) auditRemovedMCPEntries(addDiag func(entity.Diagnostic)) {
	targets := []struct {
		path    string
		section string
		system  string
	}{
		{"~/.config/opencode/opencode.json", "mcp", "OpenCode"},
		{"~/.commandcode/mcp.json", "mcpServers", "CommandCode"},
	}
	for _, tgt := range targets {
		expanded, err := uc.fsManager.ExpandUserPath(tgt.path)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(expanded)
		if err != nil {
			continue
		}
		var root map[string]json.RawMessage
		if err := json.Unmarshal(data, &root); err != nil {
			continue
		}
		var servers map[string]json.RawMessage
		if err := json.Unmarshal(root[tgt.section], &servers); err != nil {
			continue
		}
		var stale []string
		for _, name := range removedMCPEntries {
			if _, ok := servers[name]; ok {
				stale = append(stale, name)
			}
		}
		if len(stale) > 0 {
			addDiag(entity.Warn(
				tgt.system,
				"Removed MCP entries",
				fmt.Sprintf("%s still declares removed MCP server(s): %s — envctl no longer provisions them", tgt.path, strings.Join(stale, ", ")),
				"run 'envctl run shell' to re-provision, or delete the stale block(s) by hand",
			))
		}
	}
}

// auditAgentsIdentityCoverage warns when no AGENTS.md manifest variant matches
// this host (e.g. an unmatched distro after the debian/ubuntu + arch/cachyos
// split): provisioning would silently skip the global rules file and doctor
// would otherwise stay green.

func (uc *DoctorAuditUseCase) auditAgentsIdentityCoverage(addDiag func(entity.Diagnostic), configFiles []entity.ConfigFile) {
	matched := false
	for _, cf := range configFiles {
		if !strings.HasSuffix(cf.Destination, "AGENTS.md") {
			continue
		}
		if entity.MatchesOS(cf.OS) {
			matched = true
			break
		}
	}
	if !matched {
		addDiag(entity.Warn(
			"OpenCode",
			"AGENTS.md (identity coverage)",
			"no AGENTS.md manifest variant matches this OS/distro — global rules would not be provisioned",
			"add a matching manifest variant or report the distro gap",
		))
	}
}

func (uc *DoctorAuditUseCase) auditCommandCodeAgents(ccConfigDir string, addDiag func(entity.Diagnostic)) {
	agentsDir := filepath.Join(ccConfigDir, "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return
	}

	var blocking, advisories []string
	valid := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		id := strings.TrimSuffix(name, ".md")
		if commandCodeReservedAgentNames[id] {
			continue
		}

		content, readErr := os.ReadFile(filepath.Join(agentsDir, name))
		if readErr != nil {
			blocking = append(blocking, name+" (unreadable)")
			continue
		}
		block, blockOK := skillFrontmatterBlock(content)
		fm, fmOK := parseSkillFrontmatter(content)
		if !blockOK || !fmOK || strings.TrimSpace(fm.Name) != id {
			blocking = append(blocking, name+" (frontmatter 'name' missing or different from the filename)")
			continue
		}
		for _, issue := range validateCommandCodeAgentFrontmatter(block) {
			problem := fmt.Sprintf("%s (%s: %s)", name, issue.Field, issue.Problem)
			if issue.Status == entity.DiagWarning {
				blocking = append(blocking, problem)
				continue
			}
			advisories = append(advisories, problem)
		}
		valid++
	}

	if len(blocking) > 0 {
		addDiag(entity.Warn(
			"CommandCode",
			"Agents",
			fmt.Sprintf("%d agent problem(s) that block loading or silently change behavior: %s", len(blocking), strings.Join(blocking, "; ")),
			"Fix the reported frontmatter fields (or remove the stale file), then run 'envctl commandcode'",
		))
		return
	}

	if len(advisories) > 0 {
		addDiag(entity.Info(
			"CommandCode",
			"Agents",
			fmt.Sprintf("%d note(s) on otherwise valid agents: %s", len(advisories), strings.Join(advisories, "; ")),
			"informational: compare with the installed CommandCode version before changing anything",
		))
		return
	}

	addDiag(entity.OK(
		"CommandCode",
		"Agents",
		fmt.Sprintf("%d custom agent definition(s) valid", valid),
	))
}

// auditSkillsContentParity verifies every embedded skill source file
// byte-for-byte against the deployed tree. The frontmatter audit cannot see
// content drift (a stale deploy or a hand edit with valid frontmatter passes
// it), so without this a tree that never converged stays green. Files the
// sources don't declare (backups, local additions) are ignored: only the
// managed set is audited.
func (uc *DoctorAuditUseCase) auditSkillsContentParity(system, skillsDir string, addDiag func(entity.Diagnostic)) {
	if !uc.fsManager.Exists(skillsDir) {
		return // the tree check owns a missing directory
	}
	skills, err := uc.manifestRepo.LoadSkills()
	if err != nil {
		return // manifest unreadable: its own loader diagnostics own that failure
	}
	targets := make(map[string]string, len(skills))
	for _, s := range skills {
		if !s.Enabled || !s.AppliesToOS(runtime.GOOS) {
			continue
		}
		dir := s.Name
		if s.TargetDir != "" {
			dir = s.TargetDir
		}
		targets[s.Name] = dir
	}
	var bad []string
	checked := 0
	walkErr := uc.manifestRepo.WalkSkillSources(runtime.GOOS, func(skill, rel string, data []byte) error {
		checked++
		//nolint:gosec // G703: skill/rel come from the embedded manifest walk.
		live, err := os.ReadFile(filepath.Join(skillsDir, targets[skill], rel))
		if err != nil || !bytes.Equal(live, data) {
			bad = append(bad, skill+"/"+rel)
		}
		return nil
	})
	if walkErr != nil {
		return
	}
	if len(bad) > 0 {
		shown := bad
		suffix := ""
		if len(bad) > 5 {
			shown = bad[:5]
			suffix = fmt.Sprintf(" and %d more", len(bad)-5)
		}
		addDiag(entity.Warn(
			system,
			"Skills content",
			fmt.Sprintf("%d of %d skill file(s) differ from the embedded sources: %s%s", len(bad), checked, strings.Join(shown, ", "), suffix),
			"run 'envctl run skills' to re-sync",
		))
		return
	}
	addDiag(entity.OK(
		system,
		"Skills content",
		fmt.Sprintf("%d skill file(s) match the embedded sources", checked),
	))
}

// auditSkillTree validates one agent's deployed skills: presence first, then the
// frontmatter its loader reads. Both agents share this implementation because a
// skill whose frontmatter is rejected is silently ignored at runtime — an
// existence-only check would call that tree healthy.

func (uc *DoctorAuditUseCase) auditSkillTree(system, skillsDir string, addDiag func(entity.Diagnostic)) {
	if skillsDir == "" {
		return
	}
	if !uc.fsManager.Exists(skillsDir) {
		addDiag(entity.Warn(
			system,
			"Skills",
			"Skills directory not found",
			"Run 'envctl run skills' to deploy skills",
		))
		return
	}

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		addDiag(entity.Warn(
			system,
			"Skills",
			fmt.Sprintf("Skills directory unreadable: %v", err),
			"Check the directory permissions, then run 'envctl run skills'",
		))
		return
	}

	deployed := 0
	var rejected []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		name := entry.Name()
		content, readErr := os.ReadFile(filepath.Join(skillsDir, name, "SKILL.md"))
		if readErr != nil {
			rejected = append(rejected, name+" (SKILL.md missing)")
			continue
		}
		fm, ok := parseSkillFrontmatter(content)
		if !ok {
			rejected = append(rejected, name+" (frontmatter missing or invalid YAML)")
			continue
		}
		if vErr := validateSkillFrontmatter(name, fm); vErr != nil {
			rejected = append(rejected, fmt.Sprintf("%s (%v)", name, vErr))
			continue
		}
		deployed++
	}

	total := deployed + len(rejected)
	switch {
	case len(rejected) > 0:
		addDiag(entity.Warn(
			system,
			"Skills",
			fmt.Sprintf("%d of %d deployed skills will not load: %s", len(rejected), total, strings.Join(rejected, "; ")),
			"Fix the SKILL.md frontmatter (name must match the directory, description must be non-empty), then run 'envctl run skills'",
		))
	case deployed > 0:
		addDiag(entity.OK(
			system,
			"Skills",
			fmt.Sprintf("%d skills deployed, frontmatter valid", deployed),
		))
	default:
		addDiag(entity.Warn(
			system,
			"Skills",
			"Skills directory exists but holds no skill",
			"Run 'envctl run skills' to deploy the manifest skills",
		))
	}

	if total == 0 {
		return
	}
	manifestSkills, err := uc.manifestRepo.LoadSkills()
	if err != nil {
		return
	}
	expected := 0
	for _, s := range manifestSkills {
		if s.Enabled && s.AppliesToOS(runtime.GOOS) {
			expected++
		}
	}
	if expected > 0 && total != expected {
		addDiag(entity.Warn(
			system,
			"Skills",
			fmt.Sprintf("%d skills deployed but the manifest declares %d", total, expected),
			"Run 'envctl run skills' to re-sync the deployed skills with the manifest",
		))
	}
}
