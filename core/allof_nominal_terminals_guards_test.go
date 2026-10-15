package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestAllOfNominalTerminalsNoProjectedOrDiscardedOwner(t *testing.T) {
	for _, tc := range []struct{ body, path string }{
		{`"Owner":{"properties":{"node":{"allOf":[{"type":"object"}]}}}`, "Owner/properties/node"},
		{`"Owner":{"properties":{"node":{"allOf":[{"properties":{"id":{"type":"string","const":"fixed"}}}]}}}`, "Owner/properties/node"},
		{`"Owner":{"allOf":[{"properties":{"node":{"allOf":[{"properties":{"id":{"type":"string"}}}]}}},{"properties":{"node":{"type":"string"}}}]}`, "Owner/allOf/0/properties/node"},
		{`"Owner":{"allOf":[{"oneOf":[{"properties":{"kind":{"type":"string","const":"a"},"node":{"allOf":[{"properties":{"id":{"type":"string"}}}]}}},{"properties":{"kind":{"type":"string","const":"b"},"label":{"type":"string"}}}]},{"oneOf":[{"properties":{"side":{"type":"string","const":"x"}}},{"properties":{"side":{"type":"string","const":"y"}}}]}]}`, "Owner/allOf/0/oneOf/0/properties/node"},
		{`"Owner":{"allOf":[{"oneOf":[{"properties":{"kind":{"type":"string","const":"a"},"id":{"type":"string"}}},{"properties":{"kind":{"type":"string","const":"b"},"name":{"type":"string"}}}]},{"properties":{"extra":{"type":"string"}}}]},"Hops":{"properties":{"hop":{"$ref":"#/components/schemas/Owner"}}}`, "Hops/properties/hop"},
		{`"Owner":{"allOf":[{"type":"object"}]},"Hops":{"properties":{"hop":{"$ref":"#/components/schemas/Owner"}}}`, "Hops/properties/hop"},
	} {
		t.Run(tc.path+tc.body, func(t *testing.T) {
			input := schemas(`{` + tc.body + `,"Alias":{"$ref":"#/components/schemas/` + tc.path + `"}}`)
			if tc.path == "Hops/properties/hop" && strings.Contains(tc.body, `"oneOf"`) {
				// The original product-union input now binds its retained sealed
				// declaration. Keep the input and assert its canonical API explicitly.
				r := report(t, input, "Dto")
				if !strings.Contains(source(t, r, "AliasDto"), "= org.example.models.models.OwnerDto\n") || !strings.Contains(source(t, r, "OwnerDto"), "public sealed class OwnerDto") {
					t.Fatal("product terminal lost canonical union")
				}
				return
			}
			if tc.path == "Hops/properties/hop" && tc.body == `"Owner":{"allOf":[{"type":"object"}]},"Hops":{"properties":{"hop":{"$ref":"#/components/schemas/Owner"}}}` {
				// Preserve the original empty allOf root input; actual retained
				// empty ordinary Model now grants the independent raw source role.
				r := report(t, input, "Dto")
				if !strings.Contains(source(t, r, "AliasDto"), "= kotlinx.serialization.json.JsonElement\n") {
					t.Fatal("raw allOf source graduation lost backing")
				}
				return
			}
			aliasFailure(t, input, "E_REF_UNSUPPORTED", "/components/schemas/Alias/$ref")
		})
	}
	all := `"Owner":{"allOf":[{"properties":{"id":{"type":"string"}}},{"properties":{"next":{"$ref":"#/components/schemas/Owner"}}}]},"Hops":{"properties":{"hop":{"$ref":"#/components/schemas/Owner"}}},"Alias":{"$ref":"#/components/schemas/Hops/properties/hop"}`
	aliasFailure(t, schemas(`{`+all+`,"Consumer":{"allOf":[{"$ref":"#/components/schemas/Alias"},{"properties":{"extra":{"type":"string"}}}]}}`), "E_REF_UNSUPPORTED", "/components/schemas/Consumer/allOf/0")
	borrowedAllOfModelGraduation(t, schemas(`{`+all+`,"Extra":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}}}},"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Extra/properties/rows"}}}}`), "org.example.models.models.OwnerDto")
	aliasFailure(t, schemas(`{"Alias":{"$ref":"#/components/schemas/Owner/properties/hop"},"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/Alias"}}}}`), "E_REF_CYCLE", "/components/schemas/Alias/$ref")
}

func TestAllOfNominalTerminalNullableIntentAndFullSourceHeight(t *testing.T) {
	node := `{"allOf":[{"properties":{"id":{"type":"string"}},"required":["id"]},{"properties":{"other":{"type":"string"}}}],"nullable":true}`
	for _, checked := range []bool{false, true} {
		req := request(schemas(`{"Owner":{"properties":{"node":`+node+`}},"Alias":{"$ref":"#/components/schemas/Owner/properties/node"},"Maybe":{"$ref":"#/components/schemas/Alias"},"False":{"$ref":"#/components/schemas/Maybe","nullable":false},"ReNull":{"$ref":"#/components/schemas/False","nullable":true},"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/False"},"renull":{"$ref":"#/components/schemas/ReNull"}},"required":["value","renull"]}}`), "Dto")
		req.Options.Validation = checked
		r, e := core.GenerateWithDiagnostics(req)
		if e != nil {
			t.Fatal(e)
		}
		for _, tc := range []struct {
			name     string
			nullable bool
		}{{"Alias", true}, {"Maybe", true}, {"False", false}, {"ReNull", true}} {
			want := "= org.example.models.models.OwnerDto.Node"
			if tc.nullable {
				want += "?"
			}
			want += "\n"
			if !strings.Contains(source(t, r, tc.name+"Dto"), want) {
				t.Fatal(tc, source(t, r, tc.name+"Dto"))
			}
		}
		for _, named := range []bool{false, true} {
			// n original hop fields include the final h0 reference. The incoming
			// Alias edge adds one step for both internal and named terminals.
			limit := 47
			for _, typed := range []bool{false, true} {
				bound := limit
				if typed {
					bound -= 2
				}
				for _, n := range []int{bound, bound + 1} {
					fields := []string{}
					for i := range n {
						target := "Owner/properties/node"
						if named {
							target = "AllModel"
						}
						if i > 0 {
							target = fmt.Sprint("Hops/properties/h", i-1)
						}
						fields = append(fields, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s"}`, i, target))
					}
					body := `"Owner":{"properties":{"node":` + node + `}},"AllModel":{"allOf":[{"properties":{"id":{"type":"string"}}},{"properties":{"more":{"type":"string"}}}]},"Hops":{"properties":{` + strings.Join(fields, ",") + `}},"Alias":{"$ref":"#/components/schemas/Hops/properties/h` + fmt.Sprint(n-1) + `","nullable":false}`
					if typed {
						body += `,"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Alias","nullable":false}},"required":["value"]}`
					}
					req := request(schemas(`{`+body+`}`), "Dto")
					req.Options.Validation = checked
					r, e := core.GenerateWithDiagnostics(req)
					if n == bound {
						if e != nil {
							t.Fatal(checked, named, typed, n, e)
						}
					} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0 {
						t.Fatal(checked, named, typed, n, e)
					}
				}
			}
		}
		for _, n := range []int{240, 241} {
			fields := []string{}
			for i := range n {
				fields = append(fields, fmt.Sprintf(`"p%d":{"type":"string"}`, i))
			}
			req := request(schemas(`{"Owner":{"properties":{"node":{"allOf":[{"properties":{`+strings.Join(fields, ",")+`}}]}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/node"}}`), "Dto")
			req.Options.Validation = checked
			r, e := core.GenerateWithDiagnostics(req)
			if n == 240 {
				if e != nil {
					t.Fatal(e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
				t.Fatal(n, e)
			}
		}
	}
}
