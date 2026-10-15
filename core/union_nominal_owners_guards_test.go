package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestUnionNominalNoFoldScratchDiscardedOrAmbiguousOwner(t *testing.T) {
	for _, tc := range []struct{ owner, path string }{
		{`{"properties":{"choice":{"oneOf":[{"type":"string"},{"type":"string"}]}}}`, "Owner/properties/choice"},
		{`{"properties":{"choice":{"oneOf":[{"properties":{"id":{"type":"string"}}},{"type":"null"}]}}}`, "Owner/properties/choice"},
		{`{"allOf":[{"properties":{"choice":{"oneOf":[{"type":"string"},{"type":"boolean"}]}}},{"properties":{"choice":{"type":"string"}}}]}`, "Owner/allOf/0/properties/choice"},
		{`{"allOf":[{"oneOf":[{"properties":{"kind":{"type":"string","const":"a"},"choice":{"oneOf":[{"type":"string"},{"type":"boolean"}]}}},{"properties":{"kind":{"type":"string","const":"b"},"label":{"type":"string"}}}]},{"oneOf":[{"properties":{"side":{"type":"string","const":"x"}}},{"properties":{"side":{"type":"string","const":"y"}}}]}]}`, "Owner/allOf/0/oneOf/0/properties/choice"},
		{`{"properties":{"id":{"type":"string"}},"additionalProperties":{"oneOf":[{"type":"string"},{"type":"boolean"}]}}`, "Owner/additionalProperties"},
	} {
		t.Run(tc.path+tc.owner, func(t *testing.T) {
			aliasFailure(t, schemas(`{"Owner":`+tc.owner+`,"Alias":{"$ref":"#/components/schemas/`+tc.path+`"}}`), "E_REF_UNSUPPORTED", "/components/schemas/Alias/$ref")
			late := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Owner":` + tc.owner + `}},"paths":{"/union":{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/` + tc.path + `"}}}}}}}}}`)
			aliasFailure(t, late, "E_REF_UNSUPPORTED", "/paths/~1union/get/responses/200/content/application~1json/schema")
		})
	}
}

func TestUnionNominalSourceHeightAndBranchLimits(t *testing.T) {
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
						target := "Owner/properties/choice"
						if named {
							target = "Choice"
						}
						if i > 0 {
							target = fmt.Sprint("Hops/properties/h", i-1)
						}
						fields = append(fields, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s"}`, i, target))
					}
					body := `"Owner":{"properties":{"choice":{"oneOf":[{"type":"string"},{"type":"boolean"},{"type":"null"}]}}},"Choice":{"oneOf":[{"type":"string"},{"type":"boolean"},{"type":"null"}]},"Hops":{"properties":{` + strings.Join(fields, ",") + `}},"Alias":{"$ref":"#/components/schemas/Hops/properties/h` + fmt.Sprint(n-1) + `","nullable":false}`
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
			branches := []string{}
			for i := range n {
				branches = append(branches, fmt.Sprintf(`{"properties":{"kind":{"type":"string","const":"v%d"},"id":{"type":"string"}},"required":["id"]}`, i))
			}
			req := request(schemas(`{"Owner":{"properties":{"choice":{"oneOf":[`+strings.Join(branches, ",")+`]}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/choice"}}`), "Dto")
			req.Options.Validation = checked
			r, e := core.GenerateWithDiagnostics(req)
			if n == 240 {
				if e != nil {
					t.Fatal(e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_UNION_LIMIT" || len(r.Artifacts.Files) != 0 {
				t.Fatal(n, e)
			}
		}
	}
}
