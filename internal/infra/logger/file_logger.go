package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/eajdias/win11-new/internal/domain/repository"
)

type fileLogger struct {
	mu          sync.Mutex
	logFilePath string
	file        *os.File
}

// NewFileLogger creates a persistent disk logger in ~/.win11-new/logs/ or a specified directory.
func NewFileLogger(customDir string) (repository.Logger, error) {
	logDir := customDir
	if logDir == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			userHome = os.Getenv("USERPROFILE")
			if userHome == "" {
				userHome = "."
			}
		}
		logDir = filepath.Join(userHome, ".win11-new", "logs")
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}

	timestamp := time.Now().Format("20060102-150405")
	logFilePath := filepath.Join(logDir, fmt.Sprintf("win11-new-%s.log", timestamp))

	f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", logFilePath, err)
	}

	l := &fileLogger{
		logFilePath: logFilePath,
		file:        f,
	}

	// Write session header
	header := fmt.Sprintf(
		"================================================================================\n"+
			"win11-new Execution Log Session Started: %s\n"+
			"Target Host: %s | User: %s | OS: %s\n"+
			"================================================================================\n\n",
		time.Now().Format(time.RFC3339),
		getHostname(),
		getUsername(),
		getOSInfo(),
	)
	_, _ = f.WriteString(header)

	return l, nil
}

func sanitizeText(s string) string {
	return strings.ReplaceAll(s, "\x00", "")
}

func (l *fileLogger) writeEntry(level, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	line := fmt.Sprintf("[%s] [%-5s] %s\n", timestamp, level, sanitizeText(message))
	_, _ = l.file.WriteString(line)
}

func (l *fileLogger) Info(format string, args ...any) {
	l.writeEntry("INFO", fmt.Sprintf(format, args...))
}

func (l *fileLogger) Warn(format string, args ...any) {
	l.writeEntry("WARN", fmt.Sprintf(format, args...))
}

func (l *fileLogger) Error(format string, args ...any) {
	l.writeEntry("ERROR", fmt.Sprintf(format, args...))
}

func (l *fileLogger) Debug(format string, args ...any) {
	l.writeEntry("DEBUG", fmt.Sprintf(format, args...))
}

func (l *fileLogger) LogCommand(cmd string, args []string, exitCode int, output string, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	cmdLine := cmd
	if len(args) > 0 {
		cmdLine = fmt.Sprintf("%s %s", cmd, strings.Join(args, " "))
	}

	statusStr := "SUCCESS"
	if exitCode != 0 || err != nil {
		statusStr = fmt.Sprintf("FAILED (ExitCode: %d)", exitCode)
	}

	entry := fmt.Sprintf("[%s] [EXEC ] Command: %s | Result: %s\n", timestamp, cmdLine, statusStr)
	if err != nil {
		entry += fmt.Sprintf("               Error: %v\n", err)
	}
	if strings.TrimSpace(output) != "" {
		cleanedOutput := sanitizeText(strings.TrimSpace(output))
		indented := strings.ReplaceAll(cleanedOutput, "\n", "\n               | ")
		entry += fmt.Sprintf("               Output: | %s\n", indented)
	}

	_, _ = l.file.WriteString(entry)
}

func (l *fileLogger) LogIdempotency(system, target string, skipped bool, reason string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	action := "APPLY-CHANGE"
	if skipped {
		action = "IDEMPOTENT-SKIP"
	}

	line := fmt.Sprintf("[%s] [%-15s] [%s] %s -> %s\n", timestamp, action, system, target, sanitizeText(reason))
	_, _ = l.file.WriteString(line)
}

func (l *fileLogger) GetLogFilePath() string {
	return l.logFilePath
}

func (l *fileLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		footer := fmt.Sprintf(
			"\n================================================================================\n"+
				"win11-new Execution Log Session Closed: %s\n"+
				"================================================================================\n",
			time.Now().Format(time.RFC3339),
		)
		_, _ = l.file.WriteString(footer)
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}

func getHostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

func getUsername() string {
	u := os.Getenv("USERNAME")
	if u == "" {
		u = os.Getenv("USER")
	}
	return u
}

func getOSInfo() string {
	return fmt.Sprintf("Windows (%s/%s)", os.Getenv("PROCESSOR_ARCHITECTURE"), os.Getenv("OS"))
}
