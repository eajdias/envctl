package usecase

import (
	"strings"
)

// mergeKwinrcCompositing ensures the [Compositing] section of a KDE kwinrc
// carries the fullscreen compositing bypass (AllowBlockCompositing=true +
// UnredirectFullscreen=true), preserving every other section and key. It is
// idempotent: an already-configured file comes back unchanged. Keeping the
// merge scoped to those two keys is what makes provisioning safe — a blind
// overwrite of kwinrc would discard the user's desktop preferences.
func mergeKwinrcCompositing(data []byte) string {
	lines := strings.Split(string(data), "\n")
	out := make([]string, 0, len(lines)+3)
	inCompositing := false
	haveBlock := false
	haveUnredirect := false

	for _, raw := range lines {
		line := raw
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			// Leaving the section: flush any missing keys before the next one.
			if inCompositing {
				out = append(out, missingCompositingKeys(haveBlock, haveUnredirect)...)
			}
			inCompositing = trimmed == "[Compositing]"
			haveBlock, haveUnredirect = false, false
			out = append(out, line)
			continue
		}
		if inCompositing {
			switch trimmed {
			case "AllowBlockCompositing=true":
				haveBlock = true
			case "UnredirectFullscreen=true":
				haveUnredirect = true
			}
		}
		out = append(out, line)
	}
	if inCompositing {
		out = append(out, missingCompositingKeys(haveBlock, haveUnredirect)...)
	}

	merged := strings.Join(out, "\n")
	if !strings.Contains(merged, "[Compositing]") {
		merged += "\n\n[Compositing]\nAllowBlockCompositing=true\nUnredirectFullscreen=true\n"
	}
	return merged
}

// missingCompositingKeys returns the bypass keys the [Compositing] section
// still lacks, ready to append.
func missingCompositingKeys(haveBlock, haveUnredirect bool) []string {
	var missing []string
	if !haveBlock {
		missing = append(missing, "AllowBlockCompositing=true")
	}
	if !haveUnredirect {
		missing = append(missing, "UnredirectFullscreen=true")
	}
	return missing
}

// scxLoaderConfig renders the sched_ext loader config the validated host
// applies: bpfland in Auto mode. The scheduler stays with the package default
// when the file already exists — the merge is a full overwrite of a tiny
// file, but only through the privileged apply step.
func scxLoaderConfig() []byte {
	return []byte(`default_sched = "scx_bpfland"
default_mode = "Auto"
`)
}

// lactConfigFor renders the LACT GPU control config for a detected AMD
// device. The fan curve is the conservative Polaris profile the validated
// host runs (ramp to 70°C, aggressive 75°C+, ceiling 85°C); performance level
// stays auto so clocks are only raised when the workload asks. The device id
// varies per machine — it always comes from the sysfs probe, never from a
// hardcoded PCI id.
func lactConfigFor(deviceID string) []byte {
	return []byte(`version: 7
daemon:
  log_level: info
  admin_group: wheel
  disable_clocks_cleanup: false
apply_settings_timer: 5
current_profile: null
auto_switch_profiles: false
gpus:
  "` + deviceID + `":
    fan_control_enabled: true
    fan_control_settings:
      mode: curve
      temperature_key: edge
      interval_ms: 500
      curve:
        40: 0.2
        55: 0.35
        65: 0.55
        75: 0.8
        85: 1.0
    performance_level: auto
`)
}

// applyCmdlineParams adds the wanted kernel parameters to an existing cmdline
// line, keeping every parameter already present and the parameter order of
// the original. Parameters are matched by their full `key=value` or bare key
// form; a wanted list is idempotent (second call is a no-op).
func applyCmdlineParams(current string, wanted []string) string {
	already := strings.Fields(current)
	have := make(map[string]bool, len(already))
	for _, p := range already {
		have[p] = true
	}
	var missing []string
	for _, w := range wanted {
		if !have[w] {
			missing = append(missing, w)
		}
	}
	if len(missing) == 0 {
		return current
	}
	merged := make([]string, 0, len(already)+len(missing))
	merged = append(merged, already...)
	merged = append(merged, missing...)
	return strings.Join(merged, " ")
}

// edenURL returns the pinned Eden AppImage download URL, choosing the legacy
// build (pre-AVX2 CPUs) or the standard build by the host capability.
// Version and asset names are pinned: the validated host runs v0.2.1 legacy.
func edenURL(avx2 bool, version string) string {
	base := "https://git.eden-emu.dev/eden-emu/eden/releases/download/" + version + "/"
	if avx2 {
		return base + "Eden-Linux-v" + version + "-gcc-standard.AppImage"
	}
	return base + "Eden-Linux-v" + version + "-legacy-gcc-standard.AppImage"
}

// applyLimineCmdline applies the wanted kernel parameters to every
// KERNEL_CMDLINE entry of a /etc/default/limine file. Limine entries look
// like `KERNEL_CMDLINE[default]+="<cmdline>"`; each quoted cmdline is merged
// with applyCmdlineParams and re-quoted in place. Entries already carrying
// the wanted parameters come back unchanged, so the file is only rewritten
// when something actually changes.
func applyLimineCmdline(data []byte, wanted []string) []byte {
	lines := strings.Split(string(data), "\n")
	changed := false
	for i, line := range lines {
		if !strings.Contains(line, "KERNEL_CMDLINE") {
			continue
		}
		openIdx := strings.Index(line, "\"")
		if openIdx < 0 {
			continue // malformed entry; leave it alone
		}
		closeIdx := strings.LastIndex(line, "\"")
		if closeIdx <= openIdx {
			continue
		}
		quoted := line[openIdx+1 : closeIdx]
		merged := applyCmdlineParams(quoted, wanted)
		if merged == quoted {
			continue
		}
		lines[i] = line[:openIdx+1] + merged + line[closeIdx:]
		changed = true
	}
	if !changed {
		return data
	}
	return []byte(strings.Join(lines, "\n"))
}