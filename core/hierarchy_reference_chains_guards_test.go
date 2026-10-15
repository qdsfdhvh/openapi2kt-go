package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func hierarchyInternalChainInput(hops int, consumer bool) []byte {
	nodes := []string{`"Root":{"nullable":true,"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"kind"}},"First":{"properties":{"kind":{"type":"string","const":"first"},"id":{"type":"string"}},"required":["id"]},"Other":{"properties":{"kind":{"type":"string","const":"other"},"id":{"type":"string"}},"required":["id"]}`}
	fields := []string{}
	for i := range hops {
		target := "Root"
		if i > 0 {
			target = fmt.Sprint("Owner/properties/h", i-1)
		}
		fields = append(fields, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s"}`, i, target))
	}
	nodes = append(nodes, `"Owner":{"properties":{`+strings.Join(fields, ",")+`}}`)
	nodes = append(nodes, fmt.Sprintf(`"Alias":{"$ref":"#/components/schemas/Owner/properties/h%d","nullable":false}`, hops-1))
	if consumer {
		nodes = append(nodes, `"Holder":{"properties":{"narrow":{"$ref":"#/components/schemas/Alias","nullable":false}},"required":["narrow"]}`)
	}
	return schemas("{" + strings.Join(nodes, ",") + "}")
}

func TestNamedHierarchyInternalChainDepthAndFalseCache(t *testing.T) {
	for _, validation := range []bool{false, true} {
		for _, consumer := range []bool{false, true} {
			limit := 47 // Incoming internal reference + original hop refs.
			if consumer {
				limit = 45
			} // Field depth + typed named alias edge.
			for _, hops := range []int{limit, limit + 1} {
				req := request(hierarchyInternalChainInput(hops, consumer), "Dto")
				req.Options.Validation = validation
				r, e := core.GenerateWithDiagnostics(req)
				if hops == limit {
					if e != nil || !strings.Contains(source(t, r, "AliasDto"), "= org.example.models.models.RootDto\n") {
						t.Fatal(validation, consumer, hops, e)
					}
				} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0 {
					t.Fatal(validation, consumer, hops, r, e)
				}
			}
		}
	}
}

func TestNamedHierarchyInternalChainSlotsAndCatalogCycles(t *testing.T) {
	base := `"Root":{"nullable":true,"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"kind"}},"First":{"properties":{"kind":{"type":"string","const":"first"},"id":{"type":"string"}},"required":["id"]},"Other":{"properties":{"kind":{"type":"string","const":"other"},"id":{"type":"string"}},"required":["id"]},"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/Root"}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/hop"}`
	for _, validation := range []bool{false, true} {
		for _, n := range []int{240, 241} {
			fields, required := []string{}, []string{}
			for i := range n {
				fields = append(fields, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Alias"}`, i))
				required = append(required, fmt.Sprintf(`"p%d"`, i))
			}
			req := request(schemas(`{`+base+`,"Wide":{"properties":{`+strings.Join(fields, ",")+`},"required":[`+strings.Join(required, ",")+`]}}`), "Dto")
			req.Options.Validation = validation
			r, e := core.GenerateWithDiagnostics(req)
			if n == 240 {
				if e != nil || !strings.Contains(source(t, r, "WideDto"), "public val p239: org.example.models.models.RootDto? = null,") {
					t.Fatal(n, validation, e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
				t.Fatal(n, validation, r, e)
			}
		}
		cycle := request(schemas(`{"Alias":{"$ref":"#/components/schemas/Owner/properties/a"},"Owner":{"properties":{"a":{"$ref":"#/components/schemas/Alias"}}}}`), "Dto")
		cycle.Options.Validation = validation
		r, e := core.GenerateWithDiagnostics(cycle)
		code := "E_SCHEMA_UNSUPPORTED"
		if validation {
			code = "E_REF_CYCLE"
		}
		if e == nil || e.(core.Diagnostics)[0].Code != code || len(r.Artifacts.Files) != 0 {
			t.Fatal(validation, r, e)
		}
	}
}

func TestNamedHierarchyInternalChainUnownedAndCycleGuards(t *testing.T) {
	base := `"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"kind"}},"Leaf":{"properties":{"kind":{"type":"string","const":"leaf"},"node":{"properties":{"id":{"type":"string"}}}}},"Other":{"properties":{"kind":{"type":"string","const":"other"},"id":{"type":"string"}}}`
	for _, terminal := range []string{"Leaf/properties/node", "Choice", "Empty", "Color"} {
		input := schemas(`{` + base + `,"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/` + terminal + `"}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/hop"},"Choice":{"oneOf":[{"properties":{"a":{"type":"string"}}},{"properties":{"b":{"type":"string"}}}]},"Empty":{"type":"object"},"Color":{"type":"string","enum":["a","b"]}}`)
		if terminal == "Leaf/properties/node" {
			// The retained lexical nested Model is now an actual nominal owner.
			r := report(t, input, "Dto")
			if !strings.Contains(source(t, r, "AliasDto"), "= org.example.models.models.LeafDto.Node\n") {
				t.Fatal("actual hierarchy nested owner lost")
			}
			continue
		}
		if terminal == "Choice" {
			r := report(t, input, "Dto")
			if !strings.Contains(source(t, r, "AliasDto"), "= org.example.models.models.ChoiceDto\n") {
				t.Fatal("actual union terminal graduation")
			}
			continue
		}
		if terminal == "Color" {
			// Preserve the original named-enum input and canonical vocabulary.
			r := report(t, input, "Dto")
			if !strings.Contains(source(t, r, "AliasDto"), "= org.example.models.models.ColorDto\n") {
				t.Fatal("actual named enum terminal graduation")
			}
			continue
		}
		if terminal == "Empty" {
			// Original input remains; the separate raw role is transparent JSON,
			// not a nominal hierarchy/source constructor.
			r := report(t, input, "Dto")
			if !strings.Contains(source(t, r, "AliasDto"), "= kotlinx.serialization.json.JsonElement\n") {
				t.Fatal("raw chain graduation lost carrier")
			}
			continue
		}
		aliasFailure(t, input, "E_REF_UNSUPPORTED", "/components/schemas/Alias/$ref")
	}
	aliasFailure(t, schemas(`{`+base+`,"Owner":{"properties":{"a":{"$ref":"#/components/schemas/Owner/properties/b"},"b":{"$ref":"#/components/schemas/Owner/properties/a"}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/a"}}`), "E_REF_UNSUPPORTED", "/components/schemas/Alias/$ref")
	aliasFailure(t, schemas(`{`+base+`,"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/Root"}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/hop"},"Borrow":{"allOf":[{"$ref":"#/components/schemas/Alias"},{"properties":{"id":{"type":"string"}}}]}}`), "E_REF_UNSUPPORTED", "/components/schemas/Borrow/allOf/0")
	// Discovery must still validate every original branch rather than hiding
	// a bad target under a canonical classifier binding.
	aliasFailure(t, schemas(`{`+strings.Replace(base, `"type":"string","const":"leaf"`, `"type":"integer","const":7`, 1)+`,"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/Root"}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/hop"}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Leaf/properties/kind")
}
