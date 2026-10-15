package core_test

import (
	"encoding/json"
	"openapi-kt/core"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test-only IO. The product core never reads this environment or writes output.
func TestExportNameLockKotlinProbe(t *testing.T) {
	out := os.Getenv("OPENAPI_KT_NAME_LOCK_PROBE_DIR")
	if out == "" {
		t.Skip("explicit probe directory not set")
	}
	input := nameLockInput(t)
	var d, paths map[string]json.RawMessage
	_ = json.Unmarshal(input, &d)
	_ = json.Unmarshal(d["paths"], &paths)
	for _, owner := range []string{"abacus", "aardvark"} {
		paths["/"+owner] = json.RawMessage(`{"get":{"operationId":"` + owner + `","responses":{"200":{"content":{"application/json":{"schema":{"type":"object","properties":{"other":{"type":"string"}},"required":["other"]}}}}}}}`)
	}
	var alpha, post map[string]json.RawMessage
	_ = json.Unmarshal(paths["/alpha"], &alpha)
	_ = json.Unmarshal(alpha["post"], &post)
	post["operationId"] = json.RawMessage(`"aapear"`)
	alpha["post"], _ = json.Marshal(post)
	paths["/aapear"], _ = json.Marshal(alpha)
	d["paths"], _ = json.Marshal(paths)
	expanded, _ := json.Marshal(d)
	for _, c := range []struct {
		dir, pkg  string
		qualified bool
		style     core.ModelStyle
	}{{"short-data", "org.example.hoist", false, core.DataClass}, {"short-plain", "org.example.hoistplain", false, core.PlainClass}, {"qualified-data", "org.example.qualifiedhoist", true, core.DataClass}, {"qualified-plain", "org.example.qualifiedhoistplain", true, core.PlainClass}} {
		req := request(input, "Dto")
		req.Options.PackageName = c.pkg
		req.Options.HoistSharedShapes = true
		req.Options.LockHoistedNames = true
		req.Options.QualifyHoistedNames = c.qualified
		req.Options.ModelStyle = c.style
		req.Options.Validation = true
		req.Options.DeduplicateModels = true
		before, e := core.GenerateWithDiagnostics(req)
		if e != nil {
			t.Fatal(e)
		}
		req.Document = expanded
		req.PreviousNameLock = before.Artifacts.NameLock
		after, e := core.GenerateWithDiagnostics(req)
		if e != nil {
			t.Fatal(e)
		}
		write := func(path string, bytes []byte) {
			t.Helper()
			if e = os.MkdirAll(filepath.Dir(path), 0755); e != nil {
				t.Fatal(e)
			}
			if e = os.WriteFile(path, bytes, 0644); e != nil {
				t.Fatal(e)
			}
		}
		write(filepath.Join(out, c.dir+"-previous.json"), req.PreviousNameLock)
		write(filepath.Join(out, c.dir+"-current.json"), after.Artifacts.NameLock)
		write(filepath.Join(out, c.dir+"-input.json"), expanded)
		for _, f := range after.Artifacts.Files {
			if strings.Contains(f.Path, "..") || filepath.IsAbs(f.Path) {
				t.Fatal("unsafe core path", f.Path)
			}
			write(filepath.Join(out, "project/build/generated-limits", c.dir, f.Path), []byte(f.Content))
		}
	}
}
