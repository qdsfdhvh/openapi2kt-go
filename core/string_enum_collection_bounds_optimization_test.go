package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestStringEnumCollectionBoundsActualOptimizationIdentity(t *testing.T) {
	shape := `{"properties":{"state":{"type":"string","format":"date","enum":["one","two","x"],"minLength":2,"maxLength":4,"default":"one"},"codes":{"type":"array","items":{"enum":["one","two","x"],"minLength":2,"maxLength":4},"default":["one"]},"labels":{"additionalProperties":{"enum":["one","two"],"pattern":"^[a-z]+$"},"default":{"b":"two","a":"one"}}}}`
	op := func(id, shape string) string {
		return `{"get":{"operationId":"` + id + `","responses":{"200":{"content":{"application/json":{"schema":` + shape + `}}}}}}`
	}
	for _, mode := range []string{"dedup", "merge", "hoist"} {
		doc := `{"openapi":"3.1.0","paths":{"/a":` + op("alpha", shape) + `,"/b":` + op("beta", shape) + `,"/bounds":` + op("bounds", strings.Replace(shape, `"minLength":2`, `"minLength":3`, 1)) + `,"/defaults":` + op("defaults", strings.Replace(shape, `"default":["one"]`, `"default":["two"]`, 1)) + `}}`
		r := request([]byte(doc), "Dto")
		r.Options.Validation = true
		r.Options.DeduplicateModels = mode == "dedup"
		r.Options.MergeIdenticalModels = mode == "merge"
		r.Options.HoistSharedShapes = mode == "hoist"
		r.Options.LockHoistedNames = mode == "hoist"
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(mode, err)
		}
		owner := "api/models/AlphaResponseDto"
		if mode == "hoist" {
			owner = "models/ResponseDto"
		}
		canonical := source(t, got, owner)
		if !strings.Contains(source(t, got, "BetaResponseDto"), "public typealias") || strings.Contains(source(t, got, "BoundsResponseDto"), "public typealias") || strings.Contains(source(t, got, "DefaultsResponseDto"), "public typealias") {
			t.Fatal("Wrong identity", mode)
		}
		for _, leaf := range []string{"State", "CodesInner", "LabelsValue"} {
			if !strings.Contains(canonical, "org.example.models."+strings.ReplaceAll(owner, "/", ".")+"."+leaf) {
				t.Fatal("Wrong final lexical owner", mode, canonical)
			}
		}
		again, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal(mode, err)
		}
		if mode == "hoist" {
			r.PreviousNameLock = got.Artifacts.NameLock
			r.Document = []byte(strings.ReplaceAll(doc, `{"b":"two","a":"one"}`, `{"a":"one","b":"two"}`))
			again, err = core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("Canonical keys lost", err)
			}
			for _, change := range []struct{ old, new string }{{`"minLength":2`, `"minLength":3`}, {`"pattern":"^[a-z]+$"`, `"pattern":"[a-z]"`}, {`"default":["one"]`, `"default":["two"]`}, {`"default":"one"`, `"default":"two"`}} {
				r.Document = []byte(strings.ReplaceAll(doc, change.old, change.new))
				changed, err := core.GenerateWithDiagnostics(r)
				if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
					t.Fatal("Frozen check/default reused", change, err)
				}
			}
		}
	}
}
