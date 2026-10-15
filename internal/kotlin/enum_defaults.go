package kotlin

import (
	"fmt"
	"strings"

	"openapi-kt/internal/schema"
)

func enumDefaultExpression(model *schema.Model, p schema.Property, owner string) string {
	if ty, prefix, ok := enumTypedDefaultType(p, owner+"."+model.ClassName); ok {
		return enumTypedDefaultHelperName(ty) + "(" + strings.TrimPrefix(p.Default, prefix)
	}
	target := p.Type.WithoutRootNull()
	prefix := target.Name + ".fromSerialName("
	if target.Kind != schema.Named || !strings.HasPrefix(p.Default, prefix) {
		return p.Default
	}
	name := owner + "." + model.ClassName + "." + target.Name
	return enumDefaultHelperName(name) + "(" + strings.TrimPrefix(p.Default, prefix)
}

// A helper binds to the actual lexical enum classifier after hoisting/merging.
// Constructor fields can therefore never shadow its classifier expression.
func enumDefaultHelperName(name string) string {
	return "`openapi-kt default enum " + strings.ReplaceAll(name, ".", " ") + "`"
}

// Typed defaults bind using the final lexical owner after relocation. The
// planner stores only validated JSON, so optimization can rebind the actual
// enum declaration without reconstructing a classifier from source spelling.
func enumTypedDefaultType(p schema.Property, scope string) (schema.TypeRef, string, bool) {
	ty := p.Type.WithoutRootNull()
	if prefix := "`openapi-kt default enum scalar`("; strings.HasPrefix(p.Default, prefix) {
		if ty.Kind != schema.Named || ty.Element != nil {
			return schema.TypeRef{}, "", false
		}
		qualified, ok := enumDefaultTreeType(ty, scope)
		return qualified, prefix, ok
	}
	if prefix := "`openapi-kt default enum tree`("; strings.HasPrefix(p.Default, prefix) {
		if ty.Kind != schema.List && ty.Kind != schema.Map {
			return schema.TypeRef{}, "", false
		}
		qualified, ok := enumDefaultTreeType(ty, scope)
		return qualified, prefix, ok
	}
	kind := "array"
	if ty.Kind == schema.Map {
		kind = "map"
	} else if ty.Kind != schema.List {
		return schema.TypeRef{}, "", false
	}
	prefix := "`openapi-kt default enum " + kind + "`("
	if !strings.HasPrefix(p.Default, prefix) || ty.Element == nil {
		return schema.TypeRef{}, "", false
	}
	element := ty.Element.WithoutRootNull()
	if element.Kind != schema.Named {
		return schema.TypeRef{}, "", false
	}
	if !strings.Contains(element.Name, ".") {
		element.Name = scope + "." + element.Name
	}
	if ty.Element.Kind == schema.Nullable {
		element = element.Null()
	}
	ty.Element = &element
	return ty, prefix, true
}

// Rebuild only transparent container/nullable wrappers. The terminal is the
// actual final enum TypeRef, whose lexical owner may have been moved or merged.
func enumDefaultTreeType(ty schema.TypeRef, scope string) (schema.TypeRef, bool) {
	switch ty.Kind {
	case schema.Nullable, schema.List, schema.Map:
		if ty.Element == nil {
			return schema.TypeRef{}, false
		}
		element, ok := enumDefaultTreeType(*ty.Element, scope)
		if !ok {
			return schema.TypeRef{}, false
		}
		ty.Element = &element
		return ty, true
	case schema.Named:
		if !strings.Contains(ty.Name, ".") {
			ty.Name = scope + "." + ty.Name
		}
		return ty, true
	}
	return schema.TypeRef{}, false
}

func enumDefaultTreeKey(ty schema.TypeRef) string {
	switch ty.Kind {
	case schema.Nullable:
		return "nullable " + enumDefaultTreeKey(*ty.Element)
	case schema.List:
		return "list " + enumDefaultTreeKey(*ty.Element)
	case schema.Map:
		return "map " + enumDefaultTreeKey(*ty.Element)
	}
	return ty.Name
}

func enumTypedDefaultHelperName(ty schema.TypeRef) string {
	if ty.Kind == schema.Named {
		return enumDefaultHelperName("scalar " + ty.Name)
	}
	if ty.Element != nil {
		element := ty.Element.WithoutRootNull()
		if element.Kind == schema.List || element.Kind == schema.Map {
			return enumDefaultHelperName("tree " + enumDefaultTreeKey(ty))
		}
	}
	kind := "array"
	if ty.Kind == schema.Map {
		kind = "map"
	}
	name := ty.Element.WithoutRootNull().Name + " " + kind
	if ty.Element.Kind == schema.Nullable {
		name += " nullable"
	}
	return enumDefaultHelperName(name)
}

func emitEnumDefaultHelpers(declaration schema.Declaration, owner string, out *strings.Builder) {
	// Nested DTOs may borrow the same external enum. Helpers live at file scope.
	seen := map[string]bool{}
	var visit func(schema.Declaration, string)
	visit = func(d schema.Declaration, scope string) {
		switch d := d.(type) {
		case *schema.Model:
			for _, p := range d.Properties {
				if ty, _, ok := enumTypedDefaultType(p, scope+"."+d.ClassName); ok {
					name := enumTypedDefaultHelperName(ty)
					if !seen[name] {
						seen[name] = true
						fmt.Fprintf(out, "\nprivate fun %s(source: kotlin.String): %s = kotlinx.serialization.json.Json.decodeFromString(kotlinx.serialization.serializer<%s>(), source)\n", name, RenderType(ty), RenderType(ty))
					}
					continue
				}
				target := p.Type.WithoutRootNull()
				if target.Kind != schema.Named || !strings.HasPrefix(p.Default, target.Name+".fromSerialName(") {
					continue
				}
				name := scope + "." + d.ClassName + "." + target.Name
				helper := enumDefaultHelperName(name)
				if seen[helper] {
					continue
				}
				seen[helper] = true
				fmt.Fprintf(out, "\nprivate fun %s(value: kotlin.String): %s = %s.fromSerialName(value)\n", helper, name, name)
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
