package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func nullableProductSource(intent string) string {
	return `{` + intent + `"allOf":[{"oneOf":[{"properties":{"kind":{"const":"a"},"id":{"type":"string"}},"required":["id"]},{"properties":{"kind":{"const":"b"},"ok":{"type":"boolean"}},"required":["ok"]}]},{"properties":{"stamp":{"const":"shared"}}}]}`
}

func TestNullableProductUnionsRetainPhysicalCarrierAndViews(t *testing.T) {
	for _, intent := range []string{`"nullable":true,`, `"type":["object","null"],`} {
		input := schemas(`{"Product":` + nullableProductSource(intent) + `,"Alias":{"$ref":"#/components/schemas/Product"},"NullableAlias":{"$ref":"#/components/schemas/Product","nullable":true},"False":{"$ref":"#/components/schemas/NullableAlias","nullable":false},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Product"},"narrow":{"$ref":"#/components/schemas/False"}},"required":["value","narrow"]}}`)
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			rq := request(input, "Dto")
			rq.Options.ModelStyle = style
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(intent, style, err)
			}
			physical := source(t, got, "ProductDto")
			for _, part := range []string{"public sealed class ProductDto", "if (element == kotlinx.serialization.json.JsonNull)", "NullProductDto", "obj[\"kind\"]", "stamp"} {
				if !strings.Contains(physical, part) {
					t.Fatal("lost physical null carrier or product dispatch", part, physical)
				}
			}
			if !strings.Contains(source(t, got, "AliasDto"), "= org.example.models.models.ProductDto\n") || !strings.Contains(source(t, got, "NullableAliasDto"), "= org.example.models.models.ProductDto?\n") || !strings.Contains(source(t, got, "FalseDto"), "= org.example.models.models.ProductDto\n") {
				t.Fatal("product aliases changed carrier policy")
			}
		}
	}
}

func TestNullableProductUnionsInlineAndOriginalConstraints(t *testing.T) {
	for _, checked := range []bool{false, true} {
		rq := request(fieldDoc(nullableProductSource(`"nullable":true,`)), "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(checked, err)
		}
		if text := source(t, got, "RecordDto"); !strings.Contains(text, "public val value: Value? = null,") || strings.Contains(text, "NullValue") {
			t.Fatal("inline nullable view should use Kotlin nullable codec", text)
		}
	}
	bad := strings.Replace(nullableProductSource(`"nullable":true,`), `"type":"string"`, `"type":"string","format":"unsupported"`, 1)
	got, err := core.GenerateWithDiagnostics(request(schemas(`{"Product":`+bad+`}`), "Dto"))
	if err == nil || len(got.Artifacts.Files) != 0 {
		t.Fatal("nullable product hid invalid source", got, err)
	}
}
