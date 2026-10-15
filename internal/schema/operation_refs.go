package schema

import (
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// operationReference borrows a parameter/body/response object at its original
// location. It never expands or rewrites the JSON tree, visits schema payloads,
// or turns a component schema reference into an unnamed type.
func (a *analyzer) operationReference(v *document.Value, pointer string) (*document.Value, string, *diagnostic.Diagnostic) {
	seen := map[*document.Value]bool{}
	for depth := 0; ; depth++ {
		if err := object(v, pointer); err != nil {
			return nil, "", err
		}
		if !has(v, "$ref") {
			return v, pointer, nil
		}
		location := pointer + "/$ref"
		if seen[v] {
			return nil, "", diagnostic.At("E_REF_CYCLE", "HTTP reference chain is cyclic.", location)
		}
		seen[v] = true
		if depth >= MaxSchemaDepth {
			return nil, "", diagnostic.At("E_SCHEMA_DEPTH", "HTTP reference chain exceeds the supported depth.", location)
		}
		if err := operationKeys(v, []string{"$ref", "summary", "description"}, pointer); err != nil {
			return nil, "", err
		}
		ref, ok := get(v, "$ref").Text()
		if !ok {
			return nil, "", invalid("$ref must be a string.", location)
		}
		var err *diagnostic.Diagnostic
		v, pointer, err = a.localHTTPPointer(ref, location)
		if err != nil {
			return nil, "", err
		}
	}
}

func (a *analyzer) localHTTPPointer(ref, source string) (*document.Value, string, *diagnostic.Diagnostic) {
	if !strings.HasPrefix(ref, "#/") || strings.Contains(ref, "%") {
		return nil, "", diagnostic.At("E_REF_UNSUPPORTED", "Only unencoded local JSON Pointer references are supported.", source)
	}
	value := a.root
	pointer := ""
	for _, raw := range strings.Split(ref[2:], "/") {
		key, err := naming.DecodePointerSegment(raw, source)
		if err != nil {
			return nil, "", err
		}
		pointer = naming.Pointer(pointer, key)
		switch value.Kind() {
		case document.Object:
			value = get(value, key)
		case document.Array:
			valid := key != "" && (len(key) == 1 || key[0] != '0')
			for _, c := range []byte(key) {
				valid = valid && c >= '0' && c <= '9'
			}
			if !valid {
				return nil, "", invalid("Expected a canonical JSON Pointer array index.", source)
			}
			index, e := strconv.Atoi(key)
			if e != nil || index >= len(value.Items()) {
				value = nil
			} else {
				value = value.Items()[index]
			}
		default:
			value = nil
		}
		if value == nil {
			return nil, "", diagnostic.At("E_REF_MISSING", "HTTP reference target does not exist.", source)
		}
	}
	return value, pointer, nil
}
