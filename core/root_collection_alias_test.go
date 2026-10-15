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

func rootCollectionInput(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/root-collection-aliases.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}

func TestRootCollectionAliasEnvelope(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(rootCollectionInput(t), suffix)
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatalf("root collection binding flag%t: %v", flag, e)
			}
			again, e := core.GenerateWithDiagnostics(req)
			if e != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("root collection determinism", e)
			}
			for _, name := range []string{"AValues", "BValues"} {
				if !strings.Contains(source(t, r, name+suffix), "public typealias "+name+suffix+" = org.example.models.models.ZValues"+suffix+"?") {
					t.Fatal(source(t, r, name+suffix))
				}
			}
			holder := source(t, r, "Holder"+suffix)
			for _, want := range []string{"narrow: kotlin.collections.List<kotlin.String?>", "mapNarrow: kotlin.collections.Map<kotlin.String, kotlin.String?>", "mapUseNarrow: kotlin.collections.Map<kotlin.String, kotlin.String?>", "forced: org.example.models.models.ZItems" + suffix + "?", "rows: org.example.models.models.ZRows" + suffix} {
				if !strings.Contains(holder, want) {
					t.Fatalf("missing %s in %s", want, holder)
				}
			}
			for _, name := range []string{"AMapNarrow", "ANarrow"} {
				if strings.Contains(source(t, r, name+suffix), "?>?") {
					t.Fatal("root null removed but item preserved", source(t, r, name+suffix))
				}
			}
			patch := source(t, r, "SubmitInput"+suffix)
			for _, want := range []string{"Patch<kotlin.collections.List<kotlin.String?>>", "Patch<kotlin.collections.Map<kotlin.String, kotlin.String?>>", "Patch<org.example.models.models.ZItems" + suffix + ">", "requiredNarrow: kotlin.collections.Map<kotlin.String, kotlin.String?>"} {
				if !strings.Contains(patch, want) {
					t.Fatal(want, patch)
				}
			}
			api := source(t, r, "DefaultApi")
			if strings.Count(api, "public suspend fun echo(") != 1 || strings.Count(api, "public suspend fun echo2(") != 1 {
				t.Fatal("transparent erasure", api)
			}
		}
	}
}

func rootCollectionChain(n int, consumer string) []byte {
	entries := []string{`"ZValues":{"type":"array","items":{"type":"string"}}`}
	for i := range n {
		target := "ZValues"
		if i > 0 {
			target = fmt.Sprintf("A%02d", i-1)
		}
		entries = append(entries, fmt.Sprintf(`"A%02d":{"$ref":"#/components/schemas/%s"}`, i, target))
	}
	if consumer != "" {
		entries = append(entries, `"Consumer":`+consumer)
	}
	return schemas("{" + strings.Join(entries, ",") + "}")
}

func TestRootCollectionAliasSourceBudgets(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, n := range []int{47, 48} {
			req := request(rootCollectionChain(n, ""), "")
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if n == 47 {
				if e != nil {
					t.Fatal("root alias47+array1", flag, e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0 {
				t.Fatal(flag, n, r, e)
			}
		}
		for _, n := range []int{45, 46} {
			field := fmt.Sprintf(`{"properties":{"v":{"$ref":"#/components/schemas/A%02d"}}}`, n-1)
			req := request(rootCollectionChain(n, field), "")
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if n == 45 {
				if e != nil {
					t.Fatal("field+ref+45aliases+array=48", flag, e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0 {
				t.Fatal(flag, n, r, e)
			}
		}
	}
}

func TestRootCollectionAliasGuards(t *testing.T) {
	for _, tc := range []struct{ body, code string }{
		{`"A":{"$ref":"#/components/schemas/ZValues"},"ZValues":{"type":"array","items":{"$ref":"#/components/schemas/A"}}`, "E_REF_CYCLE"},
		{`"A":{"$ref":"#/components/schemas/ZMap","nullable":true},"ZMap":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/A"}}`, "E_REF_CYCLE"},
		{`"A":{"$ref":"#/components/schemas/Missing"}`, "E_REF_MISSING"},
		{`"A":{"$ref":"https://example.test/spec"}`, "E_REF_UNSUPPORTED"},
		{`"A":{"$ref":"#/components/schemas/ZValues","minItems":1},"ZValues":{"type":"array","items":{"type":"string"}}`, "E_SCHEMA_UNSUPPORTED"},
		{`"A":{"$ref":"#/components/schemas/ZValues"},"ZValues":{"type":"array","minItems":-1,"items":{"type":"string"}}`, "E_SCHEMA_INVALID"},
	} {
		req := request(schemas("{"+tc.body+"}"), "")
		req.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(req)
		if e == nil || e.(core.Diagnostics)[0].Code != tc.code || len(r.Artifacts.Files) != 0 {
			t.Fatal(tc, r, e)
		}
	}
	for _, n := range []int{16, 17} {
		leaf := `{"type":"string","minLength":1}`
		for range n {
			leaf = `{"type":"array","items":` + leaf + `}`
		}
		req := request(schemas(`{"Alias":{"$ref":"#/components/schemas/ZValues"},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Alias"}}},"ZValues":`+leaf+`}`), "")
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
	for _, n := range []int{240, 241} {
		var props []string
		for i := range n {
			props = append(props, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Alias"}`, i))
		}
		req := request(schemas(`{"Alias":{"$ref":"#/components/schemas/ZValues"},"Holder":{"properties":{`+strings.Join(props, ",")+`}},"ZValues":{"type":"array","minItems":1,"items":{"type":"string","minLength":1,"pattern":"^[a-z]+$"}}}`), "")
		req.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(req)
		if n == 240 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
			t.Fatal(n, r, e)
		}
	}
}

func TestRootCollectionAliasOverridesAndNoDrop(t *testing.T) {
	for _, flag := range []bool{false, true} {
		doc := schemas(`{"A":{"$ref":"#/components/schemas/B"},"B":{"$ref":"#/components/schemas/Z","nullable":false},"C":{"$ref":"#/components/schemas/A","nullable":true},"Holder":{"properties":{"inherit":{"$ref":"#/components/schemas/A"},"rootTrue":{"$ref":"#/components/schemas/C"},"useTrue":{"$ref":"#/components/schemas/A","nullable":true},"wrapped":{"allOf":[{"$ref":"#/components/schemas/C","nullable":false}]},"wrappedOuter":{"allOf":[{"$ref":"#/components/schemas/A","nullable":false}],"nullable":true}},"required":["inherit","rootTrue","useTrue","wrapped","wrappedOuter"]},"Z":{"type":"object","nullable":true,"additionalProperties":{"type":["string","null"]}}}`)
		req := request(doc, "Dto")
		req.Options.Validation = flag
		r, e := core.GenerateWithDiagnostics(req)
		if e != nil {
			t.Fatal(flag, e)
		}
		text := source(t, r, "HolderDto")
		for _, want := range []string{"inherit: kotlin.collections.Map<kotlin.String, kotlin.String?>", "wrapped: kotlin.collections.Map<kotlin.String, kotlin.String?>", "rootTrue: org.example.models.models.ZDto?", "useTrue: org.example.models.models.ZDto?", "wrappedOuter: org.example.models.models.ZDto?"} {
			if !strings.Contains(text, want) {
				t.Fatal(flag, want, text)
			}
		}
	}
	// Typed structured parameters retain checks even though Ktor stringifies
	// their wire representation. Ignored fixed-model extras cannot retain them.
	doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"A":{"$ref":"#/components/schemas/Z"},"Z":{"type":"object","additionalProperties":{"type":"string","minLength":1}}}},"paths":{"/raw":{"get":{"parameters":[{"name":"q","in":"query","schema":{"$ref":"#/components/schemas/A"}}],"responses":{"204":{}}}}}}`)
	if !strings.Contains(source(t, aggregationReport(t, doc), "DefaultApi"), "length >= 1") {
		t.Fatal("structured alias parameter dropped preflight checks")
	}
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z"},"Fixed":{"properties":{"id":{"type":"string"}},"additionalProperties":{"$ref":"#/components/schemas/A"}},"Z":{"type":"object","additionalProperties":{"type":"string","minLength":1}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Z/additionalProperties/minLength")
	// Narrowing inside a collection definition must not erase chain source work.
	for _, n := range []int{45, 46} {
		var catalog map[string]any
		_ = json.Unmarshal(rootCollectionChain(n, ""), &catalog)
		components := catalog["components"].(map[string]any)["schemas"].(map[string]any)
		components["ZValues"].(map[string]any)["nullable"] = true
		components["Outer"] = map[string]any{"type": "array", "items": map[string]any{"$ref": fmt.Sprintf("#/components/schemas/A%02d", n-1), "nullable": false}}
		input, _ := json.Marshal(catalog)
		req := request(input, "")
		r, e := core.GenerateWithDiagnostics(req)
		if n == 45 {
			if e != nil {
				t.Fatal("root array+ref+45alias+terminal=48", e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0 {
			t.Fatal(n, r, e)
		}
	}
}

func TestRootCollectionAliasStructuralControl(t *testing.T) {
	var doc map[string]any
	if e := json.Unmarshal(rootCollectionInput(t), &doc); e != nil {
		t.Fatal(e)
	}
	catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
	bindings := map[string]struct {
		target   string
		nullable *bool
	}{}
	var bind func(string) (string, *bool)
	bind = func(name string) (string, *bool) {
		if b, ok := bindings[name]; ok {
			return b.target, b.nullable
		}
		v := catalog[name].(map[string]any)
		ref, ok := v["$ref"].(string)
		if !ok {
			return name, nil
		}
		target, nullable := bind(strings.TrimPrefix(ref, "#/components/schemas/"))
		if flag, ok := v["nullable"].(bool); ok {
			nullable = &flag
		}
		bindings[name] = struct {
			target   string
			nullable *bool
		}{target, nullable}
		return target, nullable
	}
	for name, v := range catalog {
		if _, ok := v.(map[string]any)["$ref"]; ok {
			bind(name)
		}
	}
	for name := range bindings {
		delete(catalog, name)
	}
	var visit func(any)
	visit = func(x any) {
		switch v := x.(type) {
		case map[string]any:
			if ref, ok := v["$ref"].(string); ok {
				if b, ok := bindings[strings.TrimPrefix(ref, "#/components/schemas/")]; ok {
					v["$ref"] = "#/components/schemas/" + b.target
					if _, present := v["nullable"]; !present && b.nullable != nil {
						v["nullable"] = *b.nullable
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
	input, _ := json.Marshal(doc)
	for _, flag := range []bool{false, true} {
		req := request(input, "Dto")
		req.Options.Validation = flag
		if _, e := core.GenerateWithDiagnostics(req); e != nil {
			t.Fatal("canonical target control", flag, e)
		}
	}
}
