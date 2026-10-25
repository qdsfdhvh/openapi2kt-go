package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestDTOSpecialCollectionReferenceInlineControls(t *testing.T) {
	for _, format := range []string{"uuid", "date", "date-time"} {
		rq := request(schemas(`{"Item":{"properties":{"value":{"type":"array","items":{"type":"string","format":"`+format+`"}}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("existing inline mapped collection control", format, err)
		}
	}
}

func TestDTOSpecialCollectionReferenceGetterUnique(t *testing.T) {
	for _, format := range []string{"uuid", "date", "date-time"} {
		rq := request(schemas(`{"Value":{"type":"array","items":{"type":"string","format":"`+format+`"}},"Item":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("original referenced mapped collection getter", format, err)
		}
		rq.Document = []byte(strings.Replace(string(rq.Document), `"value":{"$ref":"#/components/schemas/Value"}`, `"value":{"$ref":"#/components/schemas/Value","default":[]}`, 1))
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("exact former UUID/default-empty ref role", format, err)
		}
	}
}

func TestDTOSpecialCollectionReferenceOwnDefaultMappedKeys(t *testing.T) {
	for _, tc := range []struct {
		format, first, second string
		dictionary            bool
	}{
		{"uuid", "550E8400-E29B-41D4-A716-446655440000", "550e8400-e29b-41d4-a716-446655440000", false},
		{"date", "2024-02-29", "2024-02-29", false},
		{"date-time", "2024-01-01T00:00:00Z", "2024-01-01T01:00:00+01:00", true},
	} {
		var target, first, second string
		if tc.dictionary {
			target = `{"type":"object","additionalProperties":{"type":"string","format":"` + tc.format + `"}}`
			first = `{"x":"` + tc.first + `"}`
			second = `{"x":"` + tc.second + `"}`
		} else {
			target = `{"type":"array","items":{"type":"string","format":"` + tc.format + `"}}`
			first = `["` + tc.first + `"]`
			second = `["` + tc.second + `"]`
		}
		rq := request(schemas(`{"Value":`+target+`,"Alias":{"$ref":"#/components/schemas/Value"},"Item":{"properties":{"value":{"$ref":"#/components/schemas/Alias","default":`+first+`}}},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true,"default":[{}, {"value":`+second+`}]}}}}`), "Dto")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Holder/properties/values/default/1" || len(r.Artifacts.Files) != 0 {
			t.Fatal("mapped temporal/UUID default equality", tc, err)
		}
	}
}

func TestDTOSpecialCollectionReferenceStrictDefaultsBeforeSuppression(t *testing.T) {
	for _, format := range []string{"uuid", "date", "date-time"} {
		rq := request(schemas(`{"Value":{"type":"array","items":{"type":"string","format":"`+format+`"}},"Item":{"properties":{"value":{"$ref":"#/components/schemas/Value","default":["invalid"]}},"required":["value"]},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || *ds[0].Pointer != "/components/schemas/Item/properties/value/default/0" || len(r.Artifacts.Files) != 0 {
			t.Fatal("strict mapped default before required suppression", format, err)
		}
	}
}
