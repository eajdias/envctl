package windows

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

// TweaksManager implements repository.WindowsTweaksManager.
type TweaksManager struct {
	logger repository.Logger
}

func NewWindowsTweaksManager(logger repository.Logger) repository.WindowsTweaksManager {
	return &TweaksManager{
		logger: logger,
	}
}

func (m *TweaksManager) CheckTweak(ctx context.Context, tweak entity.WindowsTweak) (bool, string, error) {
	switch strings.ToLower(tweak.Type) {
	case "font":
		// Check font files or oh-my-posh
		psScript := fmt.Sprintf(`
$fontPath1 = "$env:LOCALAPPDATA\Microsoft\Windows\Fonts"
$fontPath2 = "C:\Windows\Fonts"
$found = (Get-ChildItem -Path $fontPath1, $fontPath2 -Filter "*%s*" -ErrorAction SilentlyContinue | Measure-Object).Count
if ($found -gt 0) { Write-Output "INSTALLED" } else { Write-Output "MISSING" }
`, tweak.Name)
		cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psScript)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return false, "", fmt.Errorf("failed to check font %s: %w", tweak.Name, err)
		}
		if strings.Contains(string(out), "INSTALLED") {
			return true, "Font installed in Windows font directory", nil
		}
		return false, "Font not found", nil

	case "feature":
		psScript := fmt.Sprintf(`
$f = Get-WindowsOptionalFeature -Online -FeatureName "%s" -ErrorAction SilentlyContinue
if ($f -and $f.State -eq 'Enabled') { Write-Output "ENABLED" } else { Write-Output "DISABLED" }
`, tweak.Name)
		cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psScript)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return false, "", fmt.Errorf("failed to check feature %s: %w", tweak.Name, err)
		}
		if strings.Contains(string(out), "ENABLED") {
			return true, "Windows optional feature is enabled", nil
		}
		return false, "Feature is disabled or not present", nil

	default: // Registry DWord, String, Binary, etc.
		psScript := fmt.Sprintf(`
$path = "%s"
$name = "%s"
if (Test-Path $path) {
    $val = Get-ItemPropertyValue -Path $path -Name $name -ErrorAction SilentlyContinue
    if ($val -ne $null) {
        Write-Output "VALUE:$val"
    } else {
        Write-Output "VALUE_NOT_SET"
    }
} else {
    Write-Output "PATH_NOT_FOUND"
}
`, tweak.Path, tweak.Name)
		cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psScript)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return false, "", fmt.Errorf("failed to read registry %s\\%s: %w", tweak.Path, tweak.Name, err)
		}
		outStr := strings.TrimSpace(string(out))
		expectedStr := fmt.Sprintf("%v", tweak.Value)
		if strings.HasPrefix(outStr, "VALUE:") {
			actualVal := strings.TrimPrefix(outStr, "VALUE:")
			if actualVal == expectedStr {
				return true, fmt.Sprintf("Registry value matches (%s)", actualVal), nil
			}
			return false, fmt.Sprintf("Registry value mismatch (actual: %s, expected: %s)", actualVal, expectedStr), nil
		}
		return false, outStr, nil
	}
}

func (m *TweaksManager) ApplyTweak(ctx context.Context, tweak entity.WindowsTweak) error {
	switch strings.ToLower(tweak.Type) {
	case "font":
		psScript := fmt.Sprintf(`oh-my-posh font install %s`, tweak.Name)
		cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psScript)
		out, err := cmd.CombinedOutput()
		if m.logger != nil {
			m.logger.LogCommand("powershell.exe", []string{"-Command", psScript}, cmd.ProcessState.ExitCode(), string(out), err)
		}
		if err != nil {
			return fmt.Errorf("failed to install font %s: %s (%w)", tweak.Name, string(out), err)
		}
		return nil

	case "feature":
		psScript := fmt.Sprintf(`Enable-WindowsOptionalFeature -Online -FeatureName "%s" -NoRestart -ErrorAction Stop`, tweak.Name)
		cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psScript)
		out, err := cmd.CombinedOutput()
		if m.logger != nil {
			m.logger.LogCommand("powershell.exe", []string{"-Command", psScript}, cmd.ProcessState.ExitCode(), string(out), err)
		}
		if err != nil {
			return fmt.Errorf("failed to enable feature %s: %s (%w)", tweak.Name, string(out), err)
		}
		return nil

	default: // Registry
		valType := tweak.Type
		if valType == "" {
			valType = "DWord"
		}
		psScript := fmt.Sprintf(`
$path = "%s"
$name = "%s"
$val = %v
$type = "%s"
if (-not (Test-Path $path)) {
    New-Item -Path $path -Force | Out-Null
}
Set-ItemProperty -Path $path -Name $name -Value $val -Type $type -Force | Out-Null
`, tweak.Path, tweak.Name, tweak.Value, valType)
		cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psScript)
		out, err := cmd.CombinedOutput()
		if m.logger != nil {
			m.logger.LogCommand("powershell.exe", []string{"-Command", psScript}, cmd.ProcessState.ExitCode(), string(out), err)
		}
		if err != nil {
			return fmt.Errorf("failed to set registry %s\\%s: %s (%w)", tweak.Path, tweak.Name, string(out), err)
		}
		return nil
	}
}

func (m *TweaksManager) EnsureTweaks(ctx context.Context, tweaks []entity.WindowsTweak) ([]entity.Diagnostic, error) {
	var diags []entity.Diagnostic
	for _, tw := range tweaks {
		targetName := fmt.Sprintf("%s\\%s", tw.Path, tw.Name)
		if tw.Path == "" {
			targetName = fmt.Sprintf("[%s] %s", tw.Type, tw.Name)
		}

		ok, details, err := m.CheckTweak(ctx, tw)
		if err != nil {
			if m.logger != nil {
				m.logger.Error("Windows tweak check failed for %s: %v", targetName, err)
			}
			diags = append(diags, entity.Diagnostic{
				Category: entity.DiagError,
				System:   "Windows11",
				Target:   targetName,
				Details:  fmt.Sprintf("Failed to check tweak: %v", err),
			})
			continue
		}

		if ok {
			if m.logger != nil {
				m.logger.LogIdempotency("Windows11", targetName, true, "Already configured correctly: "+details)
			}
			diags = append(diags, entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Windows11",
				Target:   targetName,
				Details:  details,
			})
			continue
		}

		// Needs application
		if m.logger != nil {
			m.logger.Info("Applying Windows tweak %s (current: %s)", targetName, details)
		}
		if err := m.ApplyTweak(ctx, tw); err != nil {
			if m.logger != nil {
				m.logger.Error("Failed to apply Windows tweak %s: %v", targetName, err)
			}
			diags = append(diags, entity.Diagnostic{
				Category: entity.DiagError,
				System:   "Windows11",
				Target:   targetName,
				Details:  fmt.Sprintf("Failed to apply: %v", err),
				FixHint:  "Run terminal as Administrator if required for HKLM settings",
			})
		} else {
			if m.logger != nil {
				m.logger.LogIdempotency("Windows11", targetName, false, "Applied successfully")
			}
			diags = append(diags, entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Windows11",
				Target:   targetName,
				Details:  "Applied successfully",
			})
		}
	}
	return diags, nil
}
