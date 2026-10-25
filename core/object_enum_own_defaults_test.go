package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestObjectEnumOwnDefaultsInIndependentFactories(t *testing.T) {
	for _, shape := range []string{
		`"type":"object","properties":{"mode":{"type":"string","enum":["on","off"],"default":"on"},"count":{"type":"integer","default":7}}`,
		`"allOf":[{"properties":{"mode":{"type":"string","enum":["on","off"],"default":"on"}}},{"properties":{"count":{"type":"integer","default":7}}}]`,
	} {
		input := schemas(`{"Target":{` + shape + `,"default":{"mode":"off","count":1}},"Alias":{"$ref":"#/components/schemas/Target","nullable":true,"default":null},"Outer":{"$ref":"#/components/schemas/Alias","nullable":false,"default":{}},"Holder":{"properties":{"target":{"$ref":"#/components/schemas/Target"}}}}`)
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, checked := range []bool{false, true} {
				rq := request(input, "Dto")
				rq.Options.ModelStyle, rq.Options.Validation = style, checked
				first, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal("own String enum field default lost inside factory", err)
				}
				for _, name := range []string{"TargetDto", "AliasDto", "OuterDto"} {
					if !strings.Contains(source(t, first, name), "default") {
						t.Fatal("independent factory lost", name)
					}
				}
				if strings.Contains(source(t, first, "HolderDto"), "fun default()") {
					t.Fatal("factory inherited into holder")
				}
				again, err := core.GenerateWithDiagnostics(rq)
				if err != nil || !reflect.DeepEqual(first, again) {
					t.Fatal("factory source/determinism changed", err)
				}
			}
		}
	}
}

func TestObjectEnumOwnDefaultsKeepSourceAndValuePointers(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ input, pointer string }{
			{`{"Target":{"properties":{"mode":{"type":"string","enum":["on"],"default":"bad"}}},"Alias":{"$ref":"#/components/schemas/Target","nullable":true,"default":null}}`, "/components/schemas/Target/properties/mode/default"},
			{`{"Target":{"properties":{"mode":{"type":"string","enum":["on"],"default":"on"}}},"Alias":{"$ref":"#/components/schemas/Target","default":{"mode":"bad"}}}`, "/components/schemas/Alias/default/mode"},
			{`{"Target":{"required":["mode"],"properties":{"mode":{"type":"string","enum":["on"],"default":"on"}}},"Alias":{"$ref":"#/components/schemas/Target","default":{}}}`, "/components/schemas/Alias/default"},
		} {
			rq := request(schemas(tc.input), "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != tc.pointer {
				t.Fatal("source or annotation diagnostic moved", tc.pointer, err)
			}
		}
	}
}

func TestStructuredWrapperUnsupportedSourceKeepsOriginalPointer(t *testing.T) {
	input := schemas(`{"Target":{"type":"object","properties":{"value":{"type":"string"}},"minProperties":1},"Alias":{"$ref":"#/components/schemas/Target","nullable":true,"default":null}}`)
	rq := request(input, "Dto")
	rq.Options.Validation = true
	got, err := core.GenerateWithDiagnostics(rq)
	if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_UNSUPPORTED" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Target/minProperties" {
		t.Fatal("unsupported structured source lost original Pointer or gained permission", err)
	}
}
