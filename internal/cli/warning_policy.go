package cli

import (
	"openapi-kt/core"
	"openapi-kt/internal/diagnostic"
)

// Warning codes are exact ASCII identifiers, not patterns or comma-separated
// lists. Unknown well-formed codes are harmless and permit forward-compatible
// allowlists; they cannot suppress a warning with a different code.
func warningCode(code string) bool { return diagnostic.WarningCode(code) }

// Policy is an adapter-level acceptance decision. It never alters artifacts or
// hides warnings, and must run before output.Write (including read-only check).
func deniedWarnings(warnings core.Diagnostics, allowed []string) core.Diagnostics {
	allow := make(map[string]bool, len(allowed))
	for _, code := range allowed {
		allow[code] = true
	}
	var denied core.Diagnostics
	for _, warning := range warnings {
		if !allow[warning.Code] {
			denied = append(denied, core.Diagnostic{Code: "E_WARNING_DENIED", Message: "Generation denied warning " + warning.Code + ".", Pointer: warning.Pointer})
		}
	}
	return denied
}

// Configuration policy preserves each spec's effective default/spec/invocation
// settings. Evaluate once after the complete generated inventory is available,
// before any output transaction or migration report can be applied.
func configDeniedWarnings(specs []specReport, settings map[string]arguments, timer *metricCollector) core.Diagnostics {
	enabled := false
	for _, spec := range specs {
		enabled = enabled || settings[spec.Name].denyWarnings
	}
	if !enabled {
		return nil
	}
	end := timer.phase("warning_policy")
	defer end()
	var denied core.Diagnostics
	for _, spec := range specs {
		a := settings[spec.Name]
		if a.denyWarnings {
			denied = append(denied, deniedWarnings(spec.Warnings, a.allowWarnings)...)
		}
	}
	return denied
}
