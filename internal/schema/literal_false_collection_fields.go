package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// A Boolean additionalProperties is a keyword role with its existing JSON
// object projection. Only an actual array items slot supplies this false leaf.
type literalFalseCollectionSource struct {
	Schema, Leaf *document.Value
	Pointer      string
	Height       int
	OwnDefault   bool
}
type literalFalseCollectionPlanning struct {
	Source literalFalseCollectionSource
	Field  field
	Owner  *component
	Model  *Model
	Helper *FalseValue
}

func (a *analyzer) literalFalseCollectionSource(v *document.Value, pointer string) *literalFalseCollectionSource {
	return a.literalFalseCollectionSourceRole(v, pointer, false)
}
func (a *analyzer) literalFalseCollectionSourceRole(v *document.Value, pointer string, own bool) *literalFalseCollectionSource {
	original := v
	arrayParent := false
	for height := 0; height <= MaxSchemaDepth; height++ {
		if a.booleanFalseRoot(v) {
			if height > 0 && arrayParent {
				return &literalFalseCollectionSource{original, v, pointer, height, own}
			}
			return nil
		}
		if v == nil || v.Kind() != document.Object {
			return nil
		}
		if has(v, "default") && !(own && height == 0) {
			return nil
		}
		for _, key := range []string{"$ref", "allOf", "oneOf", "anyOf", "prefixItems", "uniqueItems"} {
			if has(v, key) {
				return nil
			}
		}
		kind, _, err := schemaType(v, pointer)
		if err != nil {
			return nil
		}
		if kind == "array" {
			v = get(v, "items")
			pointer = naming.Pointer(pointer, "items")
			arrayParent = true
		} else if kind == "object" && dictionarySchema(v) {
			v = get(v, "additionalProperties")
			pointer = naming.Pointer(pointer, "additionalProperties")
			arrayParent = false
		} else {
			return nil
		}
	}
	return nil
}
func (a *analyzer) literalFalseCollectionFieldSource(f field, owner *document.Value, pointer string, ordinary bool, common []Property, typed map[string]typedLeafField) *literalFalseCollectionSource {
	if !a.componentRoot(owner, pointer) || !ordinary || a.usage != SharedUsage || common != nil || len(typed) != 0 || has(owner, "default") && !a.componentDefaultSchemas[owner] || has(owner, "allOf") || a.hierarchySources[owner] || has(f.Schema, "readOnly") || has(f.Schema, "writeOnly") || get(get(owner, "properties"), f.Wire) != f.Schema || f.Pointer != naming.Pointer(pointer+"/properties", f.Wire) {
		return nil
	}
	return a.literalFalseCollectionSourceRole(f.Schema, f.Pointer, has(f.Schema, "default"))
}
func (a *analyzer) literalFalseCollectionElementType(v *document.Value, pointer string, depth int) (*TypeRef, *diagnostic.Diagnostic) {
	p := a.literalFalseCollectionPlanning
	if p == nil || p.Source.Leaf != v || p.Source.Pointer != pointer || a.usage != SharedUsage {
		return nil, nil
	}
	if err := depthCheck(depth, pointer); err != nil {
		return nil, err
	}
	if p.Owner == nil || a.components[p.Owner.Wire] != p.Owner || p.Model == nil || p.Model.ClassName != p.Owner.Name || p.Helper == nil {
		return nil, schemaReferenceUnsupported(pointer)
	}
	current := a.literalFalseCollectionFieldSource(p.Field, p.Owner.Schema, p.Owner.Pointer, true, nil, nil)
	if current == nil || *current != p.Source {
		return nil, schemaReferenceUnsupported(pointer)
	}
	helpers := 0
	for _, d := range p.Model.Nested {
		if d == p.Helper {
			helpers++
		}
	}
	if helpers != 1 {
		return nil, schemaReferenceUnsupported(pointer)
	}
	ty := TypeRef{Kind: Named, Name: a.options.PackageName + ".models." + p.Model.ClassName + "." + p.Helper.ClassName}
	return &ty, nil
}
