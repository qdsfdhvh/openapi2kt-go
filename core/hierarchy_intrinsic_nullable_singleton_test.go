package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestIntrinsicNullableHierarchySingletonDirectSerializerRejectsNull(t *testing.T) {
	const body = `{"Alias":{"$ref":"#/components/schemas/Empty"},"False":{"$ref":"#/components/schemas/Empty","nullable":false},"Root":{"oneOf":[{"$ref":"#/components/schemas/Empty"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag"}},"Empty":{"type":"object","nullable":true},"Other":{"properties":{"id":{"type":"string"}}}}`
	for _, plain := range []bool{false, true} {
		for _, validation := range []bool{false, true} {
			req := request(schemas(body), "Dto")
			req.Options.Validation = validation
			if plain {
				req.Options.ModelStyle = core.PlainClass
			}
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(e)
			}
			s := source(t, r, "EmptyDto")
			if !strings.Contains(s, "public data object EmptyDto : org.example.models.models.RootDto() {") || !strings.Contains(s, "if (element == kotlinx.serialization.json.JsonNull)") || !strings.Contains(s, `throw kotlinx.serialization.SerializationException("Expected a non-null hierarchy value.")`) {
				t.Fatal(plain, validation, s)
			}
			if !strings.Contains(source(t, r, "AliasDto"), "= org.example.models.models.EmptyDto?") || !strings.Contains(source(t, r, "FalseDto"), "= org.example.models.models.EmptyDto\n") {
				t.Fatal("intent changed")
			}
			// Preserve the established accepted singleton bytes for non-null input.
			control := req
			control.Document = []byte(strings.Replace(string(req.Document), `"nullable":true`, `"nullable":false`, 1))
			old, e := core.GenerateWithDiagnostics(control)
			if e != nil {
				t.Fatal(e)
			}
			if strings.Contains(source(t, old, "EmptyDto"), "Expected a non-null hierarchy value.") {
				t.Fatal("non-null historical profile changed")
			}
		}
	}
}
