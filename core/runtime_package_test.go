package core_test

import (
	"encoding/json"
	"openapi-kt/core"
	"os"
	"reflect"
	"strings"
	"testing"
)

func runtimeRequest(t *testing.T) core.GenerationRequest {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/shared-runtime.json")
	if e != nil {
		t.Fatal(e)
	}
	return core.GenerationRequest{Document: b, Format: core.JSON, Options: core.GenerationOptions{PackageName: "org.example.runtimecore", ModelNameSuffix: "Dto"}}
}
func TestRuntimePackageCoreParityAndEligibility(t *testing.T) {
	for _, validation := range []bool{false, true} {
		r := runtimeRequest(t)
		r.Options.Validation = validation
		if !validation {
			old, err := core.GenerateWithDiagnostics(r)
			r.Options.RuntimePackage = "org.example.shared.runtime"
			same, newErr := core.GenerateWithDiagnostics(r)
			if err == nil || newErr == nil || !reflect.DeepEqual(err, newErr) || !reflect.DeepEqual(old, same) {
				t.Fatal("runtime option dropped opt-out source checks", old, same, err, newErr)
			}
			var doc map[string]any
			_ = json.Unmarshal(r.Document, &doc)
			delete(doc["paths"].(map[string]any)["/submit"].(map[string]any)["post"].(map[string]any)["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)["properties"].(map[string]any)["id"].(map[string]any), "minLength")
			r.Document, _ = json.Marshal(doc)
			r.Options.RuntimePackage = ""
		}
		old, e := core.GenerateWithDiagnostics(r)
		if e != nil {
			t.Fatal(e)
		}
		r.Options.RuntimePackage = r.Options.PackageName + ".models"
		same, e := core.GenerateWithDiagnostics(r)
		if e != nil || !reflect.DeepEqual(old, same) {
			t.Fatal("explicit default changed artifacts", e)
		}
		r.Options.RuntimePackage = "org.example.shared.runtime"
		actual, e := core.GenerateWithDiagnostics(r)
		if e != nil || len(old.Artifacts.Files) != len(actual.Artifacts.Files) || !reflect.DeepEqual(old.Warnings, actual.Warnings) {
			t.Fatal(e)
		}
		for i, f := range old.Artifacts.Files {
			expected := strings.ReplaceAll(f.Content, "org.example.runtimecore.models.Patch", "org.example.shared.runtime.Patch")
			expected = strings.ReplaceAll(expected, "org.example.runtimecore.models.LongAsStringSerializer", "org.example.shared.runtime.LongAsStringSerializer")
			path := f.Path
			if strings.HasSuffix(path, "/Patch.kt") || strings.HasSuffix(path, "/LongAsStringSerializer.kt") {
				path = strings.Replace(path, "org/example/runtimecore/models/", "org/example/shared/runtime/", 1)
				expected = strings.Replace(expected, "package org.example.runtimecore.models", "package org.example.shared.runtime", 1)
			}
			found := false
			for _, g := range actual.Artifacts.Files {
				if g.Path == path {
					found = true
					if g.Content != expected {
						t.Fatal("unexpected source/wire change", i, path)
					}
				}
			}
			if !found {
				t.Fatal("lost artifact", path)
			}
		}
		again, e := core.GenerateWithDiagnostics(r)
		if e != nil || !reflect.DeepEqual(actual, again) {
			t.Fatal("nondeterminism", e)
		}
		r.Options.ExcludeAPIs = []string{"DefaultApi"}
		excluded, e := core.GenerateWithDiagnostics(r)
		if e != nil {
			t.Fatal(e)
		}
		for _, f := range excluded.Artifacts.Files {
			if strings.HasSuffix(f.Path, "/Patch.kt") {
				t.Fatal("excluded runtime")
			}
		}
	}
}
func TestRuntimePackageInvalidAndScopedCollisions(t *testing.T) {
	for _, pkg := range []string{"kotlin.user", "org..invalid", "org.class", "../unsafe", "org.CON"} {
		r := runtimeRequest(t)
		r.Document = []byte("bad JSON")
		r.Options.RuntimePackage = pkg
		out, e := core.GenerateWithDiagnostics(r)
		if e == nil || e.(core.Diagnostics)[0].Code != "E_PACKAGE_INVALID" || len(out.Artifacts.Files) != 0 {
			t.Fatal(pkg, out, e)
		}
	}
	for _, name := range []string{"Patch", "PatchSerializer", "LongAsStringSerializer"} {
		r := runtimeRequest(t)
		r.Options.ModelNameSuffix = ""
		r.Options.Validation = true
		var doc map[string]any
		_ = json.Unmarshal(r.Document, &doc)
		doc["components"].(map[string]any)["schemas"].(map[string]any)[name] = map[string]any{"properties": map[string]any{"id": map[string]any{"type": "string"}}}
		r.Document, _ = json.Marshal(doc)
		_, e := core.Generate(r)
		if e == nil || e.(core.Diagnostics)[0].Code != "E_NAME_COLLISION" || *e.(core.Diagnostics)[0].Pointer != "/components/schemas/"+name {
			t.Fatal(name, e)
		}
		r.Options.RuntimePackage = "org.example.shared.runtime"
		if _, e := core.Generate(r); e != nil {
			t.Fatal("other package should not reserve component", name, e)
		}
	}
	for _, name := range []string{"Patch", "PatchSerializer", "LongAsStringSerializer"} {
		r := runtimeRequest(t)
		r.Options.RuntimePackage = r.Options.PackageName
		r.Options.Validation = true
		r.Options.APIClientName = name
		out, e := core.GenerateWithDiagnostics(r)
		if e == nil || e.(core.Diagnostics)[0].Code != "E_NAME_COLLISION" || len(out.Artifacts.Files) != 0 {
			t.Fatal("client runtime classifier collision", name, out, e)
		}
	}
}
