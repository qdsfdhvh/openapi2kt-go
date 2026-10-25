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

func nullableObjectInput(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/root-nullable-object-aliases.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestRootNullableObjectAliasEnvelope(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(nullableObjectInput(t), suffix)
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
				if !strings.Contains(s, "= org.example.models.models.ZRecord"+suffix+"?") {
					t.Fatal(s)
				}
			}
			if s := source(t, r, "AFalse"+suffix); !strings.Contains(s, "= org.example.models.models.ZRecord"+suffix) || strings.Contains(s, "?") {
				t.Fatal(s)
			}
			holder := source(t, r, "Holder"+suffix)
			for _, name := range []string{"inherited", "forcedTrue", "wrapper"} {
				if !strings.Contains(holder, "val "+name+": org.example.models.models.ZRecord"+suffix+"? = null") {
					t.Fatal(holder)
				}
			}
			for _, name := range []string{"forcedFalse", "wrapperFalse"} {
				if !strings.Contains(holder, "val "+name+": org.example.models.models.ZRecord"+suffix+",") {
					t.Fatal(holder)
				}
			}
			patch := source(t, r, "SubmitInput"+suffix)
			for _, name := range []string{"inherited", "forcedTrue", "wrapper"} {
				if !strings.Contains(patch, "val "+name+": org.example.models.models.Patch<org.example.models.models.ZRecord"+suffix+">") {
					t.Fatal(patch)
				}
			}
			for _, name := range []string{"forcedFalse", "wrapperFalse"} {
				if !strings.Contains(patch, "val "+name+": org.example.models.models.ZRecord"+suffix+"? = null") {
					t.Fatal(patch)
				}
			}
			if !strings.Contains(patch, "val requiredNull: org.example.models.models.ZRecord"+suffix+"?,") {
				t.Fatal(patch)
			}
			api := source(t, r, "DefaultApi")
			if strings.Count(api, "public suspend fun echo(") != 1 || strings.Count(api, "public suspend fun echo2(") != 1 {
				t.Fatal("nullability doesn't change erasure", api)
			}
		}
	}
}
func TestRootNullableObjectAliasStructuralControl(t *testing.T) {
	var doc map[string]any
	if e := json.Unmarshal(nullableObjectInput(t), &doc); e != nil {
		t.Fatal(e)
	}
	catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
	intent := map[string]bool{"AInherited": true, "BNullable": true, "ATrue": true, "AFalse": false}
	for n := range intent {
		delete(catalog, n)
	}
	var visit func(any)
	visit = func(x any) {
		switch v := x.(type) {
		case map[string]any:
			if ref, ok := v["$ref"].(string); ok {
				if flag, ok := intent[strings.TrimPrefix(ref, "#/components/schemas/")]; ok {
					v["$ref"] = "#/components/schemas/ZRecord"
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

func TestRootNullableObjectAliasSourceSlotsAndGuards(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, mode := range []string{"root", "inherit", "false"} {
			limit := 48
			if mode != "root" {
				limit = 46
			}
			for _, n := range []int{limit, limit + 1} {
				consumer := ""
				if mode != "root" {
					override := ""
					if mode == "false" {
						override = `,"nullable":false`
					}
					consumer = fmt.Sprintf(`{"properties":{"v":{"$ref":"#/components/schemas/A%02d"%s}}}`, n-1, override)
				}
				b := rootSpecialChain(n, `{"properties":{"id":{"type":"string"}}}`, consumer)
				b = []byte(strings.Replace(string(b), `"A00":{"$ref":"#/components/schemas/Z"}`, `"A00":{"$ref":"#/components/schemas/Z","nullable":true}`, 1))
				req := request(b, "Dto")
				req.Options.Validation = flag
				r, e := core.GenerateWithDiagnostics(req)
				if n == limit {
					if e != nil {
						t.Fatal(flag, mode, n, e)
					}
				} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0 {
					t.Fatal(flag, mode, n, r, e)
				}
			}
		}
		for _, count := range []int{240, 241} {
			var fields []string
			for i := range count {
				override := ""
				if i%2 != 0 {
					override = `,"nullable":false`
				}
				fields = append(fields, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/A"%s}`, i, override))
			}
			req := request(schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"properties":{"id":{"type":"string"}}},"Holder":{"properties":{`+strings.Join(fields, ",")+`}}}`), "")
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if count == 240 {
				if e != nil {
					t.Fatal(e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
				t.Fatal(r, e)
			}
		}
	}
	// Ordinary builtin Long nullable aliases graduated independently.
	for _, terminal := range []string{`{"type":"string","const":"ok"}`} {
		t.Run(terminal, func(t *testing.T) {
			pointer := "/components/schemas/A"
			aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":`+terminal+`}`), "E_SCHEMA_UNSUPPORTED", pointer)
		})
	}
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/B","nullable":true},"B":{"$ref":"#/components/schemas/A","nullable":false}}`), "E_REF_CYCLE", "/components/schemas/A/$ref")
	report(t, schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"properties":{"id":{"type":"string"}}},"C":{"allOf":[{"$ref":"#/components/schemas/A"},{"properties":{"other":{"type":"boolean"}}}]}}`), "Dto")
	r := report(t, schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"B":{"$ref":"#/components/schemas/A","nullable":false},"Z":{"allOf":[{"properties":{"id":{"type":"string"}}},{"properties":{"ok":{"type":"boolean"}}}]},"C":{"allOf":[{"$ref":"#/components/schemas/B"},{"properties":{"other":{"type":"boolean"}}}]}}`), "Dto")
	if !strings.Contains(source(t, r, "ADto"), "ZDto?") || !strings.Contains(source(t, r, "CDto"), "val other") {
		t.Fatal(r)
	}
}

func TestRootNullableObjectAliasNoNullableShapeErasure(t *testing.T) {
	base := `"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"properties":{"id":{"type":"string"}}},"Other":{"properties":{"ok":{"type":"boolean"}}}`
	// Original nullable alias member now contributes a union Null carrier;
	// its real DTO source and ordinary allOf exclusion remain distinct.
	accepted := report(t, schemas(`{`+base+`,"Choice":{"oneOf":[{"$ref":"#/components/schemas/A"},{"$ref":"#/components/schemas/Other"}]}}`), "Dto")
	if text := source(t, accepted, "ChoiceDto"); !strings.Contains(text, "NullChoiceDto") || !strings.Contains(text, "public val id: kotlin.String? = null,") {
		t.Fatal("original nullable alias member lost", text)
	}
	// Preserve the historical nullable member/mapping inputs. Their target
	// now owns a physical hierarchy leaf; ordinary composition stays excluded.
	for _, input := range [][]byte{
		schemas(`{` + base + `,"Choice":{"oneOf":[{"$ref":"#/components/schemas/A"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag"}}}`),
		schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Parent":{"oneOf":[{"$ref":"#/components/schemas/Z"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag","mapping":{"leaf":"A"}}},"Z":{"properties":{"tag":{"type":"string"},"id":{"type":"string"}}},"Other":{"properties":{"tag":{"type":"string"},"ok":{"type":"boolean"}}}}`),
	} {
		for _, validation := range []bool{false, true} {
			req := request(input, "Dto")
			req.Options.Validation = validation
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil || !strings.Contains(source(t, r, "ADto"), "= org.example.models.models.ZDto?\n") || !strings.Contains(source(t, r, "ZDto"), "public data class ZDto(") {
				t.Fatal(validation, e)
			}
		}
	}
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"A":{"$ref":"#/components/schemas/Event","nullable":true},"Event":{"properties":{"event":{"type":"string","const":"value"},"data":{"type":"string"}},"required":["data"]}}},"paths":{"/events":{"get":{"operationId":"watch","responses":{"200":{"content":{"text/event-stream":{"schema":{"$ref":"#/components/schemas/A"}}}}}}}}}`)
	aliasFailure(t, input, "E_SCHEMA_UNSUPPORTED", "/paths/~1events/get/responses/200/content/text~1event-stream/schema")
}
