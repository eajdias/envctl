package usecase

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/eajdias/envctl/internal/infra/filesystem"
)

// deploySnapshotFile is the home-relative path of the deploy record index.
// It lives next to the execution logs (~/.envctl/logs), never next to the
// managed configs, so hygiene sweeps and content audits never see it.
const deploySnapshotFile = "~/.envctl/deploys.json"

// DeployRecord pins the exact bytes envctl last wrote to a destination:
// SHA256 of the post-transform content plus when it was deployed. The doctor
// uses it as the third side of the drift comparison (template vs live vs
// last-deploy), so operator customization stops looking like template drift.
type DeployRecord struct {
	SHA256 string    `json:"sha256"`
	At     time.Time `json:"at"`
}

// DeploySnapshotStore persists deploy records across runs. Reads are
// fail-open (missing or corrupt index means "no record", never an error),
// because the doctor must stay useful on machines provisioned before this
// store existed.
type DeploySnapshotStore struct {
	fs   *filesystem.FileSystemManager
	path string
}

func newDeploySnapshotStore(fs *filesystem.FileSystemManager) *DeploySnapshotStore {
	return &DeploySnapshotStore{fs: fs, path: deploySnapshotFile}
}

// Load returns destination → record. It never fails: a missing or corrupt
// index degrades to "no deploy record", which the doctor reports as INFO.
func (s *DeploySnapshotStore) Load() map[string]DeployRecord {
	out := make(map[string]DeployRecord)
	if s == nil || s.fs == nil {
		return out
	}
	expanded, err := s.fs.ExpandUserPath(s.path)
	if err != nil {
		return out
	}
	data, err := os.ReadFile(expanded)
	if err != nil {
		return out
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return make(map[string]DeployRecord)
	}
	if out == nil {
		return make(map[string]DeployRecord)
	}
	return out
}

// Record pins content as the last-deployed bytes for dest. The key is the
// manifest destination literal (e.g. "~/.config/..."), stable across runs.
func (s *DeploySnapshotStore) Record(dest string, content []byte) error {
	if s == nil || s.fs == nil {
		return nil
	}
	expanded, err := s.fs.ExpandUserPath(s.path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(expanded), 0750); err != nil {
		return err
	}
	records := s.Load()
	records[dest] = DeployRecord{SHA256: sha256Hex(content), At: time.Now().UTC()}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(expanded), ".deploys-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, expanded); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// deploySnapshotIndex returns the last-deploy record index, honoring the
// test seam. Production reads ~/.envctl/deploys.json; tests inject a map.
func (uc *DoctorAuditUseCase) deploySnapshotIndex() map[string]DeployRecord {
	if uc.deploySnapshots != nil {
		return uc.deploySnapshots()
	}
	return newDeploySnapshotStore(uc.fsManager).Load()
}

// classifyOverwriteDrift is the three-way drift check for overwrite-mode
// config files: live (dst) vs last deploy (snapshots). Callers only invoke
// it on divergence (dst != template), so the template itself is not needed:
// what matters is whether the live bytes still match the last deploy.
// Returns whether the divergence is a genuine template drift (WARN:
// safe to re-deploy) or operator customization (INFO: hands off).
func classifyOverwriteDrift(dest string, dst []byte, snapshots map[string]DeployRecord) (warn bool, details, fixHint string) {
	const reviewHint = "run 'envctl doctor --diff' to review the difference"
	snap, hasSnap := snapshots[dest]
	switch {
	case !hasSnap:
		return false,
			"Diverges from provisioned source (no deploy record — treated as operator customization, not drift)",
			reviewHint
	case sha256Hex(dst) == snap.SHA256:
		when := deployWhen(snap.At)
		return true,
			"Provisioned source updated since last deploy (" + when + "); local file untouched, safe to re-apply",
			"run 'envctl doctor --diff' to review, then 'envctl run shell' to apply"
	default:
		when := deployWhen(snap.At)
		return false,
			"Customized by operator after deploy (" + when + "); provisioning will not touch it",
			reviewHint
	}
}

func deployWhen(at time.Time) string {
	if at.IsZero() {
		return "date unknown"
	}
	return "deployed " + at.Format("2006-01-02")
}
