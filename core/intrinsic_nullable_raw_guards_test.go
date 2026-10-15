package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestIntrinsicNullableRawNoDropOrShapeBorrowing(t *testing.T) {
	for _, body := range []string{
		`"type":"object","nullable":true,"minProperties":0`,
		`"type":"object","nullable":true,"maxProperties":1`,
		`"type":"object","nullable":true,"properties":{"pin":{"type":"integer","const":7}}`,
		`"type":"object","nullable":true,"allOf":[{"type":"object"}]`,
	} {
		r, e := core.GenerateWithDiagnostics(request(schemas(`{"Z":{`+body+`}}`), "Dto"))
		if e == nil || len(r.Artifacts.Files) != 0 || len(r.Warnings) != 0 {
			t.Fatal("dropped semantics", body, r, e)
		}
	}
	base := `"Z":{"type":"object","nullable":true},"A":{"$ref":"#/components/schemas/Z","nullable":false},"Other":{"properties":{"id":{"type":"string"}}}`
	for _, name := range []string{"Z", "A"} {
		aliasFailure(t, schemas(`{`+base+`,"C":{"allOf":[{"$ref":"#/components/schemas/`+name+`","nullable":false},{"properties":{"id":{"type":"string"}}}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/allOf/0/$ref")
		aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/`+name+`"},{"$ref":"#/components/schemas/Other"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0")
		// The unchanged historical discriminator input has an actual leaf
		// declaration. Its mapped tag survives the empty source object.
		input := schemas(`{` + base + `,"C":{"oneOf":[{"$ref":"#/components/schemas/` + name + `"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag"}}}`)
		for _, validation := range []bool{false, true} {
			req := request(input, "Dto")
			req.Options.Validation = validation
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil || !strings.Contains(source(t, r, "CDto"), "public sealed class CDto {") || !strings.Contains(source(t, r, "ZDto"), "public data object ZDto : org.example.models.models.CDto() {") || !strings.Contains(source(t, r, "ZDto"), `"tag" to kotlinx.serialization.json.JsonPrimitive(value.tag.serialName)`) || !strings.Contains(source(t, r, "ADto"), "public typealias ADto = org.example.models.models.ZDto\n") {
				t.Fatal(validation, name, r, e)
			}
		}
	}
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":false},"Z":{"type":"object","nullable":true},"Parent":{"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag","mapping":{"leaf":"A"}}},"First":{"properties":{"tag":{"type":"string"},"id":{"type":"string"}}},"Other":{"properties":{"tag":{"type":"string"},"ok":{"type":"boolean"}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Parent/discriminator/mapping/leaf")
	aliasFailure(t, []byte(`{"openapi":"3.1.0","components":{"schemas":{"Z":{"type":"object","nullable":true},"A":{"$ref":"#/components/schemas/Z","nullable":false}}},"paths":{"/events":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"$ref":"#/components/schemas/A"}}}}}}}}}`), "E_SCHEMA_UNSUPPORTED", "/paths/~1events/get/responses/200/content/text~1event-stream/schema")
	aliasFailure(t, schemas(`{"Z":{"type":"object","nullable":true,"properties":[]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Z")
	aliasFailure(t, schemas(`{"Z":{"type":"object","nullable":true,"required":true}}`), "E_SCHEMA_INVALID", "/components/schemas/Z")
}
func TestIntrinsicNullableRawBudgets(t *testing.T) {
	for _, validation := range []bool{false, true} {
		for _, typed := range []bool{false, true} {
			limit := 48
			if typed {
				limit = 46
			}
			for _, n := range []int{limit, limit + 1} {
				consumer := ""
				if typed {
					consumer = fmt.Sprintf(`{"properties":{"v":{"$ref":"#/components/schemas/A%02d","nullable":false}}}`, n-1)
				}
				r := request(rootSpecialChain(n, `{"type":"object","nullable":true}`, consumer), "Dto")
				r.Options.Validation = validation
				got, e := core.GenerateWithDiagnostics(r)
				if n == limit {
					if e != nil {
						t.Fatal(n, typed, validation, e)
					}
				} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 || len(got.Warnings) != 0 {
					t.Fatal(n, got, e)
				}
			}
		}
		for _, n := range []int{240, 241} {
			var fields, required []string
			for i := range n {
				fields = append(fields, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Z"}`, i))
				required = append(required, fmt.Sprintf(`"p%d"`, i))
			}
			r := request(schemas(`{"Z":{"type":"object","nullable":true},"Holder":{"properties":{`+strings.Join(fields, ",")+`},"required":[`+strings.Join(required, ",")+`]}}`), "Dto")
			r.Options.Validation = validation
			got, e := core.GenerateWithDiagnostics(r)
			if n == 240 {
				if e != nil {
					t.Fatal(e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0 {
				t.Fatal(n, got, e)
			}
		}
		for _, n := range []int{44, 45} {
			list := `{"$ref":"#/components/schemas/A"}`
			for range n {
				list = `{"type":"array","items":` + list + `}`
			}
			b := schemas(`{"A":{"$ref":"#/components/schemas/Z"},"Z":{"type":"object","nullable":true},"ListRoot":` + list + `,"Holder":{"properties":{"v":{"$ref":"#/components/schemas/ListRoot"}}}}`)
			r := request(b, "Dto")
			r.Options.Validation = validation
			got, e := core.GenerateWithDiagnostics(r)
			control := request([]byte(strings.Replace(string(b), `"nullable":true`, `"nullable":false`, 1)), "Dto")
			control.Options.Validation = validation
			_, ce := core.GenerateWithDiagnostics(control)
			if (e == nil) != (ce == nil) {
				t.Fatal("nullable adds source work", n, e, ce)
			}
			if n == 44 {
				if e != nil {
					t.Fatal(e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
				t.Fatal(n, got, e)
			}
		}
	}
}
