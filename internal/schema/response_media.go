package schema

import (
	"strings"

	"openapi-kt/internal/document"
)

// Raw scalar media are fallbacks after each context's JSON/SSE/form preference.
// An exact media spelling wins over parameter/case variants of the same kind.
func rawValueMedia(content *document.Value) string {
	best, priority := "", 4
	for _, member := range httpMembers(content) {
		rank := 4
		base := strings.ToLower(strings.TrimSpace(strings.SplitN(member.Name, ";", 2)[0]))
		switch base {
		case "text/plain":
			rank = 1
			if member.Name == "text/plain" {
				rank = 0
			}
		case "application/octet-stream":
			rank = 3
			if member.Name == "application/octet-stream" {
				rank = 2
			}
		}
		if rank < priority {
			best, priority = member.Name, rank
		}
	}
	return best
}

func rawValueType(media string, ty TypeRef) bool {
	value := bare(ty)
	base := strings.ToLower(strings.TrimSpace(strings.SplitN(media, ";", 2)[0]))
	switch base {
	case "text/plain":
		return value.Kind == Scalar && value.Name == "kotlin.String"
	case "application/octet-stream":
		return value.Kind == Scalar && value.Name == "kotlin.ByteArray"
	default:
		return true // Existing selected JSON media keep their original type policy.
	}
}
