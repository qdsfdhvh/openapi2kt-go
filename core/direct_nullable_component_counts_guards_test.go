package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestDirectNullableComponentCountsOriginalGrammar(t *testing.T) {
	for _, root := range []string{`"type":"object","nullable":true`, `"type":["null","object"]`, `"nullable":true`} {
		for _, key := range []string{"minProperties", "maxProperties"} {
			for _, value := range []string{"-1", "-0", "1.0", "1e0", `"1"`, "null", "true", "[]", "{}", "2147483648"} {
				input := directNullableCountInput("3.1.0", root, `,"`+key+`":`+value)
				aliasFailure(t, input, "E_SCHEMA_INVALID", "/components/schemas/Record/"+key)
				rq := request(input, "")
				if _, err := core.Generate(rq); err == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Record/"+key || err.(core.Diagnostics)[0].Code != "E_SCHEMA_UNSUPPORTED" {
					t.Fatal("validation-off grammar permission changed", err)
				}
			}
		}
		aliasFailure(t, directNullableCountInput("3.1.0", root, `,"minProperties":3,"maxProperties":2`), "E_SCHEMA_INVALID", "/components/schemas/Record/maxProperties")
		// No constructor-size satisfiability inference, including min=Int.MAX
		// and max=0: these are JSON codec bounds, not construction barriers.
		for _, bounds := range []string{`,"minProperties":0`, `,"minProperties":2147483647`, `,"maxProperties":0`, `,"minProperties":0,"maxProperties":0`} {
			text := source(t, aggregationReport(t, directNullableCountInput("3.1.0", root, bounds)), "Record")
			if strings.Contains(text, "init {") || !strings.Contains(text, "generatedSerializer()") {
				t.Fatal("count became ctor check", text)
			}
		}
	}
	input := schemas(`{"Un~used/slot":{"type":["object","null"],"properties":{"value":{"type":"string"}},"maxProperties":-1}}`)
	aliasFailure(t, input, "E_SCHEMA_INVALID", "/components/schemas/Un~0used~1slot/maxProperties")
	// An unused root/null consumer cannot hide invalid own fields/defaults.
	for _, tc := range []struct{ field, tail, code string }{
		{`{"type":"string","minLength":-1}`, "/properties/value/minLength", "E_SCHEMA_INVALID"},
		{`{"type":"string","default":false}`, "/properties/value/default", "E_SCHEMA_INVALID"},
		{`{"type":"string","enum":["a"],"minLength":1}`, "/properties/value/enum", "E_SCHEMA_UNSUPPORTED"},
		{`{"$ref":"#/components/schemas/Missing"}`, "/properties/value", "E_REF_MISSING"},
	} {
		input := schemas(`{"Record":{"type":["object","null"],"minProperties":0,"properties":{"value":` + tc.field + `}}}`)
		aliasFailure(t, input, tc.code, "/components/schemas/Record"+tc.tail)
	}
	for _, tc := range []struct{ sibling, tail string }{
		{`,"required":["missing"]`, ""},
		{`,"default":null`, "/default"},
		{`,"additionalProperties":{"type":"string","minLength":1}`, "/additionalProperties/minLength"},
	} {
		code := "E_SCHEMA_UNSUPPORTED"
		if tc.tail == "" {
			code = "E_SCHEMA_INVALID"
		}
		aliasFailure(t, directNullableCountInput("3.1.0", `"type":["object","null"]`+tc.sibling, `,"minProperties":0`), code, "/components/schemas/Record"+tc.tail)
	}
}

func TestDirectNullableComponentCountsKeepExcludedRoles(t *testing.T) {
	for _, body := range []string{
		`{"nullable":true,"allOf":[{"properties":{"value":{"type":"string"}}}],"minProperties":0}`,
		`{"type":["object","null"],"properties":{"value":{"type":"string"}},"allOf":[{}],"minProperties":0}`,
		`{"allOf":[{"nullable":true,"properties":{"value":{"type":"string"}}}],"minProperties":0}`,
		`{"nullable":true,"allOf":[{"nullable":false,"minProperties":0,"properties":{"value":{"type":"string"}}}]}`,
		`{"nullable":true,"properties":{},"minProperties":0}`,
		`{"nullable":true,"type":"object","minProperties":0}`,
		`{"nullable":true,"properties":{"pin":{"const":"fixed"}},"minProperties":0}`,
		`{"$ref":"#/components/schemas/Target","nullable":true,"minProperties":0}`,
		`{"allOf":[{"$ref":"#/components/schemas/Target"}],"nullable":true,"minProperties":0}`,
		`{"nullable":true,"oneOf":[{"properties":{"value":{"type":"string"}}}],"minProperties":0}`,
	} {
		input := schemas(`{"Record":` + body + `,"Target":{"properties":{"value":{"type":"string"}}}}`)
		rq := request(input, "")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_UNSUPPORTED" || !reflect.DeepEqual(r, core.GenerationReport{}) {
			t.Fatal("excluded nullable shape gained count permission", body, r, err)
		}
	}
	count := `{"type":["object","null"],"minProperties":0,"properties":{"value":{"type":"string"}}}`
	for _, ref := range []string{"Record", "Narrow"} {
		input := schemas(`{"Record":` + count + `,"Narrow":{"$ref":"#/components/schemas/Record","nullable":false},"Borrow":{"allOf":[{"$ref":"#/components/schemas/` + ref + `","nullable":false},{"properties":{"other":{"type":"string"}}}]}}`)
		aliasFailure(t, input, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Borrow/allOf/0/$ref")
	}
	for _, tc := range []struct {
		input []byte
		at    string
	}{
		{schemas(`{"Record":` + count + `,"Union":{"oneOf":[{"$ref":"#/components/schemas/Record"},{"properties":{"other":{"type":"string"}}}]}}`), "/components/schemas/Record/minProperties"},
		{schemas(`{"Parent":{"oneOf":[{"$ref":"#/components/schemas/Record"}],"discriminator":{"propertyName":"tag"}},"Record":` + count + `}`), "/components/schemas/Record/minProperties"},
		{schemas(`{"Record":` + count + `,"Outer":{"allOf":[{"properties":{"lost":{"$ref":"#/components/schemas/Record"}}}],"properties":{"lost":{"type":"string"}}}}`), "/components/schemas/Record/minProperties"},
		{[]byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":` + count + `}},"paths":{"/filter":{"get":{"parameters":[{"name":"q","in":"query","schema":{"type":"array","items":{"$ref":"#/components/schemas/Record"}}}],"responses":{"204":{}}}}}}`), "/components/schemas/Record/minProperties"},
	} {
		aliasFailure(t, tc.input, "E_SCHEMA_UNSUPPORTED", tc.at)
	}
	// AP-only dictionaries retain their OLD typed-size contract rather than a
	// new nominal Model count serializer. No new DTO permission is inferred.
	input := schemas(`{"Bag":{"type":["object","null"],"additionalProperties":{"type":"string"},"minProperties":1}}`)
	text := source(t, aggregationReport(t, input), "Bag")
	if !strings.Contains(text, "typealias Bag") || strings.Contains(text, "generatedSerializer()") {
		t.Fatal(text)
	}
}

func TestDirectNullableComponentCountsParameterProjectionKeepsOldAdmission(t *testing.T) {
	input := func(root, counts, target string) []byte {
		return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"type":"object"` + root + counts + `,"properties":{"value":{"type":"string"}}},"Narrow":{"$ref":"#/components/schemas/Record","nullable":false}}},"paths":{"/filter":{"get":{"parameters":[{"name":"q","in":"query","schema":{"type":"array","items":{"$ref":"#/components/schemas/` + target + `"}}}],"responses":{"204":{}}}}}}`)
	}
	for _, tc := range []struct {
		name, root, counts string
		validation         bool
	}{
		{"old ordinary counted array", "", `,"minProperties":0,"maxProperties":2`, true},
		{"old nullable unbounded array off", `,"nullable":true`, "", false},
		{"old nullable unbounded array checked", `,"nullable":true`, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rq := request(input(tc.root, tc.counts, "Record"), "")
			rq.Options.Validation = tc.validation
			r, err := core.GenerateWithDiagnostics(rq)
			if err != nil || len(r.Artifacts.Files) == 0 {
				t.Fatal("new guard changed historical parameter admission", err)
			}
			if tc.counts != "" && !strings.Contains(source(t, r, "Record"), "generatedSerializer()") {
				t.Fatal("old ordinary count serializer permission lost")
			}
		})
	}
	for _, target := range []string{"Record", "Narrow"} {
		aliasFailure(t, input(`,"nullable":true`, `,"minProperties":0,"maxProperties":2`, target), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/minProperties")
	}
}

func TestDirectNullableComponentCountsIndependentBudgets(t *testing.T) {
	for _, n := range []int{120, 121} {
		var fields, required []string
		for i := range n {
			name := fmt.Sprintf("p%d", i)
			fields = append(fields, fmt.Sprintf(`"%s":{"type":"integer","format":"int64"}`, name))
			required = append(required, fmt.Sprintf(`"%s"`, name))
		}
		rq := request(schemas(`{"Record":{"nullable":true,"minProperties":0,"properties":{`+strings.Join(fields, ",")+`},"required":[`+strings.Join(required, ",")+`]}}`), "")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		if n == 120 && err != nil || n == 121 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0) {
			t.Fatal("nullable root changed physical Long constructor slots", n, err)
		}
	}
	for _, n := range []int{240, 241} {
		var fields []string
		for i := range n {
			fields = append(fields, fmt.Sprintf(`"p%d":{"type":"string"}`, i))
		}
		rq := request(schemas(`{"Record":{"type":["object","null"],"minProperties":0,"properties":{`+strings.Join(fields, ",")+`}}}`), "")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		if n == 240 && err != nil || n == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(r.Artifacts.Files) != 0) {
			t.Fatal(n, err)
		}
	}
	for _, n := range []int{16, 17} {
		leaf := `{"type":"string","minLength":1}`
		for range n {
			leaf = `{"type":"array","items":` + leaf + `}`
		}
		rq := request(schemas(`{"Record":{"nullable":true,"minProperties":0,"properties":{"value":`+leaf+`}}}`), "")
		rq.Options.Validation = true
		_, err := core.Generate(rq)
		if n == 16 && err != nil || n == 17 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal(n, err)
		}
	}
	for _, n := range []int{46, 47} {
		rq := request(rootSpecialChain(n, `{"nullable":true,"minProperties":0,"properties":{"value":{"type":"string"}}}`, fmt.Sprintf(`{"properties":{"value":{"$ref":"#/components/schemas/A%02d","nullable":false}}}`, n-1)), "")
		rq.Options.Validation = true
		_, err := core.Generate(rq)
		if n == 46 && err != nil || n == 47 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal(n, err)
		}
	}
}
