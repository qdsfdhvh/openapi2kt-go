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

func nullableTemporalInput(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/root-nullable-temporal-aliases.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestRootNullableTemporalAliasEnvelope(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(nullableTemporalInput(t), suffix)
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(flag, suffix, e)
			}
			again, e := core.GenerateWithDiagnostics(req)
			if e != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("determinism", e)
			}
			for _, n := range []string{"AInherited", "BNullable", "ATrue"} {
				s := source(t, r, n+suffix)
				if !strings.Contains(s, "= kotlin.time.Instant"+"?") {
					t.Fatal(s)
				}
			}
			if s := source(t, r, "AFalse"+suffix); !strings.Contains(s, "= kotlin.time.Instant") || strings.Contains(s, "?") {
				t.Fatal(s)
			}
			holder := source(t, r, "Holder"+suffix)
			for _, name := range []string{"inherited", "forcedTrue", "wrapper"} {
				if !strings.Contains(holder, "val "+name+": kotlin.time.Instant"+"? = null") {
					t.Fatal(holder)
				}
			}
			for _, name := range []string{"forcedFalse", "wrapperFalse"} {
				if !strings.Contains(holder, "val "+name+": kotlin.time.Instant"+",") {
					t.Fatal(holder)
				}
			}
			patch := source(t, r, "SubmitInput"+suffix)
			for _, name := range []string{"inherited", "forcedTrue", "wrapper"} {
				if !strings.Contains(patch, "val "+name+": org.example.models.models.Patch<kotlin.time.Instant"+">") {
					t.Fatal(patch)
				}
			}
			for _, name := range []string{"forcedFalse", "wrapperFalse"} {
				if !strings.Contains(patch, "val "+name+": kotlin.time.Instant"+"? = null") {
					t.Fatal(patch)
				}
			}
			if !strings.Contains(patch, "val requiredNull: kotlin.time.Instant"+"?,") {
				t.Fatal(patch)
			}
			api := source(t, r, "DefaultApi")
			if strings.Count(api, "public suspend fun echo(") != 1 || strings.Count(api, "public suspend fun echo2(") != 1 {
				t.Fatal("nullability doesn't change erasure", api)
			}
		}
	}
}
func TestRootNullableTemporalAliasStructuralControl(t *testing.T) {
	var doc map[string]any
	if e := json.Unmarshal(nullableTemporalInput(t), &doc); e != nil {
		t.Fatal(e)
	}
	catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
	intent := map[string]bool{"AInherited": true, "BNullable": true, "ATrue": true, "AFalse": false, "LNullable": true, "LInherited": true, "LFalse": false, "LTrue": true, "NativeMaybe": true}
	for n := range intent {
		delete(catalog, n)
	}
	var visit func(any)
	visit = func(x any) {
		switch v := x.(type) {
		case map[string]any:
			if ref, ok := v["$ref"].(string); ok {
				if flag, ok := intent[strings.TrimPrefix(ref, "#/components/schemas/")]; ok {
					if ref == "#/components/schemas/NativeMaybe" {
						v["$ref"] = "#/components/schemas/ZNative"
					} else if strings.HasPrefix(ref, "#/components/schemas/L") {
						v["$ref"] = "#/components/schemas/ZDay"
					} else {
						v["$ref"] = "#/components/schemas/ZStamp"
					}
					if _, has := v["nullable"]; !has {
						v["nullable"] = flag
					}
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
			t.Fatal("whole existing target/nullability control", flag, e)
		}
	}
}

func TestRootNullableTemporalAliasBudgetsAndNoDrop(t *testing.T) {
	for _, terminal := range []string{`{"type":"string","format":"date-time"}`, `{"type":"string","format":"date"}`, `{"x-native-type":"date"}`, `{"type":"string","x-native-type":"date"}`} {
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
		b := schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","format":"date-time"},"Holder":{"properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(required, ",") + `]}}`)
		r, e := core.GenerateWithDiagnostics(request(b, "Dto"))
		if n == 240 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
			t.Fatal(n, r, e)
		}
	}
	for _, terminal := range []string{`{"type":"string","format":"date","enum":["ready"]}`, `{"type":"string","format":"date-time","const":"2024-02-29T12:34:56Z"}`} {
		aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":`+terminal+`}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/A")
	}
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","format":true}}`), "E_SCHEMA_INVALID", "/components/schemas/Z")
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/B","nullable":true},"B":{"$ref":"#/components/schemas/A","nullable":false}}`), "E_REF_CYCLE", "/components/schemas/A/$ref")
	base := `"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","format":"date-time"}`
	aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/A"},{"type":"integer"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0")
	aliasFailure(t, schemas(`{`+base+`,"C":{"allOf":[{"$ref":"#/components/schemas/A"},{"properties":{"id":{"type":"string"}}}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/allOf/0/$ref")
	aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/A"}],"discriminator":{"propertyName":"tag"}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0/$ref")
	aliasFailure(t, []byte(`{"openapi":"3.1.0","components":{"schemas":{`+base+`}},"paths":{"/events":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"$ref":"#/components/schemas/A"}}}}}}}}}`), "E_SCHEMA_UNSUPPORTED", "/paths/~1events/get/responses/200/content/text~1event-stream/schema")
}
