package entity

// Diagnostic constructors centralize the audit/fixHint idiom in one place:
// every call site that used to spell the five-field literal now names the
// severity up front. fixHint is optional; an omitted hint is the struct zero
// value (""), so a migrated call site produces a byte-identical Diagnostic.
func OK(system, target, details string, fixHint ...string) Diagnostic {
	return newDiagnostic(DiagOK, system, target, details, fixHint)
}

// Info records context the operator did not ask to change: absence is
// signal, never drift, so Info never carries a warning.
func Info(system, target, details string, fixHint ...string) Diagnostic {
	return newDiagnostic(DiagInfo, system, target, details, fixHint)
}

// Warn asks the operator to look: the fourth argument is the remediation.
func Warn(system, target, details string, fixHint ...string) Diagnostic {
	return newDiagnostic(DiagWarning, system, target, details, fixHint)
}

// Error blocks the run: the fourth argument is the remediation.
func Error(system, target, details string, fixHint ...string) Diagnostic {
	return newDiagnostic(DiagError, system, target, details, fixHint)
}

func newDiagnostic(status DiagnosticStatus, system, target, details string, fixHint []string) Diagnostic {
	d := Diagnostic{Category: status, System: system, Target: target, Details: details}
	if len(fixHint) > 0 {
		d.FixHint = fixHint[0]
	}
	return d
}
