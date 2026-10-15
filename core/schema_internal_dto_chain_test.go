package core_test

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestSchemaInternalDTOChainEnvelope(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/schema-internal-dto-chains.json")
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile("../tests/fixtures/http/schema-internal-dtos.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			rq := request(input, suffix)
			rq.Options.Validation = flag
			r, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("finite nominal chain", flag, suffix, err)
			}
			again, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("chain determinism", err)
			}
			old := request(original, suffix)
			old.Options.Validation = flag
			baseline, err := core.GenerateWithDiagnostics(old)
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"Consumer" + suffix, "SubmitInput" + suffix, "DefaultApi", "WatchEvent" + suffix} {
				if source(t, r, name) != source(t, baseline, name) {
					t.Fatal("erased chain changed canonical API/wire plan", name)
				}
			}
		}
	}
}

func dtoChainDocument(hops, outer int, alias bool) []byte {
	props := []string{`"node":{"properties":{"id":{"type":"string"}},"required":["id"]}`}
	next := "node"
	for i := hops - 1; i >= 0; i-- {
		key := fmt.Sprintf("hop%d", i)
		props = append(props, fmt.Sprintf(`%q:%s`, key, pointerRef("Owner/properties/"+next)))
		next = key
	}
	field := pointerRef("Owner/properties/" + next)
	extra := ""
	if alias {
		extra = `"Rows":{"type":"array","items":` + field + `},`
		field = pointerRef("Rows")
	}
	for range outer {
		field = `{"type":"array","items":` + field + `}`
	}
	return schemas(`{"Consumer":{"properties":{"v":` + field + `}},` + extra + `"Owner":{"properties":{` + strings.Join(props, ",") + `}}}`)
}

func TestSchemaInternalDTOChainSourceDepth(t *testing.T) {
	// 46 intermediate refs + incoming ref + property = 48; stop at DTO.
	report(t, dtoChainDocument(46, 0, false), "")
	for _, tc := range []struct {
		hops, outer int
		alias       bool
	}{{47, 0, false}, {46, 1, false}, {44, 1, true}} {
		doc := dtoChainDocument(tc.hops, tc.outer, tc.alias)
		rq := request(doc, "")
		_, err := core.GenerateWithDiagnostics(rq)
		if err == nil {
			t.Fatal("chain/source alias reset depth", tc)
		}
		ds, ok := err.(core.Diagnostics)
		if !ok || ds[0].Code != "E_SCHEMA_DEPTH" {
			t.Fatal(tc, err)
		}
	}
	report(t, dtoChainDocument(43, 1, true), "")
}

func TestSchemaInternalDTOChainNullableAndExclusions(t *testing.T) {
	for _, tc := range []struct {
		first, last, consumer string
		nullable              bool
	}{
		{"", "", "", true}, {`,"nullable":false`, `,"nullable":true`, "", false}, {`,"nullable":true`, `,"nullable":false`, "", true}, {`,"nullable":true`, "", `,"nullable":false`, false},
	} {
		doc := schemas(`{"Consumer":{"required":["v"],"properties":{"v":{"$ref":"#/components/schemas/Owner/properties/first"` + tc.consumer + `}}},"Owner":{"properties":{"first":{"$ref":"#/components/schemas/Owner/properties/last"` + tc.first + `},"last":{"$ref":"#/components/schemas/Owner/properties/node"` + tc.last + `},"node":{"type":["object","null"],"properties":{"id":{"type":"string"}}}}}}`)
		r := report(t, doc, "")
		got := source(t, r, "Consumer")
		expected := "v: org.example.models.models.Owner.Node"
		if tc.nullable {
			expected += "?"
		}
		if !strings.Contains(got, expected+",") && !strings.Contains(got, expected+" = null") {
			t.Fatal(tc, got)
		}
	}
	// A nominal-looking terminal with no constructor is still rejected after planning.
	aliasFailure(t, pointerDoc(`{"$ref":"#/components/schemas/Source/properties/node"}`, pointerRef("Source/properties/leaf")), "E_REF_MISSING", "/components/schemas/Source/properties/leaf/$ref")
	aliasFailure(t, schemas(`{"Consumer":{"properties":{"v":`+pointerRef("Owner/properties/hop")+`}},"Owner":{"properties":{"hop":`+pointerRef("Owner/properties/hop")+`}}}`), "E_REF_UNSUPPORTED", "/components/schemas/Consumer/properties/v")
	// Borrowed containers remain builtin-only even when the typed chain is valid.
	// Original input graduates only to its actual ordinary DTO leaf.
	borrowedDTOGraduation(t, schemas(`{"Consumer":{"properties":{"v":`+pointerRef("Owner/properties/list")+`}},"Owner":{"properties":{"list":{"type":"array","items":`+pointerRef("Owner/properties/hop")+`},"hop":`+pointerRef("Owner/properties/node")+`,"node":{"properties":{"id":{"type":"string"}}}}}}`), "org.example.models.models.OwnerDto.Node")
}
