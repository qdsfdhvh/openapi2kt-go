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

func intrinsicNullableInput(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/intrinsic-nullable-dtos.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestIntrinsicNullableDTOEnvelope(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(intrinsicNullableInput(t), suffix)
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(flag, suffix, e)
			}
			again, e := core.GenerateWithDiagnostics(req)
			if e != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("determinism", e)
			}
			z := source(t, r, "ZRecord"+suffix)
			if !strings.Contains(z, "public data class ZRecord"+suffix+"(") || strings.Contains(z, "typealias") || strings.Contains(z, "NonNull") {
				t.Fatal("physical DTO stays canonical", z)
			}
			for _, n := range []string{"AInherited", "BNullable", "ATrue"} {
				if s := source(t, r, n+suffix); !strings.Contains(s, "= org.example.models.models.ZRecord"+suffix+"?") {
					t.Fatal(s)
				}
			}
			holder := source(t, r, "Holder"+suffix)
			for _, n := range []string{"direct", "directWrapper"} {
				if !strings.Contains(holder, "val "+n+": org.example.models.models.ZRecord"+suffix+"? = null") {
					t.Fatal(holder)
				}
			}
			for _, n := range []string{"directFalse", "directWrapperFalse"} {
				if !strings.Contains(holder, "val "+n+": org.example.models.models.ZRecord"+suffix+",") {
					t.Fatal(holder)
				}
			}
			if !strings.Contains(holder, "val other: org.example.models.models.ZOther"+suffix+"? = null") || !strings.Contains(holder, "val otherFalse: org.example.models.models.ZOther"+suffix+",") {
				t.Fatal(holder)
			}
			patch := source(t, r, "SubmitInput"+suffix)
			for _, n := range []string{"direct", "directWrapper"} {
				if !strings.Contains(patch, "val "+n+": org.example.models.models.Patch<org.example.models.models.ZRecord"+suffix+">") {
					t.Fatal(patch)
				}
			}
			for _, n := range []string{"directFalse", "directWrapperFalse"} {
				if !strings.Contains(patch, "val "+n+": org.example.models.models.ZRecord"+suffix+"? = null") {
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
func TestIntrinsicNullableDTOCanonicalControl(t *testing.T) {
	var doc map[string]any
	if e := json.Unmarshal(intrinsicNullableInput(t), &doc); e != nil {
		t.Fatal(e)
	}
	catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
	catalog["ZRecord"].(map[string]any)["nullable"] = false
	catalog["ZOther"].(map[string]any)["type"] = "object"
	catalog["BNullable"].(map[string]any)["nullable"] = true
	var visit func(any)
	visit = func(x any) {
		switch v := x.(type) {
		case map[string]any:
			if ref, ok := v["$ref"].(string); ok && (ref == "#/components/schemas/ZRecord" || ref == "#/components/schemas/ZOther") {
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
func TestIntrinsicNullableDTOBoundsAndExclusions(t *testing.T) {
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
				req := request(rootSpecialChain(n, `{"nullable":true,"properties":{"id":{"type":"string"}},"required":["id"]}`, consumer), "Dto")
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
				fields = append(fields, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Z"}`, i))
			}
			req := request(schemas(`{"Z":{"nullable":true,"properties":{"id":{"type":"string"}}},"Holder":{"properties":{`+strings.Join(fields, ",")+`}}}`), "")
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
	// Direct empty raw roots graduated under their independent transparent profile.
	for _, body := range []string{`"type":"object","allOf":[{"type":"object"}]`, `"properties":{"pin":{"type":"integer","const":7}}`} {
		aliasFailure(t, schemas(`{"Z":{"nullable":true,`+body+`}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Z")
	}
	// Preserve the original ordinary allOf exclusion input as an explicit
	// graduation; actual constructor fields now authorize its nullable root.
	allOf := schemas(`{"Z":{"nullable":true,"allOf":[{"properties":{"id":{"type":"string"}}},{"properties":{"ok":{"type":"boolean"}}}]}}`)
	for _, validation := range []bool{false, true} {
		req := request(allOf, "Dto")
		req.Options.Validation = validation
		r, e := core.GenerateWithDiagnostics(req)
		if e != nil || !strings.Contains(source(t, r, "ZDto"), "public data class ZDto(") || !strings.Contains(source(t, r, "ZDto"), "public val ok: kotlin.Boolean? = null") {
			t.Fatal("original nullable allOf constructor input", validation, r, e)
		}
	}
	// Retain the exact historical inferred-object spelling: root nullable intent
	// is explicit, with direct properties and a real constructor. Composition
	// and allOf counts are separate exclusions.
	originalCount := schemas(`{"Z":{"nullable":true,"properties":{"id":{"type":"string"}},"minProperties":0}}`)
	if text := source(t, aggregationReport(t, originalCount), "Z"); !strings.Contains(text, "generatedSerializer()") || !strings.Contains(text, "element.size < 0") {
		t.Fatal("original direct nullable own count lost", text)
	}
	if _, err := core.Generate(request(originalCount, "")); err == nil {
		t.Fatal("default-off nullable counts admitted")
	}
}

func TestIntrinsicNullableDTONoShapeErasure(t *testing.T) {
	base := `"Z":{"nullable":true,"properties":{"id":{"type":"string"}}},"Other":{"properties":{"ok":{"type":"boolean"}}},"A":{"$ref":"#/components/schemas/Z","nullable":false}`
	for _, ref := range []string{"Z", "A"} {
		intersection := report(t, schemas(`{`+base+`,"C":{"allOf":[{"$ref":"#/components/schemas/`+ref+`","nullable":false},{"properties":{"more":{"type":"boolean"}}}]}}`), "Dto")
		if !strings.Contains(source(t, intersection, "CDto"), "public val more: kotlin.Boolean? = null") {
			t.Fatal("nullable DTO intersection lost its original fields")
		}
		original := schemas(`{` + base + `,"C":{"oneOf":[{"$ref":"#/components/schemas/` + ref + `"},{"$ref":"#/components/schemas/Other"}]}}`)
		accepted := report(t, original, "Dto")
		if text := source(t, accepted, "CDto"); strings.Contains(text, "NullCDto") != (ref == "Z") || !strings.Contains(text, "public val id: kotlin.String? = null,") {
			t.Fatal("original nullable member source/narrowing lost", text)
		}
		// Preserve the historical input: an actual named hierarchy leaf now
		// retains its physical constructor while inheriting nullable intent.
		input := schemas(`{` + base + `,"C":{"oneOf":[{"$ref":"#/components/schemas/` + ref + `"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag"}}}`)
		for _, validation := range []bool{false, true} {
			req := request(input, "Dto")
			req.Options.Validation = validation
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil || !strings.Contains(source(t, r, "CDto"), "public sealed class CDto {") || !strings.Contains(source(t, r, "ZDto"), ") : org.example.models.models.CDto() {") || !strings.Contains(source(t, r, "ZDto"), "public val id: kotlin.String? = null,") || !strings.Contains(source(t, r, "ADto"), "public typealias ADto = org.example.models.models.ZDto\n") {
				t.Fatal(validation, ref, r, e)
			}
		}
	}
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Event":{"nullable":true,"properties":{"event":{"type":"string","const":"value"},"data":{"type":"string"}},"required":["data"]}}},"paths":{"/events":{"get":{"operationId":"watch","responses":{"200":{"content":{"text/event-stream":{"schema":{"$ref":"#/components/schemas/Event"}}}}}}}}}`)
	aliasFailure(t, input, "E_SCHEMA_UNSUPPORTED", "/paths/~1events/get/responses/200/content/text~1event-stream/schema")
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":false},"Z":{"nullable":true,"properties":{"id":{"type":"string"}}},"Parent":{"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag","mapping":{"leaf":"A"}}},"First":{"properties":{"tag":{"type":"string"},"id":{"type":"string"}}},"Other":{"properties":{"tag":{"type":"string"},"ok":{"type":"boolean"}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Parent/discriminator/mapping/leaf")
}
func TestIntrinsicNullableDTOIndependentNominalInteriors(t *testing.T) {
	// Nullable nominal wrappers do not alter the independent checked-edge budget.
	for _, n := range []int{16, 17} {
		leaf := `{"type":"string","minLength":1}`
		for range n {
			leaf = `{"type":"array","items":` + leaf + `}`
		}
		input := schemas(`{"Z":{"nullable":true,"properties":{"checked":` + leaf + `,"child":{"$ref":"#/components/schemas/Z"}}},"A":{"$ref":"#/components/schemas/Z","nullable":false},"Holder":{"properties":{"narrow":{"$ref":"#/components/schemas/A"}}}}`)
		req := request(input, "Dto")
		req.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(req)
		if n == 16 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0 {
			t.Fatal(n, r, e)
		}
	}
}

func TestIntrinsicNullableDTOUnusedDirectMappingNoDrop(t *testing.T) {
	body := `{"Z":{"nullable":true,"properties":{"id":{"type":"string"}}},"Parent":{"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag","mapping":{"leaf":"Z"}}},"First":{"properties":{"tag":{"type":"string"},"id":{"type":"string"}}},"Other":{"properties":{"tag":{"type":"string"},"ok":{"type":"boolean"}}}}`
	control := schemas(strings.Replace(body, `"nullable":true`, `"nullable":false`, 1))
	for _, flag := range []bool{false, true} {
		req := request(control, "Dto")
		req.Options.Validation = flag
		if _, e := core.GenerateWithDiagnostics(req); e != nil {
			t.Fatal("existing extra direct mapping control", e)
		}
	}
	aliasFailure(t, schemas(body), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Parent/discriminator/mapping/leaf")
}
