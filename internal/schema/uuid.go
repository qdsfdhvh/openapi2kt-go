package schema

import "strings"

// canonicalUUID mirrors Kotlin Uuid.parse: 32 hex digits, either compact or
// with exactly the four standard hyphens, in either ASCII case. No version or
// variant constraint is imposed by the provider. Keep generator defaults on the
// same grammar and canonical text used by Uuid.toString and its JSON serializer.
func canonicalUUID(value string) (string, bool) {
	if len(value) != 32 && len(value) != 36 {
		return "", false
	}
	var digits strings.Builder
	for i := range len(value) {
		if len(value) == 36 && (i == 8 || i == 13 || i == 18 || i == 23) {
			if value[i] != '-' {
				return "", false
			}
			continue
		}
		c := value[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return "", false
		}
		digits.WriteByte(c)
	}
	hex := strings.ToLower(digits.String())
	return hex[:8] + "-" + hex[8:12] + "-" + hex[12:16] + "-" + hex[16:20] + "-" + hex[20:], true
}
