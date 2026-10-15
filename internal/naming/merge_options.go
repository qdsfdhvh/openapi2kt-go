package naming

import (
	"strings"

	"openapi-kt/internal/diagnostic"
)

// ValidateMergeOptions is shared by core and complete CLI configuration inventory.
// Exclusions are exact generated root names, not regular expressions.
func ValidateMergeOptions(enabled bool, exclusions []string) *diagnostic.Diagnostic {
	if len(exclusions) != 0 && !enabled {
		return diagnostic.New("E_OPTIONS", "Model merge exclusions require model merging.")
	}
	for _, name := range exclusions {
		if !strings.Contains(name, ".") || ValidateOptions(name, "") != nil {
			return diagnostic.New("E_OPTIONS", "Merge exclusions must be fully qualified classifier names.")
		}
	}
	return nil
}
