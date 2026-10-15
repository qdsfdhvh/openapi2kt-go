// Package naming centralizes Kotlin identifiers, literals, and JSON Pointers.
package naming

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf16"

	"openapi-kt/internal/diagnostic"
)

var keywords = map[string]bool{
	"as": true, "break": true, "class": true, "continue": true,
	"do": true, "else": true, "false": true, "for": true, "fun": true,
	"if": true, "in": true, "interface": true, "is": true, "null": true,
	"object": true, "package": true, "return": true, "super": true,
	"this": true, "throw": true, "true": true, "try": true, "typealias": true,
	"typeof": true, "val": true, "var": true, "when": true, "while": true,
}

func asciiLetter(b byte) bool { return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' }
func asciiDigit(b byte) bool  { return b >= '0' && b <= '9' }

func identifier(value string) bool {
	if value == "" || !(asciiLetter(value[0]) || value[0] == '_') {
		return false
	}
	onlyUnderscores := true
	for i := 0; i < len(value); i++ {
		b := value[i]
		if b != '_' {
			onlyUnderscores = false
			if !asciiLetter(b) && !asciiDigit(b) {
				return false
			}
		}
	}
	return !onlyUnderscores
}

func ReservedPathSegment(value string) bool {
	upper := strings.ToUpper(value)
	switch upper {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	return len(upper) == 4 && (strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT")) && upper[3] >= '1' && upper[3] <= '9'
}

func ValidateOptions(packageName, suffix string) *diagnostic.Diagnostic {
	parts := strings.Split(packageName, ".")
	valid := packageName != "" && parts[0] != "kotlin"
	for _, part := range parts {
		valid = valid && identifier(part) && !keywords[part] && !ReservedPathSegment(part)
	}
	if !valid {
		return diagnostic.New("E_PACKAGE_INVALID", "Package must contain safe dot-separated ASCII identifiers outside the reserved kotlin namespace.")
	}
	if suffix != "" && !identifier(suffix) {
		return diagnostic.New("E_NAME_INVALID", "Model suffix must be an ASCII identifier.")
	}
	return nil
}

// ValidateNativeType checks reference syntax, not output package ownership. A
// caller-owned classifier may live in kotlin or use a filesystem-reserved word:
// no filename or declaration is created for this type.
func ValidateNativeType(value string) bool {
	for _, part := range strings.Split(value, ".") {
		if !identifier(part) || keywords[part] {
			return false
		}
	}
	return true
}

func ClassName(wire, pointer string) (string, *diagnostic.Diagnostic) {
	words := strings.FieldsFunc(wire, func(c rune) bool {
		return c > 127 || !(asciiLetter(byte(c)) || asciiDigit(byte(c)))
	})
	if len(words) == 0 {
		return "", diagnostic.At("E_NAME_INVALID", "Name must contain an ASCII letter or digit.", pointer)
	}
	var result strings.Builder
	for _, word := range words {
		result.WriteString(strings.ToUpper(word[:1]))
		result.WriteString(word[1:])
	}
	name := result.String()
	if asciiDigit(name[0]) {
		name = "N" + name
	}
	return name, nil
}

func PropertyName(wire, pointer string) (string, *diagnostic.Diagnostic) {
	name := wire
	if !identifier(wire) {
		pascal, err := ClassName(wire, pointer)
		if err != nil {
			return "", err
		}
		name = strings.ToLower(pascal[:1]) + pascal[1:]
	}
	if keywords[name] {
		name = "`" + name + "`"
	}
	return name, nil
}

// EnumMemberName uses the separate enum spelling policy, including UTF-16
// fallback names for non-ASCII/operator-only wire strings. Collision suffixes
// and serializer-owned names are assigned by the enum planner, not this helper.
func EnumMemberName(wire string) string {
	var sanitized strings.Builder
	hasAlnum := false
	for _, r := range wire {
		if r <= 127 && (asciiLetter(byte(r)) || asciiDigit(byte(r)) || r == '_') {
			sanitized.WriteString(strings.ToUpper(string(r)))
			hasAlnum = hasAlnum || r != '_'
		} else {
			sanitized.WriteByte('_')
		}
	}
	if hasAlnum {
		name := sanitized.String()
		if asciiDigit(name[0]) {
			name = "_" + name
		}
		return name
	}
	var parts []string
	for _, unit := range utf16.Encode([]rune(wire)) {
		if enumWhitespace(unit) {
			continue
		}
		if word, ok := operatorWords[unit]; ok {
			parts = append(parts, word)
		} else {
			parts = append(parts, fmt.Sprintf("U%04X", unit))
		}
	}
	if len(parts) == 0 {
		return "VALUE"
	}
	return strings.Join(parts, "_")
}

func enumWhitespace(unit uint16) bool {
	return unit >= 0x0009 && unit <= 0x000d || unit >= 0x001c && unit <= 0x0020 ||
		unit == 0x00a0 || unit == 0x1680 || unit >= 0x2000 && unit <= 0x200a ||
		unit == 0x2028 || unit == 0x2029 || unit == 0x202f || unit == 0x205f || unit == 0x3000
}

var operatorWords = map[uint16]string{
	0x21: "NOT", 0x22: "DOUBLE_QUOTE", 0x23: "HASH", 0x24: "DOLLAR",
	0x25: "PERCENT", 0x26: "AND", 0x27: "QUOTE", 0x28: "LEFT_PAREN",
	0x29: "RIGHT_PAREN", 0x2a: "STAR", 0x2b: "PLUS", 0x2c: "COMMA",
	0x2d: "MINUS", 0x2e: "DOT", 0x2f: "SLASH", 0x3a: "COLON",
	0x3b: "SEMICOLON", 0x3c: "LESS", 0x3d: "EQUAL", 0x3e: "GREATER",
	0x3f: "QUESTION", 0x40: "AT", 0x5b: "LEFT_BRACKET", 0x5c: "BACKSLASH",
	0x5d: "RIGHT_BRACKET", 0x5e: "CARET", 0x60: "BACKTICK", 0x7b: "LEFT_BRACE",
	0x7c: "OR", 0x7d: "RIGHT_BRACE", 0x7e: "TILDE",
}

// Literal returns a Kotlin string literal, not a JSON or Go string literal.
func Literal(value string) string {
	var result strings.Builder
	result.WriteByte('"')
	for _, r := range value {
		switch r {
		case '"':
			result.WriteString(`\"`)
		case '\\':
			result.WriteString(`\\`)
		case '$':
			result.WriteString(`\$`)
		case '\n':
			result.WriteString(`\n`)
		case '\r':
			result.WriteString(`\r`)
		case '\t':
			result.WriteString(`\t`)
		default:
			if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
				fmt.Fprintf(&result, "\\u%04x", r)
			} else {
				result.WriteRune(r)
			}
		}
	}
	result.WriteByte('"')
	return result.String()
}

func Pointer(parent, key string) string {
	return parent + "/" + strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1")
}

// DecodePointerSegment only decodes ~0/~1 once. Percent-encoded URI fragments
// and nested schema references have a separate, stricter component resolver.
func DecodePointerSegment(encoded, pointer string) (string, *diagnostic.Diagnostic) {
	var out strings.Builder
	for i := 0; i < len(encoded); i++ {
		if encoded[i] != '~' {
			out.WriteByte(encoded[i])
			continue
		}
		if i+1 >= len(encoded) || encoded[i+1] != '0' && encoded[i+1] != '1' {
			return "", diagnostic.At("E_SCHEMA_INVALID", "Invalid JSON Pointer escape.", pointer)
		}
		i++
		if encoded[i] == '0' {
			out.WriteByte('~')
		} else {
			out.WriteByte('/')
		}
	}
	return out.String(), nil
}
