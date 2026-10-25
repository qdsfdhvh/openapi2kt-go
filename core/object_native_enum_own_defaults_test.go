package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestObjectNativeEnumOwnPropertiesInFactories(t *testing.T) {
	properties := `"flag":{"type":"boolean","enum":[true,false],"default":false},"i32":{"type":"integer","format":"int32","enum":[-2147483648,2147483647],"default":-2147483648},"i64":{"type":"integer","format":"int64","enum":[-9223372036854775808,9223372036854775807],"default":-9223372036854775808},"f32":{"type":"number","format":"float","enum":[16777216,16777217],"default":16777217},"f64":{"type":"number","format":"double","enum":[1.5,2.5],"default":2.5}`
	for _, shape := range []string{`"properties":{` + properties + `}`, `"allOf":[{"properties":{` + properties + `}},{"properties":{"label":{"type":"string","default":"owned"}}}]`} {
		input := schemas(`{"Target":{` + shape + `,"default":{}},"Alias":{"$ref":"#/components/schemas/Target","nullable":true,"default":null},"Narrow":{"$ref":"#/components/schemas/Alias","nullable":false,"default":{"flag":true,"f32":16777216}}}`)
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				rq := request(input, "Dto")
				rq.Options.Validation, rq.Options.ModelStyle = checked, style
				first, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal("ordinary native enum own-property default lost", err)
				}
				again, err := core.GenerateWithDiagnostics(rq)
				if err != nil || !reflect.DeepEqual(first, again) {
					t.Fatal("native enum own-property default nondeterminism", err)
				}
			}
		}
	}
}

func TestObjectNativeEnumOwnOriginalValuePointersAndStrictRounding(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ input, pointer string }{
			{`{"Target":{"properties":{"f32":{"type":"number","format":"float","enum":[16777216,16777217],"default":16777216.5}}},"Alias":{"$ref":"#/components/schemas/Target","nullable":true,"default":null}}`, "/components/schemas/Target/properties/f32/default"},
			{`{"Target":{"properties":{"f32":{"type":"number","format":"float","enum":[16777216,16777217],"default":16777217}}},"Alias":{"$ref":"#/components/schemas/Target","default":{"f32":16777216.5}}}`, "/components/schemas/Alias/default/f32"},
			{`{"Target":{"required":["flag"],"properties":{"flag":{"type":"boolean","enum":[true,false],"default":false}}},"Alias":{"$ref":"#/components/schemas/Target","default":{}}}`, "/components/schemas/Alias/default"},
		} {
			rq := request(schemas(tc.input), "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != tc.pointer {
				t.Fatal("native enum own/outer literal lost strictness or Pointer", tc.pointer, err)
			}
		}
	}
}

func TestObjectNativeEnumOwnVocabularyLimitEvenForNullFactory(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, count := range []int{240, 241} {
			values := make([]int, count)
			for i := range values {
				values[i] = i
			}
			encoded, err := json.Marshal(values)
			if err != nil {
				t.Fatal(err)
			}
			rq := request(schemas(`{"Target":{"nullable":true,"properties":{"value":{"type":"integer","format":"int32","enum":`+string(encoded)+`,"default":0}},"default":null}}`), "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if count == 240 && err != nil {
				t.Fatal("exact native enum vocabulary budget rejected", err)
			}
			if count == 241 && (err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Target/properties/value/enum") {
				t.Fatal("null factory skipped native own source vocabulary budget", err)
			}
		}
	}
}

// Retain the exact E03.4 unsupported input as an explicit new positive; the
// structured source Pointer regression uses an independently unsupported count.
func TestObjectNativeEnumOwnFormerUnsupportedSource(t *testing.T) {
	rq := request(schemas(`{"Target":{"properties":{"mode":{"type":"boolean","enum":[true,false],"default":true}}},"Alias":{"$ref":"#/components/schemas/Target","nullable":true,"default":null}}`), "Dto")
	rq.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal(err)
	}
}
