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

func nullableRawInput(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/root-nullable-raw-aliases.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestRootNullableRawAliasEnvelope(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(nullableRawInput(t), suffix)
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(e)
			}
			again, e := core.GenerateWithDiagnostics(req)
			if e != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("determinism", e)
			}
			for _, n := range []string{"AInherited", "BNullable", "ATrue"} {
				if s := source(t, r, n+suffix); !strings.Contains(s, "= org.example.models.models.ZRaw"+suffix+"?") {
					t.Fatal(s)
				}
			}
			if s := source(t, r, "AFalse"+suffix); strings.Contains(s, "?") {
				t.Fatal(s)
			}
			if s := source(t, r, "ZRaw"+suffix); !strings.Contains(s, "= kotlinx.serialization.json.JsonElement") || strings.Contains(s, "data class") {
				t.Fatal(s)
			}
			s := source(t, r, "Holder"+suffix)
			if !strings.Contains(s, "val inherited: org.example.models.models.ZRaw"+suffix+"? = null") {
				t.Fatal(s)
			}
			p := source(t, r, "SubmitInput"+suffix)
			if !strings.Contains(p, "Patch<org.example.models.models.ZRaw"+suffix+">") {
				t.Fatal(p)
			}
			api := source(t, r, "DefaultApi")
			if strings.Count(api, "public suspend fun echo(") != 1 || strings.Count(api, "public suspend fun echo2(") != 1 {
				t.Fatal(api)
			}
			count := 0
			for _, w := range r.Warnings {
				if w.Code == "W_JSON_ELEMENT_APPROXIMATION" {
					count++
					if *w.Pointer != "/components/schemas/ZRaw" && *w.Pointer != "/components/schemas/ZEmpty" {
						t.Fatal(w)
					}
				}
			}
			if count != 2 {
				t.Fatal(r.Warnings)
			}
		}
	}
}
func TestRootNullableRawAliasStructuralControl(t *testing.T) {
	var d map[string]any
	if e := json.Unmarshal(nullableRawInput(t), &d); e != nil {
		t.Fatal(e)
	}
	catalog := d["components"].(map[string]any)["schemas"].(map[string]any)
	target := map[string]string{"AInherited": "ZRaw", "BNullable": "ZRaw", "ATrue": "ZRaw", "AFalse": "ZRaw", "CEmpty": "ZEmpty"}
	for n := range target {
		delete(catalog, n)
	}
	var walk func(any)
	walk = func(x any) {
		switch v := x.(type) {
		case map[string]any:
			if ref, ok := v["$ref"].(string); ok {
				if n, ok := target[strings.TrimPrefix(ref, "#/components/schemas/")]; ok {
					v["$ref"] = "#/components/schemas/" + n
					if _, exists := v["nullable"]; !exists {
						v["nullable"] = ref != "#/components/schemas/AFalse"
					}
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
			t.Fatal("whole control", e)
		}
	}
}
func TestRootNullableRawAliasDepth(t *testing.T) {
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
				b := rootSpecialChain(n, `{"type":"object"}`, consumer)
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
func TestRootNullableRawAliasNoDrop(t *testing.T) {
	base := `"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"object"}`
	aliasFailure(t, schemas(`{`+base+`,"C":{"allOf":[{"$ref":"#/components/schemas/A"},{"properties":{"id":{"type":"string"}}}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/allOf/0/$ref")
	aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/A"},{"type":"integer"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0")
	// The original nullable member has a real mapped singleton declaration.
	// Its alias remains nullable while the physical codec rejects null.
	input := schemas(`{` + base + `,"C":{"oneOf":[{"$ref":"#/components/schemas/A"}],"discriminator":{"propertyName":"tag"}}}`)
	for _, validation := range []bool{false, true} {
		req := request(input, "Dto")
		req.Options.Validation = validation
		r, e := core.GenerateWithDiagnostics(req)
		if e != nil || !strings.Contains(source(t, r, "ADto"), "= org.example.models.models.ZDto?\n") || !strings.Contains(source(t, r, "ZDto"), "public data object ZDto : org.example.models.models.CDto() {") || !strings.Contains(source(t, r, "ZDto"), "if (element == kotlinx.serialization.json.JsonNull)") {
			t.Fatal(validation, e)
		}
	}
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"allOf":[{"type":"object"}],"nullable":true}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Z")
}
