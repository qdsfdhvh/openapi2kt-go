package core

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestModelStyleDefaultAndPlainOwnedDeclarationEnvelope(t *testing.T) {
	for _, path := range []string{"../tests/fixtures/model-semantics/plain-style-baseline.json", "../tests/fixtures/model-semantics/plain-style-arrays.json", "../tests/fixtures/http/shared-runtime.json", "../tests/fixtures/http/root-object-aliases.json", "../tests/fixtures/http/root-union-aliases.json", "../tests/fixtures/http/root-hierarchy-aliases.json"} {
		t.Run(path, func(t *testing.T) {
			data, e := os.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			for _, validation := range []bool{false, true} {
				req := GenerationRequest{Document: data, Format: JSON, Options: GenerationOptions{PackageName: "org.example.styles", ModelNameSuffix: "Dto", Validation: validation}}
				base, e := GenerateWithDiagnostics(req)
				req.Options.ModelStyle = DataClass
				explicit, ee := GenerateWithDiagnostics(req)
				if e != nil || ee != nil {
					if (e == nil) != (ee == nil) || e.Error() != ee.Error() {
						t.Fatal(e, ee)
					}
					continue
				}
				if !reflect.DeepEqual(base, explicit) {
					t.Fatal("explicit data_class changed report")
				}
				req.Options.ModelStyle = PlainClass
				plain, e := GenerateWithDiagnostics(req)
				if e != nil {
					t.Fatal(e)
				}
				again, e := GenerateWithDiagnostics(req)
				if e != nil || !reflect.DeepEqual(again, plain) || !reflect.DeepEqual(base.Warnings, plain.Warnings) || len(base.Artifacts.Files) != len(plain.Artifacts.Files) {
					t.Fatal("style affected warnings/determinism/paths", e)
				}
				changed := false
				for i, f := range plain.Artifacts.Files {
					old := base.Artifacts.Files[i]
					if old.Path != f.Path {
						t.Fatal(old.Path, f.Path)
					}
					if strings.HasSuffix(f.Path, "/Patch.kt") || strings.HasSuffix(f.Path, "/LongAsStringSerializer.kt") {
						if old != f {
							t.Fatal("shared utility changed with style", f.Path)
						}
						continue
					}
					if strings.Contains(f.Content, "public data class ") {
						t.Fatal("unconverted owned class", f.Path)
					}
					if f.Content != old.Content {
						changed = true
						if !strings.Contains(f.Content, "override fun equals(") || !strings.Contains(f.Content, "override fun toString(") {
							t.Fatal("missing plain members", f.Path)
						}
					}
				}
				if !changed {
					t.Fatal("fixture didn't exercise style")
				}
			}
		})
	}
}
func TestModelStyleUnknownBeforeDocumentLoad(t *testing.T) {
	r, e := GenerateWithDiagnostics(GenerationRequest{Document: []byte("bad"), Format: JSON, Options: GenerationOptions{PackageName: "org.example.styles", ModelStyle: "unknown"}})
	if e == nil || e.(Diagnostics)[0].Code != "E_OPTIONS" || !reflect.DeepEqual(r, GenerationReport{}) {
		t.Fatal(r, e)
	}
}
