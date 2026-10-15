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

func intrinsicNullableBigintInput(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/intrinsic-nullable-bigints.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestIntrinsicNullableBigintEnvelope(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(intrinsicNullableBigintInput(t), suffix)
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(flag, suffix, e)
			}
			again, e := core.GenerateWithDiagnostics(req)
			if e != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("determinism", e)
			}
			for _, name := range []string{"ZAmount", "ZOmitted", "ZInteger"} {
				if root := source(t, r, name+suffix); !strings.Contains(root, "public typealias "+name+suffix+" = kotlin.Long?") {
					t.Fatal("publicrootnullable", root)
				}
			}
			for _, n := range []string{"AInherited", "BNullable", "ATrue"} {
				s := source(t, r, n+suffix)
				if !strings.Contains(s, "= kotlin.Long"+"?") {
					t.Fatal(s)
				}
			}
			if s := source(t, r, "AFalse"+suffix); !strings.Contains(s, "= kotlin.Long") || strings.Contains(s, "?") {
				t.Fatal(s)
			}
			annotated := "@kotlinx.serialization.Serializable(with = org.example.models.models.LongAsStringSerializer::class) kotlin.Long"
			holder := source(t, r, "Holder"+suffix)
			if strings.Count(holder, "LongAsStringSerializer::class") != 15 || !strings.Contains(holder, "List<kotlin.Long?>") || !strings.Contains(holder, "Map<kotlin.String, kotlin.Long?>") {
				t.Fatal("field vs list/map codec", holder)
			}
			for _, name := range []string{"inherited", "forcedTrue", "wrapper", "direct", "directWrapper", "directOmitted"} {
				if !strings.Contains(holder, "val "+name+": "+annotated+"? = null") {
					t.Fatal(holder)
				}
			}
			for _, name := range []string{"forcedFalse", "wrapperFalse", "directFalse", "directWrapperFalse", "directOmittedFalse"} {
				if !strings.Contains(holder, "val "+name+": "+annotated+",") {
					t.Fatal(holder)
				}
			}
			patch := source(t, r, "SubmitInput"+suffix)
			for _, name := range []string{"inherited", "forcedTrue", "wrapper", "direct", "directWrapper", "directOmitted"} {
				if !strings.Contains(patch, "val "+name+": org.example.models.models.Patch<"+annotated+">") {
					t.Fatal(patch)
				}
			}
			for _, name := range []string{"forcedFalse", "wrapperFalse", "directFalse", "directWrapperFalse", "directOmittedFalse"} {
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
func TestIntrinsicNullableBigintCanonicalControl(t *testing.T) {
	var doc map[string]any
	if e := json.Unmarshal(intrinsicNullableBigintInput(t), &doc); e != nil {
		t.Fatal(e)
	}
	catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
	catalog["ZAmount"].(map[string]any)["type"] = "string"
	catalog["ZOmitted"].(map[string]any)["nullable"] = false
	catalog["ZInteger"].(map[string]any)["type"] = "integer"
	catalog["BNullable"].(map[string]any)["nullable"] = true
	var visit func(any)
	visit = func(x any) {
		switch v := x.(type) {
		case map[string]any:
			if ref, ok := v["$ref"].(string); ok && (ref == "#/components/schemas/ZAmount" || ref == "#/components/schemas/ZOmitted" || ref == "#/components/schemas/ZInteger") {
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
func TestIntrinsicNullableBigintBudgetsAndNoDrop(t *testing.T) {
	for _, terminal := range []string{`{"type":"string","nullable":true,"x-native-type":"bigint"}`, `{"nullable":true,"x-native-type":"bigint"}`, `{"type":"integer","nullable":true,"x-native-type":"bigint"}`} {
		for _, flag := range []bool{false, true} {
			req := request(schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":`+terminal+`}`), "Dto")
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil || !strings.Contains(source(t, r, "ADto"), "kotlin.Long?") {
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
				b := rootSpecialChain(n, `{"type":"string","nullable":true,"x-native-type":"bigint"}`, consumer)
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
	for _, flag := range []bool{false, true} {
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
				b := schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","nullable":true,"x-native-type":"bigint"},"Holder":{"properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(required, ",") + `]}}`)
				req := request(b, "Dto")
				req.Options.Validation = flag
				r, e := core.GenerateWithDiagnostics(req)
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
	base := `"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","nullable":true,"x-native-type":"bigint"}`
	aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/A"},{"type":"integer"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0")
	aliasFailure(t, schemas(`{`+base+`,"C":{"allOf":[{"$ref":"#/components/schemas/A"},{"properties":{"id":{"type":"string"}}}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/allOf/0/$ref")
	aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/A"}],"discriminator":{"propertyName":"tag"}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0/$ref")
	aliasFailure(t, []byte(`{"openapi":"3.1.0","components":{"schemas":{`+base+`}},"paths":{"/events":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"$ref":"#/components/schemas/A"}}}}}}}}}`), "E_SCHEMA_UNSUPPORTED", "/paths/~1events/get/responses/200/content/text~1event-stream/schema")

	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/B","nullable":true},"B":{"$ref":"#/components/schemas/A","nullable":false}}`), "E_REF_CYCLE", "/components/schemas/A/$ref")
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","x-native-type":true}}`), "E_SCHEMA_INVALID", "/components/schemas/Z/x-native-type")
}

func TestIntrinsicNullableBigintMethodSlots(t *testing.T) {
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
				b := []byte(`{"openapi":"3.1.0","components":{"schemas":{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","nullable":true,"x-native-type":"bigint"}}},"paths":{"/wide":{"get":{"parameters":[` + strings.Join(params, ",") + `],"responses":` + response + `}}}}`)
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

func TestIntrinsicNullableBigintBorrowingNoDrop(t *testing.T) {
	base := `"Z":{"type":"string","x-native-type":"bigint","nullable":true},"A":{"$ref":"#/components/schemas/Z","nullable":false}`
	for _, ref := range []string{"Z", "A"} {
		aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/`+ref+`"},{"type":"integer"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0")
		aliasFailure(t, schemas(`{`+base+`,"C":{"allOf":[{"$ref":"#/components/schemas/`+ref+`"},{"properties":{"id":{"type":"string"}}}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/allOf/0/$ref")
		aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/`+ref+`"}],"discriminator":{"propertyName":"tag"}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0/$ref")
		aliasFailure(t, []byte(`{"openapi":"3.1.0","components":{"schemas":{`+base+`}},"paths":{"/events":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"$ref":"#/components/schemas/`+ref+`"}}}}}}}}}`), "E_SCHEMA_UNSUPPORTED", "/paths/~1events/get/responses/200/content/text~1event-stream/schema")
		aliasFailure(t, schemas(`{`+base+`,"Parent":{"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag","mapping":{"x":"`+ref+`"}}},"First":{"properties":{"tag":{"type":"string"}}},"Other":{"properties":{"tag":{"type":"string"}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Parent/discriminator/mapping/x")
	}
	// Plain nullable String roots graduate under intrinsic primitive contracts.
}
