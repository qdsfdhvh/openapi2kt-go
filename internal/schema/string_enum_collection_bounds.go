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

// Discover one original leaf in a finite inline List/Map path. Admission still
// belongs to the ordinary field planner and validates every original node.
// Single-layer fields keep their established planner and emitted bytes.
func nestedStringEnumBoundsLeaf(v *document.Value) *document.Value {
	return inlineStringEnumBoundsLeaf(v, 2)
}

// Collection roots and ordinary fields select their own minimum path length;
// the returned node alone grants no source or declaration admission.
func inlineStringEnumBoundsLeaf(v *document.Value, minimumEdges int) *document.Value {
	leaf := inlineStringEnumCollectionLeaf(v, minimumEdges)
	if leaf != nil && directStringEnumFieldBounds(leaf) {
		return leaf
	}
	return nil
}

func inlineStringEnumCollectionLeaf(v *document.Value, minimumEdges int) *document.Value {
	for edges := 0; edges <= MaxSchemaDepth; edges++ {
		if v == nil || v.Kind() != document.Object || has(v, "$ref") || has(v, "prefixItems") {
			return nil
		}
		if has(v, "enum") {
			kind, _, err := schemaType(v, "")
			if edges >= minimumEdges && err == nil && (kind == "" || kind == "string") && lexicalStringEnumSchema(v) {
				return v
			}
			return nil
		}
		if bareArrayDefaultSchema(v) {
			v = get(v, "items")
		} else if dictionarySchema(v) {
			v = get(v, "additionalProperties")
		} else {
			return nil
		}
	}
	return nil
}

func (a *analyzer) stringEnumCollectionLeaf(v *document.Value, name, pointer string, nested *[]Declaration, used map[string]bool, depth int) (TypeRef, *diagnostic.Diagnostic) {
	return a.checkedStringEnumLeaf(v, name, pointer, nested, used, depth, false)
}

func directStringEnumFieldBounds(v *document.Value) bool {
	if !has(v, "enum") || !hasPrimitiveBounds(v) {
		return false
	}
	kind, _, err := schemaType(v, "")
	return err == nil && (kind == "" || kind == "string")
}

// Both direct ordinary DTO roles retain lexical String codecs. Metadata never
// selects a date, UUID, byte-array or provider codec for these enum uses.
func (a *analyzer) checkedStringEnumLeaf(v *document.Value, name, pointer string, nested *[]Declaration, used map[string]bool, depth int, ownDefault bool) (TypeRef, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, pointer); err != nil {
		return TypeRef{}, err
	}
	keys := []string{"type", "nullable", "enum", "format", "x-native-type"}
	if ownDefault {
		keys = append(keys, "default")
	}
	if a.options.Validation {
		keys = append(keys, primitiveBoundKeys...)
	}
	enum, err := buildStringEnumAllowedKeys(name, v, pointer, keys)
	if err != nil {
		return TypeRef{}, err
	}
	checks, err := a.checkedEnumValidation(v, pointer)
	if err != nil {
		return TypeRef{}, err
	}
	if checks != nil {
		a.validationSources[checks] = pointer
		a.emissionHints[checks] = true // Keep wide new field checks out of JVM constructors.
	}
	if ownDefault {
		if err := a.checkedStringEnumDefault(v, pointer, checks); err != nil {
			return TypeRef{}, err
		}
	}
	_, nullable, _ := schemaType(v, pointer)
	if nested != nil {
		if err := reserveNested(used, name, pointer); err != nil {
			return TypeRef{}, err
		}
		a.recordEnumOrigin(enum, v, pointer, nullable)
		*nested = append(*nested, enum)
		a.checkedEnumSources[v].Declarations[enum] = true
	}
	ty := TypeRef{Kind: Named, Name: name, Validation: checks}
	if nullable {
		ty = ty.Null()
	}
	return ty, nil
}

func (a *analyzer) checkedStringEnumDefault(v *document.Value, pointer string, checks *PrimitiveValidation) *diagnostic.Diagnostic {
	value := get(v, "default")
	if value == nil {
		return nil
	}
	kind, nullable, err := schemaType(v, pointer)
	if err != nil {
		return err
	}
	at := naming.Pointer(pointer, "default")
	if value.Kind() == document.Null {
		if !nullable {
			return invalid("A null default requires a nullable schema.", at)
		}
		return nil
	}
	if err := a.enumDefaultMemberAt(v, value, kind, pointer, at); err != nil {
		return err
	}
	return validateDefaultChecks(value, v, checks, "kotlin.String", pointer)
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
