package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func directNullableCountInput(version, root, bounds string) []byte {
	return []byte(`{"openapi":"` + version + `","components":{"schemas":{"Record":{` + root + bounds + `,"properties":{"value":{"type":"string"}}}}}}`)
}

func TestDirectNullableComponentCountsNoCountControl(t *testing.T) {
	for _, tc := range []struct{ version, root string }{
		{"3.0.3", `"type":"object","nullable":true`},
		{"3.1.0", `"type":["object","null"]`},
	} {
		for _, validation := range []bool{false, true} {
			rq := request(directNullableCountInput(tc.version, tc.root, ""), "")
			rq.Options.Validation = validation
			r, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !strings.Contains(source(t, r, "Record"), "public data class Record(") {
				t.Fatal(tc, validation, r, err)
			}
			t.Logf("accepted original no-count control: %s %s validation=%t", tc.version, tc.root, validation)
		}
	}
}

func TestDirectNullableComponentCountsFixtureCodecAPIAndReuse(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/direct-nullable-component-counts.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, merge := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.Validation, rq.Options.MergeIdenticalModels, rq.Options.ModelStyle = true, merge, style
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(err)
			}
			if text := source(t, got, "RecordDto"); !strings.Contains(text, "generatedSerializer()") || strings.Contains(text, "typealias") || strings.Contains(text, "init {") {
				t.Fatal(text)
			}
			if text := source(t, got, "RecordTwinDto"); strings.Contains(text, "typealias") != merge {
				t.Fatal("equivalent counts/classifier reuse", text)
			}
			for _, name := range []string{"DefaultedDto", "ZeroDto", "CheckedDto"} {
				if strings.Contains(source(t, got, name), "typealias") {
					t.Fatal("different retained semantics merged", name)
				}
			}
			patch := source(t, got, "PatchInputDto")
			for _, token := range []string{"requiredNull: org.example.models.models.RecordDto?,", "note: org.example.models.models.Patch<org.example.models.models.RecordDto>", "narrow: org.example.models.models.RecordDto? = null"} {
				if !strings.Contains(patch, token) {
					t.Fatal(token, patch)
				}
			}
			if !strings.Contains(source(t, got, "NarrowDto"), "= org.example.models.models.RecordDto\n") || !strings.Contains(source(t, got, "RenullDto"), "= org.example.models.models.RecordDto?\n") {
				t.Fatal("nearest nullable alias intent changed")
			}
			changed := request([]byte(strings.Replace(string(input), `"nullable": true, "minProperties": 1, "maxProperties": 2`, `"nullable": true, "minProperties": 1, "maxProperties": 3`, 1)), "Dto")
			changed.Options = rq.Options
			distinct, err := core.GenerateWithDiagnostics(changed)
			if err != nil || strings.Contains(source(t, distinct, "RecordTwinDto"), "typealias") {
				t.Fatal("different effective root count lost in merging", err)
			}
		}
	}
}

func directNullableCountOperations(maximum string) []byte {
	counted := `{"type":["object","null"],"minProperties":0,"maxProperties":` + maximum + `,"properties":{"value":{"type":"string"}}}`
	model := `{"properties":{"direct":{"$ref":"#/components/schemas/Record"},"owned":` + counted + `}}`
	op := `{"get":{"operationId":"alpha","responses":{"200":{"content":{"application/json":{"schema":` + model + `}}}}}}`
	return []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":` + counted + `}},"paths":{"/alpha":` + op + `,"/beta":` + strings.ReplaceAll(op, `"alpha"`, `"beta"`) + `}}`)
}

func TestDirectNullableComponentCountsOwnedLockAndOptimizedIdentity(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, qualified := range []bool{false, true} {
			rq := request(directNullableCountOperations("1"), "Dto")
			rq.Options = core.GenerationOptions{PackageName: rq.Options.PackageName, ModelNameSuffix: "Dto", ModelStyle: style, Validation: true, HoistSharedShapes: true, LockHoistedNames: true, QualifyHoistedNames: qualified, DeduplicateModels: true, MergeIdenticalModels: true}
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil || len(got.Artifacts.NameLock) == 0 || !strings.Contains(source(t, got, "BetaResponseDto"), "typealias") {
				t.Fatal(err)
			}
			rq.PreviousNameLock = got.Artifacts.NameLock
			again, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("lock replay drift", err)
			}
			rq.Document = directNullableCountOperations("2")
			changed, err := core.GenerateWithDiagnostics(rq)
			if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
				t.Fatal("owned count identity drift ignored", err)
			}
		}
	}
}

func TestDirectNullableComponentCountsAdmission(t *testing.T) {
	for _, tc := range []struct{ version, root string }{
		{"3.0.3", `"type":"object","nullable":true`},
		{"3.1.0", `"type":["object","null"]`},
	} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			rq := request(directNullableCountInput(tc.version, tc.root, `,"minProperties":1,"maxProperties":2`), "Dto")
			rq.Options.Validation = true
			rq.Options.ModelStyle = style
			r, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				diagnostics, ok := err.(core.Diagnostics)
				if !ok || len(diagnostics) != 1 || diagnostics[0].Pointer == nil {
					t.Fatal(err)
				}
				t.Errorf("direct explicit nullable component counts should generate (%s %s): %v at %s", tc.version, style, err, *diagnostics[0].Pointer)
				continue
			}
			text := source(t, r, "RecordDto")
			for _, token := range []string{"generatedSerializer()", "element.size < 1", "element.size > 2", "decodeFromJsonElement(delegate, element)", "encodeToJsonElement(delegate, value)"} {
				if !strings.Contains(text, token) {
					t.Fatal(token, text)
				}
			}
			again, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("determinism", err)
			}
		}
	}
}
