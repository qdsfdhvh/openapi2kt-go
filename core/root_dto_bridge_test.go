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

func TestRootDTOBridgeEnvelope(t *testing.T) {
	for _, kind := range []string{"named", "internal"} {
		input, e := os.ReadFile("../tests/fixtures/http/root-" + kind + "-dto-bridges.json")
		if e != nil {
			t.Fatal(e)
		}
		control, e := os.ReadFile("../tests/fixtures/http/schema-" + kind + "-dto-alias-chains.json")
		if e != nil {
			t.Fatal(e)
		}
		for _, validation := range []bool{false, true} {
			for _, suffix := range []string{"", "Dto"} {
				rq := request(input, suffix)
				rq.Options.Validation = validation
				r, e := core.GenerateWithDiagnostics(rq)
				if e != nil {
					t.Fatal(kind, validation, suffix, e)
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !reflect.DeepEqual(r, again) {
					t.Fatal("nondeterministic bridge", e)
				}
				rq.Document = control
				before, e := core.GenerateWithDiagnostics(rq)
				if e != nil {
					t.Fatal(e)
				}
				for _, name := range []string{"NodeAlias", "NodeAliasChain", "MaybeAlias", "MaybeAliasChain", "RowAliasChain", "MapAliasChain", "CollectionAliasUse", "Consumer", "SubmitInput", "DefaultApi", "WatchEvent"} {
					tail := suffix
					if name == "DefaultApi" {
						tail = ""
					}
					if source(t, r, name+tail) != source(t, before, name+tail) {
						t.Fatal("changed canonical signature/wire", kind, name, suffix)
					}
				}
			}
		}
	}
}

// incoming root ref + internal hops + named refs all contribute to cached work.
func rootDTOBridgeDocument(internal, named, outer int, cached bool) []byte {
	props := []string{}
	next := "Node"
	aliases := []string{`"Node":{"properties":{"id":{"type":"string"}},"required":["id"]}`}
	for i := 0; i < named; i++ {
		key := fmt.Sprintf("Z%d", i)
		aliases = append(aliases, fmt.Sprintf(`%q:%s`, key, pointerRef(next)))
		next = key
	}
	props = append(props, `"terminal":`+pointerRef(next))
	next = "terminal"
	for i := 0; i < internal; i++ {
		key := fmt.Sprintf("hop%d", i)
		props = append(props, fmt.Sprintf(`%q:%s`, key, pointerRef("Owner/properties/"+next)))
		next = key
	}
	aliases = append(aliases, `"Alias":`+pointerRef("Owner/properties/"+next))
	field := pointerRef("Alias")
	if cached {
		aliases = append(aliases, `"Rows":{"type":"array","items":`+field+`}`)
		field = pointerRef("Rows")
	}
	for range outer {
		field = `{"type":"array","items":` + field + `}`
	}
	return schemas(`{"Consumer":{"properties":{"v":` + field + `}},"Owner":{"properties":{` + strings.Join(props, ",") + `}},` + strings.Join(aliases, ",") + `}`)
}

func TestRootDTOBridgeDepthCacheAndFalseNarrowing(t *testing.T) {
	for _, cached := range []bool{false, true} {
		hops := 43
		if cached {
			hops = 41
		}
		for _, validation := range []bool{false, true} {
			for _, more := range []struct{ internal, named, outer int }{{hops, 1, 0}, {hops + 1, 1, 0}, {hops, 2, 0}, {hops, 1, 1}} {
				for _, narrow := range []bool{false, true} {
					input := rootDTOBridgeDocument(more.internal, more.named, more.outer, cached)
					if narrow {
						s := strings.Replace(string(input), `"Z0":`+pointerRef("Node"), `"Z0":{"$ref":"#/components/schemas/Node","nullable":true}`, 1)
						s = strings.Replace(s, `"terminal":`+pointerRef("Z0"), `"terminal":{"$ref":"#/components/schemas/Z0","nullable":false}`, 1)
						input = []byte(s)
					}
					rq := request(input, "")
					rq.Options.Validation = validation
					_, e := core.GenerateWithDiagnostics(rq)
					if more.internal == hops && more.named == 1 && more.outer == 0 {
						if e != nil {
							t.Fatal(cached, narrow, e)
						}
						continue
					}
					ds, ok := e.(core.Diagnostics)
					if !ok || ds[0].Code != "E_SCHEMA_DEPTH" {
						t.Fatal("bridge erased hops", cached, narrow, more, e)
					}
				}
			}
		}
	}
}

func TestRootDTOBridgeNullablePriority(t *testing.T) {
	for _, tc := range []struct {
		terminal, named, hop, alias, use string
		nullable                         bool
	}{
		{"", "", "", "", "", false}, {`,"nullable":true`, "", "", "", "", true},
		{`,"nullable":true`, `,"nullable":false`, "", "", "", false},
		{"", `,"nullable":true`, `,"nullable":false`, "", "", false},
		{"", `,"nullable":false`, `,"nullable":true`, "", "", true},
		{"", `,"nullable":true`, "", `,"nullable":false`, "", false},
		{"", "", `,"nullable":false`, `,"nullable":true`, "", true},
		{"", "", "", `,"nullable":true`, `,"nullable":false`, false},
		{"", "", `,"nullable":false`, "", `,"nullable":true`, true},
	} {
		for _, validation := range []bool{false, true} {
			doc := schemas(`{"Alias":{"$ref":"#/components/schemas/Owner/properties/hop"` + tc.alias + `},"ZAlias":{"$ref":"#/components/schemas/Node"` + tc.named + `},"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/ZAlias"` + tc.hop + `}}},"Node":{"properties":{"id":{"type":"string"}}` + tc.terminal + `},"Consumer":{"required":["v"],"properties":{"v":{"$ref":"#/components/schemas/Alias"` + tc.use + `}}}}`)
			rq := request(doc, "Dto")
			rq.Options.Validation = validation
			r, e := core.GenerateWithDiagnostics(rq)
			if e != nil {
				t.Fatal(tc, e)
			}
			marker := "v: org.example.models.models.NodeDto"
			if tc.nullable {
				marker += "? = null"
			} else {
				marker += ","
			}
			if !strings.Contains(source(t, r, "ConsumerDto"), marker) {
				t.Fatal("nullable priority", tc, source(t, r, "ConsumerDto"))
			}
		}
	}
}

func TestRootDTOBridgeCycleAndExcludedTargets(t *testing.T) {
	for _, validation := range []bool{false, true} {
		input := schemas(`{"Alias":{"$ref":"#/components/schemas/Owner/properties/hop"},"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/ZAlias"}}},"ZAlias":{"$ref":"#/components/schemas/Alias"}}`)
		rq := request(input, "")
		rq.Options.Validation = validation
		_, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		code := "E_SCHEMA_UNSUPPORTED"
		if validation {
			code = "E_REF_CYCLE"
		}
		if !ok || ds[0].Code != code {
			t.Fatal("mixed transparent cycle", validation, e)
		}
	}
	for _, terminal := range []string{`{"type":"string"}`, `{"type":"object"}`, `{"properties":{"id":{"type":"string","const":"fixed"}}}`, `{"allOf":[{"properties":{"id":{"type":"string"}}}]}`, `{"oneOf":[{"properties":{"id":{"type":"string"}}}]}`, `{"type":"array","items":{"type":"string"}}`} {
		input := schemas(`{"Alias":{"$ref":"#/components/schemas/Owner/properties/hop"},"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/Node"}}},"Node":` + terminal + `}`)
		if terminal == `{"type":"string"}` {
			// Preserve the original input; builtin chains now keep native scalar backing.
			for _, checked := range []bool{false, true} {
				req := request(input, "Dto")
				req.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(req)
				if err != nil || !strings.Contains(source(t, got, "AliasDto"), "public typealias AliasDto = kotlin.String\n") {
					t.Fatal("builtin root bridge graduation", err)
				}
			}
			continue
		}
		if strings.HasPrefix(terminal, `{"allOf"`) || strings.HasPrefix(terminal, `{"oneOf"`) {
			for _, checked := range []bool{false, true} {
				req := request(input, "Dto")
				req.Options.Validation = checked
				r, e := core.GenerateWithDiagnostics(req)
				if e != nil || !strings.Contains(source(t, r, "AliasDto"), "= org.example.models.models.NodeDto\n") {
					t.Fatal("actual allOf terminal graduation", e)
				}
			}
			continue
		}
		if terminal == `{"type":"object"}` {
			// Preserve the original named-empty input; raw authority is the
			// retained zero-property/zero-constant Model, never nominal ownership.
			got := report(t, input, "Dto")
			if !strings.Contains(source(t, got, "AliasDto"), "= kotlinx.serialization.json.JsonElement\n") {
				t.Fatal("original raw root bridge lost backing")
			}
			continue
		}
		if terminal == `{"type":"array","items":{"type":"string"}}` {
			got := report(t, input, "Dto")
			if !strings.Contains(source(t, got, "AliasDto"), "= kotlin.collections.List<kotlin.String>\n") {
				t.Fatal("original named collection root bridge lost backing")
			}
			continue
		}
		aliasFailure(t, input, "E_REF_UNSUPPORTED", "/components/schemas/Alias/$ref")
	}
	base := `"Alias":{"$ref":"#/components/schemas/Owner/properties/hop"},"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/Node"}}},"Node":{"properties":{"id":{"type":"string"}}}`
	// Retain the exact former shape refusal as an explicit named-terminal bridge positive.
	accepted := report(t, schemas(`{`+base+`,"Consumer":{"allOf":[{"$ref":"#/components/schemas/Alias"},{"properties":{"extra":{"type":"string"}}}]}}`), "Dto")
	if !strings.Contains(source(t, accepted, "ConsumerDto"), "id: kotlin.String? = null") {
		t.Fatal("original bridge intersection lost real named fields")
	}
	// Original input graduates only to its actual ordinary DTO leaf.
	borrowedDTOGraduation(t, schemas(`{`+base+`,"Owner2":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}}}},"Consumer":{"properties":{"v":{"$ref":"#/components/schemas/Owner2/properties/rows"}}}}`), "org.example.models.models.NodeDto")
	aliasFailure(t, schemas(`{"Alias":{"$ref":"#/components/schemas/Owner/properties/hop"},"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/Node"}}},"Node":{"properties":{"id":{"$ref":"#/components/schemas/Literal"}}},"Literal":{"type":"string","const":"fixed"}}`), "E_REF_UNSUPPORTED", "/components/schemas/Alias/$ref")
}

func TestRootDTOBridgeHierarchyAndSlots(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/root-hierarchy-aliases.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(input, &doc); err != nil {
		t.Fatal(err)
	}
	catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
	catalog["BridgeOwner"] = map[string]any{"properties": map[string]any{"hop": map[string]any{"$ref": "#/components/schemas/ZFlux"}}}
	catalog["AliasNode"] = map[string]any{"$ref": "#/components/schemas/BridgeOwner/properties/hop"}
	encoded, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	// Keep the original rejected bridge input. Its terminal is an actual
	// named hierarchy leaf and now shares the canonical physical binding.
	for _, validation := range []bool{false, true} {
		req := request(encoded, "Dto")
		req.Options.Validation = validation
		r, e := core.GenerateWithDiagnostics(req)
		if e != nil || !strings.Contains(source(t, r, "AliasNodeDto"), "public typealias AliasNodeDto = org.example.models.models.ZFluxDto\n") || !strings.Contains(source(t, r, "ZFluxDto"), ") : org.example.models.models.ZChannelDto() {") {
			t.Fatal(validation, e)
		}
	}
	for _, count := range []int{240, 241} {
		fields := make([]string, count)
		for i := range fields {
			fields[i] = fmt.Sprintf(`%q:%s`, fmt.Sprintf("p%d", i), pointerRef("Alias"))
		}
		doc := schemas(`{"Alias":{"$ref":"#/components/schemas/Owner/properties/hop"},"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/Node"}}},"Node":{"properties":{"id":{"type":"string","minLength":2}}},"Consumer":{"properties":{` + strings.Join(fields, ",") + `}}}`)
		if count == 240 {
			r := aggregationReport(t, doc)
			if !strings.Contains(source(t, r, "Node"), "length >= 2") {
				t.Fatal("canonical validation lost")
			}
		} else {
			aliasFailure(t, doc, "E_MODEL_LIMIT", "/components/schemas/Consumer")
		}
	}
	parameter := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Alias":{"$ref":"#/components/schemas/Owner/properties/hop"},"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/Node"}}},"Node":{"properties":{"id":{"type":"string","minLength":2}}}}},"paths":{"/v":{"get":{"parameters":[{"in":"query","name":"v","schema":{"properties":{"node":{"$ref":"#/components/schemas/Alias"}}}}],"responses":{"204":{}}}}}}`)
	aliasFailure(t, parameter, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Node/properties/id/minLength")
}

func FuzzRootDTOBridgeWork(f *testing.F) {
	for _, seed := range [][3]int{{0, 0, 0}, {43, 1, 0}, {44, 1, 0}, {41, 1, 0}, {3, 3, 3}} {
		f.Add(uint8(seed[0]), uint8(seed[1]), uint8(seed[2]), false)
	}
	f.Fuzz(func(t *testing.T, h, n, o uint8, cached bool) {
		input := rootDTOBridgeDocument(int(h)%46, int(n)%4, int(o)%4, cached)
		rq := request(input, "")
		first, e := core.GenerateWithDiagnostics(rq)
		second, e2 := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(first, second) || fmt.Sprint(e) != fmt.Sprint(e2) {
			t.Fatal("unstable bridge")
		}
		if e != nil {
			var ds core.Diagnostics
			if !json.Valid(input) || !strings.Contains(fmt.Sprint(e), "E_SCHEMA_DEPTH") {
				t.Fatal("unexpected bounded graph error", ds, e)
			}
		}
	})
}
