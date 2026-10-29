package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func enumBoundsDiagnostic(t *testing.T, doc []byte, checked bool, code, pointer string) {
	t.Helper()
	r := request(doc, "Dto")
	r.Options.Validation = checked
	got, err := core.GenerateWithDiagnostics(r)
	if err == nil {
		t.Fatal("Expected rejection", string(doc))
	}
	d := err.(core.Diagnostics)[0]
	if d.Code != code || d.Pointer == nil || *d.Pointer != pointer || !reflect.DeepEqual(got, core.GenerationReport{}) {
		t.Fatal(code, pointer, err, got)
	}
}

func TestStringEnumCollectionBoundsFormerGuards(t *testing.T) {
	// Original rejected sources graduate verbatim; validation off still rejects.
	for _, field := range []string{
		`{"type":"array","items":{"type":"string","enum":["a"],"minLength":1}}`,
		`{"type":"object","additionalProperties":{"type":"string","enum":["a"],"minLength":1}}`,
		`{"type":"array","items":{"enum":["one"],"minLength":1},"default":[]}`,
		`{"additionalProperties":{"enum":["one"],"pattern":"x"},"default":{}}`,
	} {
		r := request(fieldDoc(field), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || !strings.Contains(source(t, got, "RecordDto"), "serialName.let") {
			t.Fatal(field, err)
		}
		r.Options.Validation = false
		if _, err := core.GenerateWithDiagnostics(r); err == nil {
			t.Fatal("Disabled checks admitted")
		}
	}
}

func TestStringEnumCollectionBoundsOriginalConstraintsAndDefaults(t *testing.T) {
	for _, slot := range []string{"items", "additionalProperties"} {
		wrap := func(leaf, def string) string {
			prefix := `{"type":"array","items":`
			if slot == "additionalProperties" {
				prefix = `{"additionalProperties":`
			}
			return prefix + leaf + def + `}`
		}
		base := "/components/schemas/Record/properties/value"
		for _, tc := range []struct{ leaf, def, code, tail string }{
			{`{"enum":["one"],"minLength":-1}`, "", "E_SCHEMA_INVALID", "/" + slot + "/minLength"},
			{`{"enum":["one"],"maxLength":1.0}`, "", "E_SCHEMA_INVALID", "/" + slot + "/maxLength"},
			{`{"enum":["one"],"minLength":2147483648}`, "", "E_SCHEMA_INVALID", "/" + slot + "/minLength"},
			{`{"enum":["one"],"minLength":4,"maxLength":2}`, "", "E_SCHEMA_INVALID", "/" + slot + "/maxLength"},
			{`{"enum":["one"],"pattern":false}`, "", "E_SCHEMA_INVALID", "/" + slot + "/pattern"},
			{`{"enum":["one"],"pattern":"["}`, "", "E_SCHEMA_INVALID", "/" + slot + "/pattern"},
			{`{"enum":["one"],"pattern":"(?=x)"}`, "", "E_SCHEMA_UNSUPPORTED", "/" + slot + "/pattern"},
			{`{"enum":["one"],"pattern":"a{1001}"}`, "", "E_SCHEMA_INVALID", "/" + slot + "/pattern"},
			{`{"enum":["one"],"minLength":1,"maximum":5}`, "", "E_SCHEMA_UNSUPPORTED", "/" + slot + "/maximum"},
			{`{"enum":[null,"one","one"],"nullable":true,"minLength":1}`, "", "E_SCHEMA_INVALID", "/" + slot + "/enum/2"},
			{`{"enum":["one"],"minLength":1,"default":"one"}`, "", "E_SCHEMA_UNSUPPORTED", "/" + slot + "/default"},
		} {
			if tc.leaf == `{"enum":["one"],"minLength":1,"default":"one"}` {
				rq := request(fieldDoc(wrap(tc.leaf, tc.def)), "Dto")
				rq.Options.Validation = true
				inlineEnumChildGraduation(t, rq, `,"default":"one"`, `,"default":"other"`, base+"/"+slot+"/default")
				continue
			}
			enumBoundsDiagnostic(t, fieldDoc(wrap(tc.leaf, tc.def)), true, tc.code, base+tc.tail)
		}
		for _, tc := range []struct{ leaf, member, code string }{
			{`{"enum":["one","x"],"minLength":2}`, `"x"`, "E_SCHEMA_INVALID"},
			{`{"enum":["one","longer"],"maxLength":4}`, `"longer"`, "E_SCHEMA_INVALID"},
			{`{"enum":["one","two"],"pattern":"^one$"}`, `"two"`, "E_SCHEMA_INVALID"},
			{`{"enum":["one"],"minLength":1}`, `"unknown"`, "E_SCHEMA_INVALID"},
			{`{"enum":["one"],"minLength":1}`, `false`, "E_SCHEMA_INVALID"},
			{`{"enum":["one"],"minLength":1}`, `null`, "E_SCHEMA_INVALID"},
			{`{"enum":["é"],"pattern":"."}`, `"é"`, "E_SCHEMA_UNSUPPORTED"},
		} {
			def, tail := `,"default":[`+tc.member+`]`, "/default/0"
			if slot == "additionalProperties" {
				def = `,"default":{"z/~":` + tc.member + `,"a":false}`
				tail = "/default/z~1~0"
			}
			// Optional and required annotations both validate before initializer suppression.
			for _, required := range []bool{true, false} {
				doc := fieldDoc(wrap(tc.leaf, def))
				if !required {
					doc = []byte(strings.Replace(string(doc), `,"required":["value"]`, "", 1))
				}
				enumBoundsDiagnostic(t, doc, true, tc.code, base+tail)
			}
		}
		for _, def := range []string{`,"default":[]`, `,"default":null`} {
			if slot == "additionalProperties" && def != `,"default":null` {
				def = `,"default":{}`
			}
			field := wrap(`{"enum":["one"],"minLength":4,"maxLength":2}`, def)
			field = strings.Replace(field, `{`, `{"nullable":true,`, 1)
			enumBoundsDiagnostic(t, fieldDoc(field), true, "E_SCHEMA_INVALID", base+"/"+slot+"/maxLength")
		}
	}
	// JSON keyword order remains source ordered when validation is disabled.
	for _, keys := range []string{`"pattern":"x","minLength":1`, `"maxLength":3,"pattern":"x"`} {
		first := "pattern"
		if strings.HasPrefix(keys, `"maxLength"`) {
			first = "maxLength"
		}
		enumBoundsDiagnostic(t, fieldDoc(`{"type":"array","items":{"enum":["one"],`+keys+`},"default":[]}`), false, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/properties/value/items/"+first)
	}
}

func TestStringEnumCollectionBoundsNoDropAndExcludedScopes(t *testing.T) {
	field := `{"type":"array","items":{"enum":["one","x"],"minLength":2},"default":["one"]}`
	for _, schema := range []string{
		`{"Use":{"properties":{"value":{"allOf":[` + field + `]}}}}`,
		`{"Choice":{"oneOf":[{"properties":{"tag":{"const":"a"},"value":` + strings.Replace(field, `,"default":["one"]`, "", 1) + `}},{"properties":{"tag":{"const":"b"}}}]}}`,
		`{"Use":{"allOf":[{"properties":{"value":` + field + `}},{"properties":{"value":{"type":"string"}}}]}}`,
		`{"Use":{"properties":{"id":{"type":"string"}},"additionalProperties":` + field + `}}`,
	} {
		r := request(schemas(schema), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("Expanded role/lost source", schema, err)
		}
	}
	// Ordinary allOf may retain a single direct field, but cannot aggregate/drop checks.
	r := request(schemas(`{"Use":{"allOf":[{"properties":{"value":`+field+`}}],"properties":{"id":{"type":"string"}}}}`), "Dto")
	r.Options.Validation = true
	if got, err := core.GenerateWithDiagnostics(r); err != nil || !strings.Contains(source(t, got, "UseDto"), "serialName.let") {
		t.Fatal(err)
	}
	for _, leaf := range []string{
		`{"enum":["one"],"x-kotlin-value-class":true,"minLength":2}`,
	} {
		r.Document = fieldDoc(`{"type":"array","items":` + leaf + `}`)
		if _, err := core.GenerateWithDiagnostics(r); err == nil {
			t.Fatal("Expanded special role", leaf)
		}
	}
	stringEnumUniqueGraduation(t, fieldDoc(strings.Replace(field, `,"default"`, `,"uniqueItems":true,"default"`, 1)))
	// A reference cannot turn a checked collection DTO into an unchecked raw projection.
	r.Document = []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"codes":` + field + `}}}},"paths":{"/raw":{"post":{"requestBody":{"content":{"application/json":{"schema":{"properties":{"raw":{"anyOf":[{"$ref":"#/components/schemas/Holder"},{"type":"boolean"}]}}}}}},"responses":{"204":{}}}}}}`)
	if _, err := core.GenerateWithDiagnostics(r); err == nil {
		t.Fatal("Raw ref dropped checks")
	}
}

func TestStringEnumCollectionBoundsPatternAndMemberBudgets(t *testing.T) {
	for _, pattern := range []string{strings.Repeat("a", 1025), strings.Repeat("a?", 257)} {
		enumBoundsDiagnostic(t, fieldDoc(`{"type":"array","items":{"enum":["one"],"pattern":"`+pattern+`"},"default":[]}`), true, "E_SCHEMA_INVALID", "/components/schemas/Record/properties/value/items/pattern")
	}
	for _, count := range []int{240, 241} {
		members := make([]string, count)
		for i := range members {
			members[i] = fmt.Sprintf(`"v%d"`, i)
		}
		r := request(fieldDoc(`{"type":"array","items":{"enum":[`+strings.Join(members, ",")+`],"minLength":1},"default":[]}`), "Dto")
		r.Options.Validation = true
		_, err := core.GenerateWithDiagnostics(r)
		if count == 240 && err != nil || count == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_ENUM_LIMIT") {
			t.Fatal(count, err)
		}
	}
	for _, count := range []int{240, 241} {
		fields := make([]string, count)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"f%d":{"type":"array","items":{"enum":["one"],"minLength":2}}`, i)
		}
		r := request(schemas(`{"Wide":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto")
		r.Options.Validation = true
		_, err := core.GenerateWithDiagnostics(r)
		if count == 240 && err != nil || count == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal(count, err)
		}
	}
}
