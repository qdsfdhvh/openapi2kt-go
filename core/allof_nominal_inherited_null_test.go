package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestAllOfNominalTerminalInheritedShapeNullAndFalse(t *testing.T) {
	for _, annotate := range []string{"", `,"nullable":false`} {
		node := `{"allOf":[{"nullable":true,"properties":{"id":{"type":"string"}},"required":["id"]},{"properties":{"more":{"type":"string"}}}]` + annotate + `}`
		input := schemas(`{"Owner":{"properties":{"node":` + node + `}},"Alias":{"$ref":"#/components/schemas/Owner/properties/node"},"Maybe":{"$ref":"#/components/schemas/Alias"},"False":{"$ref":"#/components/schemas/Maybe","nullable":false},"Hops":{"properties":{"hop":{"$ref":"#/components/schemas/Owner/properties/node"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/hop"},"Consumer":{"properties":{"direct":{"$ref":"#/components/schemas/Owner/properties/node"},"nested":{"$ref":"#/components/schemas/Hops/properties/hop"}},"required":["direct","nested"]}}`)
		for _, checked := range []bool{false, true} {
			req := request(input, "Dto")
			req.Options.Validation = checked
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(e)
			}
			suffix := "?"
			if annotate != "" {
				suffix = ""
			}
			for _, alias := range []string{"Alias", "Maybe", "Bridge"} {
				if !strings.Contains(source(t, r, alias+"Dto"), "= org.example.models.models.OwnerDto.Node"+suffix+"\n") {
					t.Fatal("inherited null intent", checked, annotate, source(t, r, alias+"Dto"))
				}
			}
			if !strings.Contains(source(t, r, "FalseDto"), "= org.example.models.models.OwnerDto.Node\n") {
				t.Fatal("false did not narrow")
			}
			if !strings.Contains(source(t, r, "ConsumerDto"), "public val direct: org.example.models.models.OwnerDto.Node"+suffix) {
				t.Fatal("typed source shape null lost")
			}
		}
	}
}
