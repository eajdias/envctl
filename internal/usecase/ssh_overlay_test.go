package usecase

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/eajdias/envctl"
)

// The expected outputs are the exact templates that used to be shipped as
// configs/ssh-config and configs/ssh-config.linux before the base+overlay
// consolidation: the deployed per-OS result must stay byte-identical.
const sshWindowsDeployedGolden = `# ~/.ssh/config - Optimized OpenSSH Client Configuration
# Managed by envctl

# Global Performance & Resiliency Defaults
Host *
    # Keepalive to prevent drops on idle VPS connections
    ServerAliveInterval 30
    ServerAliveCountMax 3

    # TCP keepalive and compression
    TCPKeepAlive yes
    Compression yes

    # Security & Agent
    StrictHostKeyChecking ask
    ForwardAgent no
    AddKeysToAgent yes

    # Identity file default lookup
    IdentityFile ~/.ssh/id_ed25519
    IdentityFile ~/.ssh/id_rsa
    IdentityFile ~/Documents/SSH-keys/id_ed25519
    IdentityFile ~/Documents/SSH-keys/id_rsa

# GitHub SSH Optimization
Host github.com
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_ed25519
    PreferredAuthentications publickey
`

const sshLinuxDeployedGolden = `# ~/.ssh/config - Optimized OpenSSH Client Configuration (Linux)
# Managed by envctl

# Global Performance & Resiliency Defaults
Host *
    # Keepalive to prevent drops on idle VPS connections
    ServerAliveInterval 30
    ServerAliveCountMax 3

    # Reuse authenticated connections for faster repeat sessions
    ControlMaster auto
    ControlPath ~/.ssh/sockets/%r@%h:%p
    ControlPersist 10m

    # TCP keepalive and compression
    TCPKeepAlive yes
    Compression yes

    # Security & Agent
    StrictHostKeyChecking ask
    ForwardAgent no
    AddKeysToAgent yes

    # Identity file default lookup
    IdentityFile ~/.ssh/id_ed25519
    IdentityFile ~/.ssh/id_rsa

# GitHub SSH Optimization
Host github.com
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_ed25519
    PreferredAuthentications publickey
`

// firstLineDiff reports the first line where got and want diverge, keeping
// failure output actionable without adding a diff dependency.
func firstLineDiff(got []byte, want string) string {
	gotLines := bytes.Split(got, []byte("\n"))
	wantLines := bytes.Split([]byte(want), []byte("\n"))
	for i := 0; i < len(gotLines) && i < len(wantLines); i++ {
		if !bytes.Equal(gotLines[i], wantLines[i]) {
			return "first divergent line " + strconv.Itoa(i+1) +
				": got " + strconv.Quote(string(gotLines[i])) + " want " + strconv.Quote(string(wantLines[i]))
		}
	}
	if len(gotLines) != len(wantLines) {
		return "line count differs: got " + strconv.Itoa(len(gotLines)) + " want " + strconv.Itoa(len(wantLines))
	}
	return "content differs in an unexpected way"
}

func TestSSHOverlayProducesTheFormerPerOSTemplates(t *testing.T) {
	base, err := envctl.EmbeddedFS.ReadFile("configs/ssh-config")
	if err != nil {
		t.Fatalf("read base ssh-config template: %v", err)
	}

	if got := withSSHOSOverlay(base, false); !bytes.Equal(got, []byte(sshWindowsDeployedGolden)) {
		t.Errorf("windows overlay output drifted from the former template:\n%s", firstLineDiff(got, sshWindowsDeployedGolden))
	}
	if got := withSSHOSOverlay(base, true); !bytes.Equal(got, []byte(sshLinuxDeployedGolden)) {
		t.Errorf("linux overlay output drifted from the former template:\n%s", firstLineDiff(got, sshLinuxDeployedGolden))
	}
}

func TestSSHOverlayIsIdempotent(t *testing.T) {
	base, err := envctl.EmbeddedFS.ReadFile("configs/ssh-config")
	if err != nil {
		t.Fatalf("read base ssh-config template: %v", err)
	}

	deployedWindows := withSSHOSOverlay(base, false)
	if again := withSSHOSOverlay(deployedWindows, false); !bytes.Equal(again, deployedWindows) {
		t.Error("windows overlay is not idempotent")
	}

	deployedLinux := withSSHOSOverlay(base, true)
	if again := withSSHOSOverlay(deployedLinux, true); !bytes.Equal(again, deployedLinux) {
		t.Error("linux overlay is not idempotent")
	}
}

func TestSSHOverlayPassesUnknownTemplateThrough(t *testing.T) {
	content := []byte("Host custom\n    User git\n")
	if got := withSSHOSOverlay(content, false); !bytes.Equal(got, content) {
		t.Errorf("windows overlay changed an unanchored template: %q", got)
	}
	if got := withSSHOSOverlay(content, true); !bytes.Equal(got, content) {
		t.Errorf("linux overlay changed an unanchored template: %q", got)
	}
}

// A Windows checkout carries CRLF (go:embed captures checkout bytes), while the
// goldens are LF. The overlay must normalize so the deployed bytes are
// identical regardless of the builder OS (CI caught this on windows-latest).
func TestSSHOverlayNormalizesCRLFCheckout(t *testing.T) {
	base, err := envctl.EmbeddedFS.ReadFile("configs/ssh-config")
	if err != nil {
		t.Fatalf("read base ssh-config template: %v", err)
	}
	// Normalize first: on a Windows checkout base already carries CRLF and a
	// naive \n -> \r\n pass would double it to \r\r\n.
	lf := bytes.ReplaceAll(base, []byte("\r\n"), []byte("\n"))
	crlf := bytes.ReplaceAll(lf, []byte("\n"), []byte("\r\n"))
	if got := withSSHOSOverlay(crlf, false); !bytes.Equal(got, []byte(sshWindowsDeployedGolden)) {
		t.Errorf("windows overlay on CRLF checkout drifted:\n%s", firstLineDiff(got, sshWindowsDeployedGolden))
	}
	if got := withSSHOSOverlay(crlf, true); !bytes.Equal(got, []byte(sshLinuxDeployedGolden)) {
		t.Errorf("linux overlay on CRLF checkout drifted:\n%s", firstLineDiff(got, sshLinuxDeployedGolden))
	}
}
