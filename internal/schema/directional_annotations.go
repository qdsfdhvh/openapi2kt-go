package schema

import (
	"strconv"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Only schema edges are inspected. Annotation payloads such as examples and
// default JSON are opaque data, even if they contain a writeOnly key.
func (a *analyzer) directionalAnnotation(v *document.Value, pointer, key string) (string, *diagnostic.Diagnostic) {
	seen := map[*document.Value]bool{}
	work := 0
	var visit func(*document.Value, string) (string, *diagnostic.Diagnostic)
	visit = func(v *document.Value, pointer string) (string, *diagnostic.Diagnostic) {
		if v == nil || seen[v] {
			return "", nil
		}
		seen[v] = true
		work++
		if work > 16384 {
			return "", diagnostic.At("E_SCHEMA_LIMIT", "Directional schema inspection exceeds the supported limit.", pointer)
		}
		if value, _ := v.Get(key); value != nil {
			flag, _ := value.Bool()
			if flag {
				return naming.Pointer(pointer, key), nil
			}
		}
		if reference := get(v, "$ref"); reference.Kind() == document.String {
			// Annotation discovery does not own reference validation. A malformed
			// or missing edge keeps the shared planner's original diagnostic.
			if target, err := a.resolveSchema(text(reference), pointer); err == nil {
				if at, err := visit(target.Schema, target.Pointer); err != nil || at != "" {
					return at, err
				}
			}
		}
		if properties := get(v, "properties"); properties != nil {
			for _, member := range properties.Members() {
				if at, err := visit(member.Value, naming.Pointer(pointer+"/properties", member.Name)); err != nil || at != "" {
					return at, err
				}
			}
		}
		for _, edge := range []string{"items", "additionalProperties"} {
			if at, err := visit(get(v, edge), naming.Pointer(pointer, edge)); err != nil || at != "" {
				return at, err
			}
		}
		for _, edge := range []string{"allOf", "oneOf", "anyOf", "prefixItems"} {
			if children := get(v, edge); children != nil {
				for n, child := range children.Items() {
					if at, err := visit(child, naming.Pointer(naming.Pointer(pointer, edge), strconv.Itoa(n))); err != nil || at != "" {
						return at, err
					}
				}
			}
		}
		return "", nil
	}
	return visit(v, pointer)
}

func (a *analyzer) rejectWriteOnlyProjection(v *document.Value, pointer string) *diagnostic.Diagnostic {
	at, err := a.directionalAnnotation(v, pointer, "writeOnly")
	if err != nil {
		return err
	}
	if at != "" {
		return unsupported(at)
	}
	return nil
}
