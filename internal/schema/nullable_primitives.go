package schema

import (
	"openapi-kt/internal/document"
	"slices"
)

// builtinPrimitiveBinding checks the already validated canonical catalog root,
// not an alias's flattened scalar spelling (which could hide enum/native tags).
func builtinPrimitiveBinding(c *component) bool {
	if c == nil || c.Reference != "" || !slices.Contains([]string{"kotlin.String", "kotlin.Boolean", "kotlin.Int", "kotlin.Long", "kotlin.Float", "kotlin.Double"}, c.Scalar) {
		return false
	}
	kind, _, err := schemaType(c.Schema, c.Pointer)
	return err == nil && nullablePrimitiveCandidate(c.Schema, kind)
}

// Only ordinary builtin scalar roots gain transparent nullable intent. Existing
// native/enum/const/value-class profiles remain separately admitted and owned.
// Malformed format grammar is still diagnosed by the normal scalar parser.
func nullablePrimitiveCandidate(v *document.Value, kind string) bool {
	for _, key := range []string{"enum", "const", "x-native-type", "x-kotlin-value-class"} {
		if has(v, key) {
			return false
		}
	}
	format := get(v, "format")
	if has(v, "format") && format.Kind() != document.String {
		return kind == "string" || kind == "integer" || kind == "number" || kind == "boolean"
	}
	switch kind {
	case "string", "boolean":
		return text(format) == ""
	case "integer":
		return text(format) == "" || text(format) == "int32" || text(format) == "int64"
	case "number":
		return text(format) == "" || text(format) == "float" || text(format) == "double"
	default:
		return false
	}
}
