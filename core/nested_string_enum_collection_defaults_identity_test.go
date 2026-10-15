package core_test

import (
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func nestedEnumDefaultOperations(field string) []byte {
	op := `{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":` + field + `}}}}}}}}`
	return []byte(`{"openapi":"3.1.0","paths":{"/alpha":` + op + `,"/beta":` + op + `}}`)
}

func TestNestedStringEnumCollectionDefaultsCanonicalOwnerOptimizationAndFrozenIdentity(t *testing.T) {
	field := `{"additionalProperties":{"additionalProperties":{"enum":["one","two"],"nullable":true}},"default":{"z":{"z":null,"a":"two"},"a":{"~":"one","/":"two","$ref":"one","type":"two"}}}`
	for _, checked := range []bool{false, true} {
		for _, mode := range []string{"none", "dedup", "merge", "hoist", "combined"} {
			r := request(nestedEnumDefaultOperations(field), "Dto")
			r.Options.Validation = checked
			r.Options.DeduplicateModels = mode == "dedup" || mode == "combined"
			r.Options.MergeIdenticalModels = mode == "merge" || mode == "combined"
			r.Options.HoistSharedShapes = mode == "hoist" || mode == "combined"
			r.Options.LockHoistedNames = r.Options.HoistSharedShapes
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(mode, err)
			}
			all := ""
			for _, f := range got.Artifacts.Files {
				all += f.Content
			}
			owner := "org.example.models.api.models.GetAlphaResponseDto.ValueValueValue"
			if r.Options.HoistSharedShapes {
				owner = "org.example.models.models.ResponseDto.ValueValueValue"
			}
			if !strings.Contains(all, "kotlin.collections.Map<kotlin.String, kotlin.collections.Map<kotlin.String, "+owner+"?>>") || strings.Contains(all, "serializer<kotlin.collections.Map<kotlin.String, kotlin.collections.Map<kotlin.String, ValueValueValue") {
				t.Fatal("lost final lexical helper", mode, all)
			}
			if mode != "none" && !strings.Contains(all, "typealias GetBetaResponseDto =") {
				t.Fatal("no actual optimization", mode, all)
			}
			if r.Options.LockHoistedNames {
				r.PreviousNameLock = got.Artifacts.NameLock
			}
			r.Document = nestedEnumDefaultOperations(strings.Replace(field, `{"z":{"z":null,"a":"two"},"a":{"~":"one","/":"two","$ref":"one","type":"two"}}`, `{"a":{"type":"two","$ref":"one","/":"two","~":"one"},"z":{"a":"two","z":null}}`, 1))
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("deep canonical drift", mode, err)
			}
			if !r.Options.LockHoistedNames {
				continue
			}
			for _, changed := range []string{
				strings.Replace(field, `"a":"two"`, `"a":"one"`, 1),
				strings.Replace(field, `"one","two"`, `"two","one"`, 1),
				strings.Replace(field, `"nullable":true`, `"nullable":false`, 1),
			} {
				// Nullable narrowing must fail its original null annotation first.
				r.Document = nestedEnumDefaultOperations(changed)
				report, err := core.GenerateWithDiagnostics(r)
				if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" && err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" {
					t.Fatal("frozen identity lost", err)
				}
			}
			if checked {
				r.Document = nestedEnumDefaultOperations(strings.Replace(field, `"additionalProperties":{"enum"`, `"maxProperties":4,"additionalProperties":{"enum"`, 1))
				if _, err := core.GenerateWithDiagnostics(r); err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" {
					t.Fatal("deep counts lost", err)
				}
			}
		}
	}
}

func TestNestedStringEnumCollectionDefaultsRequiredControlLexicalAndDiscarded(t *testing.T) {
	field := `{"type":"array","items":{"type":"array","items":{"enum":["one","two"]}},"default":[["one"]]}`
	control := strings.Replace(field, `,"default":[["one"]]`, "", 1)
	for _, checked := range []bool{false, true} {
		r := request(fieldDoc(field), "Dto")
		r.Options.Validation = checked
		required, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(err)
		}
		r.Document = fieldDoc(control)
		pure, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(required, pure) {
			t.Fatal("required annotation changed control output", err)
		}
		optional := schemas(`{"Holder":{"properties":{"value":` + field + `}},"Use":{"properties":{"leaf":{"$ref":"#/components/schemas/Holder/properties/value/items/items"}}},"Alias":{"$ref":"#/components/schemas/Holder/properties/value/items/items"}}`)
		r.Document = optional
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal("original enum reference authority", err)
		}
		if !strings.Contains(source(t, got, "UseDto"), "org.example.models.models.HolderDto.ValueInnerInner") || !strings.Contains(source(t, got, "AliasDto"), "HolderDto.ValueInnerInner") {
			t.Fatal("alias fabricated")
		}
		r.Document = schemas(`{"Holder":{"allOf":[{"properties":{"value":` + field + `}},{"properties":{"value":{"type":"string"}}}]}}`)
		discarded, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(source(t, discarded, "HolderDto"), "ValueInnerInner") {
			t.Fatal("scratch enum leaked")
		}
	}
}
