package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func testSelfUpdateServer(t *testing.T, tag string, asset string, payload []byte, sums string) (*SelfUpdateUseCase, string) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"tag_name": %q, "assets": [{"name": %q}]}`, tag, asset)
	})
	// Serve the payload and sums under the platform asset Apply resolves,
	// so the test exercises the real download path on any OS.
	platformAsset := assetName(runtime.GOOS, runtime.GOARCH)
	mux.HandleFunc("/download/"+tag+"/"+platformAsset, func(w http.ResponseWriter, r *http.Request) {
		w.Write(payload)
	})
	mux.HandleFunc("/download/"+tag+"/"+selfUpdateSumsFile, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, sums)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	uc := NewSelfUpdateUseCase()
	uc.apiBase = server.URL + "/releases/latest"
	uc.downloadBase = server.URL + "/download"
	uc.now = func() time.Time { return time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC) }
	return uc, server.URL
}

func sumsLine(payload []byte, asset string) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]) + "  " + asset + "\n"
}

func TestSelfUpdateRejectsChecksumMismatch(t *testing.T) {
	payload := []byte("fake-binary-bytes")
	platformAsset := assetName(runtime.GOOS, runtime.GOARCH)
	uc, _ := testSelfUpdateServer(t, "v9.9.9", platformAsset, payload, strings.Repeat("0", 64)+"  "+platformAsset+"\n")
	dest := filepath.Join(t.TempDir(), "envctl")
	if err := os.WriteFile(dest, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Apply(context.Background(), dest); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("tampered sums must fail closed, got %v", err)
	}
	if got, _ := os.ReadFile(dest); string(got) != "old" {
		t.Fatalf("failed install must leave the binary untouched, got %q", got)
	}
}

func TestSelfUpdateInstallsVerifiedBinaryAndKeepsBackup(t *testing.T) {
	payload := []byte("fake-binary-bytes")
	platformAsset := assetName(runtime.GOOS, runtime.GOARCH)
	uc, _ := testSelfUpdateServer(t, "v9.9.9", platformAsset, payload, sumsLine(payload, platformAsset))
	dest := filepath.Join(t.TempDir(), "envctl")
	if err := os.WriteFile(dest, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	res, err := uc.Apply(context.Background(), dest)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if res.To != "v9.9.9" || res.Asset != platformAsset {
		t.Errorf("result = %+v, want To=v9.9.9 Asset=%s", res, platformAsset)
	}
	if got, _ := os.ReadFile(dest); string(got) != string(payload) {
		t.Errorf("dest must hold the new binary, got %q", got)
	}
	backup, err := os.ReadFile(res.Backup)
	if err != nil || string(backup) != "old" {
		t.Errorf("backup %q must hold the previous binary: %v", res.Backup, err)
	}
	if !strings.HasSuffix(res.Backup, ".bak.20261103-120000") {
		t.Errorf("backup must carry a timestamp, got %q", res.Backup)
	}
}

func TestLatestRejectsMalformedTag(t *testing.T) {
	uc, _ := testSelfUpdateServer(t, "not-a-version", "envctl-test", []byte("x"), sumsLine([]byte("x"), "envctl-test"))
	if _, err := uc.Latest(context.Background()); err == nil {
		t.Error("malformed tag_name must fail")
	}
}

func TestAssetName(t *testing.T) {
	cases := map[string]string{
		"windows/amd64": "envctl-windows-amd64.exe",
		"windows/arm64": "envctl-windows-arm64.exe",
		"linux/amd64":   "envctl-linux-amd64",
		"linux/arm64":   "envctl-linux-arm64",
	}
	for platform, want := range cases {
		parts := strings.Split(platform, "/")
		if got := assetName(parts[0], parts[1]); got != want {
			t.Errorf("assetName(%s) = %q, want %q", platform, got, want)
		}
	}
	if got := assetName("darwin", "arm64"); got != "" {
		t.Errorf("unsupported OS must yield no asset, got %q", got)
	}
}

func TestVersionLess(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"dev", "v2.1.5", true},
		{"dev", "2.1.5", true},
		{"", "v2.1.5", true},
		{"v2.1.5", "v2.1.5", false},
		{"v2.1.4", "v2.1.5", true},
		{"2.1.5", "v2.1.5", false},
		{"v2.2.0", "v2.1.5", false},
		{"v2.9.0", "v2.10.0", true},
		{"v10.0.0", "v9.9.9", false},
		{"v2.1.5", "dev", false},
		{"garbage", "v2.1.5", false},
		{"v2.1.5", "garbage", false},
	}
	for _, tc := range cases {
		if got := VersionLess(tc.a, tc.b); got != tc.want {
			t.Errorf("VersionLess(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}
