package core_test

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestAllOfDTOPropertyCountOriginal(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/allof-dto-property-counts-qualified-envelopes.json")
	if err != nil {
		t.Fatal(err)
	}
	rq := request(input, "Dto")
	rq.Options.Validation = true
	r, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatalf("desired success: %v", err)
	}
	text := source(t, r, "RecordDto")
	for _, token := range []string{"generatedSerializer()", "element.size < 2", "element.size > 2", "length >= 2", "length <= 3"} {
		if !strings.Contains(text, token) {
			t.Fatal(token, text)
		}
	}
}

func allOfCountDoc(root string, members ...string) []byte {
	return schemas(`{"Record":{"allOf":[` + strings.Join(members, ",") + `]` + root + `}}`)
}
func countMember(bounds string) string { return `{"properties":{"v":{"type":"string"}}` + bounds + `}` }
func TestAllOfDTOPropertyCountGrammarAndFold(t *testing.T) {
	for _, tc := range []struct {
		root    string
		members []string
		at      string
	}{
		{"", []string{countMember(`,"maxProperties":1`), countMember(`,"minProperties":2`)}, "allOf/1/minProperties"},
		{"", []string{countMember(`,"minProperties":3`), countMember(`,"maxProperties":2`)}, "allOf/1/maxProperties"},
		{`,"minProperties":2`, []string{countMember(`,"maxProperties":1`)}, "minProperties"},
		{`,"maxProperties":2`, []string{countMember(`,"minProperties":3`)}, "maxProperties"},
		{"", []string{countMember(`,"minProperties":3,"maxProperties":2`), countMember(`,"minProperties":0,"maxProperties":9`)}, "allOf/0/maxProperties"},
		{`,"maxProperties":-1`, []string{countMember(`,"minProperties":3`), countMember(`,"maxProperties":2`)}, "maxProperties"},
		{"", []string{countMember(`,"minProperties":3`), countMember(`,"maxProperties":2`), countMember(`,"minProperties":1.0`)}, "allOf/2/minProperties"},
	} {
		aliasFailure(t, allOfCountDoc(tc.root, tc.members...), "E_SCHEMA_INVALID", "/components/schemas/Record/"+tc.at)
	}
	for _, literal := range []string{"-1", "-0", "1.0", "1e0", `"1"`, "null", "true", "2147483648"} {
		for _, key := range []string{"minProperties", "maxProperties"} {
			for _, at := range []string{"root", "member"} {
				bounds := `,"` + key + `":` + literal
				root, member, p := bounds, countMember(""), key
				if at == "member" {
					root, member, p = "", countMember(bounds), "allOf/0/"+key
				}
				aliasFailure(t, allOfCountDoc(root, member), "E_SCHEMA_INVALID", "/components/schemas/Record/"+p)
			}
		}
	}
	for _, tc := range []struct {
		root    string
		members []string
		lo, hi  string
	}{
		{`,"maxProperties":4`, []string{countMember(`,"minProperties":2,"maxProperties":5`), countMember(`,"minProperties":1,"maxProperties":4`)}, "2", "4"},
		{`,"minProperties":2,"maxProperties":2`, []string{countMember(`,"minProperties":2,"maxProperties":2`), countMember(`,"minProperties":2,"maxProperties":2`)}, "2", "2"},
		{"", []string{countMember(`,"minProperties":0`)}, "0", ""},
		{`,"minProperties":2147483647`, []string{countMember("")}, "2147483647", ""},
		{"", []string{countMember(`,"maxProperties":0`)}, "", "0"},
	} {
		input := allOfCountDoc(tc.root, tc.members...)
		a := aggregationReport(t, input)
		b := aggregationReport(t, input)
		if !reflect.DeepEqual(a, b) {
			t.Fatal("nondeterministic")
		}
		text := source(t, a, "Record")
		if strings.Count(text, "element.size < ") != boolInt(tc.lo != "") || strings.Count(text, "element.size > ") != boolInt(tc.hi != "") {
			t.Fatal("source-count-expanded output", text)
		}
		for _, s := range []string{"element.size < " + tc.lo, "element.size > " + tc.hi} {
			if !strings.HasSuffix(s, " ") && !strings.Contains(text, s) {
				t.Fatal(s, text)
			}
		}
		if strings.Contains(text, "init {") {
			t.Fatal("ctor satisfiability inferred", text)
		}
	}
}
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func TestAllOfDTOPropertyCountEligibilityNoDrop(t *testing.T) {
	for _, tc := range []struct {
		root    string
		members []string
		at      string
	}{
		{"", []string{`{"type":"object","minProperties":0}`, countMember("")}, "allOf/0/minProperties"},
		{"", []string{`{"minProperties":0}`, countMember("")}, "allOf/0/minProperties"},
		{"", []string{`{"type":"object","properties":{},"minProperties":0}`, countMember("")}, "allOf/0/minProperties"},
		{"", []string{`{"additionalProperties":true,"minProperties":0}`, countMember("")}, "allOf/0/minProperties"},
		{`,"minProperties":0`, []string{`{"properties":{"tag":{"type":"string","const":"x"}}}`}, "minProperties"},
		{`,"minProperties":0`, []string{`{"$ref":"#/components/schemas/Record"}`}, "minProperties"},
	} {
		aliasFailure(t, allOfCountDoc(tc.root, tc.members...), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/"+tc.at)
	}
	// Preserve the formerly rejected valid nested inputs as explicit
	// whole-object count graduations; validation-off still rejects them.
	for _, input := range [][]byte{
		allOfCountDoc(`,"minProperties":0`, `{"allOf":[`+countMember("")+`]}`),
		allOfCountDoc("", `{"allOf":[`+countMember(`,"minProperties":0`)+`]}`),
	} {
		text := source(t, aggregationReport(t, input), "Record")
		if !strings.Contains(text, "generatedSerializer()") || !strings.Contains(text, "element.size < 0") {
			t.Fatal("Lost original nested whole-object count", text)
		}
		if _, err := core.Generate(request(input, "")); err == nil {
			t.Fatal("Default-off admitted original nested count")
		}
	}
	for _, tc := range []struct {
		input []byte
		at    string
	}{
		{allOfCountDoc("", countMember(`,"minProperties":0`)), "/components/schemas/Record/allOf/0/minProperties"},
		{allOfCountDoc(`,"minProperties":0`, countMember("")), "/components/schemas/Record/minProperties"},
	} {
		rq := request(tc.input, "")
		_, err := core.Generate(rq)
		d, ok := err.(core.Diagnostics)
		if !ok || len(d) != 1 || d[0].Code != "E_SCHEMA_UNSUPPORTED" || d[0].Pointer == nil || *d[0].Pointer != tc.at {
			t.Fatal(err)
		}
	}
	// Annotation/empty UNBOUNDED branches retain old acceptance.
	aggregationReport(t, allOfCountDoc(`,"minProperties":0`, `{}`, `{"type":"object","properties":{}}`, countMember("")))
	original, err := os.ReadFile("../tests/fixtures/http/allof-dto-property-counts-supported-envelopes.json")
	if err != nil {
		t.Fatal(err)
	}
	aliasFailure(t, original, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/additionalProperties")
}

func TestAllOfDTOPropertyCountCanonicalizationAndWinningFields(t *testing.T) {
	aggregate := `{"type":"object","allOf":[{"minProperties":1,"properties":{"value":{"type":"string","minLength":1}}},{"maxProperties":2,"properties":{"value":{"type":"string","maxLength":2}}}]}`
	direct := `{"type":"object","minProperties":1,"maxProperties":2,"properties":{"value":{"type":"string","minLength":1,"maxLength":2}}}`
	for _, reverse := range []bool{false, true} {
		a, b := aggregate, direct
		if reverse {
			a, b = b, a
		}
		wrap := func(v string) string { return `{"type":"object","properties":{"nested":` + v + `}}` }
		input := []byte(`{"openapi":"3.1.0","paths":{"/a":{"post":{"operationId":"same","requestBody":{"content":{"application/json":{"schema":` + wrap(a) + `}}},"responses":{"204":{}}}},"/b":{"put":{"operationId":"same","requestBody":{"content":{"application/json":{"schema":` + wrap(b) + `}}},"responses":{"204":{}}}}}}`)
		text := source(t, aggregationReport(t, input), "SameInput")
		if strings.Count(text, "generatedSerializer()") != 1 || strings.Count(text, "private fun openapiKtValidatePrimitive(") != 1 || !strings.Contains(text, "this.openapiKtValidatePrimitive(value)") {
			t.Fatal("identity/source pointers/helper OR hint", reverse, text)
		}
		changed := []byte(strings.Replace(string(input), `"maxProperties":2`, `"maxProperties":3`, 1))
		rq := request(changed, "")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_COLLISION" || len(r.Artifacts.Files) != 0 {
			t.Fatal("different effective count merged", err)
		}
	}
	input := allOfCountDoc(`,"properties":{"first":{"type":"string","minLength":2},"own":{"type":"boolean"}}`,
		`{"minProperties":0,"required":["first"],"properties":{"first":{"type":"string","minLength":1},"last":{"type":"integer"},"kind":{"type":"string","const":"x"}}}`,
		`{"maxProperties":8,"properties":{"first":{"type":"string","maxLength":4}}}`)
	text := source(t, aggregationReport(t, input), "Record")
	for _, s := range []string{"public val first: kotlin.String,", "public val last: kotlin.Int? = null", "length >= 2", "length <= 4", `SerialName("kind")`} {
		if !strings.Contains(text, s) {
			t.Fatal(s, text)
		}
	}
	if strings.Index(text, `SerialName("first")`) > strings.Index(text, `SerialName("last")`) {
		t.Fatal("first insertion position", text)
	}
	plain := allOfCountDoc("", countMember(""))
	rq := request(plain, "")
	before, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal(err)
	}
	rq.Options.Validation = true
	after, err := core.GenerateWithDiagnostics(rq)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("noBounds changed", err)
	}
}

func TestAllOfDTOPropertyCountLimits(t *testing.T) {
	for _, tc := range []struct {
		n     int
		boxed bool
		pass  bool
	}{{120, false, true}, {121, false, false}, {240, true, true}, {241, true, false}} {
		input := aggregationWide(tc.n, tc.boxed)
		input = []byte(strings.Replace(string(input), `"Wide":{`, `"Wide":{"minProperties":0,"maxProperties":240,`, 1))
		rq := request(input, "")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		if tc.pass {
			if err != nil {
				t.Fatal(tc, err)
			}
			text := source(t, r, "Wide")
			if strings.Count(text, "generatedSerializer()") != 1 || strings.Count(text, "multipleOf") != tc.n {
				t.Fatal("count/primitive checks dropped", tc)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
			t.Fatal(tc, err)
		}
	}
	for _, n := range []int{100, 5500} {
		members := make([]string, n)
		for i := range members {
			members[i] = countMember(`,"minProperties":0,"maxProperties":2`)
		}
		rq := request(allOfCountDoc("", members...), "")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		if n == 100 {
			if err != nil || strings.Count(source(t, r, "Record"), "element.size") != 2 {
				t.Fatal("fold output bounded", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_LIMIT" {
			t.Fatal("16384 budget", err)
		}
	}
	for _, depth := range []int{46, 47} {
		obj := `{"allOf":[` + countMember(`,"minProperties":0`) + `]}`
		for i := 0; i < depth; i++ {
			obj = fmt.Sprintf(`{"properties":{"layer%d":%s}}`, i, obj)
		}
		rq := request(schemas(`{"Record":`+obj+`}`), "")
		rq.Options.Validation = true
		_, err := core.Generate(rq)
		if depth == 46 && err != nil || depth == 47 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal("schema48", depth, err)
		}
	}
	for _, depth := range []int{16, 17} {
		leaf := `{"type":"string","minLength":1}`
		for i := 0; i < depth; i++ {
			leaf = `{"type":"array","items":` + leaf + `}`
		}
		input := allOfCountDoc("", `{"minProperties":0,"properties":{"v":`+leaf+`}}`)
		rq := request(input, "")
		rq.Options.Validation = true
		_, err := core.Generate(rq)
		if depth == 16 && err != nil || depth == 17 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal("collection16", depth, err)
		}
	}
}

func TestAllOfDTOPropertyCountNullableApproximationAndNominalFields(t *testing.T) {
	for _, tc := range []struct{ root, first, second, ty string }{
		{"", `true`, `false`, `Payload?`},
		{`,"nullable":false`, `true`, `false`, `Payload`},
		{`,"nullable":true`, `false`, `false`, `Payload?`},
	} {
		nested := `{"type":"object","allOf":[{"nullable":` + tc.first + `,"minProperties":0,"properties":{"v":{"type":"string"}}},{"nullable":` + tc.second + `,"maxProperties":0,"properties":{"v":{"type":"string"}}}]` + tc.root + `}`
		input := httpDoc(`{"requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"object","required":["payload"],"properties":{"payload":` + nested + `}}}}},"responses":{"204":{}}}`)
		// httpDoc uses GET; body generation remains supported independently of verb.
		text := source(t, aggregationReport(t, input), "GetPingInput")
		if !strings.Contains(text, "public val payload: "+tc.ty+",") {
			t.Fatal("root/first inline nullable changed", tc, text)
		}
	}
	input := schemas(`{"Alias":{"type":"integer","format":"int64","minimum":0},"Record":{"allOf":[{"minProperties":0,"properties":{"native":{"type":"string","format":"date-time"},"alias":{"$ref":"#/components/schemas/Alias"},"self":{"$ref":"#/components/schemas/Record"}}}],"maxProperties":3}}`)
	text := source(t, aggregationReport(t, input), "Record")
	for _, token := range []string{"kotlin.time.Instant?", "public val alias: kotlin.Long?", "org.example.models.models.Record?", "element.size > 3"} {
		if !strings.Contains(text, token) {
			t.Fatal("nested nominal fields changed", token, text)
		}
	}
}

func TestAllOfDTOPropertyCountMixedBoxedBoundary(t *testing.T) {
	for _, n := range []int{240, 241} {
		first, second, required := []string{}, []string{}, []string{}
		for i := 0; i < n; i++ {
			name := fmt.Sprintf("p%d", i)
			a, b := `{"type":["integer","null"],"format":"int64","minimum":0,"exclusiveMinimum":0}`, `{"type":["integer","null"],"format":"int64","maximum":8,"exclusiveMaximum":8,"multipleOf":2}`
			if i%2 == 1 {
				a = `{"type":["number","null"],"format":"double","minimum":0,"exclusiveMinimum":0}`
				b = `{"type":["number","null"],"format":"double","maximum":8,"exclusiveMaximum":8,"multipleOf":2}`
			}
			if i%3 == 0 {
				a = `{"type":["string","null"],"minLength":2,"pattern":"^[a-z]+$"}`
				b = `{"type":["string","null"],"maxLength":3,"pattern":"^[a-z]+$"}`
			}
			first = append(first, fmt.Sprintf(`"%s":%s`, name, a))
			second = append(second, fmt.Sprintf(`"%s":%s`, name, b))
			required = append(required, fmt.Sprintf(`"%s"`, name))
		}
		input := schemas(`{"Wide":{"allOf":[{"minProperties":0,"properties":{` + strings.Join(first, ",") + `},"required":[` + strings.Join(required, ",") + `]},{"maxProperties":240,"properties":{` + strings.Join(second, ",") + `}}]}}`)
		rq := request(input, "")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		if n == 241 {
			if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0 {
				t.Fatal("mixed241 accepted", err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		text := source(t, r, "Wide")
		for _, key := range []string{"below minimum", "exclusiveMinimum", "above maximum", "exclusiveMaximum", "multipleOf"} {
			if strings.Count(text, key) != 160 {
				t.Fatal("mixed primitive dropped", key)
			}
		}
		for _, key := range []string{"shorter than minLength", "longer than maxLength", "does not match pattern"} {
			if strings.Count(text, key) != 80 {
				t.Fatal("String rule dropped", key)
			}
		}
		if strings.Count(text, "element.size") != 2 || strings.Count(text, "private fun openapiKtValidatePrimitive") != 240 {
			t.Fatal("wrapper/helper width changed")
		}
	}
}

func TestAllOfDTOPropertyCountNullableComponentOriginalPointer(t *testing.T) {
	input := schemas(`{"Unused":{"allOf":[{"nullable":true,"minProperties":0,"properties":{"v":{"type":"string"}}}]}}`)
	aliasFailure(t, input, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Unused/allOf/0/minProperties")
}

func TestAllOfDTOPropertyCountNullableComponentMalformedExcluded(t *testing.T) {
	input := schemas(`{"Unused":{"allOf":[{"nullable":true,"minProperties":-1,"properties":{"v":{"type":"string"}}}]}}`)
	aliasFailure(t, input, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Unused/allOf/0/minProperties")
}

func TestAllOfDTOPropertyCountNullableComponentCatalogAndGrammar(t *testing.T) {
	for _, tc := range []struct{ name, bounds, root, at, code string }{
		{"Un~used/slot", `,"minProperties":0`, "", "/components/schemas/Un~0used~1slot/allOf/0/minProperties", "E_SCHEMA_UNSUPPORTED"},
		{"Unused", `,"minProperties":-1`, "", "/components/schemas/Unused/allOf/0/minProperties", "E_SCHEMA_UNSUPPORTED"},
		{"Unused", `,"maxProperties":"bad"`, "", "/components/schemas/Unused/allOf/0/maxProperties", "E_SCHEMA_UNSUPPORTED"},
		{"Unused", `,"minProperties":3,"maxProperties":2`, "", "/components/schemas/Unused/allOf/0/minProperties", "E_SCHEMA_UNSUPPORTED"},
		{"Unused", `,"minProperties":0`, `,"minProperties":0`, "/components/schemas/Unused/minProperties", "E_SCHEMA_UNSUPPORTED"},
		{"Unused", "", "", "/components/schemas/Unused", "E_SCHEMA_UNSUPPORTED"},
		{"Unused", `,"minProperties":-1`, `,"nullable":true,"minProperties":0`, "/components/schemas/Unused/allOf/0/minProperties", "E_SCHEMA_INVALID"},
		{"Unused", `,"minProperties":-1`, `,"nullable":false`, "/components/schemas/Unused/allOf/0/minProperties", "E_SCHEMA_INVALID"},
	} {
		input := schemas(`{"` + tc.name + `":{"allOf":[{"nullable":true,"properties":{"v":{"type":"string"}}` + tc.bounds + `}]` + tc.root + `}}`)
		aliasFailure(t, input, tc.code, tc.at)
	}
	allowed := schemas(`{"Unused":{"nullable":false,"allOf":[{"nullable":true,"minProperties":0,"properties":{"v":{"type":"string"}}}]}}`)
	if !strings.Contains(source(t, aggregationReport(t, allowed), "Unused"), "element.size < 0") {
		t.Fatal("explicit rootfalse approximation changed")
	}
	// Nested locations that share a wire-name prefix with a component are not
	// catalog roots; their nullable codecs and local grammar stay independent.
	for _, bounds := range []string{`,"minProperties":0`, `,"minProperties":-1`} {
		input := schemas(`{"Outer":{"properties":{"payload":{"allOf":[{"nullable":true,"properties":{"v":{"type":"string"}}` + bounds + `}]}}},"Outer/properties/payload":{"properties":{"v":{"type":"string"}}}}`)
		if strings.Contains(bounds, "-1") {
			aliasFailure(t, input, "E_SCHEMA_INVALID", "/components/schemas/Outer/properties/payload/allOf/0/minProperties")
		} else {
			text := source(t, aggregationReport(t, input), "Outer")
			if !strings.Contains(text, "public val payload: Payload? = null") || !strings.Contains(text, "element.size < 0") {
				t.Fatal("nested nullable confused with component", text)
			}
		}
	}
}
