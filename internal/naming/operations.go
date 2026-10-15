package naming

import (
	"strings"

	"openapi-kt/internal/diagnostic"
)

// Operation names use the API spelling policy, independently of DTO property
// names (which preserve underscores). Unsupported spellings fail, never become
// unsafe Kotlin or filesystem paths.
func apiWords(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool { return strings.ContainsRune("_- .;[]/<>:\\", r) })
}

func APIClassName(value, pointer string) (string, *diagnostic.Diagnostic) {
	var b strings.Builder
	for _, word := range apiWords(value) {
		b.WriteString(strings.ToUpper(word[:1]))
		b.WriteString(word[1:])
	}
	name := b.String()
	if !identifier(name) || ReservedPathSegment(name) {
		return "", diagnostic.At("E_NAME_INVALID", "API name must resolve to a safe ASCII identifier.", pointer)
	}
	return name, nil
}

func APICamelName(value, pointer string) (string, *diagnostic.Diagnostic) {
	allCaps := true
	for _, c := range []byte(value) {
		if c >= 'a' && c <= 'z' {
			allCaps = false
		}
	}
	var name string
	if allCaps {
		// The legacy all-caps rule keeps underscores/hyphens/spaces. Reject
		// spellings that still cannot form an identifier instead of renaming them.
		parts := strings.FieldsFunc(value, func(r rune) bool { return strings.ContainsRune(".;[]/<>:\\", r) })
		name = strings.ToLower(strings.Join(parts, "_"))
	} else {
		pascal, err := APIClassName(value, pointer)
		if err != nil {
			return "", err
		}
		name = strings.ToLower(pascal[:1]) + pascal[1:]
	}
	if !identifier(name) {
		return "", diagnostic.At("E_NAME_INVALID", "API name must resolve to a safe ASCII identifier.", pointer)
	}
	return name, nil
}

// Explicit client/exclusion names are identifiers, not inputs to a renaming
// algorithm. Exact class spelling matters for both call sites and filtering.
func ValidateClientOptions(client string, excluded []string) *diagnostic.Diagnostic {
	valid := func(name string) bool { return identifier(name) && !keywords[name] && !ReservedPathSegment(name) }
	if client != "" && !valid(client) {
		return diagnostic.New("E_NAME_INVALID", "API client name must be a safe ASCII Kotlin identifier.")
	}
	for _, name := range excluded {
		if !valid(name) {
			return diagnostic.New("E_NAME_INVALID", "Excluded API names must be safe ASCII Kotlin identifiers.")
		}
	}
	return nil
}

func CodeName(name string) string {
	if keywords[name] {
		return "`" + name + "`"
	}
	return name
}

func MethodName(operationID *string, method, path, pointer string) (string, *diagnostic.Diagnostic) {
	if operationID == nil {
		parts := []string{method}
		for _, segment := range strings.Split(path, "/") {
			if segment != "" && !strings.HasPrefix(segment, "{") && !strings.HasPrefix(segment, "@") {
				parts = append(parts, segment)
			}
		}
		return APICamelName(strings.Join(parts, "_"), pointer)
	}
	segments := strings.Split(*operationID, ".")
	tag := segments[0]
	if len(segments) > 1 {
		segments = segments[1:]
	}
	rest := func(s string) bool {
		switch s {
		case "get", "list", "create", "update", "delete", "patch", "put", "post":
			return true
		}
		return false
	}
	if len(segments) == 1 {
		s := segments[0]
		special := rest(s)
		switch s {
		case "set", "in", "is", "as", "return", "when", "while", "for", "do", "if", "else", "try", "catch", "finally":
			special = true
		}
		if special {
			return APICamelName(s+"_"+tag, pointer)
		}
		if identifier(s) {
			return s, nil
		}
		return APICamelName(s, pointer)
	}
	last := segments[len(segments)-1]
	if rest(last) {
		segments = append([]string{last}, segments[:len(segments)-1]...)
	}
	return APICamelName(strings.Join(segments, "_"), pointer)
}
