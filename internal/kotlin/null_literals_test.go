package kotlin

import (
	"strings"
	"testing"

	"openapi-kt/internal/schema"
)

func TestNullLiteralAliasesDoNotRewriteWireOrUserLiteralText(t *testing.T) {
	model := &schema.Model{ClassName: "Holder", Constants: []schema.Constant{
		{Name: "fixed", WireName: "kotlinx.serialization.json.JsonNull", Type: schema.TypeRef{Kind: schema.Scalar, Name: "kotlinx.serialization.json.JsonNull"}, Initializer: "kotlinx.serialization.json.JsonNull"},
		{Name: "kotlinx", WireName: "kotlinx", Type: schema.TypeRef{Kind: schema.Scalar, Name: "kotlin.String"}, Initializer: `"kotlinx.serialization.json.JsonNull"`},
	}}
	files := Emit([]schema.Declaration{model}, "org.example", schema.EmissionHints{})
	if len(files) != 1 {
		t.Fatal(files)
	}
	text := files[0].Content
	if !strings.Contains(text, `("kotlinx.serialization.json.JsonNull")`) || !strings.Contains(text, `public val kotlinx: kotlin.String = "kotlinx.serialization.json.JsonNull"`) || !strings.Contains(text, "import kotlinx.serialization.EncodeDefault as ΩopenapiKtNullEncodeDefault") || strings.Contains(text, "import kotlinx.serialization.*") {
		t.Fatal(text)
	}
}
