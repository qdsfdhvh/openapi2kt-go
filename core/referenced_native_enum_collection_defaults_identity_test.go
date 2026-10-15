package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func referencedNativeEnumOperations(value string) []byte {
	field := `{"$ref":"#/components/schemas/Rows","default":` + value + `}`
	doc := nestedBuiltinDefaultOperations(field)
	return []byte(strings.TrimSuffix(string(doc), "}") + `,"components":{"schemas":{"Code":{"type":"integer","format":"int32","enum":[1,2]},"Rows":{"additionalProperties":{"type":"array","items":{"$ref":"#/components/schemas/Code","nullable":true}}}}}}`)
}

func TestReferencedNativeEnumCollectionFieldDefaultsIdentity(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, mode := range []string{"dedup", "merge", "combined"} {
				r := request(referencedNativeEnumOperations(`{"z":[],"a":[1,null]}`), "Dto")
				r.Options.Validation = checked
				r.Options.ModelStyle = style
				r.Options.DeduplicateModels = mode != "merge"
				r.Options.MergeIdenticalModels = mode != "dedup"
				r.Options.HoistSharedShapes = mode == "combined"
				r.Options.LockHoistedNames = mode == "combined"
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(err)
				}
				canonical, alias := "AlphaResponseDto", "BetaResponseDto"
				if mode == "combined" {
					canonical, alias = "ResponseDto", "AlphaResponseDto"
				}
				if !strings.Contains(source(t, got, alias), "typealias "+alias) || !strings.Contains(source(t, got, canonical), "serializer<org.example.models.models.RowsDto>()") {
					t.Fatal("actual optimized alias/codecs lost")
				}
				r.PreviousNameLock = got.Artifacts.NameLock
				r.Document = referencedNativeEnumOperations(`{"a":[1,null],"z":[]}`)
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("canonical keys changed identity", err)
				}
				for _, value := range []string{`{"a":[2,null],"z":[]}`, `{"a":[null,1],"z":[]}`} {
					r.Document = referencedNativeEnumOperations(value)
					changed, err := core.GenerateWithDiagnostics(r)
					if mode == "combined" {
						if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
							t.Fatal("lexeme/array order lost", err)
						}
					} else if err != nil || reflect.DeepEqual(got, changed) {
						t.Fatal("effective identity lost", err)
					}
				}
			}
		}
	}
	for _, role := range []string{"required", "patch"} {
		holder := `{"required":["v"],"properties":{"v":{"$ref":"#/components/schemas/Codes","default":[1]}}}`
		doc := schemas(`{"Codes":{"type":"array","items":{"type":"integer","format":"int32","enum":[1,2]}},"Holder":` + holder + `}`)
		if role == "patch" {
			doc = []byte(`{"openapi":"3.1.0","components":{"schemas":{"Codes":{"type":"array","nullable":true,"items":{"type":"integer","format":"int32","enum":[1,2]}}}},"paths":{"/p":{"patch":{"requestBody":{"content":{"application/json":{"schema":{"properties":{"v":{"$ref":"#/components/schemas/Codes","default":[1]}}}}}},"responses":{"204":{}}}}}}`)
		}
		first, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if err != nil {
			t.Fatal(err)
		}
		second, err := core.GenerateWithDiagnostics(request([]byte(strings.ReplaceAll(string(doc), `"default":[1]`, `"default":[2]`)), "Dto"))
		if err != nil || !reflect.DeepEqual(first, second) {
			t.Fatal("suppressed annotation invented identity", role, err)
		}
	}
}
