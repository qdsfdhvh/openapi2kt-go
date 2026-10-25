package core_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzNameLockProjection(f *testing.F) {
	f.Add(uint8(2), uint8(1), uint8(0), false, true)
	f.Add(uint8(6), uint8(31), uint8(1), true, false)
	f.Add(uint8(4), uint8(7), uint8(2), false, false)
	f.Fuzz(func(t *testing.T, width, selected, mutation uint8, plain, checked bool) {
		n := 2 + int(width%6)
		paths := map[string]any{}
		for i := 0; i < n; i++ {
			paths[fmt.Sprintf("/op%d", i)] = map[string]any{"get": map[string]any{
				"operationId": fmt.Sprintf("operation%d", i),
				"responses": map[string]any{"200": map[string]any{"content": map[string]any{
					"application/json": map[string]any{"schema": map[string]any{
						"properties": map[string]any{"value": map[string]any{"type": "string"}},
						"required":   []string{"value"},
					}},
				}}},
			}}
		}
		doc := map[string]any{"openapi": "3.1.0", "paths": paths, "components": map[string]any{
			"schemas": map[string]any{"Stable": map[string]any{"properties": map[string]any{"label": map[string]any{"type": "string"}}}},
		}}
		input, _ := json.Marshal(doc)
		rq := request(input, "Dto")
		rq.Options.HoistSharedShapes, rq.Options.LockHoistedNames = true, true
		rq.Options.Validation = checked
		if plain {
			rq.Options.ModelStyle = core.PlainClass
		}
		before, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(err)
		}
		retained := 0
		for i := 0; i < n; i++ {
			path := fmt.Sprintf("/op%d", i)
			if selected&(1<<i) == 0 {
				delete(paths, path)
				continue
			}
			retained++
			if mutation%3 == 1 {
				op := paths[path].(map[string]any)["get"].(map[string]any)
				op["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)["properties"].(map[string]any)["value"].(map[string]any)["type"] = "integer"
			}
			if mutation%3 == 2 {
				paths[path+"/new"] = paths[path]
				delete(paths, path)
			}
		}
		rq.Document, _ = json.Marshal(doc)
		rq.PreviousNameLock = before.Artifacts.NameLock
		rq.ProjectPreviousNameLock = true
		inputCopy, lockCopy := bytes.Clone(rq.Document), bytes.Clone(rq.PreviousNameLock)
		after, err := core.GenerateWithDiagnostics(rq)
		again, retry := core.GenerateWithDiagnostics(rq)
		if !bytes.Equal(inputCopy, rq.Document) || !bytes.Equal(lockCopy, rq.PreviousNameLock) || !reflect.DeepEqual(after, again) || !reflect.DeepEqual(err, retry) {
			t.Fatal("projection mutated input or was nondeterministic")
		}
		if mutation%3 != 0 && retained > 0 {
			if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || len(after.Artifacts.Files) != 0 || after.Artifacts.NameLock != nil {
				t.Fatal("projection accepted changed/new occurrence", err)
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		if retained > 0 && source(t, before, "ResponseDto") != source(t, after, "ResponseDto") {
			t.Fatal("projection changed canonical declaration")
		}
		rq.ProjectPreviousNameLock = false
		rq.PreviousNameLock = after.Artifacts.NameLock
		replay, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(after, replay) {
			t.Fatal("projected state cannot ordinarily replay", err)
		}
	})
}
