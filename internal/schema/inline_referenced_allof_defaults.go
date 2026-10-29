package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Recognize only an inline container path followed by actual reference edges
// ending at allOf. This is routing, not admission: the shared type plan and
// final retained nominal/source checks still authorize the actual constructor.
func (a *analyzer) inlineReferencedAllOfCollection(v *document.Value) bool {
	return a.inlineReferencedAllOfElement(v) != nil
}

// Return the original element reference, not its resolved terminal. A caller
// can scope an object property's source-walk permission to this exact node.
func (a *analyzer) inlineReferencedAllOfElement(v *document.Value) *document.Value {
	if has(v, "$ref") {
		return nil
	}
	var reference *document.Value
	for edges := 0; edges <= MaxSchemaDepth && v != nil; edges++ {
		if edges > 0 && has(v, "$ref") {
			source, _, err := a.defaultSource(text(get(v, "$ref")), "")
			if err != nil {
				return nil
			}
			if reference == nil {
				reference = v
			}
			v = source
			continue
		}
		if reference != nil {
			if has(v, "allOf") {
				return reference
			}
			return nil
		}
		kind, _, err := schemaType(v, "")
		if err != nil {
			return nil
		}
		switch {
		case kind == "array":
			v = get(v, "items")
		case dictionarySchema(v):
			v = get(v, "additionalProperties")
		default:
			return nil
		}
	}
	return nil
}

func (a *analyzer) deferredFieldDefault(v *document.Value, pointer string) bool {
	if has(v, "prefixItems") && !has(v, "$ref") {
		return a.tupleCollectionDeferred(v)
	}
	return has(v, "$ref") || a.inlineReferencedAllOfCollection(v) || a.inlineReferencedScalarCollection(v, pointer)
}

// Route only an original inline finite container with a referenced scalar
// terminal. The ordinary field, shared type planner and strict default walk
// still authorize its actual type and original annotation.
func (a *analyzer) inlineReferencedScalarCollection(v *document.Value, pointer string) bool {
	return !has(v, "$ref") && a.dtoUniqueBuiltinCollectionReferenceSource(v, pointer)
}

// Plan the original inline collection type without validating the value yet.
// Element names/owners are authoritative only after the final nominal barrier.
func (a *analyzer) inlineReferencedAllOfDefaultType(f field, ordinary bool, name string, nested *[]Declaration, used map[string]bool) (TypeRef, *PrimitiveValidation, *diagnostic.Diagnostic) {
	if !ordinary {
		return TypeRef{}, nil, unsupported(naming.Pointer(f.Pointer, "default"))
	}
	if nested == nil {
		scratch := []Declaration{}
		nested = &scratch
	}
	if used == nil {
		used = map[string]bool{}
	}
	if a.objectDefaultSchemas == nil {
		a.objectDefaultSchemas = map[*document.Value]bool{}
	}
	a.objectDefaultSchemas[f.Schema] = true
	return a.modelPropertyType(f.Schema, name, f.Pointer, nested, used, f.Depth)
}
