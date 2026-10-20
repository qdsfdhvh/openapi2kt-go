package core_test

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

// UUID uses the existing String contract, including Patch, typed collections,
// internal hops, HTTP and SSE. Removing its format must preserve every artifact.
func TestNullableUUIDStringParity(t *testing.T) {
	for _, tc := range []struct{ fixture, root string }{
		{"intrinsic-nullable-primitives", "ZString"},
		{"root-nullable-primitives", "ZBaseString"},
		{"builtin-reference-chains", "Text"},
	} {
		input, err := os.ReadFile("../tests/fixtures/http/" + tc.fixture + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var doc map[string]any
		if err := json.Unmarshal(input, &doc); err != nil {
			t.Fatal(err)
		}
		// Re-encode both controls identically to preserve property order parity.
		input, err = json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
		catalog[tc.root].(map[string]any)["format"] = "uuid"
		uuid, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				for _, suffix := range []string{"", "Dto"} {
					r := request(input, suffix)
					r.Options.Validation, r.Options.ModelStyle = checked, style
					control, err := core.GenerateWithDiagnostics(r)
					if err != nil {
						t.Fatal(err)
					}
					r.Document = uuid
					got, err := core.GenerateWithDiagnostics(r)
					if err != nil || !reflect.DeepEqual(got, control) {
						t.Fatal("nullable UUID changed String API/wire/source ownership", tc.fixture, checked, style, suffix, err)
					}
					again, err := core.GenerateWithDiagnostics(r)
					if err != nil || !reflect.DeepEqual(got, again) {
						t.Fatal("UUID chain nondeterminism", err)
					}
				}
			}
		}
	}
}

func TestNullableUUIDDefaultOwners(t *testing.T) {
	for _, intrinsic := range []string{"", `,"nullable":true`, `,"type":["string","null"]`} {
		// A type array replaces the explicit type instead of duplicating it.
		shape := `"type":"string","format":"uuid"` + intrinsic
		if strings.Contains(intrinsic, `"type"`) {
			shape = `"format":"uuid"` + intrinsic
		}
		for _, targetDefault := range []string{`"target"`, `null`} {
			if intrinsic == "" && targetDefault == "null" {
				continue
			}
			body := `{"T":{` + shape + `,"minLength":2,"default":` + targetDefault + `},"M":{"$ref":"#/components/schemas/T","nullable":true,"default":null},"N":{"$ref":"#/components/schemas/M","nullable":false,"default":"outer"},"Again":{"$ref":"#/components/schemas/N","nullable":true,"default":null},"Plain":{"$ref":"#/components/schemas/N"},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/N"}}}}`
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				r := request(schemas(body), "Dto")
				r.Options.Validation, r.Options.ModelStyle = true, style
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(intrinsic, targetDefault, err)
				}
				for _, name := range []string{"MDto", "AgainDto"} {
					if !strings.Contains(source(t, got, name), "= kotlin.String?") {
						t.Fatal("nullable alias lost nearest intent", name)
					}
				}
				if !strings.Contains(source(t, got, "NDto"), "= kotlin.String\n") {
					t.Fatal("false alias did not narrow", source(t, got, "NDto"))
				}
				r.Document = schemas(strings.Replace(body, `,"default":`+targetDefault, "", 1))
				control, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"MDto", "NDto", "AgainDto", "PlainDto", "HolderDto"} {
					if source(t, got, name) != source(t, control, name) {
						t.Fatal("UUID default was inherited by another owner", name)
					}
				}
			}
		}
	}
	for _, tc := range []struct{ shape, value, tail string }{
		{`"type":"string","nullable":true,"minLength":-1`, `null`, "minLength"},
		{`"type":["string","null"],"pattern":"["`, `null`, "pattern"},
		{`"type":"string","nullable":true,"minLength":2`, `"x"`, "default"},
		{`"type":["string","null"]`, `3`, "default"},
	} {
		r := request(schemas(`{"T":{`+tc.shape+`,"format":"uuid","default":`+tc.value+`}}`), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) == 0 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/T/"+tc.tail || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("nullable UUID default hid malformed source/value", tc, err)
		}
	}
}
