package schema

import (
	"encoding/json"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// This role follows original root-reference edges into admitted finite mapped
// scalar collection trees. Native enum leaves require the original source role
// and validation; reference leaves, nominal values and default inheritance keep
// their separate admission boundaries.
// A reference role may begin at the consuming field or below its original
// inline containers. Pure inline trees retain their existing separate role.
func (a *analyzer) dtoUniqueBuiltinCollectionReferenceSource(v *document.Value, p string) bool {
	if !dtoUniqueCollectionTreeHasReference(v, 0) {
		return false
	}
	return a.dtoUniqueCollectionReferenceTreeSource(v, p, 0, true, true, map[*document.Value]bool{})
}

func dtoUniqueCollectionTreeHasReference(v *document.Value, depth int) bool {
	if v == nil || depth > MaxSchemaDepth {
		return false
	}
	if has(v, "$ref") {
		return true
	}
	if has(v, "items") {
		return dtoUniqueCollectionTreeHasReference(get(v, "items"), depth+1)
	}
	if dictionarySchema(v) {
		return dtoUniqueCollectionTreeHasReference(get(v, "additionalProperties"), depth+1)
	}
	return false
}

func (a *analyzer) dtoUniqueCollectionReferenceTreeSource(v *document.Value, p string, depth int, root, field bool, active map[*document.Value]bool) bool {
	if v == nil || v.Kind() != document.Object || depth > MaxSchemaDepth || active[v] {
		return false
	}
	active[v] = true
	defer delete(active, v)
	if has(v, "$ref") {
		keys := []string{"$ref", "nullable"}
		if field {
			keys = append(keys, "default")
		}
		if checkKeys(v, keys, p) != nil {
			return false
		}
		target, at, err := a.defaultSource(text(get(v, "$ref")), p)
		if err != nil {
			return false
		}
		return a.dtoUniqueCollectionReferenceTreeSource(target, at, depth+1, root, false, active)
	}
	for _, key := range []string{"oneOf", "anyOf", "allOf", "const", "x-native-type", "x-kotlin-value-class"} {
		if has(v, key) {
			return false
		}
	}
	if has(v, "default") && !field {
		return false
	}
	if has(v, "enum") {
		return !root && (a.options.Validation && a.uniqueNativeEnumSource(v, p, depth) || a.uniqueStringEnumSource(v, p, depth))
	}
	kind, _, err := schemaType(v, p)
	if err != nil {
		return false
	}
	edge := ""
	if kind == "array" {
		edge = "items"
	} else if dictionarySchema(v) && (kind == "" || kind == "object") {
		edge = "additionalProperties"
	}
	if edge != "" {
		keys := a.collectionKeys([]string{"type", "nullable", edge})
		if field {
			keys = append(keys, "default")
		}
		if checkKeys(v, keys, p) != nil {
			return false
		}
		return a.dtoUniqueCollectionReferenceTreeSource(get(v, edge), naming.Pointer(p, edge), depth+1, false, false, active)
	}
	return !root && a.uniqueScalarSource(v, p, depth)
}

func dtoUniqueBuiltinCollectionReferenceType(t TypeRef, depth int) bool {
	if depth > MaxSchemaDepth || t.Serializer != "" {
		return false
	}
	if t.Kind == Nullable || t.Kind == Named && t.Element != nil {
		return t.Element != nil && dtoUniqueBuiltinCollectionReferenceType(*t.Element, depth+1)
	}
	return (t.Kind == List || t.Kind == Map) && dtoUniqueBuiltinCollectionTreeType(t, depth)
}

func dtoUniqueBuiltinCollectionTreeType(t TypeRef, depth int) bool {
	if depth > MaxSchemaDepth || t.Serializer != "" {
		return false
	}
	if t.Kind == Nullable || t.Kind == Named && t.Element != nil || t.Kind == List || t.Kind == Map {
		return t.Element != nil && dtoUniqueBuiltinCollectionTreeType(*t.Element, depth+1)
	}
	return uniqueScalarType(t) || uniqueStringEnumType(t)
}

// Preserve every physical name/view/check while ignoring only resource hop
// accounting, which remains enforced separately by original source traversal.
func dtoUniqueCollectionReferenceTypeSignature(t TypeRef) string {
	var normalized func(TypeRef, int) (TypeRef, bool)
	normalized = func(t TypeRef, depth int) (TypeRef, bool) {
		if depth > MaxSchemaDepth {
			return TypeRef{}, false
		}
		t.SchemaSteps = 0
		if t.Element != nil {
			next := depth
			if t.Kind != Nullable {
				next++
			}
			child, ok := normalized(*t.Element, next)
			if !ok {
				return TypeRef{}, false
			}
			t.Element = &child
		}
		return t, true
	}
	ty, ok := normalized(t, 0)
	if !ok {
		return ""
	}
	raw, _ := json.Marshal(ty)
	return string(raw)
}

// The key is an abstract mapped descriptor from the original source, separate
// from the forward physical reference. The final getter and original alias
// graph must prove it after binding; this does not modify an alias backing.
func (a *analyzer) dtoUniqueBuiltinCollectionReferenceMappedType(f field, physical TypeRef) (TypeRef, *diagnostic.Diagnostic) {
	v, at, steps := f.Schema, f.Pointer, 0
	for has(v, "$ref") {
		if steps >= MaxSchemaDepth {
			return TypeRef{}, schemaReferenceUnsupported(f.Pointer)
		}
		target, p, err := a.defaultSource(text(get(v, "$ref")), at)
		if err != nil {
			return TypeRef{}, err
		}
		v, at = target, p
		steps++
	}
	probe := *a
	probe.warnings = nil
	probe.validationSources = map[*PrimitiveValidation]string{}
	probe.collectionSources = map[*CollectionValidation]string{}
	probe.emissionHints = EmissionHints{}
	probe.aliasUses = map[collectionAliasUse]struct{}{}
	var nested []Declaration
	var mapped TypeRef
	var err *diagnostic.Diagnostic
	if dtoUniqueStringEnumTreeSource(v) {
		mapped, err = dtoUniqueStringEnumTreeType(v, at, steps)
	} else {
		mapped, _, err = probe.modelPropertyType(v, "DTOUnique", at, &nested, map[string]bool{}, steps)
	}
	if err != nil {
		return TypeRef{}, err
	}
	if len(nested) != 0 || !dtoUniqueBuiltinCollectionReferenceType(mapped, 0) {
		return TypeRef{}, unsupported(f.Pointer)
	}
	var view func(TypeRef) TypeRef
	view = func(ty TypeRef) TypeRef {
		if ty.Kind == Nullable {
			return view(*ty.Element).Null()
		}
		if ty.Kind == Named && ty.Element != nil {
			if ty.ArrayAliasView == NonNullArrayAlias {
				return mapped.WithoutRootNull()
			}
			return mapped
		}
		return ty
	}
	return view(physical), nil
}
