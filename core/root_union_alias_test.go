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

func rootUnionInput(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/root-union-aliases.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestRootUnionAliasEnvelope(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(rootUnionInput(t), suffix)
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal("root union bindings", flag, suffix, e)
			}
			again, e := core.GenerateWithDiagnostics(req)
			if e != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("determinism", e)
			}
			for _, pair := range [][2]string{{"AChoice", "ZChoice"}, {"BChoice", "ZChoice"}, {"AKind", "ZKind"}, {"AClosed", "ZClosed"}} {
				if !strings.Contains(source(t, r, pair[0]+suffix), "public typealias "+pair[0]+suffix+" = org.example.models.models."+pair[1]+suffix) {
					t.Fatal(source(t, r, pair[0]+suffix))
				}
			}
			if !strings.Contains(source(t, r, "ZChoice"+suffix), "Unknown") || !strings.Contains(source(t, r, "ZKind"+suffix), "NullZKind"+suffix) {
				t.Fatal("canonical root backends lost")
			}
			api := source(t, r, "DefaultApi")
			if strings.Count(api, "public suspend fun echo(") != 1 || strings.Count(api, "public suspend fun echo2(") != 1 {
				t.Fatal("canonical nominal erasure", api)
			}
			patch := source(t, r, "SubmitInput"+suffix)
			if !strings.Contains(patch, "Patch<org.example.models.models.ZChoice"+suffix+">") || !strings.Contains(patch, "Patch<org.example.models.models.ZKind"+suffix+">") {
				t.Fatal(patch)
			}
		}
	}
}
func TestRootUnionAliasSourceAndExclusions(t *testing.T) {
	terminal := `{"oneOf":[{"type":"string"},{"type":"boolean"},{"type":"null"}]}`
	for _, flag := range []bool{false, true} {
		for _, n := range []int{48, 49} {
			req := request(rootSpecialChain(n, terminal, ""), "")
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if n == 48 {
				if e != nil {
					t.Fatal(e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0 {
				t.Fatal(n, r, e)
			}
		}
		for _, n := range []int{46, 47} {
			holder := fmt.Sprintf(`{"properties":{"v":{"$ref":"#/components/schemas/A%02d"}}}`, n-1)
			req := request(rootSpecialChain(n, terminal, holder), "")
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if n == 46 {
				if e != nil {
					t.Fatal(e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || *e.(core.Diagnostics)[0].Pointer != "/components/schemas/Holder/properties/v" || len(r.Artifacts.Files) != 0 {
				t.Fatal(n, r, e)
			}
		}
	}
	// The unchanged own-nullable union input now retains the canonical Null
	// carrier backing while its independent typed alias accepts Kotlin null.
	for _, checked := range []bool{false, true} {
		req := request(schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":`+terminal+`}`), "Dto")
		req.Options.Validation = checked
		r, e := core.GenerateWithDiagnostics(req)
		if e != nil || !strings.Contains(source(t, r, "ADto"), "= org.example.models.models.ZDto?\n") || !strings.Contains(source(t, r, "ZDto"), "NullZDto") {
			t.Fatal("nullable canonical union graduation", e)
		}
	}
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/Z"},"Z":{"oneOf":[{"type":"string"},{"type":"string"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Z/oneOf")
	// Alias-only cycles retain historical diagnostics; malformed terminal and
	// original hierarchy/raw exclusions remain exercised by existing suites.
	aliasFailure(t, schemas(`{"A":{"$ref":"#/components/schemas/B"},"B":{"$ref":"#/components/schemas/A"}}`), "E_REF_CYCLE", "/components/schemas/A/$ref")
	for _, n := range []int{240, 241} {
		branches := []string{`{"type":"null"}`}
		for i := 1; i < n; i++ {
			branches = append(branches, fmt.Sprintf(`{"properties":{"kind":{"type":"string","const":"v%d"},"id":{"type":"string"}},"required":["id"]}`, i))
		}
		req := request(schemas(`{"A":{"$ref":"#/components/schemas/Z"},"Z":{"oneOf":[`+strings.Join(branches, ",")+`]}}`), "")
		r, e := core.GenerateWithDiagnostics(req)
		if n == 240 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_UNION_LIMIT" || len(r.Artifacts.Files) != 0 {
			t.Fatal(n, r, e)
		}
	}
}

func TestRootUnionAliasNominalAndNullableTerminal(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(schemas(`{"A":{"$ref":"#/components/schemas/Z"},"Z":{"nullable":true,"oneOf":[{"$ref":"#/components/schemas/Node"},{"$ref":"#/components/schemas/Other"}]},"Node":{"properties":{"kind":{"type":"string","const":"node"},"child":{"$ref":"#/components/schemas/A","nullable":true},"children":{"type":"array","items":{"$ref":"#/components/schemas/A"}}}},"Other":{"properties":{"kind":{"type":"string","const":"other"},"flag":{"type":"boolean"}}}}`), suffix)
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(flag, suffix, e)
			}
			root := source(t, r, "Z"+suffix)
			if !strings.Contains(root, "NullZ"+suffix) || !strings.Contains(root, "org.example.models.models.Z"+suffix+"?") || !strings.Contains(root, "List<org.example.models.models.Z"+suffix+">") {
				t.Fatal(root)
			}
			if strings.Contains(source(t, r, "A"+suffix), "Z"+suffix+"?") {
				t.Fatal("Native Null carrier must remain named/non-null")
			}
		}
	}
}

func TestRootUnionAliasConstructorSlots(t *testing.T) {
	for _, n := range []int{120, 121} {
		props := []string{}
		required := []string{}
		for i := 0; i < n; i++ {
			props = append(props, fmt.Sprintf(`"p%d":{"type":"integer","format":"int64"}`, i))
			required = append(required, fmt.Sprintf(`"p%d"`, i))
		}
		input := schemas(`{"A":{"$ref":"#/components/schemas/Z"},"Z":{"oneOf":[{"properties":{` + strings.Join(props, ",") + `},"required":[` + strings.Join(required, ",") + `]},{"properties":{"kind":{"type":"string","const":"small"},"enabled":{"type":"boolean"}},"required":["enabled"]}]}}`)
		r, e := core.GenerateWithDiagnostics(request(input, "Dto"))
		if n == 120 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !strings.HasPrefix(*e.(core.Diagnostics)[0].Pointer, "/components/schemas/Z/oneOf/0") || len(r.Artifacts.Files) != 0 {
			t.Fatal(n, r, e)
		}
	}
}

func TestRootUnionAliasStructuralControl(t *testing.T) {
	var doc map[string]any
	if e := json.Unmarshal(rootUnionInput(t), &doc); e != nil {
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
