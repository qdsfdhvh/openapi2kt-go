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

func intrinsicNullableBytesInput(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/intrinsic-nullable-bytes.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestIntrinsicNullableBytesEnvelope(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(intrinsicNullableBytesInput(t), suffix)
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(flag, suffix, e)
			}
			again, e := core.GenerateWithDiagnostics(req)
			if e != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("determinism", e)
			}
			for _, name := range []string{"ZBytes", "ZBinary", "ZByteAgain"} {
				if root := source(t, r, name+suffix); !strings.Contains(root, "public typealias "+name+suffix+" = kotlin.ByteArray?") {
					t.Fatal("publicrootnullable", root)
				}
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
			annotated := "kotlin.ByteArray"
			holder := source(t, r, "Holder"+suffix)
			if !strings.Contains(holder, "List<kotlin.ByteArray?>") || !strings.Contains(holder, "Map<kotlin.String, kotlin.ByteArray?>") {
				t.Fatal(holder)
			}

			for _, name := range []string{"inherited", "forcedTrue", "wrapper", "direct", "directWrapper", "directBinary"} {
				if !strings.Contains(holder, "val "+name+": "+annotated+"? = null") {
					t.Fatal(holder)
				}
			}
			for _, name := range []string{"forcedFalse", "wrapperFalse", "directFalse", "directWrapperFalse", "directBinaryFalse"} {
				if !strings.Contains(holder, "val "+name+": "+annotated+",") {
					t.Fatal(holder)
				}
			}
			patch := source(t, r, "SubmitInput"+suffix)
			for _, name := range []string{"inherited", "forcedTrue", "wrapper", "direct", "directWrapper", "directBinary"} {
				if !strings.Contains(patch, "val "+name+": org.example.models.models.Patch<"+annotated+">") {
					t.Fatal(patch)
				}
			}
			for _, name := range []string{"forcedFalse", "wrapperFalse", "directFalse", "directWrapperFalse", "directBinaryFalse"} {
				if !strings.Contains(patch, "val "+name+": "+annotated+"? = null") {
					t.Fatal(patch)
				}
			}
			if !strings.Contains(patch, "val requiredNull: "+annotated+"?,") {
				t.Fatal(patch)
			}
			api := source(t, r, "DefaultApi")
			if strings.Count(api, "public suspend fun echo(") != 1 || strings.Count(api, "public suspend fun echo2(") != 1 {
				t.Fatal("nullability doesn't change erasure", api)
			}
		}
	}
}
func TestIntrinsicNullableBytesCanonicalControl(t *testing.T) {
	var doc map[string]any
	if e := json.Unmarshal(intrinsicNullableBytesInput(t), &doc); e != nil {
		t.Fatal(e)
	}
	catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
	catalog["ZBytes"].(map[string]any)["nullable"] = false
	catalog["ZBinary"].(map[string]any)["type"] = "string"
	catalog["ZByteAgain"].(map[string]any)["nullable"] = false
	catalog["BNullable"].(map[string]any)["nullable"] = true
	var visit func(any)
	visit = func(x any) {
		switch v := x.(type) {
		case map[string]any:
			if ref, ok := v["$ref"].(string); ok && (ref == "#/components/schemas/ZBytes" || ref == "#/components/schemas/ZBinary" || ref == "#/components/schemas/ZByteAgain") {
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
func TestIntrinsicNullableBytesBudgetsAndNoDrop(t *testing.T) {
	for _, terminal := range []string{`{"type":"string","nullable":true,"format":"byte"}`, `{"type":"string","nullable":true,"format":"binary"}`, `{"x-native-type":"date"}`, `{"type":"string","x-native-type":"date"}`} {
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
		b := schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","nullable":true,"format":"byte"},"Holder":{"properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(required, ",") + `]}}`)
		r, e := core.GenerateWithDiagnostics(request(b, "Dto"))
		if n == 240 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
			t.Fatal(n, r, e)
		}
	}
	for _, terminal := range []string{`{"type":"string","nullable":true,"format":"binary","enum":["ready"]}`, `{"type":"string","nullable":true,"format":"byte","const":"2024-02-29T12:34:56Z"}`} {
		pointer := "/components/schemas/A"
		if strings.Contains(terminal, `"nullable":true`) {
			pointer = "/components/schemas/Z"
		}
		aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":`+terminal+`}`), "E_SCHEMA_UNSUPPORTED", pointer)
	}
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","format":true}}`), "E_SCHEMA_INVALID", "/components/schemas/Z")
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/B","nullable":true},"B":{"$ref":"#/components/schemas/A","nullable":false}}`), "E_REF_CYCLE", "/components/schemas/A/$ref")
	base := `"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","nullable":true,"format":"byte"}`
	aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/A"},{"type":"integer"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0")
	aliasFailure(t, schemas(`{`+base+`,"C":{"allOf":[{"$ref":"#/components/schemas/A"},{"properties":{"id":{"type":"string"}}}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/allOf/0/$ref")
	aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/A"}],"discriminator":{"propertyName":"tag"}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0/$ref")
	aliasFailure(t, []byte(`{"openapi":"3.1.0","components":{"schemas":{`+base+`}},"paths":{"/events":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"$ref":"#/components/schemas/A"}}}}}}}}}`), "E_SCHEMA_UNSUPPORTED", "/paths/~1events/get/responses/200/content/text~1event-stream/schema")
}

func TestIntrinsicNullableBytesMethodSlots(t *testing.T) {
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
			r, e := core.GenerateWithDiagnostics(request([]byte(`{"openapi":"3.1.0","components":{"schemas":{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","nullable":true,"format":"byte"}}},"paths":{"/wide":{"get":{"parameters":[`+strings.Join(params, ",")+`],"responses":`+response+`}}}}`), "Dto"))
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
func TestIntrinsicNullableBytesBorrowingNoDrop(t *testing.T) {
	base := `"Z":{"type":"string","format":"byte","nullable":true},"A":{"$ref":"#/components/schemas/Z","nullable":false}`
	for _, ref := range []string{"Z", "A"} {
		aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/`+ref+`"},{"type":"integer"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0")
		aliasFailure(t, schemas(`{`+base+`,"C":{"allOf":[{"$ref":"#/components/schemas/`+ref+`"},{"properties":{"id":{"type":"string"}}}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/allOf/0/$ref")
		aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/`+ref+`"}],"discriminator":{"propertyName":"tag"}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0/$ref")
		aliasFailure(t, []byte(`{"openapi":"3.1.0","components":{"schemas":{`+base+`}},"paths":{"/events":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"$ref":"#/components/schemas/`+ref+`"}}}}}}}}}`), "E_SCHEMA_UNSUPPORTED", "/paths/~1events/get/responses/200/content/text~1event-stream/schema")
		aliasFailure(t, schemas(`{`+base+`,"Parent":{"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag","mapping":{"x":"`+ref+`"}}},"First":{"properties":{"tag":{"type":"string"}}},"Other":{"properties":{"tag":{"type":"string"}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Parent/discriminator/mapping/x")
	}
	// Plain nullable String roots graduate under intrinsic primitive contracts.
}

func TestIntrinsicNullableBytesProfileExclusions(t *testing.T) {
	for _, body := range []string{`{"type":"string","nullable":true,"format":"byte","enum":["x"]}`, `{"type":"string","nullable":true,"format":"binary","const":"x"}`, `{"type":["string","null"],"format":"byte","x-kotlin-value-class":false}`} {
		aliasFailure(t, schemas(`{"Z":`+body+`}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Z")
	}
}
