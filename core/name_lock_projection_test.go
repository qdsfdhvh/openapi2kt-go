package core_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func projectedLockReports(t *testing.T, input []byte, pkg string, style core.ModelStyle, checked bool) (core.GenerationReport, core.GenerationReport) {
	t.Helper()
	rq := request(input, "Dto")
	rq.Options.PackageName = pkg
	rq.Options.ModelStyle = style
	rq.Options.Validation = checked
	rq.Options.HoistSharedShapes = true
	rq.Options.LockHoistedNames = true
	rq.Options.APIClientName = "SplitClient"
	before, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]json.RawMessage
	var paths map[string]json.RawMessage
	if err := json.Unmarshal(input, &doc); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(doc["paths"], &paths); err != nil {
		t.Fatal(err)
	}
	delete(paths, "/b")
	doc["paths"], _ = json.Marshal(paths)
	rq.Document, _ = json.Marshal(doc)
	rq.PreviousNameLock = before.Artifacts.NameLock
	strict, failure := core.GenerateWithDiagnostics(rq)
	if failure == nil || len(strict.Artifacts.Files) != 0 || failure.(core.Diagnostics)[0].Code != "E_NAME_LOCK_REMOVED" {
		t.Fatal("ordinary generation bypassed missing occurrence", failure)
	}
	rq.ProjectPreviousNameLock = true
	original := bytes.Clone(rq.Document)
	lock := bytes.Clone(rq.PreviousNameLock)
	after, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rq.Document, original) || !bytes.Equal(rq.PreviousNameLock, lock) {
		t.Fatal("projection mutated caller bytes")
	}
	if source(t, before, "ResponseDto") != source(t, after, "ResponseDto") {
		t.Fatal("singleton changed canonical Kotlin")
	}
	rq.ProjectPreviousNameLock = false
	rq.PreviousNameLock = after.Artifacts.NameLock
	replay, err := core.GenerateWithDiagnostics(rq)
	if err != nil || !reflect.DeepEqual(after, replay) {
		t.Fatal("projected singleton cannot ordinarily regenerate", err)
	}
	return before, after
}

func TestNameLockProjectionCore(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/split-adoption-source.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		projectedLockReports(t, input, "org.example.projection", style, true)
	}
	rq := request(input, "Dto")
	rq.Options.HoistSharedShapes = true
	rq.Options.LockHoistedNames = true
	rq.ProjectPreviousNameLock = true
	if report, err := core.GenerateWithDiagnostics(rq); err == nil || len(report.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_OPTIONS" {
		t.Fatal("projection accepted no previous state", err)
	}
}

func TestExportNameLockProjection(t *testing.T) {
	output := os.Getenv("OPENAPI_KT_PROJECTION_OUTPUT")
	if output == "" {
		t.Skip("explicit Kotlin projection exporter only")
	}
	input, err := os.ReadFile(os.Getenv("OPENAPI_KT_PROJECTION_INPUT"))
	if err != nil {
		t.Fatal(err)
	}
	_, report := projectedLockReports(t, input, "org.example.singleprojection", core.ModelStyle(os.Getenv("OPENAPI_KT_PROJECTION_STYLE")), os.Getenv("OPENAPI_KT_PROJECTION_CHECKED") == "on")
	for _, file := range report.Artifacts.Files {
		path := filepath.Join(output, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(file.Content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}
