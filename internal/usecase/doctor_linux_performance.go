package usecase

import (
	"fmt"
	"strings"
)

func formatPerformanceBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	divisor := uint64(unit)
	exponent := 0
	for value := bytes / unit; value >= unit && exponent < 4; value /= unit {
		divisor *= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(divisor), "KMGTP"[exponent])
}

func valueOrUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}
