package core_test

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestSchemaInternalCollectionPointerQualifiedEnvelope(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/schema-internal-collection-pointers.json")
	if err != nil {
		t.Fatal(err)
	}
	rq := request(input, "Dto")
	rq.Options.Validation = true
	a, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatalf("desired success for complete collection envelope: %v", err)
	}
	b, err := core.GenerateWithDiagnostics(rq)
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("nondeterministic collection borrowing", err)
	}
	record := source(t, a, "RecordDto")
	for _, want := range []string{"values: kotlin.collections.List<kotlin.String?>", "entries: kotlin.collections.Map<kotlin.String, kotlin.Int?>", "inherited: kotlin.collections.List<kotlin.String?>?", "has fewer entries than minItems", "has more entries than maxProperties", "duplicate entries for uniqueItems", "length >= 2", "generatedSerializer()"} {
		if !strings.Contains(record, want) {
			t.Errorf("missing %q in %s", want, record)
		}
	}
	patch := source(t, a, "PatchCollectionsInputDto")
	if !strings.Contains(patch, "Patch<kotlin.collections.List<kotlin.String?>>") || !strings.Contains(patch, "requiredNull: kotlin.collections.List<kotlin.String?>?") {
		t.Fatal("borrowed inherited Patch/required nullable", patch)
	}
}

// This is precisely the original T1 array-negative input, now covered under
// both flags; only its old table row may graduate after baseline qualification.
func TestSchemaInternalCollectionPointerPreciseArrayBothFlags(t *testing.T) {
	input := pointerDoc(`{"type":"array","items":{"type":"string"}}`, pointerRef("Source/properties/leaf"))
	for _, validation := range []bool{false, true} {
		rq := request(input, "")
		rq.Options.Validation = validation
		r, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatalf("exact unbounded target validation=%t: %v", validation, err)
		}
		if s := source(t, r, "Consumer"); !strings.Contains(s, "v: kotlin.collections.List<kotlin.String>?") {
			t.Fatal(s)
		}
	}
}

func TestSchemaInternalCollectionPointerGrammarAndSources(t *testing.T) {
	at := "/components/schemas/Source/properties/leaf"
	use := "/components/schemas/Consumer/properties/v"
	for _, tc := range []struct{ target, code, pointer string }{
		{`{"type":"array"}`, "E_SCHEMA_INVALID", at},
		{`{"type":"array","items":false}`, "E_SCHEMA_INVALID", at + "/items"},
		{`{"type":"array","items":{"type":"string"},"minItems":-1}`, "E_SCHEMA_INVALID", at + "/minItems"},
		{`{"type":"array","items":{"type":"string"},"minItems":3,"maxItems":2}`, "E_SCHEMA_INVALID", at + "/maxItems"},
		{`{"type":"array","items":{"type":"string"},"nullable":"no"}`, "E_SCHEMA_INVALID", at + "/nullable"},
		{`{"type":"array","items":{"type":"string","minLength":-1}}`, "E_SCHEMA_INVALID", at + "/items/minLength"},
		{`{"additionalProperties":{"type":"string"},"required":["x"]}`, "E_SCHEMA_INVALID", at + "/required"},
		{`{"type":"array","items":{"type":"string"},"default":[]}`, "E_SCHEMA_UNSUPPORTED", at + "/default"},
		{`{"type":"array","items":{"type":"string"},"prefixItems":[]}`, "E_SCHEMA_UNSUPPORTED", at + "/prefixItems"},
		{`{"additionalProperties":true}`, "E_REF_UNSUPPORTED", use},
		{`{"additionalProperties":false}`, "E_REF_UNSUPPORTED", use},
		{`{"additionalProperties":{}}`, "E_REF_UNSUPPORTED", use},
		{`{"type":"array","items":{"type":"string","enum":["ok"]}}`, "E_REF_UNSUPPORTED", use},
		{`{"type":"array","items":{"type":"string","const":"ok"}}`, "E_REF_UNSUPPORTED", use},
		{`{"type":"array","items":{"type":"string","format":"date"}}`, "E_REF_UNSUPPORTED", use},
		{`{"type":"array","items":{"type":"null"}}`, "E_REF_UNSUPPORTED", use},
		{`{"type":"array","items":{"type":"number"},"uniqueItems":true}`, "E_SCHEMA_UNSUPPORTED", at + "/uniqueItems"},
		{`{"type":"array","items":{"type":"array","items":{"type":"integer"}},"uniqueItems":true}`, "E_SCHEMA_UNSUPPORTED", at + "/uniqueItems"},
		{`{"type":"array","items":{"type":"integer"},"uniqueItems":"no"}`, "E_SCHEMA_INVALID", at + "/uniqueItems"},
		{`{"type":"array","items":{"$ref":"#/components/schemas/Source/properties/leaf"}}`, "E_REF_UNSUPPORTED", at + "/items"},
		{`{"type":"array","items":{"$ref":"#/components/schemas/Source"}}`, "E_REF_UNSUPPORTED", at + "/items"},
	} {
		t.Run(tc.target, func(t *testing.T) {
			if tc.target == `{"type":"array","items":{"$ref":"#/components/schemas/Source"}}` {
				borrowedDTOGraduation(t, pointerDoc(tc.target, pointerRef("Source/properties/leaf")), "org.example.models.models.SourceDto")
				return
			}
			aliasFailure(t, pointerDoc(tc.target, pointerRef("Source/properties/leaf")), tc.code, tc.pointer)
		})
	}
	failure(t, pointerDoc(`{"type":"array","minItems":1,"items":{"type":"string"}}`, pointerRef("Source/properties/leaf")), "E_SCHEMA_UNSUPPORTED", at+"/minItems")
	// False uniqueness is no enforcing check, not permission for invalid children.
	aggregationReport(t, pointerDoc(`{"type":"array","items":{"type":"number"},"uniqueItems":false}`, pointerRef("Source/properties/leaf")))
	aliasFailure(t, pointerDoc(`{"type":"array","items":{"type":"string","enum":["ok"]},"uniqueItems":false}`, pointerRef("Source/properties/leaf")), "E_REF_UNSUPPORTED", use)
}

func TestSchemaInternalCollectionPointerDepthAndSlots(t *testing.T) {
	for _, depth := range []int{48, 49} {
		field := pointerRef("Source/properties/leaf")
		for range depth - 3 {
			field = `{"type":"array","items":` + field + `}`
		}
		input := pointerDoc(`{"type":"array","items":{"type":"string"}}`, field)
		if depth == 48 {
			report(t, input, "")
		} else {
			failure(t, input, "E_SCHEMA_DEPTH", "/components/schemas/Source/properties/leaf/items")
		}
	}
	for _, depth := range []int{16, 17} {
		field := pointerRef("Source/properties/leaf")
		for range depth - 1 {
			field = `{"type":"array","items":` + field + `}`
		}
		input := pointerDoc(`{"type":"array","minItems":1,"items":{"type":"string","minLength":1}}`, field)
		if depth == 16 {
			aggregationReport(t, input)
		} else {
			aliasFailure(t, input, "E_SCHEMA_DEPTH", "/components/schemas/Source/properties/leaf")
		}
	}
	for _, n := range []int{240, 241} {
		var props []string
		for i := range n {
			props = append(props, fmt.Sprintf(`"p%d":%s`, i, pointerRef("Source/properties/leaf")))
		}
		input := schemas(`{"Consumer":{"properties":{` + strings.Join(props, ",") + `}},"Source":{"properties":{"leaf":{"type":"array","minItems":1,"items":{"type":"string","minLength":1}}}}}`)
		if n == 240 {
			aggregationReport(t, input)
		} else {
			aliasFailure(t, input, "E_MODEL_LIMIT", "/components/schemas/Consumer")
		}
	}
}

func TestSchemaInternalCollectionPointerHTTPNoDropAndSelection(t *testing.T) {
	prefix := `{"openapi":"3.1.0","components":{"schemas":{"Source":{"properties":{"leaf":{"type":"array","minItems":1,"items":{"type":"string"}}}}}},"paths":`
	for _, paths := range []string{
		`{"/v":{"get":{"parameters":[{"in":"query","name":"v","schema":{"properties":{"values":{"$ref":"#/components/schemas/Source/properties/leaf"}}}}],"responses":{"204":{}}}}}`,
		`{"/v":{"post":{"requestBody":{"content":{"application/json":{"schema":{"type":"array","items":{"properties":{"values":{"$ref":"#/components/schemas/Source/properties/leaf"}}}}}}},"responses":{"204":{}}}}}`,
	} {
		aliasFailure(t, []byte(prefix+paths+`}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Source/properties/leaf/minItems")
	}
	// An excluded API and unselected JSON media cannot conceal bad target checks.
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Source":{"properties":{"leaf":{"type":"array","minItems":-1,"items":{"type":"string"}}}}}},"paths":{"/v":{"get":{"tags":["Ignored"],"responses":{"200":{"content":{"application/json":{"schema":{"type":"string"}},"application/problem+json":{"schema":{"$ref":"#/components/schemas/Source/properties/leaf"}}}}}}}}}`)
	rq := request(input, "")
	rq.Options.Validation = true
	rq.Options.ExcludeAPIs = []string{"IgnoredApi"}
	r, err := core.GenerateWithDiagnostics(rq)
	ds, ok := err.(core.Diagnostics)
	if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || *ds[0].Pointer != "/components/schemas/Source/properties/leaf/minItems" || len(r.Artifacts.Files) != 0 || len(r.Warnings) != 0 {
		t.Fatal("unselected/excluded checks dropped", r, err)
	}
}

func TestSchemaInternalCollectionPointerNoDropAndHelperRetention(t *testing.T) {
	for _, consumer := range []string{
		`{"allOf":[{"properties":{"v":{"$ref":"#/components/schemas/Source/properties/leaf"}}},{"properties":{"v":{"type":"string"}}}]}`,
		`{"properties":{"v":{"type":"string"}},"additionalProperties":{"$ref":"#/components/schemas/Source/properties/leaf"}}`,
	} {
		aliasFailure(t, schemas(`{"Consumer":`+consumer+`,"Source":{"properties":{"leaf":{"type":"array","minItems":1,"items":{"type":"string"}}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Source/properties/leaf/minItems")
	}
	wrap := func(field string) string { return `{"properties":{"nested":{"properties":{"v":` + field + `}}}}` }
	direct := `{"type":"array","minItems":1,"maxItems":3,"uniqueItems":true,"items":{"type":"string","minLength":1}}`
	internal := pointerRef("Source/properties/leaf")
	for _, reverse := range []bool{false, true} {
		a, b := direct, internal
		if reverse {
			a, b = b, a
		}
		input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Source":{"properties":{"leaf":` + direct + `}}}},"paths":{"/a":{"post":{"operationId":"same","requestBody":{"content":{"application/json":{"schema":` + wrap(a) + `}}},"responses":{"204":{}}}},"/b":{"put":{"operationId":"same","requestBody":{"content":{"application/json":{"schema":` + wrap(b) + `}}},"responses":{"204":{}}}}}}`)
		text := source(t, aggregationReport(t, input), "SameInput")
		if strings.Count(text, "private fun openapiKtValidatePrimitive") != 1 || !strings.Contains(text, "duplicate entries for uniqueItems") || !strings.Contains(text, "value: kotlin.collections.List<kotlin.String>?") {
			t.Fatal("typed leaf hint OR retention", reverse, text)
		}
	}
}
