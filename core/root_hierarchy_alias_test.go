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

func rootHierarchyInput(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/root-hierarchy-aliases.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestRootHierarchyAliasEnvelope(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			req := request(rootHierarchyInput(t), suffix)
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(flag, suffix, e)
			}
			again, e := core.GenerateWithDiagnostics(req)
			if e != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("determinism", e)
			}
			for _, pair := range [][2]string{{"ASignal", "ZSignal"}, {"BSignal", "ZSignal"}, {"AChannel", "ZChannel"}, {"AFlux", "ZFlux"}, {"BFlux", "ZFlux"}, {"AQuiet", "ZQuiet"}, {"AStatusAB", "ZStatus"}} {
				s := source(t, r, pair[0]+suffix)
				if !strings.Contains(s, "public typealias "+pair[0]+suffix+" = org.example.models.models."+pair[1]+suffix) {
					t.Fatal(s)
				}
			}
			root := source(t, r, "ZSignal"+suffix)
			child := source(t, r, "ZChannel"+suffix)
			leaf := source(t, r, "ZFlux"+suffix)
			if !strings.Contains(root, "ZChannel"+suffix+".serializer()") || !strings.Contains(child, "ZFlux"+suffix+".serializer()") || strings.Contains(child, "ignored-alias") || !strings.Contains(leaf, "ZChannel"+suffix+"Phase") || !strings.Contains(leaf, "ZChannel"+suffix+"()") {
				t.Fatal(root, child, leaf)
			}
			for _, s := range []string{root, child, leaf} {
				if strings.Contains(s, "AFlux"+suffix) || strings.Contains(s, "ASignal"+suffix) {
					t.Fatal("duplicated hierarchy identity", s)
				}
			}
			api := source(t, r, "DefaultApi")
			if strings.Count(api, "public suspend fun echo(") != 1 || strings.Count(api, "public suspend fun echo2(") != 1 {
				t.Fatal(api)
			}
			patch := source(t, r, "SubmitInput"+suffix)
			if !strings.Contains(patch, "Patch<org.example.models.models.ZSignal"+suffix+">") || !strings.Contains(patch, "Patch<org.example.models.models.ZFlux"+suffix+">") {
				t.Fatal(patch)
			}
		}
	}
}
func TestRootHierarchyAliasCanonicalErrors(t *testing.T) {
	for _, tc := range []struct{ body, code, pointer string }{
		{`{"Alias":{"$ref":"#/components/schemas/Leaf"},"Leaf":{"properties":{"tag":{"type":"string"}}},"Root":{"oneOf":[{"$ref":"#/components/schemas/Alias"},{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"tag"}}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Root/oneOf/1/$ref"},
		{`{"Alias":{"$ref":"#/components/schemas/Leaf"},"Leaf":{"properties":{"tag":{"type":"string"}}},"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"},{"$ref":"#/components/schemas/Alias"}],"discriminator":{"propertyName":"tag"}}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Root/oneOf/1/$ref"},
		{`{"Alias":{"$ref":"#/components/schemas/Root"},"Root":{"oneOf":[{"$ref":"#/components/schemas/Alias"}],"discriminator":{"propertyName":"tag"}}}`, "E_REF_CYCLE", "/components/schemas/Root/oneOf/0/$ref"},
		{`{"Alias":{"$ref":"#/components/schemas/Other"},"Leaf":{"properties":{"tag":{"type":"string"}}},"Other":{"properties":{"id":{"type":"string"}}},"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"tag","mapping":{"bad":"Alias"}}}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Root/discriminator/mapping/bad"},
		{`{"Alias":{"$ref":"#/components/schemas/Leaf"},"Leaf":{"properties":{"tag":{"type":"string"}}},"A":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"tag"}},"B":{"oneOf":[{"$ref":"#/components/schemas/Alias"}],"discriminator":{"propertyName":"tag"}}}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/B/oneOf/0/$ref"},
	} {
		for _, flag := range []bool{false, true} {
			req := request(schemas(tc.body), "")
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if e == nil || len(r.Artifacts.Files) != 0 {
				t.Fatal(flag, tc, r, e)
			}
			d := e.(core.Diagnostics)[0]
			if d.Code != tc.code || *d.Pointer != tc.pointer {
				t.Fatal(flag, tc, d)
			}
		}
	}
	// Unmapped aliases use canonical fallback wire, never alias spelling/suffix.
	s := source(t, report(t, schemas(`{"ALeaf":{"$ref":"#/components/schemas/ZLeaf"},"ZLeaf":{"properties":{"tag":{"type":"string"}}},"Other":{"properties":{"tag":{"type":"string"}}},"Root":{"oneOf":[{"$ref":"#/components/schemas/ALeaf"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag"}}}`), "Dto"), "RootDto")
	if !strings.Contains(s, `tag == "zleaf"`) || strings.Contains(s, `tag == "aleaf"`) || strings.Contains(s, `tag == "ZLeafDto"`) {
		t.Fatal(s)
	}
}
func hierarchyLeafChain(n int, extra string) []byte {
	nodes := []string{`"ZLeaf":{"properties":{"tag":{"type":"string"},"id":{"type":"string"}},"required":["id"]}`, `"Other":{"properties":{"tag":{"type":"string"}}}`}
	for i := range n {
		target := "ZLeaf"
		if i > 0 {
			target = fmt.Sprintf("A%02d", i-1)
		}
		nodes = append(nodes, fmt.Sprintf(`"A%02d":{"$ref":"#/components/schemas/%s"}`, i, target))
	}
	if extra != "" {
		nodes = append(nodes, extra)
	}
	return schemas("{" + strings.Join(nodes, ",") + "}")
}
func TestRootHierarchyAliasSourceCacheAndSlots(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, typed := range []bool{false, true} {
			limit := 48
			if typed {
				limit = 46
			}
			for _, n := range []int{limit, limit + 1} {
				parts := []string{`"ZRoot":{"oneOf":[{"$ref":"#/components/schemas/ZLeaf"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag"}}`, `"ZLeaf":{"properties":{"tag":{"type":"string"},"id":{"type":"string"}}}`, `"Other":{"properties":{"tag":{"type":"string"}}}`}
				for i := range n {
					target := "ZRoot"
					if i > 0 {
						target = fmt.Sprintf("A%02d", i-1)
					}
					parts = append(parts, fmt.Sprintf(`"A%02d":{"$ref":"#/components/schemas/%s"}`, i, target))
				}
				if typed {
					parts = append(parts, fmt.Sprintf(`"Holder":{"properties":{"v":{"$ref":"#/components/schemas/A%02d"}}}`, n-1))
				}
				req := request(schemas("{"+strings.Join(parts, ",")+"}"), "")
				req.Options.Validation = flag
				r, e := core.GenerateWithDiagnostics(req)
				if n == limit {
					if e != nil {
						t.Fatal(e)
					}
				} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0 {
					t.Fatal(flag, typed, n, r, e)
				} else if typed && *e.(core.Diagnostics)[0].Pointer != "/components/schemas/Holder/properties/v" {
					t.Fatal(e)
				}
			}
		}
	}
	for _, flag := range []bool{false, true} {
		for _, n := range []int{46, 47} {
			root := fmt.Sprintf(`"Root":{"oneOf":[{"$ref":"#/components/schemas/A%02d"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag"}}`, n-1)
			req := request(hierarchyLeafChain(n, root), "")
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if n == 46 {
				if e != nil {
					t.Fatal(e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || *e.(core.Diagnostics)[0].Pointer != "/components/schemas/Root/oneOf/0/$ref" || len(r.Artifacts.Files) != 0 {
				t.Fatal(n, r, e)
			}
		}
	}
	// AChild subtree is cached first. ZParent must include its erased edge hops.
	nodes := `"AChild":{"oneOf":[{"$ref":"#/components/schemas/ALeaf"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag"}},"ALeaf":{"$ref":"#/components/schemas/ZLeaf"},"ZLeaf":{"properties":{"family":{"type":"string","const":"nested"},"tag":{"type":"string"}}},"Other":{"properties":{"family":{"type":"string","const":"nested"},"tag":{"type":"string"}}},"OuterLeaf":{"properties":{"family":{"type":"string","const":"other"}}}`
	for _, n := range []int{44, 45} {
		parts := []string{nodes}
		for i := range n {
			target := "AChild"
			if i > 0 {
				target = fmt.Sprintf("B%02d", i-1)
			}
			parts = append(parts, fmt.Sprintf(`"B%02d":{"$ref":"#/components/schemas/%s"}`, i, target))
		}
		parts = append(parts, fmt.Sprintf(`"ZParent":{"oneOf":[{"$ref":"#/components/schemas/B%02d"},{"$ref":"#/components/schemas/OuterLeaf"}],"discriminator":{"propertyName":"family","mapping":{"nested":"AChild","other":"OuterLeaf"}}}`, n-1))
		r, e := core.GenerateWithDiagnostics(request(schemas("{"+strings.Join(parts, ",")+"}"), ""))
		if n == 44 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || *e.(core.Diagnostics)[0].Pointer != "/components/schemas/ZParent/oneOf/0/$ref" || len(r.Artifacts.Files) != 0 {
			t.Fatal(n, r, e)
		}
	}
	for _, count := range []int{119, 120} {
		props := []string{`"tag":{"type":"string","const":"wide"}`, `"flag":{"type":"boolean"}`}
		required := []string{`"flag"`}
		for i := range count {
			props = append(props, fmt.Sprintf(`"p%d":{"type":"integer","format":"int64"}`, i))
			required = append(required, fmt.Sprintf(`"p%d"`, i))
		}
		input := schemas(`{"AWide":{"$ref":"#/components/schemas/ZWide"},"ZWide":{"properties":{` + strings.Join(props, ",") + `},"required":[` + strings.Join(required, ",") + `]},"Small":{"properties":{"tag":{"type":"string","const":"small"}}},"Root":{"oneOf":[{"$ref":"#/components/schemas/AWide"},{"$ref":"#/components/schemas/Small"}],"discriminator":{"propertyName":"tag","mapping":{"wide":"AWide","small":"Small"}}}}`)
		r, e := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count == 119 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
			t.Fatal(count, r, e)
		}
	}
}

func TestRootHierarchyAliasMappingHeightAndNullExclusions(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, n := range []int{46, 47} {
			root := fmt.Sprintf(`"Root":{"oneOf":[{"$ref":"#/components/schemas/ZLeaf"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag","mapping":{"leaf":"A%02d"}}}`, n-1)
			req := request(hierarchyLeafChain(n, root), "Dto")
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if n == 46 {
				if e != nil {
					t.Fatal(e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || *e.(core.Diagnostics)[0].Pointer != "/components/schemas/Root/discriminator/mapping/leaf" || len(r.Artifacts.Files) != 0 {
				t.Fatal(n, r, e)
			}
		}
	}
	for _, tc := range []struct{ body, pointer string }{
		{`{"Alias":{"$ref":"#/components/schemas/Root","nullable":true},"Root":{"oneOf":[{"$ref":"#/components/schemas/ZLeaf"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag"}},"ZLeaf":{"properties":{"tag":{"type":"string"}}},"Other":{"properties":{"tag":{"type":"string"}}}}`, "/components/schemas/Alias"},
		{`{"Alias":{"$ref":"#/components/schemas/Root"},"Root":{"nullable":true,"oneOf":[{"$ref":"#/components/schemas/ZLeaf"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"tag"}},"ZLeaf":{"properties":{"tag":{"type":"string"}}},"Other":{"properties":{"tag":{"type":"string"}}}}`, "/components/schemas/Root"},
	} {
		for _, flag := range []bool{false, true} {
			req := request(schemas(tc.body), "")
			req.Options.Validation = flag
			r, e := core.GenerateWithDiagnostics(req)
			if tc.pointer == "/components/schemas/Alias" {
				// The unchanged historical nullable-alias input graduates under
				// the independent nullable hierarchy alias contract. Intrinsic
				// root nullability below has its own positive physical-class check.
				if e != nil || !strings.Contains(source(t, r, "Alias"), "public typealias Alias = org.example.models.models.Root?") {
					t.Fatal(flag, r, e)
				}
				continue
			}
			// The second unchanged historical input now has an intrinsically
			// nullable hierarchy root, retaining its non-null physical class.
			if e != nil || !strings.Contains(source(t, r, "Alias"), "public typealias Alias = org.example.models.models.Root?") || !strings.Contains(source(t, r, "Root"), "public sealed class Root {") {
				t.Fatal(flag, r, e)
			}
		}
	}
}

func TestRootHierarchyAliasBranchAndErasedWorkLimits(t *testing.T) {
	for _, count := range []int{240, 241} {
		nodes := []string{}
		branches := []string{}
		for i := range count {
			nodes = append(nodes, fmt.Sprintf(`"A%03d":{"$ref":"#/components/schemas/Z%03d"},"Z%03d":{"properties":{"tag":{"type":"string","const":"v%d"}}}`, i, i, i, i))
			branches = append(branches, fmt.Sprintf(`{"$ref":"#/components/schemas/A%03d"}`, i))
		}
		nodes = append(nodes, `"Root":{"oneOf":[`+strings.Join(branches, ",")+`],"discriminator":{"propertyName":"tag"}}`)
		r, e := core.GenerateWithDiagnostics(request(schemas("{"+strings.Join(nodes, ",")+"}"), ""))
		if count == 240 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_UNION_LIMIT" || *e.(core.Diagnostics)[0].Pointer != "/components/schemas/Root/oneOf" || len(r.Artifacts.Files) != 0 {
			t.Fatal(count, r, e)
		}
	}
	var nodes []string
	for i := range 5462 {
		nodes = append(nodes, fmt.Sprintf(`"Root%05d":{"oneOf":[{"$ref":"#/components/schemas/A%05d"}],"discriminator":{"propertyName":"tag"}},"A%05d":{"$ref":"#/components/schemas/Leaf%05d"},"Leaf%05d":{"properties":{"tag":{"type":"string"}}}`, i, i, i, i, i))
	}
	failure(t, schemas("{"+strings.Join(nodes, ",")+"}"), "E_SCHEMA_LIMIT", "/components/schemas/Root05461/oneOf/0/$ref")
}

func TestRootHierarchyAliasStructuralControl(t *testing.T) {
	var doc map[string]any
	if e := json.Unmarshal(rootHierarchyInput(t), &doc); e != nil {
		t.Fatal(e)
	}
	catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
	var decode func(string) string
	decode = func(ref string) string {
		return strings.ReplaceAll(strings.ReplaceAll(strings.TrimPrefix(ref, "#/components/schemas/"), "~1", "/"), "~0", "~")
	}
	var resolve func(string) string
	resolve = func(n string) string {
		v := catalog[n].(map[string]any)
		if ref, ok := v["$ref"].(string); ok {
			return resolve(decode(ref))
		}
		return n
	}
	bindings := map[string]string{}
	for n, v := range catalog {
		if _, ok := v.(map[string]any)["$ref"]; ok {
			bindings[n] = resolve(n)
		}
	}
	for n := range bindings {
		delete(catalog, n)
	}
	rewrite := func(ref string) string {
		if target, ok := bindings[decode(ref)]; ok {
			return "#/components/schemas/" + target
		}
		return ref
	}
	var visit func(any)
	visit = func(x any) {
		switch v := x.(type) {
		case map[string]any:
			if ref, ok := v["$ref"].(string); ok {
				v["$ref"] = rewrite(ref)
			}
			if d, ok := v["discriminator"].(map[string]any); ok {
				if mapping, ok := d["mapping"].(map[string]any); ok {
					for key, target := range mapping {
						mapping[key] = rewrite(target.(string))
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
	// Constraint itself is the established validation-opt-in surface.
	for _, flag := range []bool{false, true} {
		if !flag {
			delete(catalog["ZFlux"].(map[string]any)["properties"].(map[string]any)["note"].(map[string]any), "minLength")
		} else {
			catalog["ZFlux"].(map[string]any)["properties"].(map[string]any)["note"].(map[string]any)["minLength"] = 1
		}
		input, _ := json.Marshal(doc)
		req := request(input, "Dto")
		req.Options.Validation = flag
		if _, e := core.GenerateWithDiagnostics(req); e != nil {
			t.Fatal("whole canonical control", flag, e)
		}
	}
}
