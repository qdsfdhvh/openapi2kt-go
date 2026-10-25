package core_test

import (
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func FuzzAllOfDirectionalModels(f *testing.F) {
	f.Add(uint8(0), false, false)
	f.Add(uint8(6), true, true)
	f.Fuzz(func(t *testing.T, count uint8, checked, optimized bool) {
		input := directionalDoc(`"id":{"type":"string","readOnly":true},"password":{"type":"string","writeOnly":true},"name":{"type":"string"}`)
		var root map[string]any
		if e := json.Unmarshal(input, &root); e != nil {
			t.Fatal(e)
		}
		catalog := root["components"].(map[string]any)["schemas"].(map[string]any)
		catalog["Base"] = catalog["Account"]
		catalog["BaseAlias"] = map[string]any{"$ref": "#/components/schemas/Base"}
		branches := []any{map[string]any{"$ref": "#/components/schemas/BaseAlias"}}
		for i := 0; i < int(count%7)+1; i++ {
			field := map[string]any{"type": "string"}
			if i%2 == 0 {
				field["readOnly"] = true
			} else {
				field["writeOnly"] = true
			}
			branches = append(branches, map[string]any{"properties": map[string]any{fmt.Sprint("v", i): field}})
		}
		catalog["Account"] = map[string]any{"allOf": branches}
		input, e := json.Marshal(root)
		if e != nil {
			t.Fatal(e)
		}
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		rq.Options.DeduplicateModels = optimized
		rq.Options.HoistSharedShapes = optimized
		rq.Options.MergeIdenticalModels = optimized
		rq.Options.LockHoistedNames = optimized
		first, e := core.GenerateWithDiagnostics(rq)
		if e != nil {
			t.Fatal(e)
		}
		second, e := core.GenerateWithDiagnostics(rq)
		if e != nil || !reflect.DeepEqual(first, second) {
			t.Fatal("allOfdirectional nondeterminism", e)
		}
		req, res := source(t, first, "SaveAccountRequestDto"), source(t, first, "SaveAccountPublicResponseDto")
		for i := 0; i < int(count%7)+1; i++ {
			wire := fmt.Sprint("v", i)
			if i%2 == 0 {
				if !strings.Contains(req, wire+": org.example.models.models.Patch<kotlin.String>") {
					t.Fatal("readOnly source notPatch", wire)
				}
			} else if strings.Contains(res, "SerialName(\""+wire+"\")") {
				t.Fatal("writeOnly leaked", wire)
			}
		}
	})
}
