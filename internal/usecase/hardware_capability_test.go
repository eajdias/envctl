package usecase

import (
	"os"
	"path/filepath"
	"testing"
)

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