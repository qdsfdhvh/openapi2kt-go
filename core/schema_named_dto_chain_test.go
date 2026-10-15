package core_test

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestSchemaNamedDTOChainEnvelope(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/schema-named-dto-chains.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, validation := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			rq := request(input, suffix)
			rq.Options.Validation = validation
			r, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("named DTO chain", err)
			}
			again, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("determinism", err)
			}
			for _, name := range []string{"Consumer" + suffix, "SubmitInput" + suffix, "DefaultApi", "WatchEvent" + suffix} {
				text := source(t, r, name)
				if !strings.Contains(text, "org.example.models.models.") || strings.Contains(text, "TreeKey"+suffix+".Node") {
					t.Fatal("chain must bind canonical named classifier", name, text)
				}
			}
			if !strings.Contains(source(t, r, "Consumer"+suffix), "node: org.example.models.models.Node"+suffix+",") {
				t.Fatal("canonical Node")
			}
			if !strings.Contains(source(t, r, "SubmitInput"+suffix), "Patch<org.example.models.models.Maybe"+suffix+">") {
				t.Fatal("inherited request Patch")
			}
		}
	}
}

func namedDTOChainDocument(hops, outer int, cached bool) []byte {
	props := []string{`"terminal":{"$ref":"#/components/schemas/Node"}`}
	next := "terminal"
	for i := hops - 1; i >= 0; i-- {
		key := fmt.Sprintf("hop%d", i)
		props = append(props, fmt.Sprintf(`%q:%s`, key, pointerRef("Owner/properties/"+next)))
		next = key
	}
	value := pointerRef("Owner/properties/" + next)
	extra := ""
	if cached {
		extra = `"Rows":{"type":"array","items":` + value + `},`
		value = pointerRef("Rows")
	}
	for range outer {
		value = `{"type":"array","items":` + value + `}`
	}
	return schemas(`{"Consumer":{"properties":{"v":` + value + `}},` + extra + `"Owner":{"properties":{` + strings.Join(props, ",") + `}},"Node":{"properties":{"id":{"type":"string"}},"required":["id"]}}`)
}

func TestSchemaNamedDTOChainDepth(t *testing.T) {
	// Property depth 1 + incoming ref + 45 intermediate refs + final named ref = 48.
	report(t, namedDTOChainDocument(45, 0, false), "")
	for _, tc := range []struct {
		hops, outer int
		cached      bool
	}{{46, 0, false}, {45, 1, false}, {43, 1, true}} {
		_, err := core.GenerateWithDiagnostics(request(namedDTOChainDocument(tc.hops, tc.outer, tc.cached), ""))
		if err == nil {
			t.Fatal("lost named terminal/source height", tc)
		}
		ds, ok := err.(core.Diagnostics)
		if !ok || ds[0].Code != "E_SCHEMA_DEPTH" {
			t.Fatal(tc, err)
		}
	}
	report(t, namedDTOChainDocument(42, 1, true), "")
}

func TestSchemaNamedDTOChainWidth(t *testing.T) {
	for _, count := range []int{240, 241} {
		fields := make([]string, count)
		for i := range fields {
			fields[i] = fmt.Sprintf(`%q:%s`, fmt.Sprintf("p%d", i), pointerRef("Owner/properties/hop"))
		}
		doc := schemas(`{"Consumer":{"properties":{` + strings.Join(fields, ",") + `}},"Owner":{"properties":{"hop":` + pointerRef("Node") + `}},"Node":{"properties":{"id":{"type":"string"}}}}`)
		if count == 240 {
			report(t, doc, "Dto")
		} else {
			failure(t, doc, "E_MODEL_LIMIT", "/components/schemas/Consumer")
		}
	}
}

func FuzzNamedDTOChainDepth(f *testing.F) {
	for _, seed := range []int{0, 42, 45, 46, 48} {
		f.Add(seed, 0, false)
	}
	f.Fuzz(func(t *testing.T, hops, outer int, cached bool) {
		if hops < 0 || hops > 50 || outer < 0 || outer > 4 {
			t.Skip()
		}
		rq := request(namedDTOChainDocument(hops, outer, cached), "Dto")
		a, ea := core.GenerateWithDiagnostics(rq)
		b, eb := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(ea, eb) {
			t.Fatal("nondeterministic bounded reference chain")
		}
		if ea != nil && len(a.Artifacts.Files) != 0 {
			t.Fatal("failed chain produced artifacts")
		}
	})
}

func TestSchemaNamedDTOChainNullableAndGuards(t *testing.T) {
	for _, tc := range []struct {
		root, hop, use string
		nullable       bool
	}{
		{"", "", "", false}, {`,"nullable":true`, "", "", true}, {`,"nullable":true`, `,"nullable":false`, "", false},
		{"", `,"nullable":true`, "", true}, {`,"nullable":true`, `,"nullable":true`, `,"nullable":false`, false},
	} {
		doc := schemas(`{"Consumer":{"required":["v"],"properties":{"v":{"$ref":"#/components/schemas/Owner/properties/hop"` + tc.use + `}}},"Owner":{"properties":{"hop":{"$ref":"#/components/schemas/Node"` + tc.hop + `}}},"Node":{"type":"object"` + tc.root + `,"properties":{"id":{"type":"string"}}}}`)
		text := source(t, report(t, doc, ""), "Consumer")
		want := "v: org.example.models.models.Node"
		if tc.nullable {
			want += "?"
		}
		if !strings.Contains(text, want+",") && !strings.Contains(text, want+" = null") {
			t.Fatal(tc, text)
		}
	}
	for _, terminal := range []string{`{"type":"string"}`, `{"type":"array","items":{"type":"string"}}`, `{"type":"object"}`, `{"allOf":[{"properties":{"id":{"type":"string"}}}]}`, `{"$ref":"#/components/schemas/DTO"}`, `{"oneOf":[{"type":"string"},{"type":"boolean"}]}`} {
		doc := schemas(`{"Consumer":{"properties":{"v":` + pointerRef("Owner/properties/hop") + `}},"Owner":{"properties":{"hop":` + pointerRef("Node") + `}},"Node":` + terminal + `,"DTO":{"properties":{"id":{"type":"string"}}}}`)
		t.Run(terminal, func(t *testing.T) {
			if terminal == `{"type":"string"}` {
				// Keep the earlier input and assert the newly admitted builtin backing.
				for _, checked := range []bool{false, true} {
					r := request(doc, "Dto")
					r.Options.Validation = checked
					got, err := core.GenerateWithDiagnostics(r)
					if err != nil || !strings.Contains(source(t, got, "ConsumerDto"), "v: kotlin.String? = null") {
						t.Fatal("named builtin chain graduation", err)
					}
				}
				return
			}
			if strings.HasPrefix(terminal, `{"allOf"`) || strings.HasPrefix(terminal, `{"oneOf"`) {
				// Original source now has an actual retained named allOf constructor.
				for _, checked := range []bool{false, true} {
					req := request(doc, "Dto")
					req.Options.Validation = checked
					r, e := core.GenerateWithDiagnostics(req)
					if e != nil || !strings.Contains(source(t, r, "ConsumerDto"), "v: org.example.models.models.NodeDto? = null") {
						t.Fatal("actual named allOf terminal graduation", e)
					}
				}
				return
			}
			if terminal == `{"$ref":"#/components/schemas/DTO"}` {
				// Explicit graduation of the unchanged named DTO alias terminal.
				for _, validation := range []bool{false, true} {
					rq := request(doc, "Dto")
					rq.Options.Validation = validation
					r, err := core.GenerateWithDiagnostics(rq)
					if err != nil || !strings.Contains(source(t, r, "ConsumerDto"), "v: org.example.models.models.DTODto? = null") {
						t.Fatal("canonical named-alias chain graduation", err)
					}
				}
				return
			}
			if terminal == `{"type":"object"}` {
				got := report(t, doc, "Dto")
				if !strings.Contains(source(t, got, "ConsumerDto"), "v: kotlinx.serialization.json.JsonElement? = null") {
					t.Fatal("original named raw typed chain backing lost")
				}
				return
			}
			if terminal == `{"type":"array","items":{"type":"string"}}` {
				got := report(t, doc, "Dto")
				if !strings.Contains(source(t, got, "ConsumerDto"), "v: kotlin.collections.List<kotlin.String>? = null") {
					t.Fatal("original named collection typed chain lost backing")
				}
				return
			}
			aliasFailure(t, doc, "E_REF_UNSUPPORTED", "/components/schemas/Consumer/properties/v")
		})
	}
	// Syntactic DTOs that become const-only cannot manufacture a classifier.
	aliasFailure(t, schemas(`{"Consumer":{"properties":{"v":`+pointerRef("Owner/properties/hop")+`}},"Owner":{"properties":{"hop":`+pointerRef("Node")+`}},"Literal":{"type":"string","const":"fixed"},"Node":{"properties":{"id":`+pointerRef("Literal")+`}}}`), "E_REF_UNSUPPORTED", "/components/schemas/Consumer/properties/v")
	// Named owner fields and raw projections cannot discard terminal checks.
	constrained := schemas(`{"Consumer":{"properties":{"v":` + pointerRef("Owner/properties/hop") + `}},"Owner":{"properties":{"hop":` + pointerRef("Node") + `}},"Node":{"properties":{"id":{"type":"string","minLength":2}}}}`)
	failure(t, constrained, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Node/properties/id/minLength")
	aggregationReport(t, constrained)
	parameter := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Owner":{"properties":{"hop":` + pointerRef("Node") + `}},"Node":{"properties":{"id":{"type":"string","minLength":2}}}}},"paths":{"/v":{"get":{"parameters":[{"in":"query","name":"v","schema":{"properties":{"node":` + pointerRef("Owner/properties/hop") + `}}}],"responses":{"204":{}}}}}}`)
	aliasFailure(t, parameter, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Node/properties/id/minLength")
	// Borrowed finite collection targets retain their builtin-only role.
	// Original input graduates only to its actual ordinary DTO leaf.
	borrowedDTOGraduation(t, schemas(`{"Consumer":{"properties":{"v":`+pointerRef("Owner/properties/list")+`}},"Owner":{"properties":{"list":{"type":"array","items":`+pointerRef("Owner/properties/hop")+`},"hop":`+pointerRef("Node")+`}},"Node":{"properties":{"id":{"type":"string"}}}}`), "org.example.models.models.NodeDto")
}
