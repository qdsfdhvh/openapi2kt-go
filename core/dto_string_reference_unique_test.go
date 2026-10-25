package core_test

import (
	"testing"

	"openapi-kt/core"
)

func TestDTOStringEnumReferenceGetterUnique(t *testing.T) {
	for _, definition := range []string{`{"Value":{"type":"string","enum":["ready","done"]}}`, `{"Value":{"$ref":"#/components/schemas/Hops/properties/leaf"},"Hops":{"properties":{"leaf":{"enum":["ready","done"]}}}}`} {
		rq := request(schemas(definition[:len(definition)-1]+`,"Item":{"properties":{"value":{"$ref":"#/components/schemas/Value","default":"ready"}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		if _, e := core.GenerateWithDiagnostics(rq); e != nil {
			t.Fatal("String enum original ref getter", definition, e)
		}
	}
}

func TestDTOStringReferenceFormerBoundary(t *testing.T) {
	rq := request(schemas(`{"Value":{"type":"string","enum":["a","b"]},"Item":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
	rq.Options.Validation = true
	if _, e := core.GenerateWithDiagnostics(rq); e != nil {
		t.Fatal("exact former String reference getter boundary", e)
	}
}

func TestDTOStringReferenceOwnDefaultsAndStrictVocabulary(t *testing.T) {
	for _, tc := range []struct{ definition, values string }{
		{`{"Value":{"type":"string","enum":["ready","done"]}}`, `[{}, {"value":"ready"}]`},
		{`{"Value":{"$ref":"#/components/schemas/Hops/properties/leaf"},"Hops":{"properties":{"leaf":{"enum":["ready","done"]}}}}`, `[{}, {"value":"ready"}]`},
	} {
		rq := request(schemas(tc.definition[:len(tc.definition)-1]+`,"Item":{"properties":{"value":{"$ref":"#/components/schemas/Value","default":"ready"}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true,"default":`+tc.values+`}}}}`), "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/values/default/1" || len(r.Artifacts.Files) != 0 {
			t.Fatal("String reference initializer mapped key", e)
		}
	}
	rq := request(schemas(`{"Value":{"type":"string","enum":["ready","done"]},"Item":{"properties":{"value":{"$ref":"#/components/schemas/Value","default":"future"}},"required":["value"]},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
	rq.Options.Validation = true
	r, e := core.GenerateWithDiagnostics(rq)
	ds, ok := e.(core.Diagnostics)
	if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Item/properties/value/default" || len(r.Artifacts.Files) != 0 {
		t.Fatal("String reference own default strict vocabulary", e)
	}
}
