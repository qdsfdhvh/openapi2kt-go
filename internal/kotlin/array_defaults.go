package kotlin

import (
	"fmt"
	"slices"
	"strings"

	"openapi-kt/internal/schema"
)

// Parse a validated literal inside a private file helper. Passing one JSON
// string avoids unbounded vararg initializers and keeps package qualifiers out
// of constructor scopes with legitimate kotlin/kotlinx field names. Maps and
// lists use distinct helper names even when their primitive leaves match.
func emitArrayDefaultHelpers(declaration schema.Declaration, out *strings.Builder) {
	types := map[string]string{}
	var visit func(schema.Declaration)
	visit = func(d schema.Declaration) {
		switch d := d.(type) {
		case *schema.Model:
			for _, p := range d.Properties {
				if !strings.HasPrefix(p.Default, "`openapi-kt default array ") && !strings.HasPrefix(p.Default, "`openapi-kt default map ") && !strings.HasPrefix(p.Default, "`openapi-kt default tree ") {
					continue
				}
				ty := p.Type.WithoutRootNull()
				if (ty.Kind != schema.List && ty.Kind != schema.Map && ty.Kind != schema.Named) || ty.Element == nil {
					continue
				}
				index := strings.Index(p.Default, "`(")
				if index < 0 {
					continue
				}
				types[p.Default[:index+1]] = RenderType(ty)
			}
			for _, n := range d.Nested {
				visit(n)
			}
		case *schema.ObjectUnion:
			for _, v := range d.Variants {
				visit(v)
			}
		case *schema.PrimitiveUnion:
			if d.Object != nil {
				visit(d.Object)
			}
		case *schema.Hierarchy:
			if d.Leaf != nil {
				visit(d.Leaf)
			}
		}
	}
	visit(declaration)
	names := make([]string, 0, len(types))
	for name := range types {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		fmt.Fprintf(out, "\nprivate fun %s(source: kotlin.String): %s = kotlinx.serialization.json.Json.decodeFromString(kotlinx.serialization.serializer<%s>(), source)\n", name, types[name], types[name])
	}
}
