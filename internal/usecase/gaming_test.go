package usecase

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/infra/filesystem"
)

func TestMergeKwinrcCompositing(t *testing.T) {
	// No [Compositing] section at all: the merge adds it with both keys.
	basic := "[General]\nTheme=default\n"
	got := mergeKwinrcCompositing([]byte(basic))
	if !strings.Contains(got, "[Compositing]") {
		t.Fatalf("expected a [Compositing] section to be added, got:\n%s", got)
	}
	if !strings.Contains(got, "AllowBlockCompositing=true") ||
		!strings.Contains(got, "UnredirectFullscreen=true") {
		t.Errorf("expected both compositing keys, got:\n%s", got)
	}
	if !strings.Contains(got, "[General]") || !strings.Contains(got, "Theme=default") {
		t.Errorf("expected the existing section to be preserved, got:\n%s", got)
	}

	// Section present, keys missing: only the keys are added.
	partial := "[Compositing]\nOpenGLIsUnsafe=false\n"
	got = mergeKwinrcCompositing([]byte(partial))
	if !strings.Contains(got, "OpenGLIsUnsafe=false") {
		t.Errorf("expected existing [Compositing] keys to be preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "AllowBlockCompositing=true") ||
		!strings.Contains(got, "UnredirectFullscreen=true") {
		t.Errorf("expected both keys to be added, got:\n%s", got)
	}

	// Already fully configured: idempotent no-op.
	done := "[Compositing]\nAllowBlockCompositing=true\nUnredirectFullscreen=true\n"
	if got := mergeKwinrcCompositing([]byte(done)); got != done {
		t.Errorf("expected a no-op on an already-configured file, got:\n%s", got)
	}

	// Empty input still yields a valid section.
	got = mergeKwinrcCompositing(nil)
	if !strings.Contains(got, "[Compositing]") {
		t.Errorf("expected a section even on empty input, got:\n%s", got)
	}
}

func TestScxLoaderConfig(t *testing.T) {
	got := string(scxLoaderConfig())
	if !strings.Contains(got, `default_sched = "scx_bpfland"`) {
		t.Errorf("expected bpfland scheduler, got:\n%s", got)
	}
	if !strings.Contains(got, `default_mode = "Auto"`) {
		t.Errorf("expected Auto mode, got:\n%s", got)
	}
}

func TestLactConfigFor(t *testing.T) {
	got := string(lactConfigFor("1002:6FDF-1002:0B31-0000:01:00.0"))
	if !strings.Contains(got, "1002:6FDF-1002:0B31-0000:01:00.0") {
		t.Errorf("expected the device id in the config, got:\n%s", got)
	}
	if !strings.Contains(got, "fan_control_enabled: true") {
		t.Errorf("expected fan control enabled, got:\n%s", got)
	}
	if !strings.Contains(got, "performance_level: auto") {
		t.Errorf("expected auto performance level, got:\n%s", got)
	}
	if !strings.Contains(got, "85: 1.0") {
		t.Errorf("expected the fan curve ceiling, got:\n%s", got)
	}
	// The config must parse as YAML (the LACT daemon reads strict YAML).
	var probe map[string]interface{}
	if err := yaml.Unmarshal([]byte(got), &probe); err != nil {
		t.Fatalf("LACT config is not valid YAML: %v\n%s", err, got)
	}
}

func TestApplyCmdlineParams(t *testing.T) {
	wanted := []string{"preempt=full", "split_lock_detect=off", "zswap.enabled=0"}

	// Already present: no-op.
	done := "quiet nowatchdog preempt=full split_lock_detect=off zswap.enabled=0"
	if got := applyCmdlineParams(done, wanted); got != done {
		t.Errorf("expected a no-op on the full cmdline, got %q", got)
	}

	// Missing some: only the missing ones are appended.
	partial := "quiet nowatchdog preempt=full"
	got := applyCmdlineParams(partial, wanted)
	for _, w := range wanted {
		if !strings.Contains(got, w) {
			t.Errorf("expected %s in the merged cmdline, got %q", w, got)
		}
	}
	if got != "quiet nowatchdog preempt=full split_lock_detect=off zswap.enabled=0" {
		t.Errorf("unexpected merge output: %q", got)
	}

	// Empty current: built from scratch.
	if got := applyCmdlineParams("", wanted); got != "preempt=full split_lock_detect=off zswap.enabled=0" {
		t.Errorf("unexpected build from empty: %q", got)
	}
}

func TestEdenURL(t *testing.T) {
	legacy := edenURL(false, "v0.2.1")
	if !strings.Contains(legacy, "legacy-gcc-standard") {
		t.Errorf("expected the legacy build for CPUs without AVX2, got %q", legacy)
	}
	standard := edenURL(true, "v0.2.1")
	if !strings.Contains(standard, "-gcc-standard.AppImage") || strings.Contains(standard, "legacy") {
		t.Errorf("expected the standard build for AVX2 CPUs, got %q", standard)
	}
}

func TestParsePCIIDFromUevent(t *testing.T) {
	// The real host uevent: LACT v7 keys GPUs by the full id, not the bare
	// PCI_ID — the subsys and slot must be appended or the curve stays inert.
	full := "PCI_CLASS=30000\nPCI_ID=1002:6FDF\nPCI_SUBSYS_ID=1002:0B31\nPCI_SLOT_NAME=0000:01:00.0\n"
	if got := parsePCIIDFromUevent(full, "fallback"); got != "1002:6FDF-1002:0B31-0000:01:00.0" {
		t.Errorf("expected the full LACT device id, got %q", got)
	}
	// Missing fields: whatever is available, with the fallback for a bare file.
	if got := parsePCIIDFromUevent("PCI_ID=1002:6FDF\n", "fallback"); got != "1002:6FDF" {
		t.Errorf("expected the PCI id when subsys/slot are absent, got %q", got)
	}
	if got := parsePCIIDFromUevent("DRIVER=amdgpu\n", "fallback"); got != "fallback" {
		t.Errorf("expected the fallback when no PCI_ID, got %q", got)
	}
}

func TestApplyLimineCmdline(t *testing.T) {
	wanted := []string{"preempt=full", "zswap.enabled=0"}

	// One entry missing both params: merged in place, other lines untouched.
	input := "KERNEL_CMDLINE[default]+=\"quiet nowatchdog zswap.enabled=0\"\n# comment\nKERNEL_CMDLINE[fallback]+=\"quiet\"\n"
	got := applyLimineCmdline([]byte(input), wanted)
	merged := string(got)
	for _, w := range wanted {
		if !strings.Contains(merged, w) {
			t.Errorf("expected %s in the merged file, got:\n%s", w, merged)
		}
	}
	if !strings.Contains(merged, "quiet nowatchdog zswap.enabled=0 preempt=full") {
		t.Errorf("expected the missing param appended to the default entry, got:\n%s", merged)
	}
	if !strings.Contains(merged, "KERNEL_CMDLINE[fallback]+=\"quiet preempt=full zswap.enabled=0\"") {
		t.Errorf("expected the fallback entry merged too, got:\n%s", merged)
	}
	if !strings.Contains(merged, "# comment") {
		t.Errorf("expected comments to be preserved, got:\n%s", merged)
	}

	// Already fully configured: byte-identical no-op.
	done := "KERNEL_CMDLINE[default]+=\"quiet preempt=full zswap.enabled=0\"\n"
	if got := applyLimineCmdline([]byte(done), wanted); string(got) != done {
		t.Errorf("expected a no-op on a configured file, got:\n%s", got)
	}

	// Malformed entry without quotes is left alone, not crashed on.
	malformed := "KERNEL_CMDLINE[default]+=quiet\n"
	if got := applyLimineCmdline([]byte(malformed), wanted); string(got) != malformed {
		t.Errorf("expected a malformed entry to be untouched, got:\n%s", got)
	}
}

func TestEdenSmokeFailsOnIllegalInstruction(t *testing.T) {
	if runtime.GOOS != "linux" || !entity.MatchesOS("arch,cachyos") {
		t.Skip("gaming presence gate only resolves on Arch/CachyOS")
	}
	uc := NewProvisionGamingTuningUseCase(filesystem.NewFileSystemManager(), &mockLogger{})
	// A script that prints the AVX2 fingerprint and exits 132 is what a
	// wrong-build AppImage looks like to the smoke test.
	fake := filepath.Join(t.TempDir(), "eden.SIGILL")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\necho 'Illegal instruction (core dumped)' >&2\nexit 132\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := uc.smokeEden(context.Background(), fake)
	if err == nil || !strings.Contains(err.Error(), "not executable on this CPU") {
		t.Errorf("expected an illegal-instruction failure to be named, got %v", err)
	}
}

func TestEdenSmokeAcceptsARunningBuild(t *testing.T) {
	if runtime.GOOS != "linux" || !entity.MatchesOS("arch,cachyos") {
		t.Skip("gaming presence gate only resolves on Arch/CachyOS")
	}
	uc := NewProvisionGamingTuningUseCase(filesystem.NewFileSystemManager(), &mockLogger{})
	fake := filepath.Join(t.TempDir(), "eden.run")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nprintf 'Eden v0.2.1'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := uc.smokeEden(context.Background(), fake); err != nil {
		t.Errorf("expected a healthy build to pass the smoke test, got %v", err)
	}
}


func TestCpuHasAVX2(t *testing.T) {
	withAVX2 := "processor\t: 0\nvendor_id\t: GenuineIntel\nflags\t\t: fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush dts acpi mmx fxsr sse sse2 ss ht tm pbe syscall nx pdpe1gb rdtscp lm constant_tsc art arch_perfmon pebs bts rep_good nopl xtopology nonstop_tsc cpuid aperfmperf pni pclmulqdq dtes64 monitor ds_cpl vmx est tm2 ssse3 sdbg fma cx16 xtpr pdcm pcid sse4_1 sse4_2 x2apic movbe popcnt tsc_deadline_timer aes xsave avx f16c rdrand lahf_lm abm 3dnowprefetch cpuid_fault epb invpcid_single ssbd ibrs ibpb stibp ibrs_enhanced tpr_shadow vnmi flexpriority ept vpid ept_ad fsgsbase tsc_adjust bmi1 avx2 smep bmi2 erms invpcid avx512f avx512dq rdseed adx smap avx512ifma clflushopt clwb intel_pt avx512cd sha_ni avx512bw avx512vl xsaveopt xsavec xgetbv1 xsaves split_lock_detect dtherm ida arat pln pts hwp hwp_act_window hwp_epp hwp_pkg_req hwp_notify hwp_dynamic hwp_hwp_pkg_req umip pku ospke avx512_vnni md_clear flush_l1d arch_capabilities\n"
	if !cpuHasAVX2(withAVX2) {
		t.Error("expected AVX2 to be detected from the flags line")
	}

	// Sandy Bridge (the validated host): no avx2 anywhere in flags.
	sansAVX2 := "processor\t: 0\nvendor_id\t: GenuineIntel\nflags\t\t: fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush dts acpi mmx fxsr sse sse2 ss ht tm pbe syscall nx pdpe1gb rdtscp lm constant_tsc arch_perfmon pebs bts rep_good nopl xtopology nonstop_tsc cpuid aperfmperf pni pclmulqdq dtes64 monitor ds_cpl vmx est tm2 ssse3 sdbg fma cx16 xtpr pdcm pcid sse4_1 sse4_2 x2apic movbe popcnt tsc_deadline_timer aes xsave avx f16c rdrand lahf_lm abm 3dnowprefetch cpuid_fault epb ssbd ibrs ibpb stibp tpr_shadow vnmi flexpriority ept vpid fsgsbase tsc_adjust bmi1 smep bmi2 erms invpcid rdseed adx smap clflushopt clwb intel_pt sha_ni xsaveopt xsavec xgetbv1 xsaves dtherm ida arat pln pts hwp hwp_act_window hwp_epp hwp_pkg_req hwp_notify hwp_dynamic hwp_hwp_pkg_req umip pku ospke md_clear flush_l1d arch_capabilities\n"
	if cpuHasAVX2(sansAVX2) {
		t.Error("expected no AVX2 on Sandy Bridge flags")
	}

	// A line that merely mentions avx2 as a substring (e.g. a comment or a
	// model name) must not count.
	noFlagsLine := "model name\t: Intel(R) Core(TM) i7-2600 with avx2 mention\n"
	if cpuHasAVX2(noFlagsLine) {
		t.Error("expected avx2 outside a flags line to not count")
	}

	if cpuHasAVX2("") {
		t.Error("expected an empty cpuinfo to report no AVX2")
	}
}

func TestAmdgpuDevices(t *testing.T) {
	root := t.TempDir()
	if got := amdgpuDevices(root); len(got) != 0 {
		t.Errorf("expected no devices in an empty sysfs root, got %v", got)
	}

	// AMD card: vendor 0x1002.
	amdCard := filepath.Join(root, "card0", "device")
	if err := os.MkdirAll(amdCard, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(amdCard, "vendor"), []byte("0x1002\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Intel card: vendor 0x8086.
	intelCard := filepath.Join(root, "card1", "device")
	if err := os.MkdirAll(intelCard, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(intelCard, "vendor"), []byte("0x8086\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A render node without a device dir must be skipped, not fail.
	if err := os.MkdirAll(filepath.Join(root, "renderD128"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := amdgpuDevices(root)
	if len(got) != 1 || got[0] != amdCard {
		t.Errorf("expected only the AMD device %q, got %v", amdCard, got)
	}
}

func TestHostAmdgpuDevice(t *testing.T) {
	oldDir := amdgpuSysfsDir
	t.Cleanup(func() { amdgpuSysfsDir = oldDir })

	// No AMD GPU: empty.
	amdgpuSysfsDir = t.TempDir()
	if got := hostAmdgpuDevice(); got != "" {
		t.Errorf("expected no device on an empty sysfs, got %q", got)
	}

	// AMD present: its device path is returned.
	amdCard := filepath.Join(amdgpuSysfsDir, "card0", "device")
	if err := os.MkdirAll(amdCard, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(amdCard, "vendor"), []byte("0x1002\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := hostAmdgpuDevice(); got != amdCard {
		t.Errorf("expected the AMD device %q, got %q", amdCard, got)
	}
}
