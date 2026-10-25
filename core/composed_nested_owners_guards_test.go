package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestComposedNestedOwnersCanonicalBorrowedCopiesAndLateUse(t *testing.T) {
	original := `"Source":{"allOf":[{"properties":{"node":{"properties":{"id":{"type":"string"},"next":{"$ref":"#/components/schemas/Source/allOf/0/properties/node"}},"required":["id"]}}},{"properties":{"extra":{"type":"string"}}}]}`
	borrowed := `,"BorrowLeft":{"allOf":[{"$ref":"#/components/schemas/Source"},{"properties":{"left":{"type":"boolean"}}}]},"BorrowRight":{"allOf":[{"$ref":"#/components/schemas/Source"},{"properties":{"right":{"type":"boolean"}}}]}`
	for _, alias := range []bool{false, true} {
		body := original + borrowed
		if alias {
			body += `,"Alias":{"$ref":"#/components/schemas/Source/allOf/0/properties/node"}`
		}
		input := []byte(`{"openapi":"3.1.0","components":{"schemas":{` + body + `}},"paths":{"/node":{"get":{"operationId":"getNode","responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Source/allOf/0/properties/node"}}}}}}}}}`)
		r := report(t, input, "Dto")
		if !strings.Contains(source(t, r, "SourceDto"), "next: org.example.models.models.SourceDto.Node? = null") || !strings.Contains(source(t, r, "BorrowLeftDto"), "next: org.example.models.models.SourceDto.Node? = null") {
			t.Fatal("borrowed occurrence became owner")
		}
		if alias && !strings.Contains(source(t, r, "AliasDto"), "= org.example.models.models.SourceDto.Node\n") {
			t.Fatal("canonical alias lost")
		}
		client := source(t, r, "DefaultApi")
		if !strings.Contains(client, "org.example.models.models.SourceDto.Node") {
			t.Fatal("first HTTP use did not bind retained lexical model", client)
		}
	}
}

func TestComposedNestedOwnersAmbiguousDiscardedAndScratchTargets(t *testing.T) {
	for _, tc := range []struct{ owner, target string }{
		{`{"allOf":[{"oneOf":[{"properties":{"kind":{"type":"string","const":"a"},"node":{"properties":{"id":{"type":"string"}}}}},{"properties":{"kind":{"type":"string","const":"b"},"label":{"type":"string"}}}]},{"oneOf":[{"properties":{"side":{"type":"string","const":"x"}}},{"properties":{"side":{"type":"string","const":"y"}}}]}]}`, "Owner/allOf/0/oneOf/0/properties/node"},
		{`{"allOf":[{"properties":{"node":{"properties":{"id":{"type":"string"}}}}},{"properties":{"node":{"type":"string"}}}]}`, "Owner/allOf/0/properties/node"},
		{`{"allOf":[{"properties":{"node":{"properties":{"id":{"type":"string"}}}}},{"properties":{"node":{"properties":{"other":{"type":"string"}}}}}]}`, "Owner/allOf/0/properties/node"},
		{`{"allOf":[{"properties":{"id":{"type":"string"}},"additionalProperties":{"properties":{"id":{"type":"string"}}}},{"properties":{"extra":{"type":"string"}}}]}`, "Owner/allOf/0/additionalProperties"},
		{`{"allOf":[{"properties":{"node":{"properties":{"pin":{"type":"string","const":"x"}}}}},{"properties":{"extra":{"type":"string"}}}]}`, "Owner/allOf/0/properties/node"},
		{`{"allOf":[{"properties":{"node":{"type":"object"}}},{"properties":{"extra":{"type":"string"}}}]}`, "Owner/allOf/0/properties/node"},
	} {
		t.Run(tc.target+tc.owner, func(t *testing.T) {
			input := schemas(`{"Source":{"properties":{"node":{"properties":{"id":{"type":"string"}}}}},"Owner":` + tc.owner + `,"Alias":{"$ref":"#/components/schemas/` + tc.target + `"}}`)
			aliasFailure(t, input, "E_REF_UNSUPPORTED", "/components/schemas/Alias/$ref")
			late := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Source":{"properties":{"node":{"properties":{"id":{"type":"string"}}}}},"Owner":` + tc.owner + `}},"paths":{"/node":{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/` + tc.target + `"}}}}}}}}}`)
			aliasFailure(t, late, "E_REF_UNSUPPORTED", "/paths/~1node/get/responses/200/content/application~1json/schema")
		})
	}
	owner := `"Owner":{"allOf":[{"properties":{"node":{"properties":{"id":{"type":"string"}}}}},{"properties":{"extra":{"type":"string"}}}]}`
	for _, consumer := range []string{`{"allOf":[{"$ref":"#/components/schemas/Owner/allOf/0/properties/node"},{"properties":{"extra":{"type":"string"}}}]}`, `{"oneOf":[{"$ref":"#/components/schemas/Owner/allOf/0/properties/node"},{"properties":{"id":{"type":"string"}}}]}`} {
		t.Run("shape"+consumer, func(t *testing.T) {
			input := schemas(`{` + owner + `,"Consumer":` + consumer + `}`)
			if strings.HasPrefix(consumer, `{"allOf"`) {
				r := report(t, input, "Dto")
				if !strings.Contains(source(t, r, "ConsumerDto"), "id: kotlin.String? = null") {
					t.Fatal("original nested composedallOf fields lost")
				}
			} else {
				aliasFailure(t, input, "E_REF_UNSUPPORTED", "/components/schemas/Consumer/oneOf/0")
			}
		})
	}
}

func TestComposedNestedOwnersDepthAndSlots(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, typed := range []bool{false, true} {
			limit := 47
			if typed {
				limit = 45
			}
			for _, n := range []int{limit, limit + 1} {
				fields := []string{}
				for i := range n {
					target := "Owner/allOf/0/properties/node"
					if i > 0 {
						target = fmt.Sprint("Hops/properties/h", i-1)
					}
					fields = append(fields, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s"}`, i, target))
				}
				body := `"Owner":{"allOf":[{"properties":{"node":{"properties":{"id":{"type":"string"}}}}},{"properties":{"extra":{"type":"string"}}}]},"Hops":{"properties":{` + strings.Join(fields, ",") + `}},"Alias":{"$ref":"#/components/schemas/Hops/properties/h` + fmt.Sprint(n-1) + `","nullable":false}`
				if typed {
					body += `,"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Alias","nullable":false}},"required":["value"]}`
				}
				req := request(schemas(`{`+body+`}`), "Dto")
				req.Options.Validation = checked
				r, e := core.GenerateWithDiagnostics(req)
				if n == limit {
					if e != nil || !strings.Contains(source(t, r, "AliasDto"), "= org.example.models.models.OwnerDto.Node\n") {
						t.Fatal(checked, typed, n, e)
					}
				} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0 {
					t.Fatal(n, e)
				}
			}
		}
		for _, n := range []int{240, 241} {
			fields := []string{}
			for i := range n {
				fields = append(fields, fmt.Sprintf(`"p%d":{"type":"string"}`, i))
			}
			req := request(schemas(`{"Owner":{"allOf":[{"properties":{"node":{"properties":{`+strings.Join(fields, ",")+`}}}},{"properties":{"extra":{"type":"string"}}}]},"Alias":{"$ref":"#/components/schemas/Owner/allOf/0/properties/node"}}`), "Dto")
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
