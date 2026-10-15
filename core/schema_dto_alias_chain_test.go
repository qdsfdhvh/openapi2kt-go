package core_test

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestSchemaDTOAliasChainEnvelope(t *testing.T) {
	for _, named := range []bool{false, true} {
		kind := "internal"
		if named {
			kind = "named"
		}
		input, err := os.ReadFile("../tests/fixtures/http/schema-" + kind + "-dto-alias-chains.json")
		if err != nil {
			t.Fatal(err)
		}
		baseline, err := os.ReadFile("../tests/fixtures/http/schema-" + kind + "-dto-chains.json")
		if err != nil {
			t.Fatal(err)
		}
		for _, validation := range []bool{false, true} {
			for _, suffix := range []string{"", "Dto"} {
				rq := request(input, suffix)
				rq.Options.Validation = validation
				r, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal(kind, validation, suffix, err)
				}
				again, err := core.GenerateWithDiagnostics(rq)
				if err != nil || !reflect.DeepEqual(r, again) {
					t.Fatal("determinism", err)
				}
				old := request(baseline, suffix)
				old.Options.Validation = validation
				before, err := core.GenerateWithDiagnostics(old)
				if err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"Consumer" + suffix, "SubmitInput" + suffix, "DefaultApi", "WatchEvent" + suffix} {
					if source(t, r, name) != source(t, before, name) {
						t.Fatal("alias chain changed canonical API/wire", kind, name)
					}
				}
				target := "Node" + suffix
				if !named {
					target = "TreeKey" + suffix + ".Node"
				}
				if !strings.Contains(source(t, r, "NodeAliasChain"+suffix), "public typealias NodeAliasChain"+suffix+" = org.example.models.models."+target) {
					t.Fatal("copied alias owner")
				}
				for _, canonical := range []string{"RootRows" + suffix + "Inner", "RootMap" + suffix + "Value"} {
					if !strings.Contains(source(t, r, "CollectionAliasUse"+suffix), "org.example.models.models."+canonical) {
						t.Fatal("collection physical alias chain owner", canonical)
					}
				}
			}
		}
	}
}

func dtoAliasChainDocument(internalHops, namedHops, outer int, cached, internalTarget bool) []byte {
	target := "Node"
	if internalTarget {
		target = "Owner/properties/node"
	}
	aliases := []string{}
	for i := 0; i < namedHops; i++ {
		key := fmt.Sprintf("A%02d", i)
		aliases = append(aliases, fmt.Sprintf(`%q:%s`, key, pointerRef(target)))
		target = key
	}
	props := []string{`"node":{"properties":{"id":{"type":"string"}},"required":["id"]}`, `"terminal":` + pointerRef(target)}
	next := "terminal"
	for i := internalHops - 1; i >= 0; i-- {
		key := fmt.Sprintf("hop%d", i)
		props = append(props, fmt.Sprintf(`%q:%s`, key, pointerRef("Owner/properties/"+next)))
		next = key
	}
	field := pointerRef("Owner/properties/" + next)
	extra := ""
	if cached {
		extra = `"Rows":{"type":"array","items":` + field + `},`
		field = pointerRef("Rows")
	}
	for range outer {
		field = `{"type":"array","items":` + field + `}`
	}
	aliases = append(aliases, `"Node":{"properties":{"id":{"type":"string"}},"required":["id"]}`)
	return schemas(`{"Consumer":{"properties":{"v":` + field + `}},` + extra + `"Owner":{"properties":{` + strings.Join(props, ",") + `}},` + strings.Join(aliases, ",") + `}`)
}

func TestSchemaDTOAliasChainDepthAndCache(t *testing.T) {
	for _, nested := range []bool{false, true} {
		for _, cached := range []bool{false, true} {
			hops := 44
			outer := 0
			if cached {
				hops = 41
				outer = 1
			}
			report(t, dtoAliasChainDocument(hops, 1, outer, cached, nested), "")
			for _, more := range []struct{ internal, named, outer int }{{hops + 1, 1, outer}, {hops, 2, outer}, {hops, 1, outer + 1}} {
				_, err := core.GenerateWithDiagnostics(request(dtoAliasChainDocument(more.internal, more.named, more.outer, cached, nested), ""))
				ds, ok := err.(core.Diagnostics)
				if !ok || ds[0].Code != "E_SCHEMA_DEPTH" {
					t.Fatal("lost alias/internal height", nested, cached, more, err)
				}
			}
			// True alias intent followed by nearest internal false must not
			// erase work when a transparent collection caches the narrowed type.
			for _, extra := range []int{0, 1} {
				doc := dtoAliasChainDocument(hops+extra, 1, outer, cached, nested)
				target := "Node"
				if nested {
					target = "Owner/properties/node"
				}
				text := strings.Replace(string(doc), `"A00":`+pointerRef(target), `"A00":{"$ref":"#/components/schemas/`+target+`","nullable":true}`, 1)
				text = strings.Replace(text, `"terminal":`+pointerRef("A00"), `"terminal":{"$ref":"#/components/schemas/A00","nullable":false}`, 1)
				if extra == 0 {
					report(t, []byte(text), "")
					continue
				}
				_, err := core.GenerateWithDiagnostics(request([]byte(text), ""))
				ds, ok := err.(core.Diagnostics)
				if !ok || ds[0].Code != "E_SCHEMA_DEPTH" {
					t.Fatal("nullable aliases erased height", err)
				}
			}
		}
	}
}

func TestSchemaDTOAliasChainNullablePriority(t *testing.T) {
	for _, nested := range []bool{false, true} {
		for _, tc := range []struct {
			root, inner, outer, hop, use string
			nullable                     bool
		}{
			{"", "", "", "", "", false}, {`,"nullable":true`, "", "", "", "", true},
			{`,"nullable":true`, `,"nullable":false`, "", "", "", false},
			{"", `,"nullable":true`, `,"nullable":false`, "", "", false},
			{"", `,"nullable":false`, `,"nullable":true`, "", "", true},
			{"", `,"nullable":true`, "", `,"nullable":false`, "", false},
			{"", "", `,"nullable":false`, `,"nullable":true`, "", true},
			{"", `,"nullable":true`, "", `,"nullable":true`, `,"nullable":false`, false},
		} {
			target := "Node"
			node := `{"type":"object"` + tc.root + `,"properties":{"id":{"type":"string"}}}`
			owner := `"hop":{"$ref":"#/components/schemas/Outer"` + tc.hop + `}`
			if nested {
				target = "Owner/properties/node"
				owner += `,"node":` + node
				node = `{"properties":{"unrelated":{"type":"string"}}}`
			}
			doc := schemas(`{"Consumer":{"required":["v"],"properties":{"v":{"$ref":"#/components/schemas/Owner/properties/hop"` + tc.use + `}}},"Owner":{"properties":{` + owner + `}},"Inner":{"$ref":"#/components/schemas/` + target + `"` + tc.inner + `},"Outer":{"$ref":"#/components/schemas/Inner"` + tc.outer + `},"Node":` + node + `}`)
			text := source(t, report(t, doc, "Dto"), "ConsumerDto")
			want := "v: org.example.models.models.NodeDto"
			if nested {
				want = "v: org.example.models.models.OwnerDto.Node"
			}
			if tc.nullable {
				want += "?"
			}
			if !strings.Contains(text, want+",") && !strings.Contains(text, want+" = null") {
				t.Fatal(nested, tc, text)
			}
		}
	}
}

func TestSchemaDTOAliasChainWidth(t *testing.T) {
	for _, nested := range []bool{false, true} {
		for _, count := range []int{240, 241} {
			fields := make([]string, count)
			for i := range fields {
				fields[i] = fmt.Sprintf(`%q:%s`, fmt.Sprintf("p%d", i), pointerRef("Owner/properties/hop"))
			}
			target := "Node"
			owner := `"hop":` + pointerRef("Alias")
			if nested {
				target = "Owner/properties/node"
				owner += `,"node":{"properties":{"id":{"type":"string"}}}`
			}
			doc := schemas(`{"Consumer":{"properties":{` + strings.Join(fields, ",") + `}},"Owner":{"properties":{` + owner + `}},"Alias":` + pointerRef(target) + `,"Node":{"properties":{"id":{"type":"string"}}}}`)
			if count == 240 {
				report(t, doc, "Dto")
			} else {
				failure(t, doc, "E_MODEL_LIMIT", "/components/schemas/Consumer")
			}
		}
	}
}

func FuzzDTOAliasChainSourceWork(f *testing.F) {
	for _, seed := range []int{0, 41, 43, 44, 45} {
		f.Add(seed, 1, 0, false, false)
	}
	f.Fuzz(func(t *testing.T, hops, named, outer int, cached, nested bool) {
		if hops < 0 || hops > 48 || named < 1 || named > 8 || outer < 0 || outer > 3 {
			t.Skip()
		}
		rq := request(dtoAliasChainDocument(hops, named, outer, cached, nested), "Dto")
		a, ea := core.GenerateWithDiagnostics(rq)
		b, eb := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(ea, eb) {
			t.Fatal("nondeterministic alias chain")
		}
		if ea != nil && len(a.Artifacts.Files) != 0 {
			t.Fatal("failed alias graph produced artifacts")
		}
	})
}

func TestSchemaDTOAliasChainExclusionsAndValidation(t *testing.T) {
	for _, terminal := range []string{`{"type":"string"}`, `{"type":"array","items":{"type":"string"}}`, `{"type":"object"}`, `{"allOf":[{"properties":{"id":{"type":"string"}}}]}`, `{"oneOf":[{"type":"string"},{"type":"boolean"}]}`} {
		doc := schemas(`{"Consumer":{"properties":{"v":` + pointerRef("Owner/properties/hop") + `}},"Owner":{"properties":{"hop":` + pointerRef("Alias") + `}},"Alias":` + pointerRef("Node") + `,"Node":` + terminal + `}`)
		t.Run(terminal, func(t *testing.T) {
			if terminal == `{"type":"string"}` {
				// The unchanged legacy alias remains opt-in; only the new chain graduates.
				r := request(doc, "Dto")
				r.Options.Validation = true
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil || !strings.Contains(source(t, got, "ConsumerDto"), "v: kotlin.String? = null") {
					t.Fatal("builtin alias chain graduation", err)
				}
				failure(t, doc, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Alias")
				return
			}
			if strings.HasPrefix(terminal, `{"allOf"`) || strings.HasPrefix(terminal, `{"oneOf"`) {
				for _, checked := range []bool{false, true} {
					req := request(doc, "Dto")
					req.Options.Validation = checked
					r, e := core.GenerateWithDiagnostics(req)
					if e != nil || !strings.Contains(source(t, r, "ConsumerDto"), "v: org.example.models.models.NodeDto? = null") {
						t.Fatal("allOf named alias terminal graduation", e)
					}
				}
				return
			}
			if terminal == `{"type":"object"}` {
				got := report(t, doc, "Dto")
				if !strings.Contains(source(t, got, "ConsumerDto"), "v: kotlinx.serialization.json.JsonElement? = null") {
					t.Fatal("original raw alias typed chain backing lost")
				}
				return
			}
			if terminal == `{"type":"array","items":{"type":"string"}}` {
				got := report(t, doc, "Dto")
				if !strings.Contains(source(t, got, "ConsumerDto"), "v: kotlin.collections.List<kotlin.String>? = null") {
					t.Fatal("original named collection typed alias chain lost backing")
				}
				return
			}
			aliasFailure(t, doc, "E_REF_UNSUPPORTED", "/components/schemas/Consumer/properties/v")
		})
	}
	// Explicit graduation: preserve the earlier root/internal/named-alias input
	// and assert its original canonical owner in both validation modes.
	bridge := schemas(`{"Alias":` + pointerRef("Owner/properties/hop") + `,"Owner":{"properties":{"hop":` + pointerRef("NodeAlias") + `}},"NodeAlias":` + pointerRef("Node") + `,"Node":{"properties":{"id":{"type":"string"}}}}`)
	for _, validation := range []bool{false, true} {
		rq := request(bridge, "Dto")
		rq.Options.Validation = validation
		r, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !strings.Contains(source(t, r, "AliasDto"), "public typealias AliasDto = org.example.models.models.NodeDto") {
			t.Fatal("root DTO bridge graduation", err)
		}
	}
	// Borrowed containers never enable nominal terminal admission.
	// Original input graduates only to its actual ordinary DTO leaf.
	borrowedDTOGraduation(t, schemas(`{"Consumer":{"properties":{"v":`+pointerRef("Owner/properties/list")+`}},"Owner":{"properties":{"list":{"type":"array","items":`+pointerRef("Owner/properties/hop")+`},"hop":`+pointerRef("Alias")+`}},"Alias":`+pointerRef("Node")+`,"Node":{"properties":{"id":{"type":"string"}}}}`), "org.example.models.models.NodeDto")
	constrained := schemas(`{"Consumer":{"properties":{"v":` + pointerRef("Owner/properties/hop") + `}},"Owner":{"properties":{"hop":` + pointerRef("Alias") + `}},"Alias":` + pointerRef("Node") + `,"Node":{"properties":{"id":{"type":"string","minLength":2}}}}`)
	failure(t, constrained, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Node/properties/id/minLength")
	aggregationReport(t, constrained)
	parameter := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Owner":{"properties":{"hop":` + pointerRef("Alias") + `}},"Alias":` + pointerRef("Node") + `,"Node":{"properties":{"id":{"type":"string","minLength":2}}}}},"paths":{"/v":{"get":{"parameters":[{"in":"query","name":"v","schema":{"properties":{"node":` + pointerRef("Owner/properties/hop") + `}}}],"responses":{"204":{}}}}}}`)
	aliasFailure(t, parameter, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Node/properties/id/minLength")
	// An alias to syntactic object fields that lower to const-only must fail
	// against final declarations, not create a nominal classifier on demand.
	aliasFailure(t, schemas(`{"Consumer":{"properties":{"v":`+pointerRef("Owner/properties/hop")+`}},"Owner":{"properties":{"hop":`+pointerRef("Alias")+`}},"Alias":`+pointerRef("Node")+`,"Node":{"properties":{"id":`+pointerRef("Literal")+`}},"Literal":{"type":"string","const":"fixed"}}`), "E_REF_UNSUPPORTED", "/components/schemas/Consumer/properties/v")
	aliasFailure(t, schemas(`{"Consumer":{"properties":{"v":`+pointerRef("Owner/properties/hop")+`}},"Owner":{"properties":{"hop":`+pointerRef("Alias")+`}},"Alias":`+pointerRef("Alias2")+`,"Alias2":`+pointerRef("Alias")+`}`), "E_REF_CYCLE", "/components/schemas/Alias/$ref")
}
