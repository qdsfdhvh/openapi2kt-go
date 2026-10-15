package schema

import (
	"strings"
	"unicode"
)

// Hierarchy fallback tags lowercase original component keys, not generated
// names. Full Unicode lowercase differs from Go's simple-case conversion for
// dotted I and context-sensitive final sigma; suffixes never enter this path.
func lowerHierarchyWire(value string) string {
	runes := []rune(value)
	var out strings.Builder
	for index, r := range runes {
		if r == 'İ' {
			out.WriteString("i\u0307")
			continue
		}
		if r == 'Σ' {
			before, after := false, false
			for i := index - 1; i >= 0; i-- {
				if caseIgnorable(runes[i]) {
					continue
				}
				before = isCased(runes[i])
				break
			}
			for i := index + 1; i < len(runes); i++ {
				if caseIgnorable(runes[i]) {
					continue
				}
				after = isCased(runes[i])
				break
			}
			if before && !after {
				out.WriteRune('ς')
				continue
			}
		}
		out.WriteRune(unicode.ToLower(r))
	}
	return out.String()
}
func isCased(r rune) bool {
	return unicode.IsUpper(r) || unicode.IsLower(r) || unicode.IsTitle(r) || unicode.Is(unicode.Properties["Other_Uppercase"], r) || unicode.Is(unicode.Properties["Other_Lowercase"], r)
}
func caseIgnorable(r rune) bool {
	if unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf, unicode.Lm, unicode.Sk) {
		return true
	}
	// Unicode Word_Break MidLetter / MidNumLet / Single_Quote supplement
	// the general-category Case_Ignorable classes above.
	switch r {
	case '\'', '.', ':', '\u00b7', '\u0387', '\u05f4', '\u2018', '\u2019', '\u2024', '\u2027', '\ufe13', '\ufe52', '\ufe55', '\uff07', '\uff0e', '\uff1a':
		return true
	}
	return false
}
