package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestHierarchyNestedOwnersNoScratchOrShapeGuesses(t *testing.T) {
	root := `"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag"}},"Other":{"properties":{"id":{"type":"string"}}}`
	for _, tc := range []struct{ leaf, target string }{
		{`{"properties":{"id":{"type":"string"}},"additionalProperties":{"properties":{"id":{"type":"string"}}}}`, "Leaf/additionalProperties"},
		{`{"properties":{"node":{"type":"object"},"id":{"type":"string"}}}`, "Leaf/properties/node"},
		{`{"properties":{"node":{"properties":{"pin":{"type":"string","const":"x"}}},"id":{"type":"string"}}}`, "Leaf/properties/node"},
		{`{"allOf":[{"properties":{"node":{"properties":{"id":{"type":"string"}}}}},{"properties":{"more":{"type":"string"}}}]}`, "Leaf/allOf/0/properties/node"},
		{`{"properties":{"choice":{"oneOf":[{"properties":{"node":{"properties":{"id":{"type":"string"}}}}},{"properties":{"other":{"type":"string"}}}]},"id":{"type":"string"}}}`, "Leaf/properties/choice/oneOf/0/properties/node"},
	} {
		b := schemas(`{` + root + `,"Leaf":` + tc.leaf + `,"Alias":{"$ref":"#/components/schemas/` + tc.target + `"}}`)
		canonical := map[string]string{"Leaf/allOf/0/properties/node": "LeafDto.Node", "Leaf/properties/choice/oneOf/0/properties/node": "LeafDto.Choice.Variant1.Node"}[tc.target]
		if canonical != "" {
			r := report(t, b, "Dto")
			if !strings.Contains(source(t, r, "AliasDto"), "= org.example.models.models."+canonical+"\n") {
				t.Fatal("actual composed owner lost")
			}
		} else {
			aliasFailure(t, b, "E_REF_UNSUPPORTED", "/components/schemas/Alias/$ref")
		}
		late := schemas(`{` + root + `,"Leaf":` + tc.leaf + `,"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/` + tc.target + `"}}}}`)
		if canonical != "" {
			r := report(t, late, "Dto")
			if !strings.Contains(source(t, r, "ConsumerDto"), "value: org.example.models.models."+canonical+"? = null") {
				t.Fatal("late actual composed owner lost")
			}
		} else {
			aliasFailure(t, late, "E_REF_UNSUPPORTED", "/components/schemas/Consumer/properties/value")
		}
	}
}

func TestHierarchyNestedOwnersSourceDepthAndSlots(t *testing.T) {
	for _, validation := range []bool{false, true} {
		for _, typed := range []bool{false, true} {
			limit := 47
			if typed {
				limit = 45
			}
			for _, n := range []int{limit, limit + 1} {
				fields := []string{`"node":{"properties":{"id":{"type":"string"}},"required":["id"]}`}
				for i := range n {
					ref := "Leaf/properties/node"
					if i > 0 {
						ref = fmt.Sprint("Leaf/properties/h", i-1)
					}
					fields = append(fields, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s"}`, i, ref))
				}
				body := `"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag"}},"Other":{"properties":{"id":{"type":"string"}}},"Leaf":{"properties":{` + strings.Join(fields, ",") + `}},"Alias":{"$ref":"#/components/schemas/Leaf/properties/h` + fmt.Sprint(n-1) + `","nullable":false}`
				if typed {
					body += `,"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Alias","nullable":false}},"required":["value"]}`
				}
				req := request(schemas(`{`+body+`}`), "Dto")
				req.Options.Validation = validation
				r, e := core.GenerateWithDiagnostics(req)
				if n == limit {
					if e != nil || !strings.Contains(source(t, r, "AliasDto"), "= org.example.models.models.LeafDto.Node\n") {
						t.Fatal(validation, typed, n, e)
					}
				} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0 {
					t.Fatal(validation, typed, n, r, e)
				}
			}
		}
		for _, n := range []int{239, 240} { // One inherited tag body is serialized too.
			properties := []string{`"node":{"properties":{"id":{"type":"string"}}}`}
			for i := 1; i < n; i++ {
				properties = append(properties, fmt.Sprintf(`"p%d":{"type":"string"}`, i))
			}
			b := schemas(`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag"}},"Other":{"properties":{"id":{"type":"string"}}},"Leaf":{"properties":{` + strings.Join(properties, ",") + `}},"Alias":{"$ref":"#/components/schemas/Leaf/properties/node"}}`)
			req := request(b, "Dto")
			req.Options.Validation = validation
			r, e := core.GenerateWithDiagnostics(req)
			if n == 239 {
				if e != nil {
					t.Fatal(e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
				t.Fatal(n, r, e)
			}
		}
	}
}
