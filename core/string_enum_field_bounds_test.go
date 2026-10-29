package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestDirectStringEnumFieldBounds(t *testing.T) {
	for _, meta := range []string{"", `,"format":"date"`, `,"x-native-type":"org.example.provider.Amount"`} {
		for _, def := range []string{"", `,"default":"one"`} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				field := `{"type":"string","enum":["one","x","toolong"],"minLength":2,"maxLength":4,"pattern":"^[a-z]+$"` + meta + def + `}`
				rq := request(fieldDoc(field), "Dto")
				rq.Options.Validation, rq.Options.ModelStyle = true, style
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal(meta, def, style, err)
				}
				text := source(t, got, "RecordDto")
				for _, want := range []string{"public sealed class", "class Unrecognized", ".serialName.let", "minLength", "maxLength", "containsMatchIn"} {
					if !strings.Contains(text, want) {
						t.Fatal("missing open lexical enum field check", want, text)
					}
				}
				if again, err := core.GenerateWithDiagnostics(rq); err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("bounded enum field nondeterminism", err)
				}
				rq.Options.Validation = false
				if got, err := core.GenerateWithDiagnostics(rq); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("bounded enum field bypassed validation opt-in", meta, def, err)
				}
			}
		}
	}
}

func TestMetadataStringEnumCollectionBounds(t *testing.T) {
	for _, meta := range []string{`,"format":"date"`, `,"format":"uuid"`, `,"x-native-type":"org.example.provider.Amount"`} {
		leaf := `{"type":"string","enum":["one","x"],"minLength":2` + meta + `}`
		for _, field := range []string{`{"type":"array","items":` + leaf + `,"default":["one"]}`, `{"additionalProperties":` + leaf + `,"default":{"first":"one"}}`} {
			rq := request(fieldDoc(field), "Dto")
			rq.Options.Validation = true
			if got, err := core.GenerateWithDiagnostics(rq); err != nil || !strings.Contains(source(t, got, "RecordDto"), "serialName.let") {
				t.Fatal("metadata enum collection lost lexical String bounds", meta, field, err)
			}
		}
	}
}

func TestStringEnumFieldBoundsFormerScopeGuards(t *testing.T) {
	// Preserve each original rejected source as an explicit positive migration.
	for _, body := range []string{
		`{"Record":{"properties":{"value":{"type":"string","enum":["a"],"minLength":1}},"required":["value"]}}`,
		`{"Record":{"type":["object","null"],"minProperties":0,"properties":{"value":{"type":"string","enum":["a"],"minLength":1}}}}`,
		`{"Use":{"properties":{"value":{"enum":["one"],"minLength":2}}}}`,
		`{"Use":{"properties":{"value":{"type":"array","items":{"type":"string","format":"date","enum":["2024-01-01"],"minLength":2}}}}}`,
		`{"Use":{"properties":{"value":{"type":"array","items":{"enum":["one"],"x-native-type":"org.example.External","minLength":2}}}}}`,
	} {
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("original bounded lexical source did not migrate", body, err)
		}
		found := false
		for _, f := range got.Artifacts.Files {
			found = found || strings.Contains(f.Content, "serialName.let")
		}
		if !found {
			t.Fatal("original bounded lexical source lost checks", body)
		}
	}
}

func TestStringEnumFieldBoundsOriginalDiagnostics(t *testing.T) {
	base := "/components/schemas/Record/properties/value"
	for _, tc := range []struct{ extra, code, tail string }{
		{`,"minLength":-1`, "E_SCHEMA_INVALID", "/minLength"},
		{`,"maxLength":1.0`, "E_SCHEMA_INVALID", "/maxLength"},
		{`,"minLength":4,"maxLength":2`, "E_SCHEMA_INVALID", "/maxLength"},
		{`,"pattern":"["`, "E_SCHEMA_INVALID", "/pattern"},
		{`,"pattern":"(?=x)"`, "E_SCHEMA_UNSUPPORTED", "/pattern"},
		{`,"minLength":2,"maximum":4`, "E_SCHEMA_UNSUPPORTED", "/maximum"},
		{`,"minLength":2,"default":"x"`, "E_SCHEMA_INVALID", "/default"},
		{`,"minLength":2,"default":"unknown"`, "E_SCHEMA_INVALID", "/default"},
		{`,"minLength":2,"default":null`, "E_SCHEMA_INVALID", "/default"},
		{`,"minLength":2,"format":"unsupported","default":"one"`, "E_SCHEMA_UNSUPPORTED", "/format"},
	} {
		for _, required := range []bool{false, true} {
			doc := fieldDoc(`{"enum":["one","x"]` + tc.extra + `}`)
			if !required {
				doc = []byte(strings.Replace(string(doc), `,"required":["value"]`, "", 1))
			}
			enumBoundsDiagnostic(t, doc, true, tc.code, base+tc.tail)
		}
	}
	for _, body := range []string{
		`{"Use":{"properties":{"value":{"enum":["one"],"minLength":2,"x-kotlin-value-class":true}}}}`,
		`{"Use":{"allOf":[{"properties":{"value":{"enum":["one"],"minLength":2}}},{"properties":{"value":{"type":"string"}}}]}}`,
	} {
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = true
		if got, err := core.GenerateWithDiagnostics(rq); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("bounded field granted excluded source role", body, err)
		}
	}
}

func TestStringEnumFieldBoundsPatchDefaultValidation(t *testing.T) {
	for _, required := range []bool{false, true} {
		body := `{"properties":{"state":{"type":"string","nullable":true,"format":"date","enum":["one","x"],"minLength":2,"default":"one"}}`
		if required {
			body += `,"required":["state"]`
		}
		body += `}`
		doc := []byte(`{"openapi":"3.1.0","paths":{"/update":{"patch":{"requestBody":{"required":true,"content":{"application/json":{"schema":` + body + `}}},"responses":{"204":{}}}}}}`)
		rq := request(doc, "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("checked enum Patch own-default", required, err)
		}
		found := false
		for _, f := range got.Artifacts.Files {
			found = found || strings.Contains(f.Content, "serialName.let")
		}
		if !found {
			t.Fatal("Patch lost original enum payload checks")
		}
		rq.Document = []byte(strings.Replace(string(doc), `"default":"one"`, `"default":"x"`, 1))
		got, err = core.GenerateWithDiagnostics(rq)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/paths/~1update/patch/requestBody/content/application~1json/schema/properties/state/default" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("required/Patch suppression hid invalid enum annotation", required, err)
		}
	}
}
