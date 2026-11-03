package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Release channels for self-update. The asset names are the same contract
// bootstrap.sh / bootstrap.ps1 rely on (see .goreleaser.yml): loose binaries
// envctl-<os>-<arch>[.exe] plus a covering SHA256SUMS.txt.
const (
	selfUpdateReleasesAPI  = "https://api.github.com/repos/eajdias/envctl/releases/latest"
	selfUpdateDownloadBase = "https://github.com/eajdias/envctl/releases/download"
	selfUpdateSumsFile     = "SHA256SUMS.txt"
	selfUpdateHTTPTimeout  = 60 * time.Second
)

// SelfUpdateUseCase installs the latest verified envctl release over the
// running binary. Every install keeps a timestamped backup next to the
// binary, so the previous version is always one rename away.
type SelfUpdateUseCase struct {
	client *http.Client
	now    func() time.Time
	// apiBase and downloadBase honor the production channels above; tests
	// point them at an httptest server.
	apiBase      string
	downloadBase string
}

func NewSelfUpdateUseCase() *SelfUpdateUseCase {
	return &SelfUpdateUseCase{
		client:       &http.Client{Timeout: selfUpdateHTTPTimeout},
		now:          time.Now,
		apiBase:      selfUpdateReleasesAPI,
		downloadBase: selfUpdateDownloadBase,
	}
}

// SelfUpdateResult describes an applied update.
type SelfUpdateResult struct {
	To     string // release tag installed (e.g. v2.1.5)
	Backup string // previous binary, kept next to the install
	Asset  string // release asset name installed
}

// assetName maps a GOOS/GOARCH pair to the loose release asset. Empty means
// the platform has no published binary (macOS/darwin is not a target).
func assetName(goos, goarch string) string {
	switch goos {
	case "windows":
		return "envctl-windows-" + goarch + ".exe"
	case "linux":
		return "envctl-linux-" + goarch
	default:
		return ""
	}
}

// AssetName reports the release asset for this platform ("" when the
// platform has no published binary). The CLI prints it in --check output.
func (uc *SelfUpdateUseCase) AssetName() string {
	return assetName(runtime.GOOS, runtime.GOARCH)
}

// VersionLess reports whether a is older than b. A dev build (or an empty
// version) is always older; anything else compares numerically on the
// release triple, never lexicographically ("2.10.0" is newer than "2.9.0").
// Unparseable non-dev versions are never "less", so a garbled input can
// never trigger a downgrade.
func VersionLess(a, b string) bool {
	if a == b {
		return false
	}
	if a == "" || a == "dev" {
		return true
	}
	if b == "" || b == "dev" {
		return false
	}
	ma, na, pa, oka := parseSemver(a)
	mb, nb, pb, okb := parseSemver(b)
	if !oka || !okb {
		return false
	}
	if ma != mb {
		return ma < mb
	}
	if na != nb {
		return na < nb
	}
	return pa < pb
}

// Latest resolves the newest release tag (e.g. v2.1.5) from the GitHub API.
func (uc *SelfUpdateUseCase) Latest(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uc.apiBase, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := uc.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("latest release lookup failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("latest release lookup failed: %s", resp.Status)
	}
	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("latest release lookup failed: %w", err)
	}
	tag := strings.TrimSpace(payload.TagName)
	if tag == "" {
		return "", fmt.Errorf("latest release lookup failed: empty tag_name")
	}
	if _, _, _, ok := parseSemver(tag); !ok {
		return "", fmt.Errorf("latest release lookup failed: malformed tag %q", tag)
	}
	return tag, nil
}

// Apply downloads tag's asset, verifies it against SHA256SUMS.txt, and
// installs it over dest, keeping dest.bak.YYYYMMDD-HHMMSS. The current
// binary is renamed aside first (renaming a running exe works on Windows;
// overwriting it does not), so a failed move restores the backup instead
// of bricking the install.
func (uc *SelfUpdateUseCase) Apply(ctx context.Context, dest string) (SelfUpdateResult, error) {
	latest, err := uc.Latest(ctx)
	if err != nil {
		return SelfUpdateResult{}, err
	}
	asset := assetName(runtime.GOOS, runtime.GOARCH)
	if asset == "" {
		return SelfUpdateResult{}, fmt.Errorf("self-update is not supported on %s/%s: no published binary", runtime.GOOS, runtime.GOARCH)
	}
	bin, err := uc.download(ctx, uc.downloadBase+"/"+latest+"/"+asset)
	if err != nil {
		return SelfUpdateResult{}, err
	}
	sums, err := uc.download(ctx, uc.downloadBase+"/"+latest+"/"+selfUpdateSumsFile)
	if err != nil {
		return SelfUpdateResult{}, err
	}
	want, err := checksumForAsset(string(sums), asset)
	if err != nil {
		return SelfUpdateResult{}, err
	}
	sum := sha256.Sum256(bin)
	if hex.EncodeToString(sum[:]) != want {
		return SelfUpdateResult{}, fmt.Errorf("checksum mismatch for %s: download is corrupt or tampered, refusing to install", asset)
	}

	dir := filepath.Dir(dest)
	tmp, err := os.CreateTemp(dir, ".envctl-update-*")
	if err != nil {
		return SelfUpdateResult{}, err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(bin); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return SelfUpdateResult{}, err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return SelfUpdateResult{}, err
	}
	// Executable bit for POSIX installs (Windows ignores it harmlessly). A
	// failure here aborts the install: without +x the binary would land
	// dead on arrival.
	//nolint:gosec // G302: the download is an executable binary by design; user-owned, never world-writable.
	if err := os.Chmod(tmpName, 0755); err != nil {
		os.Remove(tmpName)
		return SelfUpdateResult{}, fmt.Errorf("could not mark the download executable: %w", err)
	}

	backup := dest + ".bak." + uc.now().UTC().Format("20060102-150405")
	if err := os.Rename(dest, backup); err != nil {
		os.Remove(tmpName)
		return SelfUpdateResult{}, fmt.Errorf("could not move the current binary aside: %w", err)
	}
	if err := os.Rename(tmpName, dest); err != nil {
		//nolint:errcheck // best effort: the move already failed, the restore attempt must not mask the original error.
		os.Rename(backup, dest)
		return SelfUpdateResult{}, fmt.Errorf("could not install the new binary (previous restored): %w", err)
	}
	return SelfUpdateResult{To: latest, Backup: backup, Asset: asset}, nil
}

func (uc *SelfUpdateUseCase) download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := uc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download %s failed: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s failed: %s", url, resp.Status)
	}
	const maxAssetBytes = 256 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxAssetBytes+1))
	if err != nil {
		return nil, fmt.Errorf("download %s failed: %w", url, err)
	}
	if len(data) > maxAssetBytes {
		return nil, fmt.Errorf("download %s exceeds %d bytes, refusing", url, maxAssetBytes)
	}
	return data, nil
}

// checksumForAsset extracts the expected hex digest for asset from a
// SHA256SUMS.txt body ("<hex>[ *]<filename>" per line).
func checksumForAsset(sums, asset string) (string, error) {
	for _, line := range strings.Split(sums, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		if filepath.Base(name) != asset && name != asset {
			continue
		}
		digest := strings.ToLower(fields[0])
		if len(digest) != 64 {
			return "", fmt.Errorf("malformed checksum entry for %s", asset)
		}
		return digest, nil
	}
	return "", fmt.Errorf("asset %s not listed in %s", asset, selfUpdateSumsFile)
}
