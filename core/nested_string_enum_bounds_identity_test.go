package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNestedStringEnumBoundsOptimizationAndLocks(t *testing.T) {
	shape := `{"properties":{"trees":{"type":"array","items":{"additionalProperties":{"enum":["one","two","x"],"minLength":2,"maxLength":6,"pattern":"^[a-z]+$"}},"default":[{"b":"two","a":"one"}]}}}`
	operation := func(id, schema string) string {
		return `{"get":{"operationId":"` + id + `","responses":{"200":{"content":{"application/json":{"schema":` + schema + `}}}}}}`
	}
	for _, mode := range []string{"dedup", "merge", "hoist"} {
		doc := `{"openapi":"3.1.0","paths":{"/a":` + operation("alpha", shape) + `,"/b":` + operation("beta", shape) + `,"/bounds":` + operation("bounds", strings.Replace(shape, `"minLength":2`, `"minLength":3`, 1)) + `,"/defaults":` + operation("defaults", strings.Replace(shape, `"b":"two","a":"one"`, `"b":"one","a":"two"`, 1)) + `}}`
		rq := request([]byte(doc), "Dto")
		rq.Options.Validation = true
		rq.Options.DeduplicateModels = mode == "dedup"
		rq.Options.MergeIdenticalModels = mode == "merge"
		rq.Options.HoistSharedShapes = mode == "hoist"
		rq.Options.LockHoistedNames = mode == "hoist"
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(mode, err)
		}
		owner := "api/models/AlphaResponseDto"
		if mode == "hoist" {
			owner = "models/ResponseDto"
		}
		if !strings.Contains(source(t, got, "BetaResponseDto"), "public typealias") || strings.Contains(source(t, got, "BoundsResponseDto"), "public typealias") || strings.Contains(source(t, got, "DefaultsResponseDto"), "public typealias") {
			t.Fatal("nested check/default identity was erased", mode)
		}
		if !strings.Contains(source(t, got, owner), "org.example.models."+strings.ReplaceAll(owner, "/", ".")+".TreesInnerValue") {
			t.Fatal("nested enum helper lost actual canonical owner", mode)
		}
		if mode != "hoist" {
			continue
		}
		rq.PreviousNameLock = got.Artifacts.NameLock
		rq.Document = []byte(strings.ReplaceAll(doc, `"b":"two","a":"one"`, `"a":"one","b":"two"`))
		again, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("nested map canonicalization changed frozen identity", err)
		}
		for _, change := range []struct{ from, to string }{
			{`"minLength":2`, `"minLength":3`},
			{`"pattern":"^[a-z]+$"`, `"pattern":"[a-z]"`},
			{`"b":"two","a":"one"`, `"b":"one","a":"two"`},
		} {
			rq.Document = []byte(strings.ReplaceAll(doc, change.from, change.to))
			changed, err := core.GenerateWithDiagnostics(rq)
			if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
				t.Fatal("nested frozen shape reused changed checks/default", change, err)
			}
		}
	}
}
