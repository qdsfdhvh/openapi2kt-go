package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestInlineEnumCollectionReferenceChildOriginalGraduation(t *testing.T) {
	// The former refusal remains an exact positive control.
	body := `{"Target":{"type":"array","items":{"type":"string","enum":["a"]}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Target","default":["a"]}}}`
	for _, checked := range []bool{false, true} {
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("former inline enum collection child refusal", checked, err)
		}
	}
}

func TestInlineEnumCollectionReferenceChildDefaultIsIndependent(t *testing.T) {
	for _, tc := range []struct {
		name, leaf, target, alias, child, invalid string
	}{
		{"string", `"type":"string","enum":["a","b"]`, `"a"`, `"b"`, `"a"`, `"future"`},
		{"integer", `"type":"integer","enum":[1,2]`, `1`, `2`, `1`, `3`},
		{"number", `"type":"number","enum":[1.0,2.0]`, `1.0`, `2.0`, `1.0`, `3.0`},
		{"boolean", `"type":"boolean","enum":[true]`, `true`, `true`, `true`, `false`},
	} {
		for _, parent := range []struct{ name, schema, location string }{
			{"list", `"type":"array","items":`, "/items"},
			{"map", `"additionalProperties":`, "/additionalProperties"},
		} {
			t.Run(tc.name+"/"+parent.name, func(t *testing.T) {
				body := `{"Values":{"type":"array","items":{` + tc.leaf + `},"default":[` + tc.target + `]},` +
					`"Alias":{"$ref":"#/components/schemas/Values","default":[` + tc.alias + `]},` +
					`"Rows":{` + parent.schema + `{"$ref":"#/components/schemas/Alias","default":[` + tc.child + `]}}}`
				for _, checked := range []bool{false, true} {
					for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
						rq := request(schemas(body), "Dto")
						rq.Options.Validation, rq.Options.ModelStyle = checked, style
						annotated, err := core.GenerateWithDiagnostics(rq)
						if err != nil {
							t.Fatal("inline enum collection child rejected", checked, style, err)
						}
						rq.Document = schemas(strings.Replace(body, `,"default":[`+tc.child+`]}}}`, `}}}`, 1))
						control, err := core.GenerateWithDiagnostics(rq)
						if err != nil || !reflect.DeepEqual(annotated, control) {
							t.Fatal("child annotation changed generated output", checked, style, err)
						}
						invalid := strings.Replace(body, `"default":[`+tc.child+`]}}}`, `"default":[`+tc.invalid+`]}}}`, 1)
						carrierDefaultFailure(t, schemas(invalid), checked, "E_SCHEMA_INVALID", "/components/schemas/Rows"+parent.location+"/default/0")
					}
				}
			})
		}
	}
}

func TestInlineEnumCollectionChildDefaultBoundsAndNullable(t *testing.T) {
	body := `{"Values":{"type":"array","minItems":1,"items":{"type":"string","enum":["one","two"],"minLength":3,"pattern":"^[a-z]+$"}},` +
		`"Alias":{"$ref":"#/components/schemas/Values","nullable":false},` +
		`"Rows":{"additionalProperties":{"$ref":"#/components/schemas/Alias","nullable":true,"default":null}}}`
	rq := request(schemas(body), "Dto")
	rq.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("nullable checked enum collection child rejected", err)
	}
	for _, tc := range []struct{ before, after, pointer string }{
		{`"nullable":true,"default":null`, `"nullable":false,"default":null`, "/components/schemas/Rows/additionalProperties/default"},
		{`"default":null`, `"default":[]`, "/components/schemas/Rows/additionalProperties/default"},
		{`"default":null`, `"default":["future"]`, "/components/schemas/Rows/additionalProperties/default/0"},
	} {
		carrierDefaultFailure(t, schemas(strings.Replace(body, tc.before, tc.after, 1)), true, "E_SCHEMA_INVALID", tc.pointer)
	}
	valid := strings.Replace(body, `"default":null`, `"default":["two"]`, 1)
	rq.Document = schemas(valid)
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("valid checked enum collection child rejected", err)
	}
	withoutValidation := request(schemas(valid), "Dto")
	if _, err := core.GenerateWithDiagnostics(withoutValidation); err == nil {
		t.Fatal("checked source admitted without validation")
	}
}

func TestReferencedInlineEnumCollectionChildNestedAnnotations(t *testing.T) {
	body := `{"AOuter":{"type":"array","items":{"$ref":"#/components/schemas/BAlias","default":{"child":["two"]}}},` +
		`"BAlias":{"$ref":"#/components/schemas/CValues","default":{"alias":["one"]}},` +
		`"CValues":{"additionalProperties":{"type":"array","items":{"type":"string","format":"date","enum":["one","two"],"default":"one"}},"default":{"root":["two"]}}}`
	for _, checked := range []bool{false, true} {
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = checked
		annotated, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("nested independent enum annotations rejected", checked, err)
		}
		rq.Document = schemas(strings.Replace(body, `,"default":{"child":["two"]}`, "", 1))
		control, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(annotated, control) {
			t.Fatal("nested enum child changed generated output", checked, err)
		}
		for _, tc := range []struct{ before, after, pointer string }{
			{`"default":{"child":["two"]}`, `"default":{"child":["future"]}`, "/components/schemas/AOuter/items/default/child/0"},
			{`"default":{"alias":["one"]}`, `"default":{"alias":["future"]}`, "/components/schemas/BAlias/default/alias/0"},
			{`"default":{"root":["two"]}`, `"default":{"root":["future"]}`, "/components/schemas/CValues/default/root/0"},
			{`"default":"one"`, `"default":"future"`, "/components/schemas/CValues/additionalProperties/items/default"},
		} {
			carrierDefaultFailure(t, schemas(strings.Replace(body, tc.before, tc.after, 1)), checked, "E_SCHEMA_INVALID", tc.pointer)
		}
	}
}

func TestReferencedInlineEnumCollectionChildDefaultLimits(t *testing.T) {
	// Every independent annotation shares the existing cumulative source-work
	// budget, including retained enum classifier snapshots.
	for _, count := range []int{1, 18} {
		var body strings.Builder
		body.WriteString(`{"Values":{"type":"array","items":{"type":"string","enum":["one"]}}`)
		for i := 0; i < count; i++ {
			fmt.Fprintf(&body, `,"Holder%d":{"properties":{`, i)
			for j := 0; j < 180; j++ {
				if j != 0 {
					body.WriteByte(',')
				}
				fmt.Fprintf(&body, `"v%d":{"type":"array","items":{"$ref":"#/components/schemas/Values","default":["one"]}}`, j)
			}
			body.WriteString(`}}`)
		}
		body.WriteByte('}')
		report, err := core.GenerateWithDiagnostics(request(schemas(body.String()), "Dto"))
		if count == 1 && err != nil {
			t.Fatal("bounded enum collection children rejected", err)
		}
		if count == 18 {
			ds, ok := err.(core.Diagnostics)
			if !ok || len(ds) == 0 || ds[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{}) {
				t.Fatal("enum collection children escaped cumulative limit", err)
			}
		}
	}
}
