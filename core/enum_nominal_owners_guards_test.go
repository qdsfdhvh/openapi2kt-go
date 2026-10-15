package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestEnumNominalOwnerAndSourceGuards(t *testing.T) {
	for _, tc := range []struct{ owner, path string }{
		{`{"allOf":[{"properties":{"state":{"enum":["on","off"]}}},{"properties":{"state":{"type":"string"}}}]}`, "Owner/allOf/0/properties/state"},
		{`{"properties":{"id":{"type":"string"}},"additionalProperties":{"enum":["on","off"]}}`, "Owner/additionalProperties"},
		{`{"allOf":[{"oneOf":[{"properties":{"kind":{"type":"string","const":"a"},"state":{"enum":["on","off"]}}},{"properties":{"kind":{"type":"string","const":"b"},"label":{"type":"string"}}}]},{"oneOf":[{"properties":{"side":{"type":"string","const":"x"}}},{"properties":{"side":{"type":"string","const":"y"}}}]}]}`, "Owner/allOf/0/oneOf/0/properties/state"},
		{`{"properties":{"state":{"type":"string","format":"uuid","enum":["on","off"]}}}`, "Owner/properties/state"},
		{`{"properties":{"state":{"type":"integer","enum":[1,2]}}}`, "Owner/properties/state"},
	} {
		input := schemas(`{"Owner":` + tc.owner + `,"Alias":{"$ref":"#/components/schemas/` + tc.path + `"}}`)
		if strings.Contains(tc.owner, `"type":"integer"`) {
			// Preserve the original Pointer input as a validation-only scalar
			// bridge graduation with source membership, never a nominal enum.
			r := request(input, "Dto")
			r.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil || !strings.Contains(source(t, got, "AliasDto"), "typealias AliasDto = kotlin.Int") || !strings.Contains(source(t, got, "OwnerDto"), "value == 1 || value == 2") {
				t.Fatal(got, err)
			}
			failure(t, input, "E_REF_UNSUPPORTED", "/components/schemas/Alias/$ref")
		} else {
			aliasFailure(t, input, "E_REF_UNSUPPORTED", "/components/schemas/Alias/$ref")
		}
		late := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Owner":` + tc.owner + `}},"paths":{"/enum":{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/` + tc.path + `"}}}}}}}}}`)
		if strings.Contains(tc.owner, `"type":"integer"`) {
			r := request(late, "Dto")
			r.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil || !strings.Contains(source(t, got, "DefaultApi"), "value == 1 || value == 2") {
				t.Fatal(got, err)
			}
			failure(t, late, "E_REF_UNSUPPORTED", "/paths/~1enum/get/responses/200/content/application~1json/schema")
		} else {
			aliasFailure(t, late, "E_REF_UNSUPPORTED", "/paths/~1enum/get/responses/200/content/application~1json/schema")
		}
	}
	aliasFailure(t, schemas(`{"Owner":{"properties":{"state":{"enum":["a",null]}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/state"}}`), "E_SCHEMA_INVALID", "/components/schemas/Owner/properties/state/enum/1")
	aliasFailure(t, schemas(`{"Alias":{"$ref":"#/components/schemas/Owner/properties/hop"},"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/Alias"}}}}`), "E_REF_CYCLE", "/components/schemas/Alias/$ref")
}

func TestEnumNominalSourceHeightVocabularyAndSlotLimits(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, named := range []bool{false, true} {
			for _, typed := range []bool{false, true} {
				limit := 47
				if typed {
					limit = 45
				}
				for _, n := range []int{limit, limit + 1} {
					fields := []string{}
					for i := range n {
						target := "Owner/properties/state"
						if named {
							target = "Named"
						}
						if i > 0 {
							target = fmt.Sprint("Hops/properties/h", i-1)
						}
						fields = append(fields, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s"}`, i, target))
					}
					body := `"Owner":{"properties":{"state":{"enum":["on","off"],"nullable":true}}},"Named":{"enum":["on","off"],"nullable":true},"Hops":{"properties":{` + strings.Join(fields, ",") + `}},"Alias":{"$ref":"#/components/schemas/Hops/properties/h` + fmt.Sprint(n-1) + `","nullable":false}`
					if typed {
						body += `,"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Alias","nullable":false}},"required":["value"]}`
					}
					req := request(schemas(`{`+body+`}`), "Dto")
					req.Options.Validation = checked
					r, e := core.GenerateWithDiagnostics(req)
					if n == limit {
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
			values := []string{}
			for i := range n {
				values = append(values, fmt.Sprintf(`"v%d"`, i))
			}
			req := request(schemas(`{"Owner":{"properties":{"state":{"enum":[`+strings.Join(values, ",")+`]}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/state"}}`), "Dto")
			req.Options.Validation = checked
			r, e := core.GenerateWithDiagnostics(req)
			if n == 240 {
				if e != nil {
					t.Fatal(e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_ENUM_LIMIT" || len(r.Artifacts.Files) != 0 {
				t.Fatal(n, e)
			}
			props := []string{}
			required := []string{}
			for i := range n {
				props = append(props, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Alias"}`, i))
				required = append(required, fmt.Sprintf(`"p%d"`, i))
			}
			req = request(schemas(`{"Owner":{"properties":{"state":{"enum":["on","off"]}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/state"},"Wide":{"properties":{`+strings.Join(props, ",")+`},"required":[`+strings.Join(required, ",")+`]}}`), "Dto")
			req.Options.Validation = checked
			r, e = core.GenerateWithDiagnostics(req)
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

func TestEnumNominalBorrowedCopyAndLateUse(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Owner":{"properties":{"state":{"enum":["on","off"]}}},"Borrow":{"allOf":[{"$ref":"#/components/schemas/Owner"},{"properties":{"extra":{"type":"string"}}}]}}},"paths":{"/enum":{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Owner/properties/state"}}}}}}}}}`)
	r := report(t, input, "Dto")
	if !strings.Contains(source(t, r, "DefaultApi"), "org.example.models.models.OwnerDto.State") {
		t.Fatal("late use borrowed enum owner")
	}
}
