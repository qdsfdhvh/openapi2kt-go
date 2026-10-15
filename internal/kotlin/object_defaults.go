package kotlin

import (
	"fmt"
	"strings"

	"openapi-kt/internal/schema"
)

// Direct inline ordinary DTO object defaults are stored as a keyless marker
// plus validated canonical JSON. The emitter binds the final actual lexical
// nominal type after relocation, exactly like enum typed defaults, so hoisting
// or merging cannot leave a stale classifier or a short nested name.
const objectDefaultMarker = "`openapi-kt default model`("

// defaultExpression dispatches every stored initializer form. Object defaults
// must intercept before the enum expression fallback because both may be
// present on unrelated properties of the same model.
func defaultExpression(model *schema.Model, p schema.Property, owner string) string {
	if expression, ok := objectDefaultExpression(p, owner+"."+model.ClassName); ok {
		return expression
	}
	return enumDefaultExpression(model, p, owner)
}

// objectDefaultExpression qualifies the actual property type against the
// declaration scope. Nested DTO type names are scope-relative short names.
func objectDefaultExpression(p schema.Property, scope string) (string, bool) {
	if !strings.HasPrefix(p.Default, objectDefaultMarker) {
		return "", false
	}
	ty, ok := qualifyObjectDefaultType(p.Type.WithoutRootNull(), scope)
	if !ok {
		return p.Default, true
	}
	return objectDefaultHelperName(ty) + "(" + strings.TrimPrefix(p.Default, objectDefaultMarker), true
}

func qualifyObjectDefaultType(ty schema.TypeRef, scope string) (schema.TypeRef, bool) {
	switch ty.Kind {
	case schema.Nullable, schema.List, schema.Map:
		if ty.Element == nil {
			return schema.TypeRef{}, false
		}
		element, ok := qualifyObjectDefaultType(*ty.Element, scope)
		if !ok {
			return schema.TypeRef{}, false
		}
		ty.Element = &element
		return ty, true
	case schema.Named:
		if ty.Element != nil {
			return schema.TypeRef{}, false
		}
		if !strings.Contains(ty.Name, ".") {
			ty.Name = scope + "." + ty.Name
		}
		return ty, true
	}
	return schema.TypeRef{}, false
}

func objectDefaultHelperName(ty schema.TypeRef) string {
	return "`openapi-kt model default " + strings.ReplaceAll(objectDefaultTypeKey(ty), ".", " ") + "`"
}

// objectDefaultTypeKey distinguishes the container/nullable wrappers so a DTO,
// a list of DTOs and a dictionary of DTOs never share one helper name.
func objectDefaultTypeKey(ty schema.TypeRef) string {
	switch ty.Kind {
	case schema.Nullable:
		return "Nullable " + objectDefaultTypeKey(*ty.Element)
	case schema.List:
		return "List " + objectDefaultTypeKey(*ty.Element)
	case schema.Map:
		return "Map " + objectDefaultTypeKey(*ty.Element)
	}
	return ty.Name
}

func emitObjectDefaultHelpers(declaration schema.Declaration, owner string, out *strings.Builder) {
	seen := map[string]bool{}
	var visit func(schema.Declaration, string)
	visit = func(d schema.Declaration, scope string) {
		switch d := d.(type) {
		case *schema.Model:
			for _, p := range d.Properties {
				if !strings.HasPrefix(p.Default, objectDefaultMarker) {
					continue
				}
				ty, ok := qualifyObjectDefaultType(p.Type.WithoutRootNull(), scope+"."+d.ClassName)
				if !ok {
					continue
				}
				name := objectDefaultHelperName(ty)
				if seen[name] {
					continue
				}
				seen[name] = true
				fmt.Fprintf(out, "\nprivate fun %s(source: kotlin.String): %s = kotlinx.serialization.json.Json.decodeFromString(kotlinx.serialization.serializer<%s>(), source)\n", name, RenderType(ty), RenderType(ty))
			}
			for _, n := range d.Nested {
				visit(n, scope+"."+d.ClassName)
			}
		case *schema.ObjectUnion:
			for _, v := range d.Variants {
				visit(v, scope+"."+d.ClassName)
			}
		case *schema.PrimitiveUnion:
			if d.Object != nil {
				visit(d.Object, scope)
			}
		case *schema.Hierarchy:
			if d.Leaf != nil {
				visit(d.Leaf, scope)
			}
		}
	}
	visit(declaration, owner)
}
