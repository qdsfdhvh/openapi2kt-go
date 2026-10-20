package kotlin

import (
	"fmt"
	"strings"

	"openapi-kt/internal/naming"
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
	jsonDefault, objectDefault := false, false
	var visit func(schema.Declaration, string)
	visit = func(d schema.Declaration, scope string) {
		switch d := d.(type) {
		case *schema.Model:
			// Empty models project to raw aliases whose top-level factory decodes
			// through the alias. Ordinary model null factories return the literal.
			if d.ComponentDefault != "" && (d.ComponentDefault != "null" || modelUsesRawAlias(d)) {
				name := componentDefaultHelperName(scope, d.ClassName)
				if !seen[name] {
					seen[name] = true
					ty := schema.TypeRef{Kind: schema.Named, Name: d.ClassName}
					if d.Nullable {
						ty = ty.Null()
					}
					fmt.Fprintf(out, "\nprivate fun %s(source: kotlin.String): %s = kotlinx.serialization.json.Json.decodeFromString(kotlinx.serialization.serializer<%s>(), source)\n", name, RenderType(schema.TypeRef{Kind: schema.Named, Name: d.ClassName}), RenderType(schema.TypeRef{Kind: schema.Named, Name: d.ClassName}))
				}
			}
			for _, p := range d.Properties {
				if strings.HasPrefix(p.Default, "`openapi-kt json default`(") {
					jsonDefault = true
					continue
				}
				if strings.HasPrefix(p.Default, "`openapi-kt object default`(") {
					objectDefault = true
					continue
				}
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
		case *schema.ValueAlias:
			if d.ComponentDefault != "" {
				name := componentDefaultHelperName(scope, d.ClassName)
				if !seen[name] {
					seen[name] = true
					ty := RenderType(schema.TypeRef{Kind: schema.Named, Name: d.ClassName})
					fmt.Fprintf(out, "\nprivate fun %s(source: kotlin.String): %s = kotlinx.serialization.json.Json.decodeFromString(kotlinx.serialization.serializer<%s>(), source)\n", name, ty, ty)
				}
			}
		case *schema.StringEnum:
			if d.ComponentDefault == "null" {
				// The physical enum classifier stays non-null; only this opt-in
				// factory returns the nullable root value from its source schema.
				fmt.Fprintf(out, "\n// The OpenAPI component default for this schema.\npublic fun `%s default`(): %s? = null\n", d.ClassName, d.ClassName)
			} else if d.ComponentDefault != "" {
				name := componentDefaultHelperName(scope, d.ClassName)
				if !seen[name] {
					seen[name] = true
					ty := RenderType(schema.TypeRef{Kind: schema.Named, Name: d.ClassName})
					fmt.Fprintf(out, "\n// The OpenAPI component default for this schema.\npublic fun `%s default`(): %s = %s(%s)\n", d.ClassName, ty, name, naming.Literal(d.ComponentDefault))
					fmt.Fprintf(out, "\nprivate fun %s(source: kotlin.String): %s = kotlinx.serialization.json.Json.decodeFromString(kotlinx.serialization.serializer<%s>(), source)\n", name, ty, ty)
				}
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
	if jsonDefault {
		out.WriteString("\nprivate fun `openapi-kt json default`(source: kotlin.String): kotlinx.serialization.json.JsonElement = kotlinx.serialization.json.Json.parseToJsonElement(source)\n")
	}
	if objectDefault {
		out.WriteString("\nprivate fun `openapi-kt object default`(source: kotlin.String): kotlinx.serialization.json.JsonObject = kotlinx.serialization.json.Json.parseToJsonElement(source) as kotlinx.serialization.json.JsonObject\n")
	}
}

// componentDefaultHelperName keys the factory helper by the component's fully
// qualified name so equal short names in different packages never collide.
func componentDefaultHelperName(owner, className string) string {
	return "`openapi-kt component default " + strings.ReplaceAll(owner+"."+className, ".", " ") + "`"
}

// emitComponentDefaultFactory writes the opt-in `default()` companion factory.
// It is additive: no property gains an initializer, so the constructor,
// presence classification and serializer are byte-identical to a component
// without the annotation.
func emitComponentDefaultFactory(model *schema.Model, indent int, owner string, out *strings.Builder) {
	// indent already addresses the class body, so members sit one level deeper.
	pad := strings.Repeat("    ", indent)
	returnType := RenderType(schema.TypeRef{Kind: schema.Named, Name: model.ClassName})
	if model.Nullable {
		returnType += "?"
	}
	fmt.Fprintf(out, "\n%spublic companion object {\n", pad)
	fmt.Fprintf(out, "%s    // The OpenAPI component default for this schema.\n", pad)
	if model.ComponentDefault == "null" {
		// A null literal needs no decode helper and must not go through the
		// non-null component serializer.
		fmt.Fprintf(out, "%s    public fun default(): %s = null\n", pad, returnType)
		fmt.Fprintf(out, "%s}\n", pad)
		return
	}
	fmt.Fprintf(out, "%s    public fun default(): %s =\n", pad, returnType)
	fmt.Fprintf(out, "%s        %s(%s)\n", pad, componentDefaultHelperName(owner, model.ClassName), naming.Literal(model.ComponentDefault))
	fmt.Fprintf(out, "%s}\n", pad)
}
