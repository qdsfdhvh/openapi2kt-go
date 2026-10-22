package core_test

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestArrayComponentEnvelopeAndForwardChecks(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/array-components.json")
	if err != nil {
		t.Fatal(err)
	}
	rq := request(input, "Dto")
	rq.Options.Validation = true
	r, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatalf("root arrays should generate: %v", err)
	}
	again, err := core.GenerateWithDiagnostics(rq)
	if err != nil || !reflect.DeepEqual(r, again) {
		t.Fatal("array root determinism", err)
	}
	for _, tc := range []struct{ name, want string }{
		{"ZValuesDto", "public typealias ZValuesDto = kotlin.collections.List<kotlin.String?>"},
		{"RowsDto", "public typealias RowsDto = kotlin.collections.List<RowsDtoInner>"},
		{"StatesDto", "public typealias StatesDto = kotlin.collections.List<StatesDtoInner>"},
		{"ACarrierDto", "values: org.example.models.models.ZValuesDto,"},
		{"ACarrierDto", "length >= 2"},
		{"ACarrierDto", "duplicate entries for uniqueItems"},
		{"ACarrierDto", "kotlin.collections.List<org.example.models.models.ZValuesDto>"},
		{"ACarrierDto", "kotlin.collections.Map<kotlin.String, org.example.models.models.ZValuesDto>"},
		{"RowsDtoInner", "children: org.example.models.models.RowsDto?"},
		{"RowsDtoInner", "has fewer entries than minItems"},
		{"SubmitInputDto", "Patch<org.example.models.models.ZValuesDto>"},
		{"SubmitInputDto", "requiredNull: org.example.models.models.ZValuesDto?,"},
	} {
		if s := source(t, r, tc.name); !strings.Contains(s, tc.want) {
			t.Errorf("missing %q in %s", tc.want, s)
		}
	}
}

func TestArrayComponentOriginalBothFlags(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/p1/invalid/array-component.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []bool{false, true} {
		rq := request(input, "Dto")
		rq.Options.Validation = flag
		r, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(flag, err)
		}
		if len(r.Artifacts.Files) != 1 || len(r.Warnings) != 0 || !strings.Contains(source(t, r, "RecordDto"), "public typealias RecordDto = kotlin.collections.List<kotlin.String>") {
			t.Fatal(r)
		}
	}
}

func TestArrayComponentGuardsAndCycles(t *testing.T) {
	floatingUniqueGraduation(t, schemas(`{"Root":{"type":"array","items":{"type":"number"},"uniqueItems":true}}`))
	for _, tc := range []struct{ schema, code, at string }{
		{`{"type":"array"}`, "E_SCHEMA_INVALID", "/components/schemas/Root"},
		{`{"type":"array","items":false}`, "E_SCHEMA_INVALID", "/components/schemas/Root/items"},
		{`{"type":"array","items":{"type":"string"},"minItems":-1}`, "E_SCHEMA_INVALID", "/components/schemas/Root/minItems"},
		{`{"type":"array","items":{"type":"string","minLength":-1}}`, "E_SCHEMA_INVALID", "/components/schemas/Root/items/minLength"},
		{`{"type":"array","items":{"$ref":"#/components/schemas/Root"}}`, "E_REF_CYCLE", "/components/schemas/Root"},
	} {
		aliasFailure(t, schemas(`{"Root":`+tc.schema+`}`), tc.code, tc.at)
	}
	aliasFailure(t, schemas(`{"A":{"type":"array","items":{"$ref":"#/components/schemas/B"}},"B":{"additionalProperties":{"$ref":"#/components/schemas/A"}}}`), "E_REF_CYCLE", "/components/schemas/A")
	// Original document-root array is invalid JSON input shape, not a schema component.
	failure(t, []byte(`[]`), "E_DOCUMENT_INVALID", "")
	failure(t, schemas(`{"Root":{"type":"array","minItems":1,"items":{"type":"string"}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Root/minItems")
	for _, consumer := range []string{
		`{"allOf":[{"properties":{"v":{"$ref":"#/components/schemas/ZArray"}}},{"properties":{"v":{"type":"string"}}}]}`,
		`{"properties":{"v":{"type":"string"}},"additionalProperties":{"$ref":"#/components/schemas/ZArray"}}`,
	} {
		aliasFailure(t, schemas(`{"Consumer":`+consumer+`,"ZArray":{"type":"array","minItems":1,"items":{"type":"string"}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/ZArray/minItems")
	}
}

func TestArrayComponentPeersAndSchemaDepth(t *testing.T) {
	failure(t, schemas(`{"Root":{"type":"array","items":{"properties":{"id":{"type":"string"}}}},"RootInner":{"properties":{"v":{"type":"string"}}}}`), "E_NAME_COLLISION", "/components/schemas/Root/items")
	for _, n := range []int{48, 49} {
		root := `{"type":"string"}`
		for range n {
			root = `{"type":"array","items":` + root + `}`
		}
		doc := schemas(`{"Root":` + root + `}`)
		if n == 48 {
			report(t, doc, "")
		} else {
			failure(t, doc, "E_SCHEMA_DEPTH", "/components/schemas/Root"+strings.Repeat("/items", n))
		}
	}
}

func TestArrayComponentForwardDictionaryHints(t *testing.T) {
	doc := schemas(`{"AArray":{"type":"array","items":{"$ref":"#/components/schemas/ZDictionary"}},"Consumer":{"properties":{"v":{"$ref":"#/components/schemas/AArray"}}},"ZDictionary":{"type":"object","additionalProperties":{"type":"string","minLength":1}}}`)
	r := aggregationReport(t, doc)
	if !strings.Contains(source(t, r, "Consumer"), "private fun openapiKtValidatePrimitive") {
		t.Fatal("forward aliases must be bound before root array hint propagation")
	}
}

func TestArrayComponentSlotsAndCheckedDepth(t *testing.T) {
	for _, n := range []int{240, 241} {
		var props []string
		for i := range n {
			props = append(props, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/ZArray"}`, i))
		}
		doc := schemas(`{"Consumer":{"properties":{` + strings.Join(props, ",") + `}},"ZArray":{"type":"array","minItems":1,"items":{"type":"string","minLength":1}}}`)
		if n == 240 {
			r := aggregationReport(t, doc)
			if count := strings.Count(source(t, r, "Consumer"), "private fun openapiKtValidatePrimitive"); count != n {
				t.Fatalf("wide alias consumers need %d bounded check methods, got %d", n, count)
			}
		} else {
			aliasFailure(t, doc, "E_MODEL_LIMIT", "/components/schemas/Consumer")
		}
	}
	for _, n := range []int{16, 17} {
		field := `{"$ref":"#/components/schemas/ZArray"}`
		for range n - 1 {
			field = `{"type":"array","items":` + field + `}`
		}
		doc := schemas(`{"Consumer":{"properties":{"v":` + field + `}},"ZArray":{"type":"array","minItems":1,"items":{"type":"string","minLength":1}}}`)
		if n == 16 {
			r := aggregationReport(t, doc)
			if count := strings.Count(source(t, r, "Consumer"), "private fun openapiKtValidatePrimitive"); count != 1 {
				t.Fatalf("deep alias consumer needs one bounded check method, got %d", count)
			}
		} else {
			aliasFailure(t, doc, "E_SCHEMA_DEPTH", "/components/schemas/ZArray")
		}
	}
}
