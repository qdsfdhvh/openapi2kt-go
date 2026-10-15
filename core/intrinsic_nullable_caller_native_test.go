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

func intrinsicCallerInput(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/intrinsic-nullable-caller-natives.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestIntrinsicNullableCallerNativeEnvelope(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(intrinsicCallerInput(t), suffix)
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(e)
			}
			again, e := core.GenerateWithDiagnostics(req)
			if e != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("determinism", e)
			}
			for name, typ := range map[string]string{"ZMoney": "org.example.scalarvalues.Money", "ZLong": "kotlin.Long", "ZBytes": "kotlin.ByteArray", "ZElement": "kotlinx.serialization.json.JsonElement", "ZObject": "kotlinx.serialization.json.JsonObject", "ZString": "kotlin.String", "ZInstant": "kotlin.time.Instant", "AInherited": "org.example.scalarvalues.Money"} {
				if s := source(t, r, name+suffix); !strings.Contains(s, "= "+typ+"?") {
					t.Fatal(s)
				}
			}
			s := source(t, r, "Holder"+suffix)
			for _, name := range []string{"directMoney", "directWrapped", "inherited"} {
				if !strings.Contains(s, "val "+name+": org.example.scalarvalues.Money? = null") {
					t.Fatal(s)
				}
			}
			if !strings.Contains(s, "val directFalse: org.example.scalarvalues.Money,") {
				t.Fatal(s)
			}
			p := source(t, r, "SubmitInput"+suffix)
			if !strings.Contains(p, "val directMoney: org.example.models.models.Patch<org.example.scalarvalues.Money>") {
				t.Fatal(p)
			}
		}
	}
}
func TestIntrinsicNullableCallerNativeStructuralControl(t *testing.T) {
	var d map[string]any
	if e := json.Unmarshal(intrinsicCallerInput(t), &d); e != nil {
		t.Fatal(e)
	}
	c := d["components"].(map[string]any)["schemas"].(map[string]any)
	intrinsic := map[string]bool{}
	for n, v := range c {
		m := v.(map[string]any)
		if _, ok := m["x-native-type"]; ok {
			intrinsic[n] = true
			delete(m, "nullable")
			if a, ok := m["type"].([]any); ok {
				m["type"] = a[0]
			}
		}
	}
	for _, n := range []string{"BNullable", "LMaybe", "BytesMaybe", "ElementMaybe", "ObjectMaybe", "StringMaybe", "InstantMaybe"} {
		c[n].(map[string]any)["nullable"] = true
	}
	var walk func(any)
	walk = func(x any) {
		switch v := x.(type) {
		case map[string]any:
			if ref, ok := v["$ref"].(string); ok && intrinsic[strings.TrimPrefix(ref, "#/components/schemas/")] {
				if _, exists := v["nullable"]; !exists {
					v["nullable"] = true
				}
			}
			for _, child := range v {
				walk(child)
			}
		case []any:
			for _, child := range v {
				walk(child)
			}
		}
	}
	walk(d)
	b, _ := json.Marshal(d)
	for _, flag := range []bool{false, true} {
		req := request(b, "Dto")
		req.Options.Validation = flag
		if _, e := core.GenerateWithDiagnostics(req); e != nil {
			t.Fatal("whole control", flag, e)
		}
	}
}
func TestIntrinsicNullableCallerNativeSourceBudgets(t *testing.T) {
	for _, terminal := range []string{`{"x-native-type":"org.example.scalarvalues.Money","nullable":true}`, `{"type":["string","null"],"x-native-type":"org.example.scalarvalues.Money"}`, `{"type":["integer","null"],"x-native-type":"kotlin.Long"}`} {
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
					req := request(rootSpecialChain(n, terminal, consumer), "Dto")
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
}
func TestIntrinsicNullableCallerNativeNoDrop(t *testing.T) {
	for _, terminal := range []string{`{"x-native-type":"org.example.Money","nullable":true}`, `{"type":["string","null"],"x-native-type":"kotlinx.serialization.json.JsonElement"}`} {
		base := `"A":{"$ref":"#/components/schemas/Z","nullable":false},"Z":` + terminal
		for _, target := range []string{"A", "Z"} {
			aliasFailure(t, schemas(`{`+base+`,"C":{"allOf":[{"$ref":"#/components/schemas/`+target+`"},{"properties":{"id":{"type":"string"}}}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/allOf/0/$ref")
			aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/`+target+`"},{"type":"integer"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0")
			aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/`+target+`"}],"discriminator":{"propertyName":"tag"}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0/$ref")
			aliasFailure(t, schemas(`{`+base+`,"Leaf":{"properties":{"id":{"type":"string"}}},"C":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"tag","mapping":{"bad":"#/components/schemas/`+target+`"}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/discriminator/mapping/bad")
			aliasFailure(t, []byte(`{"openapi":"3.1.0","components":{"schemas":{`+base+`}},"paths":{"/events":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"$ref":"#/components/schemas/`+target+`"}}}}}}}}}`), "E_SCHEMA_UNSUPPORTED", "/paths/~1events/get/responses/200/content/text~1event-stream/schema")
		}
	}
}
