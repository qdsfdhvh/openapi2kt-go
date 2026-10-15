package core_test

import (
	"bytes"
	"encoding/json"
	"openapi-kt/core"
	"os"
	"reflect"
	"strings"
	"testing"
)

func nameLockInput(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/deduplicate-models.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestNameLockDesiredSuccess(t *testing.T) {
	input := nameLockInput(t)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, qualified := range []bool{false, true} {
			req := request(input, "Dto")
			req.Options.Validation = true
			req.Options.HoistSharedShapes = true
			req.Options.QualifyHoistedNames = qualified
			req.Options.ModelStyle = style
			control, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(e)
			}
			req.Options.LockHoistedNames = true
			initial, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal("desired name-lock success", e)
			}
			if len(initial.Artifacts.NameLock) == 0 || !reflect.DeepEqual(control.Artifacts.Files, initial.Artifacts.Files) || !reflect.DeepEqual(control.Warnings, initial.Warnings) {
				t.Fatal("lock didn't retain whole-envelope output")
			}
			var state struct {
				Version int               `json:"version"`
				Entries []json.RawMessage `json:"entries"`
			}
			if e = json.Unmarshal(initial.Artifacts.NameLock, &state); e != nil || state.Version != 1 || len(state.Entries) != 3 {
				t.Fatal("lock state", e, string(initial.Artifacts.NameLock))
			}
			req.PreviousNameLock = initial.Artifacts.NameLock
			replay, e := core.GenerateWithDiagnostics(req)
			if e != nil || !reflect.DeepEqual(initial, replay) {
				t.Fatal("lock replay", e)
			}
			// Keep borrowed schema/property ordering; map[string]any round-tripping
			// would change constructor property order, correctly rejected by locks.
			var d, paths map[string]json.RawMessage
			_ = json.Unmarshal(input, &d)
			_ = json.Unmarshal(d["paths"], &paths)
			for _, owner := range []string{"aardvark", "abacus"} {
				paths["/"+owner] = json.RawMessage(`{"get":{"operationId":"` + owner + `","responses":{"200":{"content":{"application/json":{"schema":{"type":"object","properties":{"other":{"type":"string"}},"required":["other"]}}}}}}}`)
			}
			d["paths"], _ = json.Marshal(paths)
			req.Document, _ = json.Marshal(d)
			added, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(e)
			}
			if source(t, added, "BetaResponseDto") != source(t, initial, "BetaResponseDto") {
				t.Fatal("unrelated earlier group stole old name")
			}
			if !bytes.Equal(req.PreviousNameLock, initial.Artifacts.NameLock) {
				t.Fatal("mutated input state")
			}
			// Same shape at an earlier new owner must preserve old qualified shared name.
			_ = json.Unmarshal(input, &d)
			paths = nil
			_ = json.Unmarshal(d["paths"], &paths)
			var alpha, post map[string]json.RawMessage
			_ = json.Unmarshal(paths["/alpha"], &alpha)
			_ = json.Unmarshal(alpha["post"], &post)
			post["operationId"] = json.RawMessage(`"aardvark"`)
			alpha["post"], _ = json.Marshal(post)
			paths["/aardvark"], _ = json.Marshal(alpha)
			d["paths"], _ = json.Marshal(paths)
			req.Document, _ = json.Marshal(d)
			peer, e := core.GenerateWithDiagnostics(req)
			if e != nil {
				t.Fatal(e)
			}
			if source(t, peer, "BetaResponseDto") != source(t, initial, "BetaResponseDto") || !strings.Contains(source(t, peer, "AardvarkResponseDto"), "public typealias") {
				t.Fatal("new canonical peer renamed stored class")
			}
		}
	}
}
