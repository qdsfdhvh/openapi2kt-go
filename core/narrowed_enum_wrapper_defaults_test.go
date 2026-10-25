package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNarrowedEnumWrapperOwnDefaultsUseOriginalNullableVocabulary(t *testing.T) {
	for _, tc := range []struct{ schema, value string }{
		{`{"enum":["on",null,"off"]}`, `"on"`},
		{`{"type":["string","null"],"enum":["on",null,"off"]}`, `"on"`},
		{`{"type":"boolean","nullable":true,"enum":[true,null,false]}`, `true`},
		{`{"type":["integer","null"],"enum":[1,null,2]}`, `1`},
		{`{"type":"integer","format":"int64","nullable":true,"enum":[9223372036854775807,null,0]}`, `9223372036854775807`},
		{`{"type":"number","format":"float","nullable":true,"enum":[0.1,null,1.0]}`, `0.1`},
		{`{"type":"number","nullable":true,"enum":[0.1,null,1.0]}`, `0.1`},
	} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			input := schemas(`{"State":` + tc.schema + `,"Narrow":{"$ref":"#/components/schemas/State","nullable":false,"default":` + tc.value + `},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Narrow"}}}}`)
			r := request(input, "Dto")
			r.Options.Validation, r.Options.ModelStyle = true, style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(tc.schema, style, err)
			}
			if !strings.Contains(source(t, got, "NarrowDto"), "default") || strings.Contains(source(t, got, "NarrowDto"), "typealias NarrowDto = org.example.models.models.StateDto?") {
				t.Fatal("narrowed factory lost actual non-null alias")
			}
			r.Document = []byte(strings.ReplaceAll(string(input), `,null`, ""))
			control, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, control) {
				t.Fatal("nullable source data changed narrowed factory output", tc.schema, err)
			}
		}
	}
}

func TestNarrowedEnumWrapperOwnDefaultsKeepStrictValuesAndPointers(t *testing.T) {
	for _, tc := range []struct{ schema, value, pointer string }{
		{`{"enum":["on",null,"off"]}`, `null`, "/components/schemas/Narrow/default"},
		{`{"enum":["on",null,"off"]}`, `"future"`, "/components/schemas/Narrow/default"},
		{`{"enum":["on",null,"off"],"default":"future"}`, `"on"`, "/components/schemas/State/default"},
		{`{"enum":["on",null,null]}`, `"on"`, "/components/schemas/State/enum/2"},
		{`{"type":"integer","nullable":true,"enum":[1,null,2]}`, `1.5`, "/components/schemas/Narrow/default"},
		{`{"type":"number","format":"float","nullable":true,"enum":[16777217,null]}`, `16777216`, "/components/schemas/Narrow/default"},
		{`{"type":"number","nullable":true,"enum":[0.1,null]}`, `0.2`, "/components/schemas/Narrow/default"},
		{`{"type":"integer","nullable":true,"enum":[1,null,2],"minimum":2}`, `1`, "/components/schemas/Narrow/default"},
		{`{"type":"integer","nullable":true,"enum":[1,null,2],"minimum":3}`, `2`, "/components/schemas/State/enum"},
	} {
		r := request(schemas(`{"Narrow":{"$ref":"#/components/schemas/State","nullable":false,"default":`+tc.value+`},"State":`+tc.schema+`}`), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || *err.(core.Diagnostics)[0].Pointer != tc.pointer || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(tc.schema, tc.value, err)
		}
	}
}

func TestNarrowedEnumWrapperOwnDefaultsKeepRawAndByteCaps(t *testing.T) {
	for _, count := range []int{239, 240} {
		values := []string{"null"}
		for i := range count {
			values = append(values, fmt.Sprintf(`"v%d"`, i))
		}
		r := request(schemas(`{"State":{"enum":[`+strings.Join(values, ",")+`]},"Narrow":{"$ref":"#/components/schemas/State","nullable":false,"default":"v0"}}`), "Dto")
		got, err := core.GenerateWithDiagnostics(r)
		if (err == nil) != (count == 239) || err != nil && (err.(core.Diagnostics)[0].Code != "E_ENUM_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal(count, err)
		}
	}
	for _, size := range []int{32766, 32767} {
		// Whitespace uses the short VALUE member while retaining the full
		// annotation bytes; operator repetitions expand into long identifiers.
		value := `"` + strings.Repeat(" ", size) + `"`
		r := request(schemas(`{"State":{"enum":[`+value+`,null]},"Narrow":{"$ref":"#/components/schemas/State","nullable":false,"default":`+value+`}}`), "Dto")
		got, err := core.GenerateWithDiagnostics(r)
		if (err == nil) != (size == 32766) || err != nil && (err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Narrow/default" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal(size, err)
		}
	}
}
