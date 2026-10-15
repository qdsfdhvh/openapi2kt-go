package core_test

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func rootSpecialInput(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/root-special-aliases.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}

func TestRootSpecialAliasEnvelope(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(rootSpecialInput(t), suffix)
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal("special roots", flag, suffix, e)
			}
			again, e := core.GenerateWithDiagnostics(req)
			if e != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("determinism", e)
			}
			for _, pair := range [][2]string{{"AStatus", "ZStatus"}, {"BStatus", "ZStatus"}, {"AId", "ZId"}, {"ALongId", "ZLongId"}} {
				if !strings.Contains(source(t, r, pair[0]+suffix), "public typealias "+pair[0]+suffix+" = org.example.models.models."+pair[1]+suffix) {
					t.Fatal(source(t, r, pair[0]+suffix))
				}
			}
			holder := source(t, r, "Holder"+suffix)
			for _, text := range []string{"pin: kotlin.Int = 7", "LongAsStringSerializer", "date: kotlinx.datetime.LocalDate", "bytes: kotlin.ByteArray", "money: org.example.scalarvalues.Money"} {
				if !strings.Contains(holder, text) {
					t.Fatal(text, holder)
				}
			}
			api := source(t, r, "DefaultApi")
			if strings.Count(api, "public suspend fun echo(") != 1 || strings.Count(api, "public suspend fun echo2(") != 1 {
				t.Fatal("canonical wrapper erasure", api)
			}
			patch := source(t, r, "SubmitInput"+suffix)
			if !strings.Contains(patch, "Patch<org.example.models.models.ZId"+suffix+">") || !strings.Contains(patch, "LongAsStringSerializer") || !strings.Contains(patch, "pin: kotlin.Int = 7") {
				t.Fatal(patch)
			}
		}
	}
}

func TestRootSpecialAliasLimitsAndSources(t *testing.T) {
	terminals := []string{`{"type":"integer","const":7}`, `{"type":"string","enum":["ready"]}`, `{"type":"integer","format":"int64","x-kotlin-value-class":true}`, `{"type":"string","x-native-type":"bigint"}`}
	for _, terminal := range terminals {
		for _, flag := range []bool{false, true} {
			for _, n := range []int{48, 49} {
				req := request(rootSpecialChain(n, terminal, ""), "")
				req.Options.Validation = flag
				r, e := core.GenerateWithDiagnostics(req)
				if n == 48 {
					if e != nil {
						t.Fatal(n, terminal, e)
					}
				} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0 {
					t.Fatal(n, terminal, r, e)
				}
			}
			for _, n := range []int{46, 47} {
				holder := fmt.Sprintf(`{"properties":{"v":{"$ref":"#/components/schemas/A%02d"}}}`, n-1)
				req := request(rootSpecialChain(n, terminal, holder), "")
				req.Options.Validation = flag
				r, e := core.GenerateWithDiagnostics(req)
				if n == 46 {
					if e != nil {
						t.Fatal(n, terminal, e)
					}
				} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || *e.(core.Diagnostics)[0].Pointer != "/components/schemas/Holder/properties/v" || len(r.Artifacts.Files) != 0 {
					t.Fatal(n, terminal, r, e)
				}
			}
		}
	}
	for _, n := range []int{120, 121} {
		var props []string
		var required []string
		for i := range n {
			props = append(props, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/A"}`, i))
			required = append(required, fmt.Sprintf(`"p%d"`, i))
		}
		doc := schemas(`{"A":{"$ref":"#/components/schemas/Z"},"Z":{"type":"integer","format":"int64","x-kotlin-value-class":true},"Holder":{"properties":{` + strings.Join(props, ",") + `},"required":[` + strings.Join(required, ",") + `]}}`)
		req := request(doc, "")
		r, e := core.GenerateWithDiagnostics(req)
		if n == 120 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
			t.Fatal(n, r, e)
		}
	}
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z"},"Z":{"type":"integer","enum":[1,1]}}`), "E_SCHEMA_INVALID", "/components/schemas/Z/enum/1")
	report(t, schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","enum":["ready"]}}`), "Dto") // Plain nullable enum bindings graduated independently.
	// Bare builtin alias remains the historical opt-in, not a broad admission.
	failure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z"},"Z":{"type":"string"}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/A")
}

func TestRootSpecialAliasStructuralControl(t *testing.T) {
	var doc map[string]any
	if e := json.Unmarshal(rootSpecialInput(t), &doc); e != nil {
		t.Fatal(e)
	}
	catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
	bindings := map[string]string{}
	var resolve func(string) string
	resolve = func(n string) string {
		v := catalog[n].(map[string]any)
		if ref, ok := v["$ref"].(string); ok {
			return resolve(strings.TrimPrefix(ref, "#/components/schemas/"))
		}
		return n
	}
	for n, v := range catalog {
		if _, ok := v.(map[string]any)["$ref"]; ok {
			bindings[n] = resolve(n)
		}
	}
	for n := range bindings {
		delete(catalog, n)
	}
	var visit func(any)
	visit = func(x any) {
		switch v := x.(type) {
		case map[string]any:
			if ref, ok := v["$ref"].(string); ok {
				if target, ok := bindings[strings.TrimPrefix(ref, "#/components/schemas/")]; ok {
					v["$ref"] = "#/components/schemas/" + target
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
	input, _ := json.Marshal(doc)
	for _, flag := range []bool{false, true} {
		req := request(input, "Dto")
		req.Options.Validation = flag
		if _, e := core.GenerateWithDiagnostics(req); e != nil {
			t.Fatal("whole control", flag, e)
		}
	}
}

func rootSpecialChain(n int, terminal, consumer string) []byte {
	entries := []string{`"Z":` + terminal}
	for i := range n {
		target := "Z"
		if i > 0 {
			target = fmt.Sprintf("A%02d", i-1)
		}
		entries = append(entries, fmt.Sprintf(`"A%02d":{"$ref":"#/components/schemas/%s"}`, i, target))
	}
	if consumer != "" {
		entries = append(entries, `"Holder":`+consumer)
	}
	return schemas("{" + strings.Join(entries, ",") + "}")
}
