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

func nullableBytesInput(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/root-nullable-bytes-aliases.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestRootNullableBytesAliasEnvelope(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(nullableBytesInput(t), suffix)
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
				if !strings.Contains(s, "= kotlin.ByteArray"+"?") {
					t.Fatal(s)
				}
			}
			if s := source(t, r, "AFalse"+suffix); !strings.Contains(s, "= kotlin.ByteArray") || strings.Contains(s, "?") {
				t.Fatal(s)
			}
			holder := source(t, r, "Holder"+suffix)
			for _, name := range []string{"inherited", "forcedTrue", "wrapper"} {
				if !strings.Contains(holder, "val "+name+": kotlin.ByteArray"+"? = null") {
					t.Fatal(holder)
				}
			}
			for _, name := range []string{"forcedFalse", "wrapperFalse"} {
				if !strings.Contains(holder, "val "+name+": kotlin.ByteArray"+",") {
					t.Fatal(holder)
				}
			}
			patch := source(t, r, "SubmitInput"+suffix)
			for _, name := range []string{"inherited", "forcedTrue", "wrapper"} {
				if !strings.Contains(patch, "val "+name+": org.example.models.models.Patch<kotlin.ByteArray"+">") {
					t.Fatal(patch)
				}
			}
			for _, name := range []string{"forcedFalse", "wrapperFalse"} {
				if !strings.Contains(patch, "val "+name+": kotlin.ByteArray"+"? = null") {
					t.Fatal(patch)
				}
			}
			if !strings.Contains(patch, "val requiredNull: kotlin.ByteArray"+"?,") {
				t.Fatal(patch)
			}
			api := source(t, r, "DefaultApi")
			if strings.Count(api, "public suspend fun echo(") != 1 || strings.Count(api, "public suspend fun echo2(") != 1 {
				t.Fatal("nullability doesn't change erasure", api)
			}
		}
	}
}
func TestRootNullableBytesAliasStructuralControl(t *testing.T) {
	var doc map[string]any
	if e := json.Unmarshal(nullableBytesInput(t), &doc); e != nil {
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
						v["$ref"] = "#/components/schemas/ZByteAgain"
					} else if strings.HasPrefix(ref, "#/components/schemas/L") {
						v["$ref"] = "#/components/schemas/ZBinary"
					} else {
						v["$ref"] = "#/components/schemas/ZBytes"
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
func TestRootNullableBytesAliasBudgetsAndNoDrop(t *testing.T) {
	for _, terminal := range []string{`{"type":"string","format":"byte"}`, `{"type":"string","format":"binary"}`, `{"x-native-type":"date"}`, `{"type":"string","x-native-type":"date"}`} {
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
		b := schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","format":"byte"},"Holder":{"properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(required, ",") + `]}}`)
		r, e := core.GenerateWithDiagnostics(request(b, "Dto"))
		if n == 240 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
			t.Fatal(n, r, e)
		}
	}
	for _, terminal := range []string{`{"type":"string","format":"binary","enum":["ready"]}`, `{"type":"string","format":"byte","const":"2024-02-29T12:34:56Z"}`} {
		pointer := "/components/schemas/A"
		if strings.Contains(terminal, `"const"`) {
			pointer = "/components/schemas/Z/const"
		}
		aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":`+terminal+`}`), "E_SCHEMA_UNSUPPORTED", pointer)
	}
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","format":true}}`), "E_SCHEMA_INVALID", "/components/schemas/Z")
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/B","nullable":true},"B":{"$ref":"#/components/schemas/A","nullable":false}}`), "E_REF_CYCLE", "/components/schemas/A/$ref")
	base := `"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","format":"byte"}`
	aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/A"},{"type":"integer"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0")
	aliasFailure(t, schemas(`{`+base+`,"C":{"allOf":[{"$ref":"#/components/schemas/A"},{"properties":{"id":{"type":"string"}}}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/allOf/0/$ref")
	aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/A"}],"discriminator":{"propertyName":"tag"}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0/$ref")
	aliasFailure(t, []byte(`{"openapi":"3.1.0","components":{"schemas":{`+base+`}},"paths":{"/events":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"$ref":"#/components/schemas/A"}}}}}}}}}`), "E_SCHEMA_UNSUPPORTED", "/paths/~1events/get/responses/200/content/text~1event-stream/schema")
}

func TestRootNullableBytesAliasMethodSlots(t *testing.T) {
	for _, stream := range []bool{false, true} {
		limit := 240
		if stream {
			limit--
		}
		for _, n := range []int{limit, limit + 1} {
			var params []string
			for i := range n {
				params = append(params, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":{"$ref":"#/components/schemas/A"}}`, i))
			}
			response := `{"204":{}}`
			if stream {
				response = `{"200":{"content":{"text/event-stream":{"schema":{"properties":{}}}}}}`
			}
			r, e := core.GenerateWithDiagnostics(request([]byte(`{"openapi":"3.1.0","components":{"schemas":{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","format":"byte"}}},"paths":{"/wide":{"get":{"parameters":[`+strings.Join(params, ",")+`],"responses":`+response+`}}}}`), "Dto"))
			if n == limit {
				if e != nil {
					t.Fatal(e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_OPERATION_LIMIT" || len(r.Artifacts.Files) != 0 {
				t.Fatal(n, r, e)
			}
		}
	}
}
func TestRootNullableBytesAliasExclusions(t *testing.T) {
	for _, body := range []string{`{"type":"string","format":"binary","enum":["any"]}`, `{"type":"string","format":"byte","x-kotlin-value-class":true}`} {
		b := schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":` + body + `}`)
		r, e := core.GenerateWithDiagnostics(request(b, "Dto"))
		if e == nil || len(r.Artifacts.Files) != 0 {
			t.Fatal(r, e)
		}
	}

}
