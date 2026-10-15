package schema

import (
	"strconv"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Raw JSON/stringification projections do not retain constructor or leaf checks.
// Inspect only schema edges (never examples/defaults), borrowing component nodes
// and breaking nominal recursion. This is a rejection guard, not expansion.
func (a *analyzer) rejectValidationProjection(v *document.Value, pointer string) *diagnostic.Diagnostic {
	if !a.options.Validation {
		return nil
	}
	seen := map[*document.Value]bool{}
	var visit func(*document.Value, string, int) *diagnostic.Diagnostic
	visit = func(v *document.Value, p string, depth int) *diagnostic.Diagnostic {
		if err := depthCheck(depth, p); err != nil {
			return err
		}
		if v == nil || v.Kind() != document.Object || seen[v] {
			return nil
		}
		seen[v] = true
		if has(v, "enum") {
			kind, _, err := schemaType(v, p)
			if err != nil {
				return err
			}
			if kind == "integer" || kind == "number" || kind == "boolean" {
				return unsupported(naming.Pointer(p, "enum"))
			}
		}
		for _, key := range append(append([]string{}, primitiveBoundKeys...), collectionBoundKeys...) {
			if has(v, key) {
				if key == "uniqueItems" {
					kind, _, err := schemaType(v, p)
					if err != nil || kind != "array" {
						return unsupported(naming.Pointer(p, key))
					}
					flag, ok := get(v, key).Bool()
					if !ok {
						return invalid("uniqueItems must be Boolean.", naming.Pointer(p, key))
					}
					if !flag {
						continue // Validated no-op has no enforcing check to lose.
					}
				}
				return unsupported(naming.Pointer(p, key))
			}
		}
		if has(v, "$ref") {
			c, err := a.resolveSchema(text(get(v, "$ref")), naming.Pointer(p, "$ref"))
			if err != nil {
				return err
			}
			return visit(c.Schema, c.Pointer, depth+1)
		}
		for _, member := range v.Members() {
			location := naming.Pointer(p, member.Name)
			switch member.Name {
			case "items", "additionalProperties":
				if err := visit(member.Value, location, depth+1); err != nil {
					return err
				}
			case "properties":
				if member.Value.Kind() == document.Object {
					for _, field := range member.Value.Members() {
						if err := visit(field.Value, naming.Pointer(location, field.Name), depth+1); err != nil {
							return err
						}
					}
				}
			case "allOf", "oneOf", "anyOf":
				if member.Value.Kind() == document.Array {
					for i, branch := range member.Value.Items() {
						if err := visit(branch, naming.Pointer(location, strconv.Itoa(i)), depth+1); err != nil {
							return err
						}
					}
				}
			}
		}
		return nil
	}
	return visit(v, pointer, 0)
}

func scalarAliasDepth(c *component, depth int, pointer string) *diagnostic.Diagnostic {
	if c.Reference != "" && depth+1+c.AliasHeight > MaxSchemaDepth {
		return diagnostic.At("E_SCHEMA_DEPTH", "Scalar alias use exceeds the supported depth.", pointer)
	}
	return nil
}
