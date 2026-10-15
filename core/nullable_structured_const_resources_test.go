package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullableStructuredConstReferencesRetainFullSourceHeightAndBodySlots(t *testing.T) {
	for _, width := range []int{240, 241} {
		fields := []string{}
		for i := range width {
			fields = append(fields, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Value"}`, i))
		}
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Value":{"const":{},"nullable":true},"Holder":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto"))
		if (err == nil) != (width == 240) || err != nil && (err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("Nullable JSON roots changed body serialization budget", width, err)
		}
	}
	for _, count := range []int{46, 47} {
		rows := []string{`"Base":{"const":[],"nullable":true}`}
		previous := "Base"
		for i := range count {
			name := fmt.Sprintf("Alias%d", i)
			nullable := ""
			if i%2 == 0 {
				nullable = `,"nullable":false`
			} else {
				nullable = `,"nullable":true`
			}
			rows = append(rows, fmt.Sprintf(`"%s":{"$ref":"#/components/schemas/%s"%s}`, name, previous, nullable))
			previous = name
		}
		rows = append(rows, fmt.Sprintf(`"Holder":{"properties":{"fixed":{"$ref":"#/components/schemas/%s"}}}`, previous))
		got, err := core.GenerateWithDiagnostics(request(schemas(`{`+strings.Join(rows, ",")+`}`), "Dto"))
		if (err == nil) != (count == 46) || err != nil && (err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("Nullable alias override dropped cached full source work", count, err)
		}
	}
}

func TestNullableStructuredConstCycleAndDefaultShapeLockRemainBounded(t *testing.T) {
	for _, validation := range []bool{false, true} {
		r := request(schemas(`{"Value":{"const":{},"nullable":true},"A":{"$ref":"#/components/schemas/B","nullable":true},"B":{"$ref":"#/components/schemas/A","nullable":false}}`), "Dto")
		r.Options.Validation = validation
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("Nullable aliases allowed an erased cycle", validation, err)
		}
	}
	op := func(id, root string) string {
		return `{"get":{"operationId":"` + id + `","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"nested":{"properties":{"label":{"type":"string"},"fixed":{"$ref":"#/components/schemas/` + root + `"}}}},"required":["nested"]}}}}}}}`
	}
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"First":{"const":{"n":1},"nullable":true},"Other":{"const":{"n":2},"nullable":true}}},"paths":{"/a":` + op("alpha", "First") + `,"/b":` + op("beta", "First") + `,"/c":` + op("gamma", "Other") + `}}`)
	r := request(input, "Dto")
	r.Options.HoistSharedShapes = true
	r.Options.DeduplicateModels = true
	r.Options.MergeIdenticalModels = true
	r.Options.LockHoistedNames = true
	got, err := core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source(t, got, "AlphaResponseDto"), "typealias") || strings.Contains(source(t, got, "GammaResponseDto"), "typealias") {
		t.Fatal("Nullable source collapsed different body defaults")
	}
	r.PreviousNameLock = got.Artifacts.NameLock
	again, err := core.GenerateWithDiagnostics(r)
	if err != nil || !reflect.DeepEqual(got, again) {
		t.Fatal("Nullable default lock replay drifted", err)
	}
	r.Document = []byte(strings.ReplaceAll(string(input), `"n":1`, `"n":3`))
	changed, err := core.GenerateWithDiagnostics(r)
	if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
		t.Fatal("Nullable source dropped retained default identity", err)
	}
}
