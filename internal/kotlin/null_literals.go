package kotlin

import (
	"strings"

	"openapi-kt/internal/schema"
)

func nullConstantExpression(c schema.Constant) string {
	if c.Type.Kind == schema.Scalar && c.Type.Name == "kotlinx.serialization.json.JsonNull" && c.Initializer == "kotlinx.serialization.json.JsonNull" {
		return "`openapi-kt null literal`()"
	}
	return c.Initializer
}

// Schema naming and native classifier validation produce ASCII identifiers.
// A Unicode import alias therefore cannot be shadowed by an admitted field.
// Rewrite emitter templates before formatting, never source strings or wire keys.
var nullLiteralSymbols = []string{
	"kotlinx.serialization.EncodeDefault", "kotlinx.serialization.SerialName",
	"kotlinx.serialization.KSerializer", "kotlinx.serialization.SerializationException",
	"kotlinx.serialization.descriptors.SerialDescriptor",
	"kotlinx.serialization.encoding.Encoder", "kotlinx.serialization.encoding.Decoder",
	"kotlinx.serialization.json.JsonElement", "kotlinx.serialization.json.JsonEncoder",
	"kotlinx.serialization.json.JsonDecoder", "kotlinx.serialization.json.JsonObject",
	"kotlinx.serialization.json.JsonPrimitive", "kotlinx.serialization.json.JsonNull",
	"kotlin.collections.mapOf",
}

func nullLiteralAlias(symbol string) string {
	return "ΩopenapiKtNull" + symbol[strings.LastIndex(symbol, ".")+1:]
}

func nullLiteralTemplate(format string, enabled bool) string {
	if enabled {
		for _, symbol := range nullLiteralSymbols {
			format = strings.ReplaceAll(format, symbol, nullLiteralAlias(symbol))
		}
	}
	return format
}

func modelHasNullConstant(model *schema.Model) bool {
	for _, c := range model.Constants {
		if nullConstantExpression(c) != c.Initializer {
			return true
		}
	}
	return false
}

func modelNeedsNullLiteralAliases(model *schema.Model) bool {
	if modelHasNullConstant(model) {
		return true
	}
	if len(model.Constants) != 0 {
		for _, p := range model.Properties {
			if p.Name == "kotlinx" || p.Name == "kotlin" {
				return true
			}
		}
		for _, c := range model.Constants {
			if c.Name == "kotlinx" || c.Name == "kotlin" {
				return true
			}
		}
	}
	return false
}

func emitNullLiteralImports(declaration schema.Declaration, out *strings.Builder) {
	if declarationNeedsNullLiteralSymbols(declaration, true) {
		for _, symbol := range nullLiteralSymbols {
			out.WriteString("import " + symbol + " as " + nullLiteralAlias(symbol) + "\n")
		}
		out.WriteByte('\n')
	}
}

func declarationNeedsNullLiteralSymbols(declaration schema.Declaration, aliases bool) bool {
	needed := false
	var visit func(schema.Declaration)
	visit = func(d schema.Declaration) {
		switch d := d.(type) {
		case *schema.Model:
			needed = needed || modelHasNullConstant(d) || aliases && modelNeedsNullLiteralAliases(d)
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
	return needed
}

// A file-level expression stays outside DTO scopes where legal fields such as
// kotlinx can shadow the package qualifier. No instance or serialized slot is added.
func emitNullLiteralHelper(declaration schema.Declaration, out *strings.Builder) {
	if declarationNeedsNullLiteralSymbols(declaration, false) {
		out.WriteString("\nprivate fun `openapi-kt null literal`(): kotlinx.serialization.json.JsonNull = kotlinx.serialization.json.JsonNull\n")
	}
}
