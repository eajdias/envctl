package usecase

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/eajdias/envctl/internal/infra/executil"
)

// mcpShimPatch maps one exact bare fragment of a provisioned MCP config to
// its absolute replacement. Fragments match literally (byte-exact, LF) so the
// deployed file keeps every other template byte intact — the same
// textual-overlay precedent as withWindowsShellOverlay.
type mcpShimPatch struct {
	old string
	new string
}

// withMCPShimOverlay rewrites bare MCP launcher fragments (mise shim names,
// bunx/npx launcher + package specs) to absolute mise-shim paths at deploy
// time, so agent child processes resolve without depending on the spawning
// process's PATH age. Rules: CRLF in, LF out; a patch applies only on an
// exact single match (ambiguous or absent fragments are skipped, never
// guessed); output must stay valid JSON or the input returns untouched.
// The function is pure and idempotent: a second run finds no bare token.
func withMCPShimOverlay(content []byte, patches []mcpShimPatch) []byte {
	// Same checkout normalization as withWindowsShellOverlay: a Windows
	// checkout carries CRLF while the fragments below are LF.
	content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	if len(patches) == 0 {
		return content
	}
	patched := content
	for _, p := range patches {
		if p.old == "" || strings.Count(string(patched), p.old) != 1 {
			continue
		}
		patched = bytes.ReplaceAll(patched, []byte(p.old), []byte(p.new))
	}
	if !json.Valid(patched) {
		return content
	}
	return patched
}

// mcpShimServer describes one mise-managed MCP server: the shim binary name
// and the exact bare fragments it replaces in each runtime's config.
type mcpShimServer struct {
	bin         string
	openCodeOld string
	cmdCodeOld  string
	cmdCodeArgs string
}

// mcpShimServers is the closed set of local MCP servers both runtimes
// provision. Versions are pinned in the templates + packages manifest and
// repeated verbatim in the fragments below — a bump touches all three, and
// TestMCPShimOverlayOnShippedTemplates fails until the table follows.
var mcpShimServers = []mcpShimServer{
	{
		bin:         "brave-search-mcp-server",
		openCodeOld: `"bunx",` + "\n" + `          "@brave/brave-search-mcp-server@2.1.4",`,
		cmdCodeOld:  `"command": "npx",` + "\n" + `      "args": ["-y", "@brave/brave-search-mcp-server@2.1.4", "--transport", "stdio"]`,
		cmdCodeArgs: `"args": ["--transport", "stdio"]`,
	},
	{
		bin:         "chrome-devtools-mcp",
		openCodeOld: `"bunx",` + "\n" + `          "chrome-devtools-mcp@1.8.0",`,
		cmdCodeOld:  `"command": "bunx",` + "\n" + `      "args": ["chrome-devtools-mcp@1.8.0", "--no-usage-statistics"]`,
		cmdCodeArgs: `"args": ["--no-usage-statistics"]`,
	},
	{
		bin:         "mcp-ssh-manager",
		openCodeOld: `"mcp-ssh-manager"`,
		cmdCodeOld:  `"command": "mcp-ssh-manager"`,
		cmdCodeArgs: "",
	},
}

// mcpShimPatches builds the deploy-time patch table for one MCP config file.
// shimDir is the machine's mise shims dir (executil.MiseShimDir) and
// exeSuffix is ".exe" on Windows, "" elsewhere. Paths deploy with forward
// slashes: valid JSON without escaping and resolvable by direct process
// spawn on both OSes. A server whose shim is absent (fresh machine before
// `run mise`) contributes no patch — the bare template fragment stays for
// the doctor package checks to report.
func mcpShimPatches(shimDir, exeSuffix, configID string) []mcpShimPatch {
	commandcode := configID == "commandcode_mcp"
	opencode := configID == "opencode_config" || configID == "opencode_config_linux"
	if !commandcode && !opencode {
		return nil
	}
	var patches []mcpShimPatch
	for _, s := range mcpShimServers {
		abs := filepath.Join(shimDir, s.bin+exeSuffix)
		if !executil.IsExecutableFile(abs) {
			continue
		}
		deployed := filepath.ToSlash(abs)
		if commandcode {
			patches = append(patches, mcpShimPatch{old: s.cmdCodeOld, new: commandcodeCommand(deployed, s.cmdCodeArgs)})
		} else {
			// Keep the trailing comma iff the bare fragment has one
			// (single-element arrays like ssh-manager carry none — a stray
			// comma is invalid JSON and the validity gate would drop the patch).
			suffix := ""
			if strings.HasSuffix(s.openCodeOld, ",") {
				suffix = ","
			}
			patches = append(patches, mcpShimPatch{old: s.openCodeOld, new: `"` + deployed + `"` + suffix})
		}
	}
	return patches
}

// deployedMCPConfig applies the deploy-time MCP shim transform shared by
// provisioning and the doctor drift comparison for one config file
// ("opencode_config", "opencode_config_linux", "commandcode_mcp"). Both
// call sites must use it — a one-sided edit silently reintroduces permanent
// drift WARN. TestDeployedMCPConfigParity locks the helper behavior
// end-to-end; keeping both call sites on it is a review-time invariant
// (grep deployedMCPConfig must show exactly the two call sites).
func deployedMCPConfig(content []byte, configID string) []byte {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return content
	}
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	return withMCPShimOverlay(content, mcpShimPatches(executil.MiseShimDir(home), suffix, configID))
}

// commandcodeCommand rebuilds a `"command": ..., "args": ...` pair with the
// absolute shim, dropping the launcher-only head (npx `-y`, package specs)
// while keeping the server's own args verbatim.
func commandcodeCommand(deployed, args string) string {
	if args == "" {
		return `"command": "` + deployed + `"`
	}
	return `"command": "` + deployed + `",` + "\n" + `      ` + args
}
