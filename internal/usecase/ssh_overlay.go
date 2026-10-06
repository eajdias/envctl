package usecase

import (
	"bytes"
)

// withSSHOSOverlay injects the OS-only lines into the single ssh-config base
// template. The old configs/ssh-config.linux copy differed from the base in
// exactly two places: the "(Linux)" header suffix plus the Control* connection
// reuse block (linux), and two extra ~/Documents/SSH-keys identity lookups
// (windows). Both are injected at deploy time from the base, so the pair can
// never drift again.
//
// Insertion is textual on stable anchors and idempotent: a template that
// already carries the OS lines passes through untouched.
func withSSHOSOverlay(content []byte, linux bool) []byte {
	// Normalize first: go:embed captures checkout bytes, and a Windows
	// checkout carries CRLF while goldens and Linux carry LF. Without this
	// the same template deploys different bytes per builder OS.
	content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	if linux {
		return withLinuxSSHOverlay(content)
	}
	return withWindowsSSHOverlay(content)
}

// windowsSSHOverlayLines are the Windows-only key lookups. The anchor is
// "IdentityFile ~/.ssh/id_rsa", which appears exactly once in the template.
var windowsSSHOverlayLines = []string{
	"    IdentityFile ~/Documents/SSH-keys/id_ed25519",
	"    IdentityFile ~/Documents/SSH-keys/id_rsa",
}

// linuxSSHOverlayLines are the Linux-only connection-reuse lines, inserted
// right after ServerAliveCountMax (the deployed linux template keeps a blank
// line before and after the block).
var linuxSSHOverlayLines = []string{
	"",
	"    # Reuse authenticated connections for faster repeat sessions",
	"    ControlMaster auto",
	"    ControlPath ~/.ssh/sockets/%r@%h:%p",
	"    ControlPersist 10m",
}

func withWindowsSSHOverlay(content []byte) []byte {
	if bytes.Contains(content, []byte("Documents/SSH-keys")) {
		return content
	}
	const anchor = "    IdentityFile ~/.ssh/id_rsa\n"
	if patched := insertAfterLine(content, anchor, windowsSSHOverlayLines); patched != nil {
		return patched
	}
	return content
}

func withLinuxSSHOverlay(content []byte) []byte {
	if bytes.Contains(content, []byte("ControlMaster auto")) {
		return content
	}
	const anchor = "    ServerAliveCountMax 3\n"
	patched := insertAfterLine(content, anchor, linuxSSHOverlayLines)
	if patched == nil {
		return content
	}
	// Header suffix: the deployed linux template marks itself "(Linux)".
	return bytes.Replace(patched, []byte("Optimized OpenSSH Client Configuration\n"),
		[]byte("Optimized OpenSSH Client Configuration (Linux)\n"), 1)
}

// insertAfterLine appends lines after the first occurrence of anchor. It
// returns nil when the anchor is missing, so the caller can pass the template
// through unchanged.
func insertAfterLine(content []byte, anchor string, lines []string) []byte {
	idx := bytes.Index(content, []byte(anchor))
	if idx < 0 {
		return nil
	}
	at := idx + len(anchor)
	patched := make([]byte, 0, contentCapacity(content, lines))
	patched = append(patched, content[:at]...)
	for _, line := range lines {
		patched = append(patched, line...)
		patched = append(patched, '\n')
	}
	patched = append(patched, content[at:]...)
	return patched
}

func contentCapacity(content []byte, lines []string) int {
	n := len(content) + 1
	for _, line := range lines {
		n += len(line) + 1
	}
	return n
}
