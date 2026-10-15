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

func TestRootObjectAliasEnvelope(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/root-object-aliases.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(input, suffix)
			req.Options.Validation = flag
			r, err := core.GenerateWithDiagnostics(req)
			if err != nil {
				t.Fatalf("root DTO aliases should generate: validation=%t suffix=%q: %v", flag, suffix, err)
			}
			again, err := core.GenerateWithDiagnostics(req)
			if err != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("root alias determinism", err)
			}
			for _, name := range []string{"ARecord", "BRecord"} {
				text := source(t, r, name+suffix)
				if !strings.Contains(text, "public typealias "+name+suffix+" = org.example.models.models.ZRecord"+suffix) || strings.Contains(text, "data class") {
					t.Fatal(text)
				}
			}
			if !strings.Contains(source(t, r, "ZRecord"+suffix), "child: org.example.models.models.ZRecord"+suffix+"? = null") {
				t.Fatal("nominal recursion must terminate at original DTO")
			}
			api := source(t, r, "DefaultApi")
			if strings.Count(api, "public suspend fun echo(") != 1 || strings.Count(api, "public suspend fun echo2(") != 1 {
				t.Fatal("alias/direct target share nominal erasure", api)
			}
			if !strings.Contains(source(t, r, "SubmitInput"+suffix), "Patch<org.example.models.models.ZRecord"+suffix+">") {
				t.Fatal("request presence lost nominal target")
			}
		}
	}
}

func rootObjectChain(n int, consumer string) []byte {
	parts := []string{`"ZRecord":{"properties":{"id":{"type":"string"}},"required":["id"]}`}
	for i := range n {
		target := "ZRecord"
		if i > 0 {
			target = fmt.Sprintf("A%02d", i-1)
		}
		parts = append(parts, fmt.Sprintf(`"A%02d":{"$ref":"#/components/schemas/%s"}`, i, target))
	}
	if consumer != "" {
		parts = append(parts, `"Consumer":`+consumer)
	}
	return schemas("{" + strings.Join(parts, ",") + "}")
}

// Empty ordinary projections are transparent JsonElement, never nominal DTOs.
func TestRootObjectAliasRawTargetsStayTransparent(t *testing.T) {
	for _, input := range []string{
		`{"Alias":{"$ref":"#/components/schemas/ZRecord"},"ZRecord":{"type":"object"}}`,
		`{"Alias":{"$ref":"#/components/schemas/ZRecord"},"ZRecord":{"allOf":[{"type":"object"},{"type":"object"}]}}`,
	} {
		for _, flag := range []bool{false, true} {
			req := request(schemas(input), "")
			req.Options.Validation = flag
			r, err := core.GenerateWithDiagnostics(req)
			if err != nil {
				t.Fatal(flag, err)
			}
			if s := source(t, r, "ZRecord"); !strings.Contains(s, "= kotlinx.serialization.json.JsonElement") {
				t.Fatal("raw target invented a nominal DTO", s)
			}
		}
	}
}

func TestRootObjectAliasGuardsAndHopBudgets(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, n := range []int{48, 49} {
			req := request(rootObjectChain(n, ""), "")
			req.Options.Validation = flag
			r, err := core.GenerateWithDiagnostics(req)
			if n == 48 {
				if err != nil || len(r.Artifacts.Files) != 49 {
					t.Fatal("bounded cached nominal binding chain", flag, n, err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0 {
				t.Fatal(flag, n, r, err)
			}
		}
	}
	for _, wrappers := range []int{22, 23} {
		field := `{"$ref":"#/components/schemas/A23"}`
		for range wrappers {
			field = `{"type":"array","items":` + field + `}`
		}
		doc := rootObjectChain(24, `{"properties":{"v":`+field+`}}`)
		if wrappers == 22 {
			report(t, doc, "")
		} else {
			failure(t, doc, "E_SCHEMA_DEPTH", "/components/schemas/Consumer/properties/v"+strings.Repeat("/items", wrappers))
		}
	}
	for _, tc := range []struct{ body, code, pointer string }{
		{`"Alias":{"$ref":"#/components/schemas/Missing"}`, "E_REF_MISSING", "/components/schemas/Alias/$ref"},
		{`"Alias":{"$ref":"https://example.test/spec.json"}`, "E_REF_UNSUPPORTED", "/components/schemas/Alias/$ref"},
		{`"Alias":{"$ref":true}`, "E_SCHEMA_INVALID", "/components/schemas/Alias/$ref"},
		{`"Alias":{"$ref":"#/components/schemas/ZRecord","required":[]},"ZRecord":{"properties":{"id":{"type":"string"}}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Alias/required"},
		{`"A":{"$ref":"#/components/schemas/B"},"B":{"$ref":"#/components/schemas/A"}`, "E_REF_CYCLE", "/components/schemas/A/$ref"},
	} {
		aliasFailure(t, schemas("{"+tc.body+"}"), tc.code, tc.pointer)
	}
	failure(t, schemas(`{"A":{"$ref":"#/components/schemas/A"}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/A")
	r := report(t, schemas(`{"Alias":{"$ref":"#/components/schemas/Z~1a~0","examples":[{"$ref":"https://example.test/opaque"}]},"Z/a~":{"properties":{"id":{"type":"string"}}}}`), "Dto")
	if !strings.Contains(source(t, r, "AliasDto"), "org.example.models.models.ZADto") {
		t.Fatal("escaped target alias")
	}
}

func TestRootObjectAliasCompositionsValidationAndSlots(t *testing.T) {
	composedTarget := schemas(`{"Alias":{"$ref":"#/components/schemas/ZComposed"},"ZComposed":{"allOf":[{"properties":{"id":{"type":"string","minLength":1}},"required":["id"]},{"properties":{"flag":{"type":"boolean"}}}]}}`)
	composed := aggregationReport(t, composedTarget)
	if !strings.Contains(source(t, composed, "Alias"), "public typealias Alias = org.example.models.models.ZComposed") || !strings.Contains(source(t, composed, "ZComposed"), "length >= 1") {
		t.Fatal("alias to composed target lost canonical declaration/checks")
	}
	doc := schemas(`{"Alias":{"$ref":"#/components/schemas/ZRecord"},"Extended":{"allOf":[{"$ref":"#/components/schemas/Alias"},{"properties":{"extra":{"type":"boolean"}}}]},"ZRecord":{"properties":{"id":{"type":"string","minLength":1}},"required":["id"]}}`)
	r := aggregationReport(t, doc)
	if !strings.Contains(source(t, r, "Extended"), "length >= 1") || !strings.Contains(source(t, r, "ZRecord"), "length >= 1") {
		t.Fatal("canonical ordinary DTO checks missing")
	}
	// A canonical DTO remains a nominal recursion/validation barrier, not a
	// transparent generic expansion. Aliases do not add constructor fields.
	for _, n := range []int{240, 241} {
		var props []string
		for i := range n {
			props = append(props, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Alias"}`, i))
		}
		doc := schemas(`{"Alias":{"$ref":"#/components/schemas/ZRecord"},"Consumer":{"properties":{` + strings.Join(props, ",") + `}},"ZRecord":{"properties":{"id":{"type":"string","minLength":1}}}}`)
		if n == 240 {
			aggregationReport(t, doc)
		} else {
			aliasFailure(t, doc, "E_MODEL_LIMIT", "/components/schemas/Consumer")
		}
	}
}

func TestRootObjectAliasStructuralControl(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/root-object-aliases.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(input, &doc); err != nil {
		t.Fatal(err)
	}
	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	delete(schemas, "ARecord")
	delete(schemas, "BRecord")
	var replace func(any)
	replace = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			if v["$ref"] == "#/components/schemas/ARecord" || v["$ref"] == "#/components/schemas/BRecord" {
				v["$ref"] = "#/components/schemas/ZRecord"
			}
			for _, child := range v {
				replace(child)
			}
		case []any:
			for _, child := range v {
				replace(child)
			}
		}
	}
	replace(doc)
	control, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []bool{false, true} {
		req := request(control, "Dto")
		req.Options.Validation = flag
		if _, err := core.GenerateWithDiagnostics(req); err != nil {
			t.Fatal("existing full-envelope named control must pass", flag, err)
		}
	}
}
