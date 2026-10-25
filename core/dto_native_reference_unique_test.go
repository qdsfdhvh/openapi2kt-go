package core_test

import (
	"testing"

	"openapi-kt/core"
)

func TestDTONativeEnumReferenceGetterUnique(t *testing.T) {
	for _, definition := range []string{`{"Value":{"type":"integer","enum":[0,1]}}`, `{"Value":{"type":"number","format":"float","enum":[0,1]},"Alias":{"$ref":"#/components/schemas/Value"}}`, `{"Value":{"$ref":"#/components/schemas/Hops/properties/leaf"},"Hops":{"properties":{"leaf":{"type":"boolean","enum":[false,true]}}}}`} {
		rq := request(schemas(definition[:len(definition)-1]+`,"Item":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		if _, e := core.GenerateWithDiagnostics(rq); e != nil {
			t.Fatal("native enum original ref getter", definition, e)
		}
	}
}

func TestDTONativeReferenceOwnDefaultMappedKeys(t *testing.T) {
	for _, tc := range []struct{ definition, field, values string }{
		{`{"Value":{"type":"number","format":"float","enum":[16777216,16777217]}}`, `{"$ref":"#/components/schemas/Value","default":16777216}`, `[{}, {"value":16777217}]`},
		{`{"Value":{"type":"boolean","enum":[false,true]},"Alias":{"$ref":"#/components/schemas/Value"}}`, `{"$ref":"#/components/schemas/Alias","default":false}`, `[{}, {"value":false}]`},
		{`{"Value":{"$ref":"#/components/schemas/Hops/properties/value"},"Hops":{"properties":{"value":{"type":"integer","format":"int64","enum":[0,1]}}}}`, `{"$ref":"#/components/schemas/Value","default":0}`, `[{}, {"value":0}]`},
	} {
		rq := request(schemas(tc.definition[:len(tc.definition)-1]+`,"Item":{"properties":{"value":`+tc.field+`}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true,"default":`+tc.values+`}}}}`), "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/values/default/1" || len(r.Artifacts.Files) != 0 {
			t.Fatal("native reference default mapped key", tc.field, e)
		}
	}
}
func TestDTONativeReferenceOriginalDefaultFailure(t *testing.T) {
	rq := request(schemas(`{"Value":{"type":"integer","enum":[0,1]},"Item":{"properties":{"value":{"$ref":"#/components/schemas/Value","default":2}},"required":["value"]},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
	rq.Options.Validation = true
	r, e := core.GenerateWithDiagnostics(rq)
	ds, ok := e.(core.Diagnostics)
	if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Item/properties/value/default" || len(r.Artifacts.Files) != 0 {
		t.Fatal("native reference own default strict vocabulary", e)
	}
}
