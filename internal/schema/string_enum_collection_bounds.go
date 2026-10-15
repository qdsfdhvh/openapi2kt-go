package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Bounds belong to the ordinary DTO collection use, not the open vocabulary's
// serializer. Only this direct inline role grants checked String enum leaves.
func directStringEnumCollectionBounds(v *document.Value) bool {
	var leaf *document.Value
	if bareArrayDefaultSchema(v) {
		leaf = get(v, "items")
	} else if dictionarySchema(v) {
		leaf = get(v, "additionalProperties")
	}
	if leaf == nil || leaf.Kind() != document.Object || !has(leaf, "enum") || !hasPrimitiveBounds(leaf) {
		return false
	}
	kind, _, err := schemaType(leaf, "")
	return err == nil && (kind == "" || kind == "string")
}

func (a *analyzer) stringEnumCollectionLeaf(v *document.Value, name, pointer string, nested *[]Declaration, used map[string]bool, depth int) (TypeRef, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, pointer); err != nil {
		return TypeRef{}, err
	}
	keys := []string{"type", "nullable", "enum"}
	if a.options.Validation {
		keys = append(keys, primitiveBoundKeys...)
	}
	enum, err := buildStringEnumAllowedKeys(name, v, pointer, keys)
	if err != nil {
		return TypeRef{}, err
	}
	checks, err := a.primitiveValidation(v, "kotlin.String", pointer)
	if err != nil {
		return TypeRef{}, err
	}
	if checks != nil {
		a.validationSources[checks] = pointer
		a.emissionHints[checks] = true // Keep wide new field checks out of JVM constructors.
	}
	_, nullable, _ := schemaType(v, pointer)
	if nested != nil {
		if err := reserveNested(used, name, pointer); err != nil {
			return TypeRef{}, err
		}
		a.recordEnumOrigin(enum, v, pointer, nullable)
		*nested = append(*nested, enum)
	}
	ty := TypeRef{Kind: Named, Name: name, Validation: checks}
	if nullable {
		ty = ty.Null()
	}
	return ty, nil
}

func (a *analyzer) stringEnumCollectionField(f field, name string, nested *[]Declaration, used map[string]bool) (TypeRef, *diagnostic.Diagnostic) {
	if err := depthCheck(f.Depth, f.Pointer); err != nil {
		return TypeRef{}, err
	}
	v := f.Schema
	kind, nullable, err := schemaType(v, f.Pointer)
	if err != nil {
		return TypeRef{}, err
	}
	ty := TypeRef{Kind: List}
	key, suffix := "items", "Inner"
	if kind == "object" {
		ty.Kind = Map
		key, suffix = "additionalProperties", "Value"
		if _, err := a.dictionaryValue(v, f.Pointer); err != nil {
			return TypeRef{}, err
		}
	} else if err := checkKeys(v, a.collectionKeys([]string{"type", "nullable", "items"}), f.Pointer); err != nil {
		return TypeRef{}, err
	}
	item, err := a.stringEnumCollectionLeaf(get(v, key), name+suffix, naming.Pointer(f.Pointer, key), nested, used, f.Depth+1)
	if err != nil {
		return TypeRef{}, err
	}
	ty.Element = &item
	ty, err = a.checkedCollection(ty, v, f.Pointer)
	if err != nil {
		return TypeRef{}, err
	}
	if nullable {
		ty = ty.Null()
	}
	return ty, nil
}
