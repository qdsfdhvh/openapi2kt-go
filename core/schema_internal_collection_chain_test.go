package core_test

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestSchemaInternalCollectionChainEnvelope(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/schema-internal-collection-chains.json")
	if err != nil {
		t.Fatal(err)
	}
	rq := request(input, "Dto")
	rq.Options.Validation = true
	a, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatalf("finite collection chains should generate: %v", err)
	}
	b, err := core.GenerateWithDiagnostics(rq)
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("collection chains changed on rerun", err)
	}
	for _, want := range []string{
		"values: kotlin.collections.List<kotlin.String?>,",
		"entries: kotlin.collections.Map<kotlin.String, kotlin.collections.List<kotlin.String?>>",
		"inherited: kotlin.collections.List<kotlin.String?>?",
		"mixed: kotlin.collections.List<kotlin.collections.Map<kotlin.String, kotlin.collections.List<kotlin.String?>>?>?",
		"length >= 2", "duplicate entries for uniqueItems", "has fewer entries than minProperties",
	} {
		if s := source(t, a, "RecordDto"); !strings.Contains(s, want) {
			t.Errorf("missing %q: %s", want, s)
		}
	}
	for _, want := range []string{
		"inherited: org.example.models.models.Patch<kotlin.collections.List<kotlin.String?>>",
		"wrapped: org.example.models.models.Patch<kotlin.collections.List<kotlin.String?>>",
		"requiredNull: kotlin.collections.List<kotlin.String?>?,",
		"requiredFalse: kotlin.collections.List<kotlin.String?>,",
	} {
		if s := source(t, a, "SubmitInputDto"); !strings.Contains(s, want) {
			t.Errorf("missing presence %q: %s", want, s)
		}
	}
}

func TestSchemaInternalCollectionChainBothFlags(t *testing.T) {
	// Explicit graduation of the historical scalar-chain -> array negative.
	for _, flag := range []bool{false, true} {
		doc := schemas(`{"Consumer":{"required":["v"],"properties":{"v":` + pointerRef("Source/properties/hop") + `}},"Source":{"properties":{"hop":` + pointerRef("Source/properties/list") + `,"list":{"type":"array","items":{"type":"string"}}}}}`)
		rq := request(doc, "")
		rq.Options.Validation = flag
		r, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(flag, err)
		}
		if s := source(t, r, "Consumer"); !strings.Contains(s, "v: kotlin.collections.List<kotlin.String>,") {
			t.Fatal(s)
		}
	}
}

func collectionChainDocument(hops, outer int) []byte {
	props := []string{`"terminal":{"type":"array","items":{"type":"string"}}`}
	next := "terminal"
	for i := hops - 1; i >= 0; i-- {
		key := fmt.Sprintf("hop%d", i)
		props = append(props, fmt.Sprintf(`%q:%s`, key, pointerRef("Source/properties/"+next)))
		next = key
	}
	field := pointerRef("Source/properties/" + next)
	for range outer {
		field = `{"type":"array","items":` + field + `}`
	}
	return schemas(`{"Consumer":{"properties":{"v":` + field + `}},"Source":{"properties":{` + strings.Join(props, ",") + `}}}`)
}

func TestSchemaInternalCollectionChainDepthAndCheckedEdges(t *testing.T) {
	for _, flag := range []bool{false, true} {
		rq := request(collectionChainDocument(45, 0), "")
		rq.Options.Validation = flag
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("48 schema boundary", flag, err)
		}
	}
	aliasFailure(t, collectionChainDocument(46, 0), "E_SCHEMA_DEPTH", "/components/schemas/Source/properties/terminal/items")
	aliasFailure(t, collectionChainDocument(45, 1), "E_SCHEMA_DEPTH", "/components/schemas/Source/properties/terminal/items")
	for _, n := range []int{16, 17} {
		props := []string{`"leaf":{"type":"string","minLength":1}`}
		next := "leaf"
		for i := n - 1; i >= 0; i-- {
			key := fmt.Sprintf("list%d", i)
			props = append(props, fmt.Sprintf(`%q:{"type":"array","minItems":1,"items":%s}`, key, pointerRef("Source/properties/"+next)))
			next = key
		}
		doc := schemas(`{"Consumer":{"properties":{"v":` + pointerRef("Source/properties/"+next) + `}},"Source":{"properties":{` + strings.Join(props, ",") + `}}}`)
		if n == 16 {
			aggregationReport(t, doc)
		} else {
			aliasFailure(t, doc, "E_SCHEMA_DEPTH", "/components/schemas/Source/properties/list16")
		}
	}
}

func TestSchemaInternalCollectionChainSlotsAndOverriddenChecks(t *testing.T) {
	for _, n := range []int{240, 241} {
		var props []string
		for i := range n {
			props = append(props, fmt.Sprintf(`"p%d":%s`, i, pointerRef("Source/properties/hop")))
		}
		doc := schemas(`{"Consumer":{"properties":{` + strings.Join(props, ",") + `}},"Source":{"properties":{"hop":` + pointerRef("Source/properties/list") + `,"list":{"type":"array","minItems":1,"items":{"type":"string","minLength":1}}}}}`)
		if n == 240 {
			aggregationReport(t, doc)
		} else {
			aliasFailure(t, doc, "E_MODEL_LIMIT", "/components/schemas/Consumer")
		}
	}
	for _, consumer := range []string{
		`{"allOf":[{"properties":{"v":` + pointerRef("Source/properties/hop") + `}},{"properties":{"v":{"type":"string"}}}]}`,
		`{"properties":{"v":{"type":"string"}},"additionalProperties":` + pointerRef("Source/properties/hop") + `}`,
	} {
		doc := schemas(`{"Consumer":` + consumer + `,"Source":{"properties":{"hop":` + pointerRef("Source/properties/list") + `,"list":{"type":"array","minItems":1,"items":{"type":"string"}}}}}`)
		aliasFailure(t, doc, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Source/properties/list/minItems")
	}
}

func TestSchemaInternalCollectionChainGuardsAndSources(t *testing.T) {
	for _, tc := range []struct{ target, code, at string }{
		{`{"type":"array","items":` + pointerRef("Source/properties/hop") + `}`, "E_REF_UNSUPPORTED", "/components/schemas/Source/properties/target/items"},
		{`{"additionalProperties":` + pointerRef("Source/properties/hop") + `}`, "E_REF_UNSUPPORTED", "/components/schemas/Source/properties/target/additionalProperties"},
		{`{"type":"array","items":` + pointerRef("Source/properties/missing") + `}`, "E_REF_MISSING", "/components/schemas/Source/properties/target/items"},
		{`{"type":"array","minItems":-1,"items":{"type":"string"}}`, "E_SCHEMA_INVALID", "/components/schemas/Source/properties/target/minItems"},
		{`{"type":"array","items":{"type":"string","minLength":-1}}`, "E_SCHEMA_INVALID", "/components/schemas/Source/properties/target/items/minLength"},
		{`{"type":"array","items":` + pointerRef("Source") + `}`, "E_REF_UNSUPPORTED", "/components/schemas/Source/properties/target/items"},
	} {
		doc := schemas(`{"Consumer":{"properties":{"v":` + pointerRef("Source/properties/hop") + `}},"Source":{"properties":{"hop":` + pointerRef("Source/properties/target") + `,"target":` + tc.target + `}}}`)
		if tc.target == `{"type":"array","items":`+pointerRef("Source")+`}` {
			borrowedDTOGraduation(t, doc, "org.example.models.models.SourceDto")
			continue
		}
		aliasFailure(t, doc, tc.code, tc.at)
	}
	// New transparent hops must not turn unchecked raw projections into validators.
	prefix := `{"openapi":"3.1.0","components":{"schemas":{"Source":{"properties":{"hop":` + pointerRef("Source/properties/list") + `,"list":{"type":"array","minItems":1,"items":{"type":"string"}}}}}},"paths":`
	paths := `{"/v":{"get":{"parameters":[{"in":"query","name":"v","schema":{"properties":{"value":` + pointerRef("Source/properties/hop") + `}}}],"responses":{"204":{}}}}}`
	aliasFailure(t, []byte(prefix+paths+`}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Source/properties/list/minItems")
	failure(t, schemas(`{"Consumer":{"properties":{"v":`+pointerRef("Source/properties/hop")+`}},"Source":{"properties":{"hop":`+pointerRef("Source/properties/list")+`,"list":{"type":"array","minItems":1,"items":{"type":"string"}}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Source/properties/list/minItems")
}
