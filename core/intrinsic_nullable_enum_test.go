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

func intrinsicNullableEnumInput(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/intrinsic-nullable-enums.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestIntrinsicNullableEnumEnvelope(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(intrinsicNullableEnumInput(t), suffix)
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(flag, suffix, e)
			}
			again, e := core.GenerateWithDiagnostics(req)
			if e != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("determinism", e)
			}
			z := source(t, r, "ZStatus"+suffix)
			if !strings.Contains(z, "public sealed class ZStatus"+suffix+" {") || strings.Contains(z, "typealias") || strings.Contains(z, "NonNull") {
				t.Fatal("physical DTO stays canonical", z)
			}
			for _, n := range []string{"AInherited", "BNullable", "ATrue"} {
				if s := source(t, r, n+suffix); !strings.Contains(s, "= org.example.models.models.ZStatus"+suffix+"?") {
					t.Fatal(s)
				}
			}
			holder := source(t, r, "Holder"+suffix)
			for _, n := range []string{"direct", "directWrapper"} {
				if !strings.Contains(holder, "val "+n+": org.example.models.models.ZStatus"+suffix+"? = null") {
					t.Fatal(holder)
				}
			}
			for _, n := range []string{"directFalse", "directWrapperFalse"} {
				if !strings.Contains(holder, "val "+n+": org.example.models.models.ZStatus"+suffix+",") {
					t.Fatal(holder)
				}
			}
			if !strings.Contains(holder, "val other: org.example.models.models.ZOther"+suffix+"? = null") || !strings.Contains(holder, "val otherFalse: org.example.models.models.ZOther"+suffix+",") {
				t.Fatal(holder)
			}
			patch := source(t, r, "SubmitInput"+suffix)
			for _, n := range []string{"direct", "directWrapper"} {
				if !strings.Contains(patch, "val "+n+": org.example.models.models.Patch<org.example.models.models.ZStatus"+suffix+">") {
					t.Fatal(patch)
				}
			}
			for _, n := range []string{"directFalse", "directWrapperFalse"} {
				if !strings.Contains(patch, "val "+n+": org.example.models.models.ZStatus"+suffix+"? = null") {
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
func TestIntrinsicNullableEnumCanonicalControl(t *testing.T) {
	var doc map[string]any
	if e := json.Unmarshal(intrinsicNullableEnumInput(t), &doc); e != nil {
		t.Fatal(e)
	}
	catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
	catalog["ZStatus"].(map[string]any)["nullable"] = false
	catalog["ZOther"].(map[string]any)["type"] = "string"
	catalog["BNullable"].(map[string]any)["nullable"] = true
	var visit func(any)
	visit = func(x any) {
		switch v := x.(type) {
		case map[string]any:
			if ref, ok := v["$ref"].(string); ok && (ref == "#/components/schemas/ZStatus" || ref == "#/components/schemas/ZOther") {
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

func TestIntrinsicNullableEnumBudgetsAndNoDrop(t *testing.T) {
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
				req := request(rootSpecialChain(n, `{"nullable":true,"enum":["ready"]}`, consumer), "Dto")
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
	for _, n := range []int{240, 241} {
		var values, fields []string
		for i := range n {
			values = append(values, fmt.Sprintf(`"v%d"`, i))
			fields = append(fields, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Z"}`, i))
		}
		r, e := core.GenerateWithDiagnostics(request(schemas(`{"Z":{"nullable":true,"enum":[`+strings.Join(values, ",")+`]}}`), "Dto"))
		if n == 240 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_ENUM_LIMIT" || len(r.Artifacts.Files) != 0 {
			t.Fatal(r, e)
		}
		r, e = core.GenerateWithDiagnostics(request(schemas(`{"Z":{"nullable":true,"enum":["ready"]},"Holder":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto"))
		if n == 240 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
			t.Fatal(r, e)
		}
	}
	for _, extra := range []string{`,"format":"date"`, `,"x-native-type":"kotlin.String"`, `,"x-kotlin-value-class":true`} {
		aliasFailure(t, schemas(`{"Z":{"nullable":true,"type":"string","enum":["ready"]`+extra+`}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Z")
	}
	// Keep the original explicit nullable null vocabulary as a positive contract.
	graduated := report(t, schemas(`{"Z":{"nullable":true,"enum":["ready",null]}}`), "Dto")
	if text := source(t, graduated, "ZDto"); !strings.Contains(text, "public data object READY") || !strings.Contains(text, "Unrecognized") || strings.Contains(text, `serialName: kotlin.String = "null"`) {
		t.Fatal(text)
	}
	base := `"Z":{"nullable":true,"enum":["ready"]},"A":{"$ref":"#/components/schemas/Z","nullable":false}`
	for _, ref := range []string{"Z", "A"} {
		aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/`+ref+`"},{"type":"integer"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0")
		aliasFailure(t, schemas(`{`+base+`,"C":{"allOf":[{"$ref":"#/components/schemas/`+ref+`"},{"properties":{"id":{"type":"string"}}}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/allOf/0/$ref")
	}
	aliasFailure(t, schemas(`{`+base+`,"Parent":{"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag","mapping":{"x":"Z"}}},"First":{"properties":{"tag":{"type":"string"}}},"Other":{"properties":{"tag":{"type":"string"}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Parent/discriminator/mapping/x")
}

func TestIntrinsicNullableEnumShapeEnvelopeGuards(t *testing.T) {
	base := `"Z":{"nullable":true,"enum":["ready"]},"A":{"$ref":"#/components/schemas/Z","nullable":false}`
	for _, ref := range []string{"Z", "A"} {
		aliasFailure(t, schemas(`{`+base+`,"Parent":{"oneOf":[{"$ref":"#/components/schemas/`+ref+`"}],"discriminator":{"propertyName":"tag"}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Parent/oneOf/0/$ref")
		input := []byte(`{"openapi":"3.1.0","components":{"schemas":{` + base + `}},"paths":{"/events":{"get":{"operationId":"watch","responses":{"200":{"content":{"text/event-stream":{"schema":{"$ref":"#/components/schemas/` + ref + `"}}}}}}}}}`)
		aliasFailure(t, input, "E_SCHEMA_UNSUPPORTED", "/paths/~1events/get/responses/200/content/text~1event-stream/schema")
	}
}
