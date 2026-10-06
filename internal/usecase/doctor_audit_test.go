package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
)



func TestMissingCmdlineParams(t *testing.T) {
	full := "quiet rw preempt=full split_lock_detect=off amdgpu.ppfeaturemask=0xffffffff zswap.enabled=0 mitigations=off"
	if missing := missingCmdlineParams(full, gamingKernelParams); len(missing) != 0 {
		t.Errorf("expected no missing params, got %v", missing)
	}
	// mitigations=off must never be required: absent is fine.
	withoutMitigations := "quiet rw preempt=full split_lock_detect=off zswap.enabled=0"
	if missing := missingCmdlineParams(withoutMitigations, gamingKernelParams); len(missing) != 0 {
		t.Errorf("expected no missing params without mitigations, got %v", missing)
	}
	partial := "quiet rw preempt=full"
	missing := missingCmdlineParams(partial, gamingKernelParams)
	if len(missing) != 2 || missing[0] != "split_lock_detect=off" || missing[1] != "zswap.enabled=0" {
		t.Errorf("expected [split_lock_detect=off zswap.enabled=0], got %v", missing)
	}
}

// realAMDGamingCmdline is the /proc/cmdline of the managed workstation, minus
// mitigations=off which is deliberately never required.
const realAMDGamingCmdline = "quiet nowatchdog splash rw rootflags=subvol=/@ root=UUID=556567b9-0e0a-4b8c-97fa-6d386b75bf07 " +
	"mitigations=off preempt=full split_lock_detect=off amdgpu.runpm=0 amdgpu.aspm=0 pcie_aspm=off " +
	"amdgpu.gpu_recovery=0 oops=panic panic=10 zswap.enabled=0"

func TestGamingAMDKernelParams_CoversRealCmdline(t *testing.T) {
	if missing := missingCmdlineParams(realAMDGamingCmdline, gamingAMDKernelParams); len(missing) != 0 {
		t.Errorf("expected no missing AMD params on the real cmdline, got %v", missing)
	}
	if missing := missingCmdlineParams("quiet preempt=full", gamingAMDKernelParams); len(missing) != len(gamingAMDKernelParams) {
		t.Errorf("expected all %d AMD params missing, got %v", len(gamingAMDKernelParams), missing)
	}
	partial := "quiet amdgpu.runpm=0 amdgpu.aspm=0 pcie_aspm=off"
	missing := missingCmdlineParams(partial, gamingAMDKernelParams)
	if len(missing) != 1 || missing[0] != "amdgpu.gpu_recovery=0" {
		t.Errorf("expected only [amdgpu.gpu_recovery=0] missing, got %v", missing)
	}
}

// TestGamingKernelParamsExcludeAMD guards the tiering: the universal list must
// stay hardware-agnostic, or an Intel or NVIDIA host warns forever.
func TestGamingKernelParamsExcludeAMD(t *testing.T) {
	for _, p := range gamingKernelParams {
		for _, amd := range gamingAMDKernelParams {
			if p == amd {
				t.Errorf("%q is in both the universal and the AMD list", p)
			}
		}
	}
	if strings.Contains(strings.Join(gamingKernelParams, " "), "amdgpu") {
		t.Errorf("universal kernel params must not mention amdgpu, got %v", gamingKernelParams)
	}
	if strings.Contains(strings.Join(gamingAMDKernelParams, " "), "panic") {
		t.Errorf("AMD kernel params must not include the panic-stability choice, got %v", gamingAMDKernelParams)
	}
}

func TestGamingPanicParamsAreOptional(t *testing.T) {
	if len(gamingPanicParams) == 0 {
		t.Fatal("expected the panic-stability params to be reported")
	}
	// Absent is the normal case on a stock kernel; it must never be a warning.
	if missing := missingCmdlineParams("quiet preempt=full", gamingPanicParams); len(missing) != len(gamingPanicParams) {
		t.Errorf("expected all panic params reported as absent, got %v", missing)
	}
	if missing := missingCmdlineParams(realAMDGamingCmdline, gamingPanicParams); len(missing) != 0 {
		t.Errorf("expected no missing panic params on the real cmdline, got %v", missing)
	}
}

func TestAmdgpuModulePresent(t *testing.T) {
	root := t.TempDir()
	if amdgpuModulePresent(filepath.Join(root, "module", "amdgpu")) {
		t.Error("expected amdgpu to be absent when the module dir is missing")
	}
	if err := os.MkdirAll(filepath.Join(root, "module", "amdgpu", "parameters"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !amdgpuModulePresent(filepath.Join(root, "module", "amdgpu")) {
		t.Error("expected amdgpu to be detected once the module dir exists")
	}
}

func TestMissingGamingConfKeys(t *testing.T) {
	both := []byte("MESA_SHADER_CACHE_MAX_SIZE=12G\nRADV_PERFTEST=gpl\n")
	if missing := missingGamingConfKeys(both); len(missing) != 0 {
		t.Errorf("expected no missing keys, got %v", missing)
	}
	onlyCache := []byte("MESA_SHADER_CACHE_MAX_SIZE=12G\n")
	if missing := missingGamingConfKeys(onlyCache); len(missing) != 1 || missing[0] != "RADV_PERFTEST=" {
		t.Errorf("expected only RADV_PERFTEST= missing, got %v", missing)
	}
	if missing := missingGamingConfKeys(nil); len(missing) != len(gamingConfRequiredKeys) {
		t.Errorf("expected all %d keys missing for an empty file, got %v", len(gamingConfRequiredKeys), missing)
	}
	// A commented-out line is not a setting; the audit must not accept it.
	commented := []byte("# MESA_SHADER_CACHE_MAX_SIZE=12G\n#RADV_PERFTEST=gpl\n")
	if missing := missingGamingConfKeys(commented); len(missing) != len(gamingConfRequiredKeys) {
		t.Errorf("expected commented keys to count as missing, got %v", missing)
	}
}

func TestPendingPacnewFiles(t *testing.T) {
	root := t.TempDir()
	if got := pendingPacnewFiles(root); len(got) != 0 {
		t.Errorf("expected no pending .pacnew in an empty tree, got %v", got)
	}

	if err := os.MkdirAll(filepath.Join(root, "pacman.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		filepath.Join("pacman.conf.pacnew"),
		filepath.Join("limine-snapper-sync.conf.pacnew"),
		filepath.Join("pacman.d", "cachyos-mirrorlist.pacnew"),
		filepath.Join("pacman.d", "not-a-pacnew.txt"),
		filepath.Join("pacman.conf"),
	} {
		if err := os.WriteFile(filepath.Join(root, rel), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got := pendingPacnewFiles(root)
	if len(got) != 3 {
		t.Fatalf("expected 3 pending .pacnew files, got %d: %v", len(got), got)
	}
	want := map[string]bool{
		"pacman.conf.pacnew":                 true,
		"limine-snapper-sync.conf.pacnew":    true,
		"pacman.d/cachyos-mirrorlist.pacnew": true,
	}
	for _, p := range got {
		if !want[p] {
			t.Errorf("unexpected file reported: %q", p)
		}
	}
	// Sorted, so the diagnostic is stable between runs.
	if !sort.StringsAreSorted(got) {
		t.Errorf("expected sorted output, got %v", got)
	}
}

func TestMultilibEnabled(t *testing.T) {
	active := "[core]\nInclude = /etc/pacman.d/mirrorlist\n\n[multilib]\nInclude = /etc/pacman.d/mirrorlist\n"
	if !multilibEnabled(active) {
		t.Errorf("expected active [multilib] to be detected")
	}
	commented := "#[multilib]\n#Include = /etc/pacman.d/mirrorlist\n"
	if multilibEnabled(commented) {
		t.Errorf("expected commented #[multilib] to be reported as disabled")
	}
	if multilibEnabled("[core]\nInclude = /etc/pacman.d/mirrorlist\n") {
		t.Errorf("expected missing [multilib] to be reported as disabled")
	}
}

func TestKwinrcCompositingEnabled(t *testing.T) {
	withBoth := []byte("[Compositing]\nAllowBlockCompositing=true\nUnredirectFullscreen=true\n")
	if !kwinrcCompositingEnabled(withBoth) {
		t.Error("expected compositing bypass to be detected when both keys are active")
	}
	partial := []byte("[Compositing]\nAllowBlockCompositing=true\n")
	if kwinrcCompositingEnabled(partial) {
		t.Error("expected partial compositing settings to be reported as not enabled")
	}
	noSection := []byte("[General]\nAllowBlockCompositing=true\nUnredirectFullscreen=true\n")
	if kwinrcCompositingEnabled(noSection) {
		t.Error("expected keys outside [Compositing] to not count")
	}
	if kwinrcCompositingEnabled(nil) {
		t.Error("expected an empty file to not count")
	}
}

func TestGamingEmulatorConfigs_CoverRealMachine(t *testing.T) {
	// The validated machine runs every emulator with a Vulkan backend. This
	// test pins the source of truth (the files the owner tuned by hand) so the
	// audit table cannot silently drift from what the real host uses.
	realConfigs := map[string]string{
		"~/.config/dolphin-emu/Dolphin.ini":      "GFXBackend = Vulkan",
		"~/.config/retroarch/retroarch.cfg":      `video_driver = "vulkan"`,
		"~/.config/ppsspp/PSP/SYSTEM/ppsspp.ini": "GraphicsBackend = 3",
		"~/.config/PCSX2/inis/PCSX2.ini":         "Renderer = 14",
		"~/.config/duckstation/settings.ini":     "Renderer = Vulkan",
		"~/.config/azahar-emu/qt-config.ini":     "graphics_api=2",
		"~/.config/eden/qt-config.ini":           "backend=1",
		"~/.config/Vita3K/config.yml":            "backend-renderer: Vulkan",
		"~/.config/Cemu/settings.xml":            "<api>1</api>",
	}
	for _, check := range gamingEmulatorConfigs {
		want, ok := realConfigs[check.path]
		if !ok {
			t.Errorf("emulator config %q (%s) is not part of the tuned machine set", check.path, check.target)
			continue
		}
		if want != check.key {
			t.Errorf("emulator config %q: expected key %q, table has %q", check.path, want, check.key)
		}
	}
}

func TestConfigHasKey(t *testing.T) {
	ini := []byte("# comment\nGFXBackend = Vulkan\n[General]\nfoo=bar\n")
	if !configHasKey(ini, "GFXBackend = Vulkan") {
		t.Error("expected the real key to be found")
	}
	if configHasKey(ini, "foo = bar") {
		t.Error("expected a key with a different spacing to not match")
	}
	if configHasKey(ini, "# comment") {
		t.Error("expected commented lines to not count")
	}
	yaml := []byte("backend-renderer: Vulkan\nresolution-multiplier: 2\n")
	if !configHasKey(yaml, "backend-renderer: Vulkan") {
		t.Error("expected the YAML key to be found")
	}
	xml := []byte("<api>3</api>\n<api>1</api>\n")
	if !configHasKey(xml, "<api>1</api>") {
		t.Error("expected the XML key to be found")
	}
	if configHasKey(nil, "anything") {
		t.Error("expected an empty file to not match")
	}
}

// mockGamingPackageManager is a minimal repository.PackageManager for the
// gaming audit tests.
type mockGamingPackageManager struct {
	available bool
	installed map[string]string
}

func (m *mockGamingPackageManager) Type() entity.PackageType { return entity.PackageTypePacman }
func (m *mockGamingPackageManager) IsAvailable(ctx context.Context) bool {
	return m.available
}
func (m *mockGamingPackageManager) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
	if v, ok := m.installed[pkg.ID]; ok {
		return true, v, nil
	}
	return false, "", nil
}
func (m *mockGamingPackageManager) Install(ctx context.Context, pkg entity.Package) error {
	return nil
}
func (m *mockGamingPackageManager) ListInstalled(ctx context.Context) ([]entity.Package, error) {
	return nil, nil
}

func TestEnvctlBinaryFresh(t *testing.T) {
	cases := []struct {
		built    string
		describe string
		want     bool
	}{
		{"v1.12.0", "v1.12.0", true},
		{"1.12.0", "v1.12.0", true},
		{"v1.12.0", "1.12.0", true},
		{"dev", "v1.12.0-9-g1f6992d", false},
		{"", "v1.12.0", false},
		{"v1.6.0", "v1.12.0-9-g1f6992d", false},
		{"v1.12.0", "", false},
	}
	for _, tc := range cases {
		if got := envctlBinaryFresh(tc.built, tc.describe); got != tc.want {
			t.Errorf("envctlBinaryFresh(%q, %q) = %v, want %v", tc.built, tc.describe, got, tc.want)
		}
	}
}

func TestAuditEnvctlFreshnessWarnsWhenStale(t *testing.T) {
	old := repoRootFinder
	t.Cleanup(func() { repoRootFinder = old })
	repoRootFinder = func() (string, error) { return t.TempDir(), nil }

	uc := &DoctorAuditUseCase{
		envctlVersion: "v1.6.0",
		gitDescribe: func(_ context.Context, _ string) (string, error) {
			return "v1.12.0-9-g1f6992d", nil
		},
	}

	var diags []entity.Diagnostic
	uc.auditEnvctlFreshness(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })

	if len(diags) != 1 {
		t.Fatalf("expected 1 freshness diagnostic, got %d: %v", len(diags), diags)
	}
	d := diags[0]
	if d.Category != entity.DiagWarning || d.System != "Envctl" || d.Target != "Binary freshness" {
		t.Errorf("unexpected diagnostic: %+v", d)
	}
	if !strings.Contains(d.FixHint, "go build") {
		t.Errorf("expected a rebuild FixHint, got %q", d.FixHint)
	}
}

func TestAuditEnvctlFreshnessSilentWhenCurrent(t *testing.T) {
	old := repoRootFinder
	t.Cleanup(func() { repoRootFinder = old })
	repoRootFinder = func() (string, error) { return t.TempDir(), nil }

	uc := &DoctorAuditUseCase{
		envctlVersion: "v1.12.0",
		gitDescribe: func(_ context.Context, _ string) (string, error) {
			return "v1.12.0", nil
		},
	}

	var diags []entity.Diagnostic
	uc.auditEnvctlFreshness(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })
	if len(diags) != 0 {
		t.Errorf("expected no diagnostic when the binary matches the checkout, got %v", diags)
	}
}

func TestAuditEnvctlFreshnessSkipsWithoutRepo(t *testing.T) {
	uc := &DoctorAuditUseCase{
		envctlVersion: "v1.6.0",
		gitDescribe: func(_ context.Context, _ string) (string, error) {
			return "v1.12.0-9-g1f6992d", nil
		},
	}
	// No repo: envctlVersion present but gitDescribe returns an error → silence.
	uc.gitDescribe = func(_ context.Context, _ string) (string, error) {
		return "", fmt.Errorf("not a git repository")
	}
	var diags []entity.Diagnostic
	uc.auditEnvctlFreshness(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })
	if len(diags) != 0 {
		t.Errorf("expected no diagnostic without a reachable repo checkout, got %v", diags)
	}
}

func TestAuditEnvctlFreshnessSkipsWithoutVersion(t *testing.T) {
	uc := &DoctorAuditUseCase{
		envctlVersion: "",
		gitDescribe: func(_ context.Context, _ string) (string, error) {
			return "v1.12.0", nil
		},
	}
	var diags []entity.Diagnostic
	uc.auditEnvctlFreshness(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })
	if len(diags) != 0 {
		t.Errorf("expected no diagnostic when the binary version is unknown, got %v", diags)
	}
}

func TestAuditOpenCodeVersionSkewParsesMajor(t *testing.T) {
	cases := []struct {
		output string
		skewed bool
	}{
		{"opencode 1.18.30", true},
		{"opencode v1.18.32", true},
		{"opencode v2.0.15", false},
		{"opencode v10.0.0", false},
		{"2.0.5", false},
		{"no numbers here", false},
	}
	for _, tc := range cases {
		version := firstVersionToken(tc.output)
		skewed := version != "" && !versionMajorAtLeast(version, 2)
		if skewed != tc.skewed {
			t.Errorf("firstVersionToken(%q) = %q, skewed = %v, want %v", tc.output, version, skewed, tc.skewed)
		}
	}
}
