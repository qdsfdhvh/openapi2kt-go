package core_test

import (
	"encoding/json"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func lockFailure(t *testing.T, req core.GenerationRequest, code string) {
	t.Helper()
	r, e := core.GenerateWithDiagnostics(req)
	if e == nil || e.(core.Diagnostics)[0].Code != code || !reflect.DeepEqual(r, core.GenerationReport{}) {
		t.Fatal(code, e, r)
	}
}
func TestNameLockRejectsDriftWithoutPartialOutput(t *testing.T) {
	input := nameLockInput(t)
	req := request(input, "Dto")
	req.Options.HoistSharedShapes = true
	req.Options.LockHoistedNames = true
	req.Options.Validation = true
	base, e := core.GenerateWithDiagnostics(req)
	if e != nil {
		t.Fatal(e)
	}
	req.PreviousNameLock = base.Artifacts.NameLock
	for _, c := range []struct{ kind, code string }{{"changed", "E_NAME_LOCK_CHANGED"}, {"removed", "E_NAME_LOCK_REMOVED"}, {"collision", "E_NAME_LOCK_CONFLICT"}} {
		r := req
		switch c.kind {
		case "changed":
			r.Document = []byte(strings.Replace(string(input), `"minLength": 2`, `"minLength": 3`, 1))
		case "removed":
			var d, paths map[string]json.RawMessage
			_ = json.Unmarshal(input, &d)
			_ = json.Unmarshal(d["paths"], &paths)
			delete(paths, "/alpha")
			d["paths"], _ = json.Marshal(paths)
			r.Document, _ = json.Marshal(d)
		case "collision":
			var d map[string]json.RawMessage
			_ = json.Unmarshal(input, &d)
			d["components"] = json.RawMessage(`{"schemas":{"Response":{"type":"object","properties":{"different":{"type":"string"}},"required":["different"]}}}`)
			r.Document, _ = json.Marshal(d)
		}
		lockFailure(t, r, c.code)
	}
	for _, change := range []string{"package", "suffix", "qualified"} {
		r := req
		switch change {
		case "package":
			r.Options.PackageName = "org.example.changed"
		case "suffix":
			r.Options.ModelNameSuffix = ""
		case "qualified":
			r.Options.QualifyHoistedNames = true
		}
		lockFailure(t, r, "E_NAME_LOCK_PROFILE")
	}
	req.Document = []byte("{")
	req.Options.HoistSharedShapes = false
	lockFailure(t, req, "E_OPTIONS")
	req.Options.HoistSharedShapes = true
	req.Options.LockHoistedNames = false
	lockFailure(t, req, "E_OPTIONS")
	req.Options.LockHoistedNames = true
	req.PreviousNameLock = []byte{}
	lockFailure(t, req, "E_NAME_LOCK_INVALID")
	req.PreviousNameLock = []byte(strings.Repeat(" ", 1<<20+1))
	lockFailure(t, req, "E_NAME_LOCK_LIMIT")
	req.Document = input
	for _, malformed := range []string{"{}", strings.Replace(string(base.Artifacts.NameLock), `"version":1`, `"version":1,"version":1`, 1), strings.Replace(string(base.Artifacts.NameLock), `"version":1`, `"version":1,"unknown":1`, 1)} {
		req.PreviousNameLock = []byte(malformed)
		lockFailure(t, req, "E_NAME_LOCK_INVALID")
	}
	req.PreviousNameLock = []byte(strings.Replace(string(base.Artifacts.NameLock), `"version":1`, `"version":2`, 1))
	lockFailure(t, req, "E_NAME_LOCK_VERSION")
}
func TestNameLockOffWholeEnvelopeAndOriginalBudgets(t *testing.T) {
	req := request(nameLockInput(t), "Dto")
	req.Options.HoistSharedShapes = true
	req.Options.Validation = true
	base, e := core.GenerateWithDiagnostics(req)
	if e != nil || base.Artifacts.NameLock != nil {
		t.Fatal(e)
	}
	req.Options.LockHoistedNames = true
	locked, e := core.Generate(req)
	if e != nil || len(locked.NameLock) == 0 || !reflect.DeepEqual(base.Artifacts.Files, locked.Files) {
		t.Fatal(e)
	}
	req.Options.LockHoistedNames = false
	off, e := core.GenerateWithDiagnostics(req)
	if e != nil || !reflect.DeepEqual(base, off) {
		t.Fatal(e)
	}
	for _, c := range []struct {
		kind  string
		limit int
	}{{"boxed", 240}, {"long", 120}, {"array", 16}, {"enum", 240}} {
		for _, n := range []int{c.limit, c.limit + 1} {
			r := request(hoistLimitDocument(c.kind, n), "Dto")
			r.Options.HoistSharedShapes = true
			r.Options.Validation = true
			a, e := core.GenerateWithDiagnostics(r)
			r.Options.LockHoistedNames = true
			b, f := core.GenerateWithDiagnostics(r)
			if n == c.limit {
				if e != nil || f != nil || !reflect.DeepEqual(a.Artifacts.Files, b.Artifacts.Files) || !reflect.DeepEqual(a.Warnings, b.Warnings) {
					t.Fatal(c, n, e, f)
				}
			} else if e == nil || f == nil || !reflect.DeepEqual(e, f) || !reflect.DeepEqual(a, b) {
				t.Fatal("lock bypasses original limit", c, n, e, f)
			}
		}
	}
}
