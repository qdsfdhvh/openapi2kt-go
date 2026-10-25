package diagnostic

import "strings"

// WarningCode recognizes exact ASCII policy identifiers. Unknown well-formed
// codes remain usable for forward-compatible allowlists, never as patterns.
func WarningCode(code string) bool {
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
