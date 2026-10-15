package kotlin

import (
	"strings"

	"openapi-kt/internal/schema"
)

// File-level parsing keeps initializer lookup outside constructor/body scopes.
// Only constants with the structured profile's validated helper expression can
// request these helpers; no schema/data strings are rewritten.
func emitStructuredConstHelpers(declaration schema.Declaration, out *strings.Builder) {
	object, array := false, false
	var visit func(schema.Declaration)
	visit = func(d schema.Declaration) {
		switch d := d.(type) {
		case *schema.Model:
			for _, c := range d.Constants {
				object = object || c.Type.Kind == schema.Scalar && c.Type.Name == "kotlinx.serialization.json.JsonObject" && strings.HasPrefix(c.Initializer, "`openapi-kt object const`(")
				array = array || c.Type.Kind == schema.Scalar && c.Type.Name == "kotlinx.serialization.json.JsonArray" && strings.HasPrefix(c.Initializer, "`openapi-kt array const`(")
			}
			for _, nested := range d.Nested {
				visit(nested)
			}
		case *schema.ObjectUnion:
			for _, variant := range d.Variants {
				visit(variant)
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
	if object {
		out.WriteString("\nprivate fun `openapi-kt object const`(source: kotlin.String): kotlinx.serialization.json.JsonObject = kotlinx.serialization.json.Json.parseToJsonElement(source) as kotlinx.serialization.json.JsonObject\n")
	}
	if array {
		out.WriteString("\nprivate fun `openapi-kt array const`(source: kotlin.String): kotlinx.serialization.json.JsonArray = kotlinx.serialization.json.Json.parseToJsonElement(source) as kotlinx.serialization.json.JsonArray\n")
	}
}
