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

func TestSchemaInternalScalarPointerQualifiedEnvelope(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/schema-internal-scalar-pointers-qualified.json")
	if err != nil {
		t.Fatal(err)
	}
	rq := request(input, "Dto")
	rq.Options.Validation = true
	a, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatalf("desired success for frozen qualified envelope: %v", err)
	}
	b, err := core.GenerateWithDiagnostics(rq)
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("nondeterministic borrowed scalar generation", err)
	}
	record := source(t, a, "RecordDto")
	for _, want := range []string{"value: kotlin.String", "nonNull: kotlin.String", "nullable: kotlin.String?", "kotlin.Int", "kotlin.Long", "kotlin.Float", "kotlin.Double", "kotlin.Boolean", "generatedSerializer()", "length >= 2", "length <= 3"} {
		if !strings.Contains(record, want) {
			t.Errorf("missing %q in %s", want, record)
		}
	}
}

func pointerDoc(target, use string) []byte {
	return schemas(`{"Consumer":{"type":"object","properties":{"v":` + use + `}},"Source":{"type":"object","properties":{"leaf":` + target + `}}}`)
}
func pointerRef(path string) string { return `{"$ref":"#/components/schemas/` + path + `"}` }

func TestSchemaInternalScalarPointerAdmissionAndSource(t *testing.T) {
	use := "/components/schemas/Consumer/properties/v"
	for _, tc := range []struct{ target, path, code, at string }{
		{`{"type":"string"}`, "Missing/properties/leaf", "E_REF_MISSING", use},
		{`{"type":"string"}`, "Source/properties/missing", "E_REF_MISSING", use},
		{`{"type":"string"}`, "Source/properties/leaf/default", "E_REF_MISSING", use},
		{`{"type":"string","example":{"type":"string"}}`, "Source/properties/leaf/example", "E_REF_UNSUPPORTED", use},
		{`{"type":"string"}`, "Source/properties", "E_REF_UNSUPPORTED", use},
		{`{"type":"object","properties":{"id":{"type":"string"}}}`, "Source/properties/leaf", "E_REF_UNSUPPORTED", use},
		{`{"type":"string","enum":["ok"]}`, "Source/properties/leaf", "E_REF_UNSUPPORTED", use},
		{`{"type":"string","const":"ok"}`, "Source/properties/leaf", "E_REF_UNSUPPORTED", use},
		{`{"type":"string","format":"uuid"}`, "Source/properties/leaf", "E_REF_UNSUPPORTED", use},
		{`{"type":"string","format":"date-time"}`, "Source/properties/leaf", "E_REF_UNSUPPORTED", use},
		{`{"type":"integer","x-native-type":"bigint"}`, "Source/properties/leaf", "E_REF_UNSUPPORTED", use},
		{`{"$ref":"#/components/schemas/Source"}`, "Source/properties/leaf", "E_REF_UNSUPPORTED", use},
		{`{"type":"string"}`, "Source/properties/le~2af", "E_SCHEMA_INVALID", use},
		{`{"type":"string"}`, "Source/properties/leaf%20", "E_REF_UNSUPPORTED", use},
		{`{"oneOf":[{"type":"string"},{"type":"null"}]}`, "Source/properties/leaf/oneOf/01", "E_SCHEMA_INVALID", use},
		{`{"oneOf":[{"type":"string"},{"type":"null"}]}`, "Source/properties/leaf/oneOf/-", "E_SCHEMA_INVALID", use},
		{`{"oneOf":[{"type":"string"},{"type":"null"}]}`, "Source/properties/leaf/oneOf/2", "E_REF_MISSING", use},
		{`{"oneOf":[{"type":"string"},{"type":"null"}]}`, "Source/properties/leaf/oneOf/9999999999999999999999999999", "E_REF_MISSING", use},
		{`{"type":"string","minLength":-1}`, "Source/properties/leaf", "E_SCHEMA_INVALID", "/components/schemas/Source/properties/leaf/minLength"},
		{`{"type":"integer","minimum":true}`, "Source/properties/leaf", "E_SCHEMA_INVALID", "/components/schemas/Source/properties/leaf/minimum"},
		{`{"type":"string","nullable":"no"}`, "Source/properties/leaf", "E_SCHEMA_INVALID", "/components/schemas/Source/properties/leaf/nullable"},
	} {
		t.Run(tc.path+tc.target, func(t *testing.T) {
			if tc.target == `{"type":"string","format":"uuid"}` {
				// Graduate the exact historical input at its original source role.
				for _, checked := range []bool{false, true} {
					rq := request(pointerDoc(tc.target, pointerRef(tc.path)), "Dto")
					rq.Options.Validation = checked
					got, err := core.GenerateWithDiagnostics(rq)
					if err != nil || !strings.Contains(source(t, got, "ConsumerDto"), "public val v: kotlin.uuid.Uuid? = null") {
						t.Fatal("borrowed UUID did not retain native backing", checked, err)
					}
				}
				return
			}
			// Explicit internal-DTO graduation: retain the original input and
			// assert canonical ownership in both modes, not a weakened negative.
			if tc.target == `{"type":"object","properties":{"id":{"type":"string"}}}` {
				for _, flag := range []bool{false, true} {
					rq := request(pointerDoc(tc.target, pointerRef(tc.path)), "Dto")
					rq.Options.Validation = flag
					r, err := core.GenerateWithDiagnostics(rq)
					if err != nil || len(r.Artifacts.Files) != 2 {
						t.Fatal("canonical DTO graduation", flag, err)
					}
					if got := source(t, r, "ConsumerDto"); !strings.Contains(got, "v: org.example.models.models.SourceDto.Leaf? = null") {
						t.Fatal(got)
					}
				}
				return
			}
			if tc.target == `{"$ref":"#/components/schemas/Source"}` {
				// Explicit named-DTO chain graduation, preserving the original
				// self-owner input and nominal recursion in both validation modes.
				for _, flag := range []bool{false, true} {
					rq := request(pointerDoc(tc.target, pointerRef(tc.path)), "Dto")
					rq.Options.Validation = flag
					r, err := core.GenerateWithDiagnostics(rq)
					if err != nil || !strings.Contains(source(t, r, "ConsumerDto"), "v: org.example.models.models.SourceDto? = null") {
						t.Fatal("named ordinary owner graduation", flag, err)
					}
				}
				return
			}
			if tc.target == `{"type":"string","enum":["ok"]}` {
				// Preserve the original enum input and verify its actual owner.
				for _, flag := range []bool{false, true} {
					rq := request(pointerDoc(tc.target, pointerRef(tc.path)), "Dto")
					rq.Options.Validation = flag
					r, err := core.GenerateWithDiagnostics(rq)
					if err != nil || !strings.Contains(source(t, r, "ConsumerDto"), "v: org.example.models.models.SourceDto.Leaf? = null") {
						t.Fatal("nested enum pointer graduation", flag, err)
					}
				}
				return
			}
			aliasFailure(t, pointerDoc(tc.target, pointerRef(tc.path)), tc.code, tc.at)
		})
	}
	input := pointerDoc(`{"type":"string","minLength":2}`, pointerRef("Source/properties/leaf"))
	failure(t, input, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Source/properties/leaf/minLength")
	// Preserve the original root input and explicitly graduate only the borrowed
	// builtin source role. Shape expansion keeps its original rejection below.
	rootAlias := schemas(`{"Alias":{"$ref":"#/components/schemas/Source/properties/leaf"},"Source":{"type":"object","properties":{"leaf":{"type":"string"}}}}`)
	for _, checked := range []bool{false, true} {
		r := request(rootAlias, "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || !strings.Contains(source(t, got, "AliasDto"), "public typealias AliasDto = kotlin.String\n") {
			t.Fatal("original borrowed root input lost canonical scalar backing", err)
		}
	}
	aliasFailure(t, schemas(`{"Consumer":{"allOf":[{"$ref":"#/components/schemas/Source/properties/leaf"},{"properties":{"local":{"type":"string"}}}]},"Source":{"type":"object","properties":{"leaf":{"type":"string"}}}}`), "E_REF_UNSUPPORTED", "/components/schemas/Consumer/allOf/0")
	original, err := os.ReadFile("../tests/fixtures/http/schema-internal-scalar-pointers.json")
	if err != nil {
		t.Fatal(err)
	}
	aliasFailure(t, original, "E_NAME_INVALID", "/components/schemas/Source/properties/")
}

func TestSchemaInternalScalarPointerNullabilityAndWrappers(t *testing.T) {
	for _, nullable := range []string{`"nullable":true`, `"type":["string","null"]`} {
		target := `{"type":"string",` + nullable + `}`
		if strings.HasPrefix(nullable, `"type"`) {
			target = `{` + nullable + `}`
		}
		for _, tc := range []struct {
			use  string
			null bool
		}{
			{pointerRef("Source/properties/leaf"), true},
			{`{"$ref":"#/components/schemas/Source/properties/leaf","nullable":false}`, false},
			{`{"allOf":[` + pointerRef("Source/properties/leaf") + `],"nullable":false}`, false},
			{`{"allOf":[{"$ref":"#/components/schemas/Source/properties/leaf","nullable":false}]}`, false},
			{`{"oneOf":[{"$ref":"#/components/schemas/Source/properties/leaf","nullable":false},{"type":"null"}]}`, true},
		} {
			input := schemas(`{"Consumer":{"type":"object","required":["v"],"properties":{"v":` + tc.use + `}},"Source":{"type":"object","properties":{"leaf":` + target + `}}}`)
			s := source(t, aggregationReport(t, input), "Consumer")
			want := "v: kotlin.String"
			if tc.null {
				want += "?"
			}
			if !strings.Contains(s, want) || (!tc.null && strings.Contains(s, "v: kotlin.String?")) {
				t.Fatal(tc, s)
			}
		}
	}
	// Numeric property keys are object keys, not canonical array indices.
	report(t, schemas(`{"Consumer":{"type":"object","properties":{"v":{"$ref":"#/components/schemas/Source/properties/01"}}},"Source":{"type":"object","properties":{"01":{"type":"boolean"}}}}`), "")
}

func TestSchemaInternalScalarPointerDepthCollectionAndSlots(t *testing.T) {
	for _, depth := range []int{48, 49} {
		field := pointerRef("Source/properties/leaf")
		for i := 0; i < depth-2; i++ {
			field = `{"type":"array","items":` + field + `}`
		}
		input := pointerDoc(`{"type":"string"}`, field)
		if depth == 48 {
			report(t, input, "")
		} else {
			at := "/components/schemas/Consumer/properties/v" + strings.Repeat("/items", depth-2)
			failure(t, input, "E_SCHEMA_DEPTH", at)
		}
	}
	for _, depth := range []int{16, 17} {
		field := pointerRef("Source/properties/leaf")
		for range depth {
			field = `{"type":"array","items":` + field + `}`
		}
		input := pointerDoc(`{"type":"string","minLength":1}`, field)
		if depth == 16 {
			aggregationReport(t, input)
		} else {
			aliasFailure(t, input, "E_SCHEMA_DEPTH", "/components/schemas/Source/properties/leaf")
		}
	}
	for _, tc := range []struct {
		n    int
		wide bool
		ok   bool
	}{{120, true, true}, {121, true, false}, {240, false, true}, {241, false, false}} {
		var props []string
		var required []string
		for i := 0; i < tc.n; i++ {
			props = append(props, fmt.Sprintf(`"p%d":%s`, i, pointerRef("Source/properties/leaf")))
			required = append(required, fmt.Sprintf(`"p%d"`, i))
		}
		extra := ""
		if tc.wide {
			extra = `,"required":[` + strings.Join(required, ",") + `]`
		}
		input := schemas(`{"Consumer":{"type":"object"` + extra + `,"properties":{` + strings.Join(props, ",") + `}},"Source":{"type":"object","properties":{"leaf":{"type":"integer","format":"int64","minimum":0,"maximum":9}}}}`)
		if tc.ok {
			aggregationReport(t, input)
		} else {
			aliasFailure(t, input, "E_MODEL_LIMIT", "/components/schemas/Consumer")
		}
	}
}

func TestSchemaInternalScalarPointerNoDropUniqueAndCatalogCounts(t *testing.T) {
	prefix := `{"openapi":"3.1.0","components":{"schemas":{"Source":{"type":"object","properties":{"leaf":{"type":"string","minLength":1}}}}},"paths":`
	for _, paths := range []string{
		`{"/v":{"get":{"parameters":[{"in":"query","name":"v","schema":{"type":"object","properties":{"leaf":{"$ref":"#/components/schemas/Source/properties/leaf"}}}}],"responses":{"204":{}}}}}`,
		`{"/v":{"post":{"requestBody":{"content":{"application/json":{"schema":{"type":"array","items":{"type":"object","properties":{"leaf":{"$ref":"#/components/schemas/Source/properties/leaf"}}}}}}},"responses":{"204":{}}}}}`,
	} {
		aliasFailure(t, []byte(prefix+paths+`}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Source/properties/leaf/minLength")
	}
	floatingUniqueGraduation(t, pointerDoc(`{"type":"number","format":"double"}`, `{"type":"array","uniqueItems":true,"items":`+pointerRef("Source/properties/leaf")+`}`))
	aggregationReport(t, pointerDoc(`{"type":["integer","null"],"minimum":0}`, `{"type":"array","uniqueItems":true,"items":`+pointerRef("Source/properties/leaf")+`}`))
	for _, name := range []string{"Nullable", "Null~able/slot"} {
		key, _ := json.Marshal(name)
		input := schemas(`{` + string(key) + `:{"allOf":[{"nullable":true,"minProperties":0,"properties":{"v":` + pointerRef("Source/properties/leaf") + `}}]},"Source":{"type":"object","properties":{"leaf":{"type":"string"}}}}`)
		at := "/components/schemas/" + strings.ReplaceAll(strings.ReplaceAll(name, "~", "~0"), "/", "~1") + "/allOf/0/minProperties"
		aliasFailure(t, input, "E_SCHEMA_UNSUPPORTED", at)
	}
}

func TestSchemaInternalScalarPointerTypedHelperCanonicalization(t *testing.T) {
	wrap := func(ref string) string {
		return `{"type":"object","properties":{"nested":{"type":"object","properties":{"value":` + ref + `,"nullable":{"allOf":[` + ref + `],"nullable":true},"list":{"type":"array","minItems":0,"maxItems":3,"uniqueItems":true,"items":` + ref + `},"map":{"type":"object","properties":{},"minProperties":0,"maxProperties":3,"additionalProperties":` + ref + `}}}}}`
	}
	named := pointerRef("Bound")
	internal := pointerRef("Source/properties/leaf")
	for _, reverse := range []bool{false, true} {
		a, b := named, internal
		if reverse {
			a, b = b, a
		}
		input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Bound":{"type":"string","minLength":1,"maxLength":2},"Source":{"type":"object","properties":{"leaf":{"type":"string","minLength":1,"maxLength":2}}}}},"paths":{"/a":{"post":{"operationId":"same","requestBody":{"content":{"application/json":{"schema":` + wrap(a) + `}}},"responses":{"204":{}}}},"/b":{"put":{"operationId":"same","requestBody":{"content":{"application/json":{"schema":` + wrap(b) + `}}},"responses":{"204":{}}}}}}`)
		text := source(t, aggregationReport(t, input), "SameInput")
		if strings.Count(text, "private fun openapiKtValidatePrimitive") != 4 || strings.Contains(text, "private inline") || !strings.Contains(text, "has fewer entries than minItems") || !strings.Contains(text, "has more entries than maxProperties") || !strings.Contains(text, "duplicate entries for uniqueItems") || !strings.Contains(text, "Patch<kotlin.String>") {
			t.Fatal("typed helper canonical OR/scopes/order", reverse, text)
		}
	}
	// Unhinted existing typed alias input keeps inlined checks and no helper.
	plain := schemas(`{"Bound":{"type":"string","minLength":1},"Consumer":{"properties":{"v":{"$ref":"#/components/schemas/Bound"}}}}`)
	if text := source(t, aggregationReport(t, plain), "Consumer"); strings.Contains(text, "openapiKtValidatePrimitive") {
		t.Fatal("unhinted output changed", text)
	}
}

func TestSchemaInternalScalarPointerDiscardedAndFixedExtras(t *testing.T) {
	for _, consumer := range []string{
		`{"allOf":[{"properties":{"v":{"$ref":"#/components/schemas/Source/properties/leaf"}}},{"properties":{"v":{"type":"string"}}}]}`,
		`{"properties":{"v":{"type":"string"}},"additionalProperties":{"$ref":"#/components/schemas/Source/properties/leaf"}}`,
	} {
		aliasFailure(t, schemas(`{"Consumer":`+consumer+`,"Source":{"properties":{"leaf":{"type":"string","minLength":1}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Source/properties/leaf/minLength")
	}
}
