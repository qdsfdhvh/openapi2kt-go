package schema

import (
	"regexp/syntax"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// validationPattern checks the declared portable source grammar, not instance
// matches. The original string is retained for Kotlin Regex.containsMatchIn.
// No Go regexp rewrite or inference about target backtracking cost is made.
func validationPattern(v *document.Value, pointer string) (string, *diagnostic.Diagnostic) {
	pattern, ok := v.Text()
	if !ok {
		return "", invalid("pattern must be a string.", pointer)
	}
	if len(pattern) > 1024 {
		return "", invalid("Pattern exceeds the supported source-size limit.", pointer)
	}
	inClass, previousQuantifier := false, false
	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]
		if ch == '\\' {
			previousQuantifier = false
			i++
			if i == len(pattern) {
				return "", invalid("Malformed pattern.", pointer)
			}
			if !strings.ContainsRune(`dDsSwWnrt\.^$|?*+()[]{}`, rune(pattern[i])) {
				return "", unsupported(pointer)
			}
			continue
		}
		if inClass {
			previousQuantifier = false
			if ch == ']' {
				inClass = false
			}
			continue
		}
		if ch == '[' {
			previousQuantifier = false
			inClass = true
			continue
		}
		if ch == '(' && i+1 < len(pattern) && pattern[i+1] == '?' && (i+2 == len(pattern) || pattern[i+2] != ':') {
			return "", unsupported(pointer)
		}
		if ch == '+' && previousQuantifier {
			return "", unsupported(pointer)
		}
		previousQuantifier = strings.ContainsRune("*+?}", rune(ch))
	}
	tree, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return "", invalid("Malformed or unsupported portable pattern.", pointer)
	}
	nodes := []*syntax.Regexp{tree}
	count := 0
	for len(nodes) != 0 {
		node := nodes[len(nodes)-1]
		nodes = nodes[:len(nodes)-1]
		count++
		if count > 256 {
			return "", invalid("Pattern exceeds the supported syntax-size limit.", pointer)
		}
		nodes = append(nodes, node.Sub...)
	}
	return pattern, nil
}
