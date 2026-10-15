package cli

import (
	"openapi-kt/core"
	"strings"
)

// Warning codes are exact ASCII identifiers, not patterns or comma-separated
// lists. Unknown well-formed codes are harmless and permit forward-compatible
// allowlists; they cannot suppress a warning with a different code.
func warningCode(code string) bool {
	if !strings.HasPrefix(code, "W_") || len(code) == 2 {
		return false
	}
	for _, ch := range code[2:] {
		if ch != '_' && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') {
			return false
		}
	}
	return true
}

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
