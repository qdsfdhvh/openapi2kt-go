package core_test

import (
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"os"
	"reflect"
	"strings"
	"testing"
)

func intrinsicNullableValueClassInput(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/intrinsic-nullable-value-classes.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestIntrinsicNullableValueClassEnvelope(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(intrinsicNullableValueClassInput(t), suffix)
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(flag, suffix, e)
			}
			again, e := core.GenerateWithDiagnostics(req)
			if e != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("determinism", e)
			}
			z := source(t, r, "ZName"+suffix)
			if !strings.Contains(z, "public value class ZName"+suffix+"(") || strings.Contains(z, "typealias") || strings.Contains(z, "NonNull") {
				t.Fatal("physical wrapper stays canonical", z)
			}
			for _, n := range []string{"AInherited", "BNullable", "ATrue"} {
				if s := source(t, r, n+suffix); !strings.Contains(s, "= org.example.models.models.ZName"+suffix+"?") {
					t.Fatal(s)
				}
			}
			holder := source(t, r, "Holder"+suffix)
			for _, n := range []string{"direct", "directWrapper"} {
				if !strings.Contains(holder, "val "+n+": org.example.models.models.ZName"+suffix+"? = null") {
					t.Fatal(holder)
				}
			}
			for _, n := range []string{"directFalse", "directWrapperFalse"} {
				if !strings.Contains(holder, "val "+n+": org.example.models.models.ZName"+suffix+",") {
					t.Fatal(holder)
				}
			}
			if !strings.Contains(holder, "val directCount: org.example.models.models.ZCount"+suffix+"? = null") || !strings.Contains(holder, "val directCountFalse: org.example.models.models.ZCount"+suffix+",") {
				t.Fatal(holder)
			}
			patch := source(t, r, "SubmitInput"+suffix)
			for _, n := range []string{"direct", "directWrapper"} {
				if !strings.Contains(patch, "val "+n+": org.example.models.models.Patch<org.example.models.models.ZName"+suffix+">") {
					t.Fatal(patch)
				}
			}
			for _, n := range []string{"directFalse", "directWrapperFalse"} {
				if !strings.Contains(patch, "val "+n+": org.example.models.models.ZName"+suffix+"? = null") {
					t.Fatal(patch)
				}
			}
			api := source(t, r, "DefaultApi")
			if strings.Count(api, "public suspend fun echo(") != 1 || strings.Count(api, "public suspend fun echo2(") != 1 {
				t.Fatal("canonical JVM erasure", api)
			}
		}
	}
}
func TestIntrinsicNullableValueClassCanonicalControl(t *testing.T) {
	var doc map[string]any
	if e := json.Unmarshal(intrinsicNullableValueClassInput(t), &doc); e != nil {
		t.Fatal(e)
	}
	catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
	catalog["ZName"].(map[string]any)["nullable"] = false
	catalog["ZCount"].(map[string]any)["type"] = "integer"
	catalog["BNullable"].(map[string]any)["nullable"] = true
	var visit func(any)
	visit = func(x any) {
		switch v := x.(type) {
		case map[string]any:
			if ref, ok := v["$ref"].(string); ok && (ref == "#/components/schemas/ZName" || ref == "#/components/schemas/ZCount") {
				if _, has := v["nullable"]; !has {
					v["nullable"] = true
				}
			}
			for _, child := range v {
				visit(child)
			}
		case []any:
			for _, child := range v {
				visit(child)
			}
		}
	}
	visit(doc)
	b, _ := json.Marshal(doc)
	for _, flag := range []bool{false, true} {
		req := request(b, "Dto")
		req.Options.Validation = flag
		if _, e := core.GenerateWithDiagnostics(req); e != nil {
			t.Fatal("whole existing canonical control", flag, e)
		}
	}
}
func TestIntrinsicNullableValueClassBudgetsAndNoDrop(t *testing.T) {
	for _, terminal := range []string{`{"type":"string","nullable":true,"x-kotlin-value-class":true}`, `{"type":"boolean","nullable":true,"x-kotlin-value-class":true}`, `{"type":"integer","nullable":true,"x-kotlin-value-class":true}`, `{"type":"integer","format":"int64","nullable":true,"x-kotlin-value-class":true}`, `{"type":"number","nullable":true,"x-kotlin-value-class":true}`, `{"type":"number","format":"float","nullable":true,"x-kotlin-value-class":true}`} {
		for _, flag := range []bool{false, true} {
			req := request(schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":`+terminal+`}`), "Dto")
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil || !strings.Contains(source(t, r, "ADto"), "ZDto?") {
				t.Fatal(r, e)
			}
		}
	}
	for _, flag := range []bool{false, true} {
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
				b := rootSpecialChain(n, `{"type":"integer","format":"int64","nullable":true,"x-kotlin-value-class":true}`, consumer)
				b = []byte(strings.Replace(string(b), `"A00":{"$ref":"#/components/schemas/Z"}`, `"A00":{"$ref":"#/components/schemas/Z","nullable":true}`, 1))
				req := request(b, "Dto")
				req.Options.Validation = flag
				r, e := core.GenerateWithDiagnostics(req)
				if n == limit {
					if e != nil {
						t.Fatal(e)
					}
				} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0 {
					t.Fatal(n, r, e)
				}
			}
		}
	}
	for _, format := range []string{"int64", "double"} {
		kind := "integer"
		if format == "double" {
			kind = "number"
		}
		for _, narrow := range []bool{false, true} {
			limit := 240
			if narrow {
				limit = 120
			}
			for _, n := range []int{limit, limit + 1} {
				var fields, required []string
				for i := range n {
					extra := ""
					if narrow {
						extra = `,"nullable":false`
					}
					fields = append(fields, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/A"%s}`, i, extra))
					required = append(required, fmt.Sprintf(`"p%d"`, i))
				}
				b := schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"` + kind + `","format":"` + format + `","nullable":true,"x-kotlin-value-class":true},"Holder":{"properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(required, ",") + `]}}`)
				r, e := core.GenerateWithDiagnostics(request(b, "Dto"))
				if n == limit {
					if e != nil {
						t.Fatal(e)
					}
				} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
					t.Fatal(n, r, e)
				}
			}
		}
	}
	base := `"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","nullable":true,"x-kotlin-value-class":true}`
	aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/A"},{"type":"integer"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0")
	aliasFailure(t, schemas(`{`+base+`,"C":{"allOf":[{"$ref":"#/components/schemas/A"},{"properties":{"id":{"type":"string"}}}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/allOf/0/$ref")
	aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/A"}],"discriminator":{"propertyName":"tag"}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0/$ref")
	aliasFailure(t, []byte(`{"openapi":"3.1.0","components":{"schemas":{`+base+`}},"paths":{"/events":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"$ref":"#/components/schemas/A"}}}}}}}}}`), "E_SCHEMA_UNSUPPORTED", "/paths/~1events/get/responses/200/content/text~1event-stream/schema")
	for _, body := range []string{`{"type":"string","nullable":true,"x-kotlin-value-class":false}`} {
		aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":`+body+`}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Z")
	}
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/B","nullable":true},"B":{"$ref":"#/components/schemas/A","nullable":false}}`), "E_REF_CYCLE", "/components/schemas/A/$ref")
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","nullable":true,"x-kotlin-value-class":true,"minLength":-1}}`), "E_SCHEMA_INVALID", "/components/schemas/Z/minLength")
}

func TestIntrinsicNullableValueClassMethodSlots(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, narrow := range []bool{false, true} {
			limit := 240
			if narrow {
				limit = 120
			}
			if stream {
				limit--
			}
			for _, n := range []int{limit, limit + 1} {
				var params []string
				for i := range n {
					extra := ""
					if narrow {
						extra = `,"nullable":false`
					}
					params = append(params, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":{"$ref":"#/components/schemas/A"%s}}`, i, extra))
				}
				response := `{"204":{}}`
				if stream {
					response = `{"200":{"content":{"text/event-stream":{"schema":{"properties":{}}}}}}`
				}
				b := []byte(`{"openapi":"3.1.0","components":{"schemas":{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"integer","format":"int64","nullable":true,"x-kotlin-value-class":true}}},"paths":{"/wide":{"get":{"parameters":[` + strings.Join(params, ",") + `],"responses":` + response + `}}}}`)
				r, e := core.GenerateWithDiagnostics(request(b, "Dto"))
				if n == limit {
					if e != nil {
						t.Fatal(e)
					}
				} else if e == nil || e.(core.Diagnostics)[0].Code != "E_OPERATION_LIMIT" || len(r.Artifacts.Files) != 0 {
					t.Fatal(n, narrow, stream, r, e)
				}
			}
		}
	}
}

func TestIntrinsicNullableValueClassBorrowingNoDrop(t *testing.T) {
	base := `"Z":{"type":"string","nullable":true,"x-kotlin-value-class":true},"A":{"$ref":"#/components/schemas/Z","nullable":false}`
	for _, ref := range []string{"Z", "A"} {
		aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/`+ref+`"},{"type":"integer"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0")
		aliasFailure(t, schemas(`{`+base+`,"C":{"allOf":[{"$ref":"#/components/schemas/`+ref+`"},{"properties":{"id":{"type":"string"}}}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/allOf/0/$ref")
		aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/`+ref+`"}],"discriminator":{"propertyName":"tag"}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0/$ref")
		aliasFailure(t, []byte(`{"openapi":"3.1.0","components":{"schemas":{`+base+`}},"paths":{"/events":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"$ref":"#/components/schemas/`+ref+`"}}}}}}}}}`), "E_SCHEMA_UNSUPPORTED", "/paths/~1events/get/responses/200/content/text~1event-stream/schema")
		aliasFailure(t, schemas(`{`+base+`,"Parent":{"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag","mapping":{"x":"`+ref+`"}}},"First":{"properties":{"tag":{"type":"string"}}},"Other":{"properties":{"tag":{"type":"string"}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Parent/discriminator/mapping/x")
	}
}
