package core_test

import (
	"bytes"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"openapi-kt/core"
)

func referencedNullStringDefaults(own bool) []byte {
	annotation := func(value string) string {
		if own {
			return `,"default":` + value
		}
		return ""
	}
	return schemas(`{
 "Status":{"type":["string","null"],"enum":[null,"on","null"]},
 "Mode":{"nullable":true,"enum":["raw",null,"wire"]},
 "Source":{"properties":{"code":{"type":["string","null"],"enum":["raw",null,"wire"]}}},
 "Alias":{"$ref":"#/components/schemas/Source/properties/code"},
 "Rows":{"type":"array","items":{"$ref":"#/components/schemas/Status"}},
 "Maps":{"additionalProperties":{"$ref":"#/components/schemas/Status"}},
 "Nested":{"type":"array","items":{"$ref":"#/components/schemas/Rows"}},
 "Holder":{"properties":{
 "scalar":{"$ref":"#/components/schemas/Status"` + annotation(`"on"`) + `},
 "nil":{"$ref":"#/components/schemas/Mode"` + annotation(`null`) + `},
 "textNull":{"$ref":"#/components/schemas/Status"` + annotation(`"null"`) + `},
 "borrowed":{"$ref":"#/components/schemas/Alias"` + annotation(`"raw"`) + `},
 "rows":{"$ref":"#/components/schemas/Rows"` + annotation(`[null,"on","null"]`) + `},
 "maps":{"$ref":"#/components/schemas/Maps"` + annotation(`{"a/~":null,"$ref":"null"}`) + `},
 "nested":{"$ref":"#/components/schemas/Nested"` + annotation(`[[null,"on"]]`) + `}
 }}}`)
}

func TestReferencedStringEnumNullMemberDefaultsOwnAnnotations(t *testing.T) {
	for _, checked := range []bool{false, true} {
		// Verbatim former referenced scalar negative, graduated separately.
		graduation := request(stringScalarDefaultDoc(`{"type":["string","null"],"enum":[null,"on"]}`, `null`, `,"nullable":true`), "Dto")
		graduation.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(graduation); err != nil {
			t.Fatal("former scalar null vocabulary guard", err)
		}
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(referencedNullStringDefaults(false), "Dto")
			r.Options.Validation, r.Options.ModelStyle = checked, style
			before, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal("accepted no-default control", err)
			}
			r.Document = referencedNullStringDefaults(true)
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal("own annotation", checked, style, err)
			}
			text := source(t, got, "HolderDto")
			for _, want := range []string{"`openapi-kt default enum scalar ", `"\"on\""`, `"\"null\""`, `"[null,\"on\",\"null\"]"`, `"[[null,\"on\"]]"`, `"{\"\$ref\":\"null\",\"a/~\":null}"`, "org.example.models.models.SourceDto.Code? ="} {
				if !strings.Contains(text, want) {
					t.Fatal(want, text)
				}
			}
			for _, name := range []string{"StatusDto", "ModeDto", "SourceDto", "RowsDto", "MapsDto", "NestedDto", "AliasDto"} {
				if source(t, got, name) != source(t, before, name) {
					t.Fatal("classifier/alias/codec drift", name)
				}
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("determinism", err)
			}
		}
	}
}

func TestReferencedStringEnumNullMemberDefaultsGrammarAndNullableViews(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ leaf, value, override, pointer string }{
			{`{"type":["string","null"],"enum":[null,"on"]}`, `null`, `,"nullable":false`, "/components/schemas/Holder/properties/a~1~0/default"},
			{`{"type":["string","null"],"enum":[null,"on"]}`, `"null"`, "", "/components/schemas/Holder/properties/a~1~0/default"},
			{`{"type":["string","null"],"enum":[null,"on"]}`, `"unknown"`, "", "/components/schemas/Holder/properties/a~1~0/default"},
			{`{"type":["string","null"],"enum":[null,"on"]}`, `true`, "", "/components/schemas/Holder/properties/a~1~0/default"},
			{`{"type":"string","enum":["on",null]}`, `null`, `,"nullable":true`, "/components/schemas/Status/enum/1"},
			{`{"type":["string","null"],"enum":[null,"on",null]}`, `null`, "", "/components/schemas/Status/enum/2"},
			{`{"type":["string","null"],"enum":[null,"on","on"]}`, `"on"`, "", "/components/schemas/Status/enum/2"},
			{`{"type":["string","null"],"enum":[null,"on",false]}`, `null`, "", "/components/schemas/Status/enum/2"},
		} {
			nativeEnumDefaultFailure(t, stringScalarDefaultDoc(tc.leaf, tc.value, tc.override), checked, "E_SCHEMA_INVALID", tc.pointer)
		}
		for _, leaf := range []string{
			`{"type":["string","null"],"enum":[null,"on"],"format":"uuid"}`,
			`{"type":["string","null"],"enum":[null,"on"],"minLength":1}`,
			`{"type":["string","null"],"enum":[null,"on"],"pattern":"on"}`,
			`{"type":["string","null"],"enum":[null,"on"],"const":"on"}`,
			`{"type":["string","null"],"enum":[null,"on"],"x-native-type":"kotlin.String"}`,
			`{"type":["string","null"],"enum":[null,"on"],"default":"on"}`,
		} {
			for _, value := range []string{`null`, `"on"`} {
				r := request(stringScalarDefaultDoc(leaf, value, ""), "Dto")
				r.Options.Validation = checked
				if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("ungranted terminal", leaf, value, err)
				}
			}
		}
		// Null is controlled by the actual layer view, not vocabulary or outer presence.
		narrowed := `{"type":"array","items":{"$ref":"#/components/schemas/Status","nullable":false}}`
		doc := schemas(`{"Status":{"type":["string","null"],"enum":[null,"on"]},"Rows":` + narrowed + `,"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Rows","nullable":true,"default":[null]}}}}`)
		nativeEnumDefaultFailure(t, doc, checked, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/value/default/0")
		r := request(bytes.Replace(doc, []byte(`"default":[null]`), []byte(`"default":["on"]`), 1), "Dto")
		r.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("narrowed non-null member", err)
		}
		r.Document = bytes.Replace(doc, []byte(`"default":[null]`), []byte(`"default":null`), 1)
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("outer null independent of narrowed leaf", err)
		}
	}
}

func TestReferencedStringEnumNullMemberDefaultsPresenceAndIdentity(t *testing.T) {
	for _, checked := range []bool{false, true} {
		// Redundant source null adds no emitted default or classifier identity.
		r := request(referencedNullStringDefaults(true), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(err)
		}
		r.Document = bytes.ReplaceAll(r.Document, []byte(`"enum":[null,"on","null"]`), []byte(`"enum":["on","null"]`))
		r.Document = bytes.ReplaceAll(r.Document, []byte(`"enum":["raw",null,"wire"]`), []byte(`"enum":["raw","wire"]`))
		same, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got, same) {
			t.Fatal("redundant null output drift", err)
		}
		for _, role := range []string{"required", "patch", "discarded"} {
			field := `{"$ref":"#/components/schemas/Status","nullable":true,"default":"on"}`
			holder := `{"required":["v"],"properties":{"v":` + field + `}}`
			if role == "patch" {
				holder = `{"properties":{"v":` + field + `}}`
			}
			if role == "discarded" {
				holder = `{"allOf":[{"properties":{"v":` + field + `}},{"properties":{"v":{"type":"string"}}}]}`
			}
			doc := schemas(`{"Status":{"type":["string","null"],"enum":[null,"on","off"]},"Holder":` + holder + `}`)
			if role == "patch" {
				doc = []byte(`{"openapi":"3.1.0","components":{"schemas":{"Status":{"type":["string","null"],"enum":[null,"on","off"]}}},"paths":{"/p":{"patch":{"requestBody":{"content":{"application/json":{"schema":` + holder + `}}},"responses":{"204":{}}}}}}`)
			}
			r.Document = doc
			first, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(role, checked, err)
			}
			r.Document = bytes.Replace(doc, []byte(`"default":"on"`), []byte(`"default":null`), 1)
			second, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(first, second) {
				t.Fatal("suppressed default identity", role, err)
			}
			r.Document = bytes.Replace(doc, []byte(`"default":"on"`), []byte(`"default":"unknown"`), 1)
			if report, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("suppression skipped original membership", role, err)
			}
		}
	}
}

func TestReferencedStringEnumNullMemberDefaultsRawVocabularyLimit(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, count := range []int{239, 240} {
			members := []string{`null`}
			for i := 0; i < count; i++ {
				members = append(members, `"v`+strconv.Itoa(i)+`"`)
			}
			r := request(stringScalarDefaultDoc(`{"type":["string","null"],"enum":[`+strings.Join(members, ",")+`]}`, `"v0"`, ""), "Dto")
			r.Options.Validation = checked
			report, err := core.GenerateWithDiagnostics(r)
			if count == 239 && err != nil || count == 240 && (err == nil || err.(core.Diagnostics)[0].Code != "E_ENUM_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{})) {
				t.Fatal("raw vocabulary limit includes null", count, checked, err)
			}
		}
	}
}
