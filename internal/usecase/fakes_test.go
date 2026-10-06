package usecase

// mockLogger implements repository.Logger for testing across usecase tests.
type mockLogger struct{}

func (m *mockLogger) Info(format string, args ...any)  {}
func (m *mockLogger) Warn(format string, args ...any)  {}
func (m *mockLogger) Error(format string, args ...any) {}
func (m *mockLogger) Debug(format string, args ...any) {}
func (m *mockLogger) LogCommand(cmd string, args []string, exitCode int, output string, err error) {
}
func (m *mockLogger) LogIdempotency(system, target string, skipped bool, reason string) {}
func (m *mockLogger) GetLogFilePath() string                                            { return "" }
func (m *mockLogger) Close() error                                                      { return nil }
