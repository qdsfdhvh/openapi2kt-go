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

func rootRawInput(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/root-raw-aliases.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func rootRawControl(t *testing.T, unique bool) []byte {
	t.Helper()
	var doc map[string]any
	if e := json.Unmarshal(rootRawInput(t), &doc); e != nil {
		t.Fatal(e)
	}
	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	bindings := map[string]string{"ARaw": "ZRaw", "BRaw": "ZRaw", "AOther": "ZOther"}
	for n := range bindings {
		delete(schemas, n)
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
	if unique {
		for path, group := range doc["paths"].(map[string]any) {
			for _, op := range group.(map[string]any) {
				o := op.(map[string]any)
				if o["operationId"] == "echo" {
					o["operationId"] = "echo" + strings.TrimPrefix(path, "/")
				}
			}
		}
	}
	b, _ := json.Marshal(doc)
	return b
}
func TestRootRawAliasStructuralControl(t *testing.T) {
	for _, flag := range []bool{false, true} {
		req := request(rootRawControl(t, true), "Dto")
		req.Options.Validation = flag
		if _, e := core.GenerateWithDiagnostics(req); e != nil {
			t.Fatal("whole canonical distinct-method control", flag, e)
		}
	}
}
func TestRootRawAliasEnvelope(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(rootRawInput(t), suffix)
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(flag, suffix, e)
			}
			again, e := core.GenerateWithDiagnostics(req)
			if e != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("determinism", e)
			}
			for _, pair := range [][2]string{{"ARaw", "ZRaw"}, {"BRaw", "ZRaw"}, {"AOther", "ZOther"}} {
				s := source(t, r, pair[0]+suffix)
				if !strings.Contains(s, "public typealias "+pair[0]+suffix+" = org.example.models.models."+pair[1]+suffix) {
					t.Fatal(s)
				}
			}
			for _, root := range []string{"ZRaw", "ZOther"} {
				s := source(t, r, root+suffix)
				if !strings.Contains(s, "= kotlinx.serialization.json.JsonElement") {
					t.Fatal("raw must not acquire nominal ctor/serializer", s)
				}
			}
			api := source(t, r, "DefaultApi")
			for _, name := range []string{"echo", "echo2", "echo3", "echo4"} {
				if strings.Count(api, "public suspend fun "+name+"(") != 1 {
					t.Fatal("actual raw erasure", api)
				}
			}
			count := 0
			for _, d := range r.Warnings {
				if d.Code == "W_JSON_ELEMENT_APPROXIMATION" {
					count++
					if *d.Pointer != "/components/schemas/ZRaw" && *d.Pointer != "/components/schemas/ZOther" {
						t.Fatal(d)
					}
				}
			}
			if count != 2 {
				t.Fatal("one warning per raw terminal, no alias copies", r.Warnings)
			}
		}
	}
}
func TestRootRawAliasSourceAndSlots(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, typed := range []bool{false, true} {
			limit := 48
			if typed {
				limit = 46
			}
			for _, n := range []int{limit, limit + 1} {
				consumer := ""
				if typed {
					consumer = fmt.Sprintf(`{"properties":{"v":{"$ref":"#/components/schemas/A%02d"}}}`, n-1)
				}
				req := request(rootSpecialChain(n, `{"type":"object"}`, consumer), "Dto")
				req.Options.Validation = flag
				r, e := core.GenerateWithDiagnostics(req)
				if n == limit {
					if e != nil {
						t.Fatal(flag, typed, e)
					}
				} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0 {
					t.Fatal(flag, typed, n, r, e)
				} else if typed && *e.(core.Diagnostics)[0].Pointer != "/components/schemas/Holder/properties/v" {
					t.Fatal(e)
				}
			}
		}
	}
	for _, count := range []int{240, 241} {
		fields := []string{}
		for i := range count {
			fields = append(fields, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/A"}`, i))
		}
		input := schemas(`{"A":{"$ref":"#/components/schemas/Z"},"Z":{"type":"object"},"Holder":{"properties":{` + strings.Join(fields, ",") + `}}}`)
		r, e := core.GenerateWithDiagnostics(request(input, ""))
		if count == 240 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
			t.Fatal(count, r, e)
		}
	}
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z"},"Z":{"allOf":[{"$ref":"#/components/schemas/Z"},{"type":"object"}]}}`), "E_REF_CYCLE", "/components/schemas/Z/allOf/0/$ref")
}

func TestRootRawAliasCollectionSourceHeight(t *testing.T) {
	for _, n := range []int{45, 46} {
		list := `{"$ref":"#/components/schemas/ZRaw"}`
		for range n {
			list = `{"type":"array","items":` + list + `}`
		}
		input := schemas(`{"Holder":{"properties":{"v":{"$ref":"#/components/schemas/ZList"}}},"ZList":` + list + `,"ZRaw":{"type":"object"}}`)
		r, e := core.GenerateWithDiagnostics(request(input, ""))
		if n == 45 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || *e.(core.Diagnostics)[0].Pointer != "/components/schemas/Holder/properties/v" || len(r.Artifacts.Files) != 0 {
			t.Fatal(n, r, e)
		}
	}
}

func TestRootRawDirectErasure(t *testing.T) {
	r, e := core.GenerateWithDiagnostics(request(rootRawControl(t, false), "Dto"))
	if e != nil {
		t.Fatal(e)
	}
	api := source(t, r, "DefaultApi")
	for _, name := range []string{"echo", "echo2", "echo3", "echo4"} {
		if strings.Count(api, "public suspend fun "+name+"(") != 1 {
			t.Fatal("distinct raw aliases and inline JSON have one real type", api)
		}
	}
}
