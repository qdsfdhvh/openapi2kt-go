package core_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func FuzzMergePrimitiveUnionRoots(f *testing.F) {
	f.Add(uint8(0), false, false, false)
	f.Add(uint8(3), true, true, true)
	f.Fuzz(func(t *testing.T, extra uint8, nullable, different, excluded bool) {
		kinds := []string{"string", "integer"}
		if extra&1 != 0 {
			kinds = append(kinds, "boolean")
		}
		if extra&2 != 0 {
			kinds = append(kinds, "number")
		}
		catalog := map[string]any{}
		for _, name := range []string{"A", "Z"} {
			members := []any{}
			for _, kind := range kinds {
				members = append(members, map[string]any{"type": kind})
			}
			if name == "Z" && different {
				members[0], members[1] = members[1], members[0]
			}
			catalog[name] = map[string]any{"oneOf": members, "nullable": nullable}
		}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": catalog}})
		original := bytes.Clone(input)
		rq := request(input, "Dto")
		rq.Options.PackageName = "org.example.unionmerge"
		rq.Options.HoistSharedShapes, rq.Options.LockHoistedNames = true, true
		before, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(err)
		}
		rq.Options.MergeIdenticalModels = true
		if excluded {
			rq.Options.ExcludeMergeModels = []string{"org.example.unionmerge.models.ZDto"}
		}
		rq.PreviousNameLock = before.Artifacts.NameLock
		first, err := core.GenerateWithDiagnostics(rq)
		second, retry := core.GenerateWithDiagnostics(rq)
		if err != nil || retry != nil || !reflect.DeepEqual(first, second) || !reflect.DeepEqual(before.Warnings, first.Warnings) || !bytes.Equal(input, original) || !bytes.Equal(before.Artifacts.NameLock, first.Artifacts.NameLock) {
			t.Fatal("union merge changed source, warnings, lock or determinism", err, retry)
		}
		merged := strings.Contains(source(t, first, "ZDto"), "public typealias ZDto =")
		if merged != (!different && !excluded) {
			t.Fatal("ordered kind/exclusion identity lost")
		}
	})
}
