package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

type performanceRepoStub struct {
	mockManifestRepo
	spec      entity.PerformanceSpec
	loadCalls int
	loadErr   error
}

func (m *performanceRepoStub) LoadPerformanceSpec(profile entity.PerformanceProfile) (entity.PerformanceSpec, error) {
	m.loadCalls++
	if m.loadErr != nil {
		return entity.PerformanceSpec{}, m.loadErr
	}
	if m.spec.Profile != profile {
		return entity.PerformanceSpec{}, errors.New("wrong performance profile")
	}
	return m.spec, nil
}

type performancePackageManager struct {
	packageType  entity.PackageType
	available    bool
	installed    map[string]string
	installCalls int
	installErr   error
}

func (m *performancePackageManager) Type() entity.PackageType { return m.packageType }

func (m *performancePackageManager) IsAvailable(context.Context) bool { return m.available }

func (m *performancePackageManager) IsInstalled(_ context.Context, pkg entity.Package) (bool, string, error) {
	if version, ok := m.installed[pkg.ID]; ok {
		return true, version, nil
	}
	return false, "", nil
}

func (m *performancePackageManager) Install(_ context.Context, _ entity.Package) error {
	m.installCalls++
	return m.installErr
}

func (m *performancePackageManager) ListInstalled(context.Context) ([]entity.Package, error) {
	return nil, nil
}

type performanceSysctlStub struct {
	calls    int
	dryRun   bool
	settings []entity.SysctlSetting
}

func (m *performanceSysctlStub) Apply(_ context.Context, settings []entity.SysctlSetting, dryRun bool) ([]entity.Diagnostic, error) {
	m.calls++
	m.dryRun = dryRun
	m.settings = append([]entity.SysctlSetting(nil), settings...)
	if dryRun {
		return []entity.Diagnostic{{Category: entity.DiagInfo, System: "Performance", Target: "sysctl", Details: "dry-run"}}, nil
	}
	return nil, nil
}

type performanceZRAMStub struct {
	calls  int
	dryRun bool
}

func (m *performanceZRAMStub) Ensure(_ context.Context, dryRun bool) ([]entity.Diagnostic, error) {
	m.calls++
	m.dryRun = dryRun
	return []entity.Diagnostic{{Category: entity.DiagInfo, System: "Performance", Target: "zram", Details: "test zram state"}}, nil
}

func newPerformanceUseCaseForTest(
	repo repository.ManifestRepository,
	manager repository.PackageManager,
	sysctl repository.SysctlManager,
	platform entity.PlatformInfo,
	zrams ...repository.ZRAMManager,
) *ProvisionPerformanceUseCase {
	packages := NewProvisionPackagesUseCase(
		repo,
		map[entity.PackageType]repository.PackageManager{manager.Type(): manager},
		&mockLogger{},
	)
	packages.platform = func() entity.PlatformInfo { return platform }
	var zram repository.ZRAMManager = &performanceZRAMStub{}
	if len(zrams) > 0 {
		zram = zrams[0]
	}
	return NewProvisionPerformanceUseCase(repo, packages, sysctl, zram, &mockLogger{}, func() entity.PlatformInfo {
		return platform
	})
}

func TestProvisionPerformanceRejectsWrongExactProfile(t *testing.T) {
	repo := &performanceRepoStub{}
	manager := &performancePackageManager{packageType: entity.PackageTypeApt, available: true, installed: map[string]string{}}
	uc := newPerformanceUseCaseForTest(repo, manager, &performanceSysctlStub{}, entity.PlatformInfo{
		GOOS: "linux", Family: entity.DistroDebian, ID: "ubuntu", VersionID: "22.04",
	})

	_, _, err := uc.ExecutePerformance(context.Background(), entity.PerformanceProfileUbuntu, false, nil)
	if err == nil {
		t.Fatal("expected Ubuntu 22.04 to be rejected")
	}
	if repo.loadCalls != 0 {
		t.Fatalf("profile rejection loaded manifest %d time(s), want 0", repo.loadCalls)
	}
}

func TestProvisionPerformanceDryRunDoesNotInstallOrApply(t *testing.T) {
	repo := &performanceRepoStub{spec: entity.PerformanceSpec{
		Profile: entity.PerformanceProfileUbuntu,
		Packages: []entity.Package{{
			ID: "systemd-zram-generator", Type: entity.PackageTypeApt, OS: "ubuntu",
			TargetDistro: "ubuntu", MinDistroVersion: "24.04",
		}},
		Sysctls: []entity.SysctlSetting{{Key: "vm.swappiness", Value: "10"}},
	}}
	manager := &performancePackageManager{packageType: entity.PackageTypeApt, available: true, installed: map[string]string{}}
	sysctl := &performanceSysctlStub{}
	zram := &performanceZRAMStub{}
	uc := newPerformanceUseCaseForTest(repo, manager, sysctl, entity.PlatformInfo{
		GOOS: "linux", Family: entity.DistroDebian, ID: "ubuntu", VersionID: "24.04",
	}, zram)

	packages, diags, err := uc.ExecutePerformance(context.Background(), entity.PerformanceProfileUbuntu, true, nil)
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}
	if len(packages) != 1 || packages[0].Status != entity.StatusMissing {
		t.Fatalf("dry-run package result = %#v, want one missing package", packages)
	}
	if len(diags) != 3 {
		t.Fatalf("dry-run diagnostics = %#v, want package, sysctl, and zram diagnostics", diags)
	}
	for _, diagnostic := range diags {
		if diagnostic.Category != entity.DiagInfo {
			t.Fatalf("dry-run diagnostic is not INFO: %#v", diagnostic)
		}
	}
	if manager.installCalls != 0 {
		t.Fatalf("dry-run called package Install %d time(s)", manager.installCalls)
	}
	if sysctl.calls != 1 || !sysctl.dryRun {
		t.Fatalf("dry-run Sysctl Apply calls = %d dryRun=%v, want one dry-run call", sysctl.calls, sysctl.dryRun)
	}
	if zram.calls != 1 || !zram.dryRun {
		t.Fatalf("dry-run zram Ensure calls = %d dryRun=%v, want one dry-run call", zram.calls, zram.dryRun)
	}
}

func TestProvisionPerformanceStopsBeforeSysctlWhenPackageFails(t *testing.T) {
	repo := &performanceRepoStub{spec: entity.PerformanceSpec{
		Profile: entity.PerformanceProfileUbuntu,
		Packages: []entity.Package{{
			ID: "systemd-zram-generator", Type: entity.PackageTypeApt, OS: "ubuntu",
			TargetDistro: "ubuntu", MinDistroVersion: "24.04",
		}},
		Sysctls: []entity.SysctlSetting{{Key: "vm.swappiness", Value: "10"}},
	}}
	manager := &performancePackageManager{
		packageType: entity.PackageTypeApt, available: true, installed: map[string]string{},
		installErr: errors.New("synthetic install failure"),
	}
	sysctl := &performanceSysctlStub{}
	uc := newPerformanceUseCaseForTest(repo, manager, sysctl, entity.PlatformInfo{
		GOOS: "linux", Family: entity.DistroDebian, ID: "ubuntu", VersionID: "24.04",
	})

	_, _, err := uc.ExecutePerformance(context.Background(), entity.PerformanceProfileUbuntu, false, nil)
	if err == nil {
		t.Fatal("expected package failure")
	}
	if sysctl.calls != 0 {
		t.Fatalf("sysctl was applied after package failure: %d call(s)", sysctl.calls)
	}
}

func TestProvisionPerformanceRejectsCachyOSSysctlBleed(t *testing.T) {
	repo := &performanceRepoStub{spec: entity.PerformanceSpec{
		Profile:  entity.PerformanceProfileCachyOS,
		Packages: []entity.Package{{ID: "zram-generator", Type: entity.PackageTypePacman, OS: "arch", TargetDistro: "cachyos"}},
		Sysctls:  []entity.SysctlSetting{{Key: "vm.swappiness", Value: "10"}},
	}}
	manager := &performancePackageManager{packageType: entity.PackageTypePacman, available: true, installed: map[string]string{}}
	uc := newPerformanceUseCaseForTest(repo, manager, &performanceSysctlStub{}, entity.PlatformInfo{
		GOOS: "linux", Family: entity.DistroArch, ID: "cachyos", VersionID: "rolling",
	})

	_, _, err := uc.ExecutePerformance(context.Background(), entity.PerformanceProfileCachyOS, false, nil)
	if err == nil {
		t.Fatal("expected CachyOS sysctl bleed to be rejected")
	}
	if manager.installCalls != 0 {
		t.Fatalf("rejected spec installed %d package(s)", manager.installCalls)
	}
}

func TestProvisionPerformanceCachyOSDoesNotApplySysctl(t *testing.T) {
	repo := &performanceRepoStub{spec: entity.PerformanceSpec{
		Profile: entity.PerformanceProfileCachyOS,
		Packages: []entity.Package{{
			ID: "zram-generator", Type: entity.PackageTypePacman, OS: "arch", TargetDistro: "cachyos",
		}},
	}}
	manager := &performancePackageManager{packageType: entity.PackageTypePacman, available: true, installed: map[string]string{"zram-generator": "1.2.1-1"}}
	sysctl := &performanceSysctlStub{}
	zram := &performanceZRAMStub{}
	uc := newPerformanceUseCaseForTest(repo, manager, sysctl, entity.PlatformInfo{
		GOOS: "linux", Family: entity.DistroArch, ID: "cachyos", VersionID: "rolling",
	}, zram)

	packages, _, err := uc.ExecutePerformance(context.Background(), entity.PerformanceProfileCachyOS, false, nil)
	if err != nil {
		t.Fatalf("CachyOS performance run failed: %v", err)
	}
	if len(packages) != 1 || packages[0].Status != entity.StatusInstalled {
		t.Fatalf("CachyOS package result = %#v, want installed package", packages)
	}
	if sysctl.calls != 0 {
		t.Fatalf("CachyOS profile called Sysctl Apply %d time(s)", sysctl.calls)
	}
	if zram.calls != 1 || zram.dryRun {
		t.Fatalf("CachyOS zram Ensure calls = %d dryRun=%v, want one real call", zram.calls, zram.dryRun)
	}
}
