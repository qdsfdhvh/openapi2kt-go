package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNestedNativeEnumCollectionDefaultsCanonicalMapsAndFrozenTypeCheckDefaultIdentity(t *testing.T) {
	field := `{"additionalProperties":{"additionalProperties":{"type":"number","format":"float","nullable":true,"enum":[1,1.000000001,1.25,2]}},"default":{"z":{"z":null,"a":1.25},"a":{"~":1,"/":2,"$ref":1.25,"type":2}}}`
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, mode := range []string{"none", "dedup", "merge", "hoist", "combined"} {
				r := request(nestedBuiltinDefaultOperations(field), "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				r.Options.DeduplicateModels = mode == "dedup" || mode == "combined"
				r.Options.MergeIdenticalModels = mode == "merge" || mode == "combined"
				r.Options.HoistSharedShapes = mode == "hoist" || mode == "combined"
				r.Options.LockHoistedNames = r.Options.HoistSharedShapes
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(mode, checked, style, err)
				}
				owner := "AlphaResponseDto"
				if r.Options.HoistSharedShapes {
					owner = "ResponseDto"
					if len(got.Artifacts.NameLock) == 0 || !strings.Contains(source(t, got, "AlphaResponseDto"), "typealias AlphaResponseDto = org.example.models.models.ResponseDto") {
						t.Fatal("no actual hoist/lock", mode)
					}
				}
				if !strings.Contains(source(t, got, owner), "serializer<kotlin.collections.Map<kotlin.String, kotlin.collections.Map<kotlin.String, kotlin.Float?>>>") {
					t.Fatal("helper lost actual native tree", mode)
				}
				if mode != "none" && !strings.Contains(source(t, got, "BetaResponseDto"), "typealias BetaResponseDto =") {
					t.Fatal("no actual optimization", mode)
				}
				r.PreviousNameLock = got.Artifacts.NameLock
				// Both alternatives are exact original vocabulary members. Only
				// after membership may mapped Float/default identity canonicalize.
				r.Document = nestedBuiltinDefaultOperations(strings.Replace(field, `{"z":{"z":null,"a":1.25},"a":{"~":1,"/":2,"$ref":1.25,"type":2}}`, `{"a":{"type":2,"$ref":1.250,"/":2,"~":1.000000001},"z":{"a":1.2500,"z":null}}`, 1))
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("deep canonical/mapped member drift", mode, checked, style, err)
				}
				if !r.Options.LockHoistedNames {
					continue
				}
				for _, changed := range []string{
					strings.Replace(field, `"a":1.25`, `"a":2`, 1),
					strings.Replace(field, `"format":"float"`, `"format":"double"`, 1),
					strings.Replace(field, `"additionalProperties":{"additionalProperties"`, `"additionalProperties":{"nullable":true,"additionalProperties"`, 1),
				} {
					r.Document = nestedBuiltinDefaultOperations(changed)
					nestedNativeEnumDefaultFailure(t, r, "E_NAME_LOCK_CHANGED", "")
				}
				// Vocabulary is checked identity only in validation mode. Off
				// still approximates supplied values without a wrapper/check.
				r.Document = nestedBuiltinDefaultOperations(strings.Replace(field, `1.25,2]`, `1.25,2,3]`, 1))
				if checked {
					nestedNativeEnumDefaultFailure(t, r, "E_NAME_LOCK_CHANGED", "")
				} else if replay, err := core.GenerateWithDiagnostics(r); err != nil || !reflect.DeepEqual(got, replay) {
					t.Fatal("default-off vocabulary acquired runtime identity", err)
				}
				if checked {
					for _, changed := range []string{
						strings.Replace(field, `"nullable":true`, `"nullable":true,"minimum":0`, 1),
						strings.Replace(field, `"additionalProperties":{"additionalProperties"`, `"additionalProperties":{"maxProperties":8,"additionalProperties"`, 1),
					} {
						r.Document = nestedBuiltinDefaultOperations(changed)
						nestedNativeEnumDefaultFailure(t, r, "E_NAME_LOCK_CHANGED", "")
					}
				}
			}
		}
	}
}

func TestNestedNativeEnumCollectionDefaultsDistinctEffectiveDefaultsDoNotMerge(t *testing.T) {
	field := nestedDefaultChain(2, `{"type":"integer","enum":[1,2]}`, `[[1]]`)
	for _, checked := range []bool{false, true} {
		for _, mode := range []string{"dedup", "merge", "hoist", "combined"} {
			r := request(nestedBuiltinDefaultOperations(field), "Dto")
			r.Options.Validation = checked
			r.Options.DeduplicateModels = mode == "dedup" || mode == "combined"
			r.Options.MergeIdenticalModels = mode == "merge" || mode == "combined"
			r.Options.HoistSharedShapes = mode == "hoist" || mode == "combined"
			index := strings.Index(string(r.Document), `"/b":`)
			r.Document = []byte(string(r.Document[:index]) + strings.Replace(string(r.Document[index:]), `[[1]]`, `[[2]]`, 1))
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(mode, checked, err)
			}
			for _, name := range []string{"AlphaResponseDto", "BetaResponseDto"} {
				text := source(t, got, name)
				if strings.Contains(text, "typealias "+name) || !strings.Contains(text, "class "+name) {
					t.Fatal("different initializers collapsed", mode, checked, name, text)
				}
			}
		}
	}
}

func TestNestedNativeEnumCollectionDefaultsRequiredAndPatchHaveNoRuntimeDefaultIdentity(t *testing.T) {
	field := nestedDefaultChain(2, `{"type":"integer","enum":[1,2]}`, `[[1]]`)
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, context := range []string{"required", "requiredNullable", "patch"} {
				base := field
				if context != "required" {
					base = strings.Replace(base, `"type":"array"`, `"type":"array","nullable":true`, 1)
				}
				build := func(f string) []byte {
					if context == "patch" {
						return httpDoc(`{"operationId":"save","requestBody":{"required":true,"content":{"application/json":{"schema":{"properties":{"value":` + f + `}}}}},"responses":{"204":{}}}`)
					}
					return fieldDoc(f)
				}
				r := request(build(base), "Dto")
				r.Options.Validation, r.Options.ModelStyle = checked, style
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(context, err)
				}
				name := "RecordDto"
				if context == "patch" {
					name = "SaveInputDto"
					if !strings.Contains(source(t, got, name), "Patch<kotlin.collections.List<kotlin.collections.List<kotlin.Int>>> = org.example.models.models.Patch.Absent") {
						t.Fatal("annotation replaced Patch.Absent", source(t, got, name))
					}
				}
				if strings.Contains(source(t, got, name), "openapi-kt default tree") {
					t.Fatal("suppressed annotation emitted a runtime helper", context)
				}
				alternates := []string{strings.Replace(base, `[[1]]`, `[[2]]`, 1)}
				if context == "requiredNullable" {
					// Historical shared nullable fields without a schema default
					// retain = null; this newly admitted annotation instead keeps
					// the required constructor argument, not a literal default.
					if !strings.Contains(source(t, got, name), "public val value: kotlin.collections.List<kotlin.collections.List<kotlin.Int>>?,") {
						t.Fatal("required nullable default became an initializer")
					}
				} else {
					alternates = append(alternates, strings.Replace(base, `,"default":[[1]]`, "", 1))
				}
				for _, alternate := range alternates {
					r.Document = build(alternate)
					control, err := core.GenerateWithDiagnostics(r)
					if err != nil || !reflect.DeepEqual(got, control) {
						t.Fatal("nonemitted default changed runtime identity", context, checked, style, err)
					}
				}
			}
		}
	}
	// Replay actual hoisted locks while changing valid required/Patch annotations.
	for _, checked := range []bool{false, true} {
		for _, patch := range []bool{false, true} {
			doc := nestedBuiltinDefaultOperations(field)
			doc = []byte(strings.ReplaceAll(string(doc), `"properties":{"value":`+field+`}`, `"properties":{"value":`+field+`},"required":["value"]`))
			if patch {
				nullable := strings.Replace(field, `"type":"array"`, `"type":"array","nullable":true`, 1)
				op := `{"post":{"operationId":"alpha","requestBody":{"required":true,"content":{"application/json":{"schema":{"properties":{"value":` + nullable + `}}}}},"responses":{"204":{}}}}`
				doc = []byte(`{"openapi":"3.1.0","paths":{"/a":` + op + `,"/b":` + strings.Replace(op, `"alpha"`, `"beta"`, 1) + `}}`)
			}
			r := request(doc, "Dto")
			r.Options.Validation, r.Options.HoistSharedShapes, r.Options.LockHoistedNames = checked, true, true
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil || len(got.Artifacts.NameLock) == 0 {
				t.Fatal(err)
			}
			r.PreviousNameLock = got.Artifacts.NameLock
			r.Document = []byte(strings.ReplaceAll(string(doc), `[[1]]`, `[[2]]`))
			replay, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, replay) {
				t.Fatal("required/Patch annotation acquired frozen default identity", patch, err)
			}
		}
	}
}
