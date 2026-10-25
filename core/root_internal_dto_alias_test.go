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

func TestRootInternalDTOAliasCanonicalEnvelope(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/root-internal-dto-aliases.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			rq := request(input, suffix)
			rq.Options.Validation = flag
			r, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatalf("canonical root/internal binding: validation=%t suffix=%q: %v", flag, suffix, err)
			}
			again, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("root/internal binding determinism", err)
			}
			for name, target := range map[string]string{
				"NodeAlias": "TreeKey" + suffix + ".Node", "ChainAlias": "TreeKey" + suffix + ".Node", "HopAlias": "TreeKey" + suffix + ".Node",
				"OptionalAlias": "TreeKey" + suffix + ".Node?", "MaybeAlias": "TreeKey" + suffix + ".Maybe?", "NarrowAlias": "TreeKey" + suffix + ".Maybe",
				"RowAlias": "RootRows" + suffix + "Inner", "MapAlias": "RootMap" + suffix + "Value",
			} {
				text := source(t, r, name+suffix)
				if !strings.Contains(text, "public typealias "+name+suffix+" = org.example.models.models."+target) || strings.Contains(text, "data class") {
					t.Fatal("alias copied or changed owner", name, text)
				}
			}
			consumer := source(t, r, "Consumer"+suffix)
			for _, want := range []string{
				"node: org.example.models.models.TreeKey" + suffix + ".Node",
				"narrowed: org.example.models.models.TreeKey" + suffix + ".Maybe",
				"inherited: org.example.models.models.TreeKey" + suffix + ".Maybe? = null",
				"kotlin.collections.List<org.example.models.models.TreeKey" + suffix + ".Node>",
			} {
				if !strings.Contains(consumer, want) {
					t.Fatal("canonical/nullable consumer", want, consumer)
				}
			}
			if !strings.Contains(source(t, r, "SubmitInput"+suffix), "Patch<org.example.models.models.TreeKey"+suffix+".Maybe>") || !strings.Contains(source(t, r, "WatchEvent"+suffix), "data: org.example.models.models.TreeKey"+suffix+".Node") {
				t.Fatal("root alias lost request or typed SSE identity")
			}
		}
	}
}

func rootInternalDTOAliasChain(n int, consumer string) []byte {
	parts := []string{`"Owner":{"properties":{"node":{"properties":{"id":{"type":"string"}},"required":["id"]}}}`}
	for i := range n {
		target := "Owner/properties/node"
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

func TestRootInternalDTOAliasFullHopAndCacheBudgets(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, n := range []int{48, 49} {
			rq := request(rootInternalDTOAliasChain(n, ""), "Dto")
			rq.Options.Validation = flag
			r, err := core.GenerateWithDiagnostics(rq)
			if n == 48 {
				if err != nil || len(r.Artifacts.Files) != 49 {
					t.Fatal("standalone bounded original hops", flag, n, err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(r.Artifacts.Files) != 0 {
				t.Fatal("root alias reset source height", flag, n, r, err)
			}
		}
	}
	for _, n := range []int{46, 47} {
		body := fmt.Sprintf(`{"properties":{"v":{"$ref":"#/components/schemas/A%02d"}}}`, n-1)
		doc := rootInternalDTOAliasChain(n, body)
		if n == 46 {
			report(t, doc, "")
		} else {
			aliasFailure(t, doc, "E_SCHEMA_DEPTH", "/components/schemas/Consumer/properties/v")
		}
	}
	for _, outer := range []int{43, 44} {
		value := `{"$ref":"#/components/schemas/Rows","nullable":false}`
		for range outer {
			value = `{"type":"array","items":` + value + `}`
		}
		doc := schemas(`{"Alias":{"$ref":"#/components/schemas/Owner/properties/node","nullable":true},"Rows":{"type":"array","nullable":true,"items":{"$ref":"#/components/schemas/Alias","nullable":false}},"Consumer":{"properties":{"v":` + value + `}},"Owner":{"properties":{"node":{"properties":{"id":{"type":"string"}}}}}}`)
		if outer == 43 {
			report(t, doc, "")
		} else {
			aliasFailure(t, doc, "E_SCHEMA_DEPTH", "/components/schemas/Consumer/properties/v"+strings.Repeat("/items", outer))
		}
	}
}

func TestRootInternalDTOAliasHierarchyOwnerRetainsActualNestedModel(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/root-hierarchy-aliases.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(input, &doc); err != nil {
		t.Fatal(err)
	}
	catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
	props := catalog["ZFlux"].(map[string]any)["properties"].(map[string]any)
	props["node"] = map[string]any{"properties": map[string]any{"id": map[string]any{"type": "string"}}}
	control, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	report(t, control, "Dto")
	catalog["AliasNode"] = map[string]any{"$ref": "#/components/schemas/ZFlux/properties/node"}
	withAlias, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	// Preserve the original input: actual retained hierarchy Leaf/Nested
	// declarations now authorize its canonical physical owner.
	for _, validation := range []bool{false, true} {
		req := request(withAlias, "Dto")
		req.Options.Validation = validation
		r, e := core.GenerateWithDiagnostics(req)
		if e != nil || !strings.Contains(source(t, r, "AliasNodeDto"), "= org.example.models.models.ZFluxDto.Node\n") || !strings.Contains(source(t, r, "ZFluxDto"), "public data class Node(") {
			t.Fatal(validation, e)
		}
	}
}

func TestRootInternalDTOAliasValidationAndSlots(t *testing.T) {
	for _, n := range []int{240, 241} {
		var fields []string
		for i := range n {
			fields = append(fields, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Alias"}`, i))
		}
		doc := schemas(`{"Alias":{"$ref":"#/components/schemas/Owner/properties/node"},"Consumer":{"properties":{` + strings.Join(fields, ",") + `}},"Owner":{"properties":{"node":{"properties":{"id":{"type":"string","minLength":2}},"required":["id"]}}}}`)
		if n == 240 {
			r := aggregationReport(t, doc)
			if !strings.Contains(source(t, r, "Owner"), "length >= 2") {
				t.Fatal("root alias dropped canonical constructor validation")
			}
		} else {
			aliasFailure(t, doc, "E_MODEL_LIMIT", "/components/schemas/Consumer")
		}
	}
	doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Alias":{"$ref":"#/components/schemas/Owner/properties/node"},"Owner":{"properties":{"node":{"properties":{"id":{"type":"string","minLength":2}}}}}}},"paths":{"/v":{"get":{"parameters":[{"in":"query","name":"v","schema":{"properties":{"node":{"$ref":"#/components/schemas/Alias"}}}}],"responses":{"204":{}}}}}}`)
	aliasFailure(t, doc, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Owner/properties/node/properties/id/minLength")
}

func TestRootInternalDTOAliasExcludedTargetsAndShapeUses(t *testing.T) {
	for _, tc := range []struct{ owner, path string }{
		{`{"properties":{"node":{}}}`, "Owner/properties/node"},
		{`{"properties":{"node":{"properties":{"id":{"type":"string","const":"fixed"}}}}}`, "Owner/properties/node"},
		{`{"properties":{"node":{"allOf":[{"properties":{"id":{"type":"string"}}}]}}}`, "Owner/properties/node"},
		{`{"allOf":[{"properties":{"node":{"properties":{"id":{"type":"string"}}}}}]}`, "Owner/allOf/0/properties/node"},
		{`{"properties":{"node":{"type":"array","items":{"type":"string"}}}}`, "Owner/properties/node"},
	} {
		input := schemas(`{"Alias":` + pointerRef(tc.path) + `,"Owner":` + tc.owner + `}`)
		if strings.Contains(tc.owner, `"node":{"type":"array"`) {
			// Preserve this original negative input as an explicit borrowed-root
			// collection positive; it does not acquire nominal shape permission.
			r := report(t, input, "Dto")
			if !strings.Contains(source(t, r, "AliasDto"), "= kotlin.collections.List<kotlin.String>\n") {
				t.Fatal("borrowed array root backing lost")
			}
			continue
		}
		if tc.path == "Owner/allOf/0/properties/node" || strings.Contains(tc.owner, `"node":{"allOf"`) {
			// Original input graduates only with actual retained constructor ownership.
			r := report(t, input, "Dto")
			if !strings.Contains(source(t, r, "AliasDto"), "= org.example.models.models.OwnerDto.Node\n") {
				t.Fatal("allOf lexical owner lost")
			}
			continue
		}
		aliasFailure(t, input, "E_REF_UNSUPPORTED", "/components/schemas/Alias/$ref")
	}
	owner := `"Alias":{"$ref":"#/components/schemas/Owner/properties/node"},"Owner":{"properties":{"node":{"properties":{"id":{"type":"string"}}}}}`
	// Keep the exact former alias-to-internal allOf input as a positive.
	intersection := report(t, schemas(`{`+owner+`,"Consumer":{"allOf":[{"$ref":"#/components/schemas/Alias"},{"properties":{"extra":{"type":"string"}}}]}}`), "Dto")
	if !strings.Contains(source(t, intersection, "ConsumerDto"), "id: kotlin.String? = null") {
		t.Fatal("original alias/internal allOf fields lost")
	}
	// Explicit graduation of the unchanged chain-to-named-internal-DTO-alias
	// input; both modes must retain its original physical constructor owner.
	doc := schemas(`{` + owner + `,"Extra":{"properties":{"hop":{"$ref":"#/components/schemas/Alias"}}},"Consumer":{"properties":{"v":{"$ref":"#/components/schemas/Extra/properties/hop"}}}}`)
	for _, validation := range []bool{false, true} {
		rq := request(doc, "Dto")
		rq.Options.Validation = validation
		r, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !strings.Contains(source(t, r, "ConsumerDto"), "v: org.example.models.models.OwnerDto.Node? = null") {
			t.Fatal("canonical internal-alias chain graduation", err)
		}
	}
	// The alias retains its original lexical owner through borrowed containers.
	borrowedDTOGraduation(t, schemas(`{`+owner+`,"Extra":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}}}},"Consumer":{"properties":{"v":{"$ref":"#/components/schemas/Extra/properties/rows"}}}}`), "org.example.models.models.OwnerDto.Node")
	// Syntax that later projects to a const-only class must not invent a DTO.
	aliasFailure(t, schemas(`{"Alias":{"$ref":"#/components/schemas/Owner/properties/node"},"Literal":{"type":"string","const":"fixed"},"Owner":{"properties":{"node":{"properties":{"id":{"$ref":"#/components/schemas/Literal"}}}}}}`), "E_REF_UNSUPPORTED", "/components/schemas/Alias/$ref")
}
