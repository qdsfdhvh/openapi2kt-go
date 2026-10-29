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

func intrinsicNullableTemporalInput(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/intrinsic-nullable-temporals.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestIntrinsicNullableTemporalEnvelope(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(intrinsicNullableTemporalInput(t), suffix)
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(flag, suffix, e)
			}
			again, e := core.GenerateWithDiagnostics(req)
			if e != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("determinism", e)
			}
			for name, scalar := range map[string]string{"ZStamp": "kotlin.time.Instant", "ZDay": "kotlinx.datetime.LocalDate", "ZNative": "kotlin.time.Instant"} {
				if z := source(t, r, name+suffix); !strings.Contains(z, "public typealias "+name+suffix+" = "+scalar+"?") {
					t.Fatal("publicrootalias mustnullable", z)
				}
			}

			for _, n := range []string{"AInherited", "BNullable", "ATrue"} {
				if s := source(t, r, n+suffix); !strings.Contains(s, "= kotlin.time.Instant"+"?") {
					t.Fatal(s)
				}
			}
			holder := source(t, r, "Holder"+suffix)
			for _, n := range []string{"direct", "directWrapper"} {
				if !strings.Contains(holder, "val "+n+": kotlin.time.Instant"+"? = null") {
					t.Fatal(holder)
				}
			}
			for _, n := range []string{"directFalse", "directWrapperFalse"} {
				if !strings.Contains(holder, "val "+n+": kotlin.time.Instant"+",") {
					t.Fatal(holder)
				}
			}
			if !strings.Contains(holder, "val directDay: kotlinx.datetime.LocalDate"+"? = null") || !strings.Contains(holder, "val directDayFalse: kotlinx.datetime.LocalDate"+",") {
				t.Fatal(holder)
			}
			patch := source(t, r, "SubmitInput"+suffix)
			for _, n := range []string{"direct", "directWrapper"} {
				if !strings.Contains(patch, "val "+n+": org.example.models.models.Patch<kotlin.time.Instant"+">") {
					t.Fatal(patch)
				}
			}
			for _, n := range []string{"directFalse", "directWrapperFalse"} {
				if !strings.Contains(patch, "val "+n+": kotlin.time.Instant"+"? = null") {
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
func TestIntrinsicNullableTemporalCanonicalControl(t *testing.T) {
	var doc map[string]any
	if e := json.Unmarshal(intrinsicNullableTemporalInput(t), &doc); e != nil {
		t.Fatal(e)
	}
	catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
	catalog["ZStamp"].(map[string]any)["nullable"] = false
	catalog["ZDay"].(map[string]any)["type"] = "string"
	catalog["ZNative"].(map[string]any)["nullable"] = false
	catalog["BNullable"].(map[string]any)["nullable"] = true
	var visit func(any)
	visit = func(x any) {
		switch v := x.(type) {
		case map[string]any:
			if ref, ok := v["$ref"].(string); ok && (ref == "#/components/schemas/ZStamp" || ref == "#/components/schemas/ZDay" || ref == "#/components/schemas/ZNative") {
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
func TestIntrinsicNullableTemporalBudgetsAndNoDrop(t *testing.T) {
	for _, terminal := range []string{`{"type":"string","nullable":true,"format":"date-time"}`, `{"type":"string","nullable":true,"format":"date"}`, `{"nullable":true,"x-native-type":"date"}`, `{"type":"string","nullable":true,"x-native-type":"date"}`} {
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
					b := rootSpecialChain(n, terminal, consumer)
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
	}
	for _, n := range []int{240, 241} {
		var fields, required []string
		for i := range n {
			fields = append(fields, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/A"}`, i))
			required = append(required, fmt.Sprintf(`"p%d"`, i))
		}
		b := schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","nullable":true,"format":"date-time"},"Holder":{"properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(required, ",") + `]}}`)
		r, e := core.GenerateWithDiagnostics(request(b, "Dto"))
		if n == 240 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
			t.Fatal(n, r, e)
		}
	}
	for _, terminal := range []string{`{"type":"string","nullable":true,"format":"date","enum":["ready"]}`, `{"type":"string","nullable":true,"format":"date-time","const":"2024-02-29T12:34:56Z"}`} {
		pointer := "/components/schemas/A"
		if strings.Contains(terminal, `"nullable":true`) {
			pointer = "/components/schemas/Z"
		}
		if metadataStringEnumFixture(terminal) {
			metadataStringEnumGraduation(t, request(schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":`+terminal+`}`), "Dto"))
			continue
		}
		aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":`+terminal+`}`), "E_SCHEMA_UNSUPPORTED", pointer)
	}

	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","format":true}}`), "E_SCHEMA_INVALID", "/components/schemas/Z")
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/B","nullable":true},"B":{"$ref":"#/components/schemas/A","nullable":false}}`), "E_REF_CYCLE", "/components/schemas/A/$ref")
	base := `"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","nullable":true,"format":"date-time"}`
	aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/A"},{"type":"integer"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0")
	aliasFailure(t, schemas(`{`+base+`,"C":{"allOf":[{"$ref":"#/components/schemas/A"},{"properties":{"id":{"type":"string"}}}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/allOf/0/$ref")
	aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/A"}],"discriminator":{"propertyName":"tag"}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0/$ref")
	aliasFailure(t, []byte(`{"openapi":"3.1.0","components":{"schemas":{`+base+`}},"paths":{"/events":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"$ref":"#/components/schemas/A"}}}}}}}}}`), "E_SCHEMA_UNSUPPORTED", "/paths/~1events/get/responses/200/content/text~1event-stream/schema")
}

func TestIntrinsicNullableTemporalBorrowingNoDrop(t *testing.T) {
	base := `"Z":{"type":"string","format":"date-time","nullable":true},"A":{"$ref":"#/components/schemas/Z","nullable":false}`
	for _, ref := range []string{"Z", "A"} {
		aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/`+ref+`"},{"type":"integer"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0")
		aliasFailure(t, schemas(`{`+base+`,"C":{"allOf":[{"$ref":"#/components/schemas/`+ref+`"},{"properties":{"id":{"type":"string"}}}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/allOf/0/$ref")
		aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/`+ref+`"}],"discriminator":{"propertyName":"tag"}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0/$ref")
		aliasFailure(t, []byte(`{"openapi":"3.1.0","components":{"schemas":{`+base+`}},"paths":{"/events":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"$ref":"#/components/schemas/`+ref+`"}}}}}}}}}`), "E_SCHEMA_UNSUPPORTED", "/paths/~1events/get/responses/200/content/text~1event-stream/schema")
		aliasFailure(t, schemas(`{`+base+`,"Parent":{"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag","mapping":{"x":"`+ref+`"}}},"First":{"properties":{"tag":{"type":"string"}}},"Other":{"properties":{"tag":{"type":"string"}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Parent/discriminator/mapping/x")
	}
	// Plain nullable String roots graduate under intrinsic primitive contracts.
}
