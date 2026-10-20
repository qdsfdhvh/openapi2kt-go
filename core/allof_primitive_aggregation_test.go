package core_test

import (
	"fmt"
	"openapi-kt/core"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestAllOfPrimitiveAggregationOriginal(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/allof-primitive-bounds-supported-envelopes.json")
	if err != nil {
		t.Fatal(err)
	}
	req := request(input, "Dto")
	req.Options.Validation = true
	r, err := core.GenerateWithDiagnostics(req)
	if err != nil {
		t.Fatalf("new success required: %v", err)
	}
	text := source(t, r, "RecordDto")
	for _, token := range []string{"length >= 2", "length <= 3", `^[a-z]+\$`} {
		if !strings.Contains(text, token) {
			t.Fatal(token, text)
		}
	}
	if strings.Count(text, `^[a-z]+\$`) != 1 {
		t.Fatal("pattern duplicated", text)
	}
}

func aggregationDoc(a, b string) []byte {
	return schemas(`{"Record":{"allOf":[{"type":"object","required":["value"],"properties":{"value":` + a + `}},{"type":"object","properties":{"value":` + b + `}}]}}`)
}
func aggregationReport(t *testing.T, input []byte) core.GenerationReport {
	t.Helper()
	req := request(input, "")
	req.Options.Validation = true
	r, err := core.GenerateWithDiagnostics(req)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func TestAllOfPrimitiveAggregationRules(t *testing.T) {
	for _, tc := range []struct{ a, b, code, at string }{
		{`{"type":"string","minLength":4}`, `{"type":"string","maxLength":3}`, "E_SCHEMA_INVALID", "1/maxLength"},
		{`{"type":"string","maxLength":1}`, `{"type":"string","minLength":2}`, "E_SCHEMA_INVALID", "1/minLength"},
		{`{"type":"string","minLength":-1}`, `{"type":"string","minLength":2}`, "E_SCHEMA_INVALID", "0/minLength"},
		{`{"type":"string","pattern":"["}`, `{"type":"string","pattern":"a"}`, "E_SCHEMA_INVALID", "0/pattern"},
		{`{"type":"string","pattern":"a"}`, `{"type":"string","pattern":"b"}`, "E_SCHEMA_UNSUPPORTED", "1/pattern"},
		{`{"type":"integer","minimum":4}`, `{"type":"integer","maximum":3}`, "E_SCHEMA_INVALID", "1/maximum"},
		{`{"type":"integer","maximum":3}`, `{"type":"integer","minimum":4}`, "E_SCHEMA_INVALID", "0/maximum"},
		{`{"type":"integer","minimum":6}`, `{"type":"integer","maximum":5,"exclusiveMaximum":4}`, "E_SCHEMA_INVALID", "1/exclusiveMaximum"},
		{`{"type":"number","minimum":0.300000000000000004}`, `{"type":"number","maximum":0.300000000000000001}`, "E_SCHEMA_INVALID", "1/maximum"},
		{`{"type":"integer","exclusiveMinimum":2}`, `{"type":"integer","exclusiveMaximum":3}`, "E_SCHEMA_INVALID", "1/exclusiveMaximum"},
		{`{"type":"integer","minimum":3,"multipleOf":2}`, `{"type":"integer","maximum":3}`, "E_SCHEMA_INVALID", "0/multipleOf"},
		{`{"type":"integer","minimum":-3}`, `{"type":"integer","maximum":-3,"multipleOf":2}`, "E_SCHEMA_INVALID", "1/multipleOf"},
		{`{"type":"integer","multipleOf":2}`, `{"type":"integer","multipleOf":3}`, "E_SCHEMA_UNSUPPORTED", "1/multipleOf"},
		{`{"type":"integer","multipleOf":1.0}`, `{"type":"integer","multipleOf":1}`, "E_SCHEMA_INVALID", "0/multipleOf"},
		{`{"type":"number","multipleOf":0}`, `{"type":"number","multipleOf":1}`, "E_SCHEMA_INVALID", "0/multipleOf"},
		{`{"type":"number","format":"float","multipleOf":1e-100}`, `{"type":"number","format":"float","multipleOf":1}`, "E_SCHEMA_INVALID", "0/multipleOf"},
		{`{"type":"number","format":"float","multipleOf":0.300000004}`, `{"type":"number","format":"float","multipleOf":0.300000001}`, "E_SCHEMA_UNSUPPORTED", "1/multipleOf"},
		{`{"type":"number","minimum":1e-513}`, `{"type":"number","maximum":1}`, "E_SCHEMA_INVALID", "0/minimum"},
		{`{"type":"integer","exclusiveMinimum":2147483647}`, `{"type":"integer"}`, "E_SCHEMA_INVALID", "0/exclusiveMinimum"},
		{`{"type":"integer","format":"int64","exclusiveMaximum":-9223372036854775808}`, `{"type":"integer","format":"int64"}`, "E_SCHEMA_INVALID", "0/exclusiveMaximum"},
	} {
		pieces := strings.Split(tc.at, "/")
		aliasFailure(t, aggregationDoc(tc.a, tc.b), tc.code, "/components/schemas/Record/allOf/"+pieces[0]+"/properties/value/"+pieces[1])
	}
	for _, pair := range [][2]string{
		{`{"type":"number","multipleOf":0.25}`, `{"type":"number","multipleOf":25e-2}`},
		{`{"type":"number","format":"float","minimum":0.300000004}`, `{"type":"number","format":"float","exclusiveMinimum":0.300000001}`},
		{`{"type":"number","minimum":0.300000000000000004}`, `{"type":"number","exclusiveMinimum":0.300000000000000001}`},
		{`{"type":"string","nullable":true,"minLength":1}`, `{"type":["string","null"],"maxLength":3}`},
	} {
		aggregationReport(t, aggregationDoc(pair[0], pair[1]))
	}
	for _, scalar := range []string{`"integer"`, `"number"`} {
		text := source(t, aggregationReport(t, aggregationDoc(`{"type":`+scalar+`,"minimum":0,"exclusiveMinimum":0}`, `{"type":`+scalar+`,"maximum":8,"exclusiveMaximum":8,"multipleOf":2}`)), "Record")
		for _, key := range []string{"below minimum", "exclusiveMinimum", "above maximum", "exclusiveMaximum", "multipleOf"} {
			if !strings.Contains(text, key) {
				t.Fatal(key, text)
			}
		}
	}
	// Same parsed pattern (JSON escaping differs), emits one rule.
	r := aggregationReport(t, aggregationDoc(`{"type":"string","pattern":"a"}`, `{"type":"string","pattern":"\u0061"}`))
	if strings.Count(source(t, r, "Record"), "toRegex()") != 1 {
		t.Fatal(r)
	}
}

func TestAllOfPrimitiveAggregationEligibilityNoDrop(t *testing.T) {
	early := `{"type":"string","minLength":1}`
	for _, later := range []string{
		`{"type":"integer"}`, `{"type":"string","format":""}`, `{"type":"string","nullable":true}`,
		`{"type":"string","enum":["a"]}`, `{"type":"string","const":"a"}`,
		`{"type":"string","x-kotlin-value-class":true}`, `{"type":"string","x-native-type":"custom.Type"}`,
		`{"type":"string","format":"date"}`, `{"type":"string","format":"byte"}`,
		`{"type":"array","items":{"type":"string"}}`, `{"oneOf":[{"type":"string"},{"type":"null"}]}`,
		`{"$ref":"#/components/schemas/Other"}`, `{"allOf":[{"$ref":"#/components/schemas/Other"}]}`,
	} {
		aliasFailure(t, aggregationDoc(early, later), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/allOf/0/properties/value/minLength")
	}
	// The exact former default-bearing rejection is now an executable positive;
	// required presence still suppresses its constructor initializer.
	graduated := source(t, aggregationReport(t, aggregationDoc(early, `{"type":"string","default":"a"}`)), "Record")
	if !strings.Contains(graduated, "val value: kotlin.String,") || !strings.Contains(graduated, "length >= 1") {
		t.Fatal("Original default-bearing allOf input lost required/check semantics", graduated)
	}
	// Preserve the original valid referenced/nested inputs as explicitly
	// graduated ordinary groups; default-off remains unsupported.
	for _, input := range []string{
		`{"Other":{"type":"object","properties":{"value":{"type":"string","minLength":1}}},"Record":{"allOf":[{"$ref":"#/components/schemas/Other"},{"properties":{"value":{"type":"string"}}}]}}`,
		`{"Record":{"allOf":[{"allOf":[{"properties":{"value":{"type":"string","minLength":1}}}]},{"properties":{"value":{"type":"string"}}}]}}`,
	} {
		req := request(schemas(input), "")
		req.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(req)
		if err != nil || !strings.Contains(source(t, got, "Record"), "length >= 1") {
			t.Fatal("Referenced/nested original bound not retained", input, err)
		}
		req.Options.Validation = false
		if _, err := core.Generate(req); err == nil {
			t.Fatal("Default-off accepted original bounded input", input)
		}
	}
	// Collection checks and non-aggregate discarded constraints keep their guard.
	aliasFailure(t, aggregationDoc(`{"type":"array","minItems":1,"items":{"type":"string"}}`, `{"type":"array","items":{"type":"string"}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/allOf/0/properties/value/minItems")
	input := aggregationDoc(early, `{"type":"string"}`)
	req := request(input, "")
	if _, err := core.Generate(req); err == nil {
		t.Fatal("default accepted bounds")
	}
	// Unused schemas and excluded consumers are still checked.
	req.Options.Validation = true
	req.Options.ExcludeAPIs = []string{"BoundsApi"}
	aggregationReport(t, input)
	original, err := os.ReadFile("../tests/fixtures/http/allof-primitive-bounds.json")
	if err != nil {
		t.Fatal(err)
	}
	req = request(original, "")
	req.Options.Validation = true
	_, err = core.Generate(req)
	if err == nil || err.(core.Diagnostics)[0].Code != "E_OPERATIONS_UNSUPPORTED" {
		t.Fatal("root HTTP allOf broadened", err)
	}
}

func TestAllOfPrimitiveAggregationLegacy(t *testing.T) {
	legacy := func(a, b string) []byte {
		return []byte(strings.Replace(string(aggregationDoc(a, b)), "3.1.0", "3.0.3", 1))
	}
	text := source(t, aggregationReport(t, legacy(`{"type":"number","minimum":1,"exclusiveMinimum":true}`, `{"type":"number","minimum":2,"maximum":4}`)), "Record")
	if !strings.Contains(text, "value > 1.0") || !strings.Contains(text, "value >= 2.0") {
		t.Fatal("modifier bound across sources", text)
	}
	aliasFailure(t, legacy(`{"type":"integer","minimum":1}`, `{"type":"integer","exclusiveMinimum":true}`), "E_SCHEMA_INVALID", "/components/schemas/Record/allOf/1/properties/value/exclusiveMinimum")
	aliasFailure(t, legacy(`{"type":"number","exclusiveMinimum":false}`, `{"type":"number","minimum":1}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/allOf/0/properties/value/exclusiveMinimum")
	aliasFailure(t, legacy(`{"type":"integer","minimum":2,"exclusiveMinimum":true}`, `{"type":"integer","maximum":2}`), "E_SCHEMA_INVALID", "/components/schemas/Record/allOf/1/properties/value/maximum")
}

func aggregationWide(n int, nullable bool) []byte {
	first, second, required := []string{}, []string{}, []string{}
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("p%d", i)
		ty := `"integer"`
		format := "int64"
		if i%2 != 0 {
			ty = `"number"`
			format = "double"
		}
		if nullable {
			ty = `[` + ty + `,"null"]`
		}
		first = append(first, fmt.Sprintf(`"%s":{"type":%s,"format":"%s","minimum":0,"exclusiveMinimum":0}`, name, ty, format))
		second = append(second, fmt.Sprintf(`"%s":{"type":%s,"format":"%s","maximum":8,"exclusiveMaximum":8,"multipleOf":2}`, name, ty, format))
		required = append(required, fmt.Sprintf(`"%s"`, name))
	}
	return schemas(`{"Wide":{"allOf":[{"properties":{` + strings.Join(first, ",") + `},"required":[` + strings.Join(required, ",") + `]},{"properties":{` + strings.Join(second, ",") + `}}]}}`)
}
func TestAllOfPrimitiveAggregationBudgets(t *testing.T) {
	for _, tc := range []struct {
		n              int
		nullable, pass bool
	}{{120, false, true}, {121, false, false}, {240, true, true}, {241, true, false}} {
		req := request(aggregationWide(tc.n, tc.nullable), "")
		req.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(req)
		if tc.pass {
			if err != nil {
				t.Fatal(tc, err)
			}
			if strings.Count(source(t, r, "Wide"), "multipleOf") != tc.n {
				t.Fatal("dropped rule")
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
			t.Fatal(tc, err)
		}
	}
	// No invented source-count16 limit; bounded output has four slots, not 100 checks.
	for _, n := range []int{100, 5500} {
		members := make([]string, n)
		for i := range members {
			members[i] = `{"properties":{"value":{"type":"number","minimum":0,"maximum":8}}}`
		}
		req := request(schemas(`{"Record":{"allOf":[`+strings.Join(members, ",")+`]}}`), "")
		req.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(req)
		if n == 100 {
			if err != nil || strings.Count(source(t, r, "Record"), "below minimum") != 1 {
				t.Fatal(err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_LIMIT" {
			t.Fatal("composition work guard", err)
		}
	}
	for _, depth := range []int{46, 47} {
		object := `{"type":"object","allOf":[{"properties":{"value":{"type":"string","minLength":1}}},{"properties":{"value":{"type":"string","maxLength":2}}}]}`
		for i := 0; i < depth; i++ {
			object = fmt.Sprintf(`{"type":"object","properties":{"layer%d":%s}}`, i, object)
		}
		req := request(schemas(`{"Record":`+object+`}`), "")
		req.Options.Validation = true
		_, err := core.Generate(req)
		if depth == 46 && err != nil || depth == 47 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal("global aggregate depth guard", depth, err)
		}
	}
	for _, depth := range []int{16, 17} {
		field := `{"type":"number","minimum":0}`
		for i := 0; i < depth; i++ {
			field = `{"type":"array","items":` + field + `}`
		}
		req := request(fieldDoc(field), "")
		req.Options.Validation = true
		_, err := core.Generate(req)
		if depth == 16 && err != nil || depth == 17 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal("collection guard", err)
		}
	}
}
func TestAllOfPrimitiveAggregationWinnerIdentityDeterminism(t *testing.T) {
	input := schemas(`{"Record":{"allOf":[{"required":["first"],"properties":{"first":{"type":"string","minLength":1},"last":{"type":"integer"}}},{"properties":{"first":{"type":"string","maxLength":4}}}],"properties":{"first":{"type":"string","description":"winning","minLength":2},"own":{"type":"boolean"}}}}`)
	a := aggregationReport(t, input)
	b := aggregationReport(t, input)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("nondeterministic")
	}
	text := source(t, a, "Record")
	if !strings.Contains(text, "public val first: kotlin.String,") || strings.Index(text, `SerialName("first")`) > strings.Index(text, `SerialName("last")`) || !strings.Contains(text, "length >= 2") {
		t.Fatal(text)
	}
	unbounded := aggregationDoc(`{"type":"string"}`, `{"type":"string"}`)
	req := request(unbounded, "")
	before, err := core.GenerateWithDiagnostics(req)
	if err != nil {
		t.Fatal(err)
	}
	after := aggregationReport(t, unbounded)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("unbounded output changed")
	}
	// Same operation-local identity at different Pointers shares its declaration.
	object := `{"type":"object","properties":{"nested":{"type":"object","allOf":[{"properties":{"value":{"type":"string","minLength":1}}},{"properties":{"value":{"type":"string","maxLength":2}}}]}}}`
	http := []byte(`{"openapi":"3.1.0","paths":{"/a":{"post":{"operationId":"same","requestBody":{"content":{"application/json":{"schema":` + object + `}}},"responses":{"204":{}}}},"/b":{"put":{"operationId":"same","requestBody":{"content":{"application/json":{"schema":` + object + `}}},"responses":{"204":{}}}}}}`)
	r := aggregationReport(t, http)
	count := 0
	for _, f := range r.Artifacts.Files {
		if strings.HasSuffix(f.Path, "SameInput.kt") {
			count++
		}
	}
	if count != 1 {
		t.Fatal("source provenance polluted declaration identity", count)
	}
}

func TestAllOfPrimitiveAggregationContextGuards(t *testing.T) {
	composed := `{"allOf":[{"properties":{"value":{"type":"string","minLength":1}}},{"properties":{"value":{"type":"string","maxLength":2}}}]}`
	for _, input := range [][]byte{
		schemas(`{"Record":{"oneOf":[` + composed + `,{"type":"object","properties":{"other":{"type":"boolean"}}}]}}`),
		schemas(`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Record"}],"discriminator":{"propertyName":"tag"}},"Record":{"allOf":[{"properties":{"tag":{"type":"string","enum":["x"]},"value":{"type":"string","minLength":1}}},{"properties":{"value":{"type":"string","maxLength":2}}}]}}`),
		httpDoc(`{"parameters":[{"name":"p","in":"query","schema":` + composed + `}],"responses":{"204":{}}}`),
		schemas(`{"Record":{"properties":{"value":{"type":"string"}},"additionalProperties":` + composed + `}}`),
	} {
		req := request(input, "")
		req.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(req)
		if err == nil || len(r.Artifacts.Files) != 0 {
			t.Fatal("erased or context-ineligible aggregation accepted", string(input), r)
		}
	}
}

func TestAllOfPrimitiveAggregationHelperCanonicalization(t *testing.T) {
	aggregate := `{"type":"object","allOf":[{"properties":{"value":{"type":"string","minLength":1}}},{"properties":{"value":{"type":"string","maxLength":2}}}]}`
	direct := `{"type":"object","properties":{"value":{"type":"string","minLength":1,"maxLength":2}}}`
	for _, reverse := range []bool{false, true} {
		a, b := aggregate, direct
		if reverse {
			a, b = b, a
		}
		wrap := func(v string) string { return `{"type":"object","properties":{"nested":` + v + `}}` }
		http := []byte(`{"openapi":"3.1.0","paths":{"/a":{"post":{"operationId":"same","requestBody":{"content":{"application/json":{"schema":` + wrap(a) + `}}},"responses":{"204":{}}}},"/b":{"put":{"operationId":"same","requestBody":{"content":{"application/json":{"schema":` + wrap(b) + `}}},"responses":{"204":{}}}}}}`)
		r := aggregationReport(t, http)
		text := source(t, r, "SameInput")
		if strings.Count(text, "private fun openapiKtValidatePrimitive(") != 1 || !strings.Contains(text, "this.openapiKtValidatePrimitive(value)") {
			t.Fatal("lost OR canonical nested helper hint", reverse, text)
		}
	}
	input := schemas(`{"Record":{"allOf":[{"properties":{"v":{"type":"integer","minimum":0}}},{"properties":{"v":{"type":"integer","maximum":2}}}],"properties":{"openapiKtValidatePrimitive":{"type":"string"},"openapiKtValidatePrimitive2":{"type":"object","properties":{"id":{"type":"string"}}},"openapiKtValidatePrimitive3":{"type":"string","const":"c"}}}}`)
	text := source(t, aggregationReport(t, input), "Record")
	if !strings.Contains(text, "private fun openapiKtValidatePrimitive4(value: kotlin.Int)") || strings.Contains(text, "private inline") || strings.Contains(text, "@kotlinx.serialization.Transient") {
		t.Fatal("helper collision/state", text)
	}
	for _, tc := range []struct {
		n        int
		nullable bool
	}{{120, false}, {240, true}} {
		text := source(t, aggregationReport(t, aggregationWide(tc.n, tc.nullable)), "Wide")
		if strings.Count(text, "private fun openapiKtValidatePrimitive") != tc.n || strings.Count(text, "this.openapiKtValidatePrimitive") != tc.n {
			t.Fatal("helper work not fixed field slots", tc)
		}
	}
}
