package core_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/testutil"
)

func TestYAMLCoreJSONParity(t *testing.T) {
	jsonInput, e := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests/fixtures/yaml/records.json"))
	if e != nil {
		t.Fatal(e)
	}
	yamlInput, e := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests/fixtures/yaml/records.yaml"))
	if e != nil {
		t.Fatal(e)
	}
	for _, version := range []string{"3.0.3", "3.1.0"} {
		j := bytes.ReplaceAll(jsonInput, []byte("3.1.0"), []byte(version))
		y := bytes.ReplaceAll(yamlInput, []byte("3.1.0"), []byte(version))
		if version == "3.0.3" {
			j = bytes.ReplaceAll(j, []byte(`"type": ["string", "null"]`), []byte(`"type": "string", "nullable": true`))
			y = bytes.ReplaceAll(y, []byte("type: [string, 'null']"), []byte("type: string, nullable: true"))
		}
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, checked := range []bool{false, true} {
				r := request(j, "Dto")
				r.Options.ModelStyle, r.Options.Validation = style, checked
				want, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(err)
				}
				for _, format := range []core.InputFormat{core.YAML, core.Auto} {
					r.Document, r.Format, r.SourceName = y, format, "source.YAML"
					got, err := core.GenerateWithDiagnostics(r)
					if err != nil || !reflect.DeepEqual(got, want) || len(got.Warnings) == 0 {
						t.Fatalf("%s/%s/%t: output/warning parity: %v", version, style, checked, err)
					}
					files, err := core.Generate(r)
					if err != nil || !reflect.DeepEqual(files, want.Artifacts) {
						t.Fatal("Generate/GenerateWithDiagnostics disagree", err)
					}
				}
			}
		}
	}
}

func TestYAMLRejectsCustomTagIdentity(t *testing.T) {
	for _, value := range []string{
		"!<!!str> 1e400", "!<!!map> {}", "!<!!seq> []",
		"!<%21%21str> 1e400", "!<%21%21map> {}", "!<%21%21seq> []",
		"!<!!%73tr> 1e400", "!%21str 1e400", "!%21map {}", "!%21seq []",
		"&a !<%21%21str> 1e400", "!<%21> text",
	} {
		t.Run(value, func(t *testing.T) {
			input := "openapi: 3.1.0\ncomponents: {schemas: {A: {type: object, properties: {v: {type: string}}}}}\nexample: " + value + "\n"
			r := request([]byte(input), "Dto")
			r.Format = core.YAML
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || !strings.Contains(err.Error(), "E_YAML_UNSUPPORTED") || len(got.Artifacts.Files) != 0 {
				t.Fatalf("custom URI must fail before emission: files=%d err=%v", len(got.Artifacts.Files), err)
			}
		})
	}
}

func TestYAMLAnalysisDiagnosticsParity(t *testing.T) {
	for _, tc := range []struct{ json, yaml string }{
		{`{"openapi":"3.1.0","components":{"schemas":{"A":{"properties":{"v":{"$ref":"#/components/schemas/Missing"}}}}}}`, "openapi: 3.1.0\ncomponents:\n  schemas:\n    A:\n      properties:\n        v: {$ref: '#/components/schemas/Missing'}"},
		{`{"openapi":"3.1.0","components":{"schemas":{"A":{"type":"unsupported"}}}}`, "openapi: 3.1.0\ncomponents: {schemas: {A: {type: unsupported}}}"},
	} {
		j := request([]byte(tc.json), "Dto")
		_, want := core.GenerateWithDiagnostics(j)
		y := j
		y.Document, y.Format = []byte(tc.yaml), core.YAML
		got, err := core.GenerateWithDiagnostics(y)
		if want == nil || !reflect.DeepEqual(err, want) || len(got.Artifacts.Files) != 0 {
			t.Fatal("analysis pointer/error parity", want, err)
		}
	}
	// Explicit JSON and extensionless auto never reinterpret YAML on failure.
	for _, format := range []core.InputFormat{core.Auto, core.JSON} {
		r := request([]byte("openapi: 3.1.0"), "Dto")
		r.Format, r.SourceName = format, "stdin"
		_, err := core.Generate(r)
		if err == nil || !strings.Contains(err.Error(), "E_JSON_INVALID") {
			t.Fatal("unexpected fallback", err)
		}
	}
}
