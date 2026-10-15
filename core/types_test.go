package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func TestDiagnosticJSONAndRootPointer(t *testing.T) {
	root := ""
	for _, tc := range []struct {
		d    core.Diagnostic
		want string
	}{
		{core.Diagnostic{Code: "E_JSON_INVALID", Message: "Invalid JSON."}, `{"code":"E_JSON_INVALID","message":"Invalid JSON."}`},
		{core.Diagnostic{Code: "E_DOCUMENT_INVALID", Message: "Object required.", Pointer: &root}, `{"code":"E_DOCUMENT_INVALID","message":"Object required.","pointer":""}`},
	} {
		encoded, err := json.Marshal(tc.d)
		if err != nil || string(encoded) != tc.want {
			t.Fatal(string(encoded), err)
		}
		var decoded core.Diagnostic
		if err := json.Unmarshal(encoded, &decoded); err != nil || !reflect.DeepEqual(decoded, tc.d) {
			t.Fatal("diagnostic lost root-vs-absent distinction", err)
		}
	}
}

func TestPublicCarriersAndDiagnosticsError(t *testing.T) {
	request := core.GenerationRequest{
		Document: []byte(`{}`), Format: core.JSON, SourceName: "input.json",
		Options: core.GenerationOptions{PackageName: "org.example.catalog", ModelNameSuffix: "Dto"},
	}
	files := core.GenerationArtifacts{Files: []core.GeneratedFile{{Path: "org/example/catalog/models/ItemDto.kt", Content: "source"}}}
	report := core.GenerationReport{Artifacts: files}
	if request.Format != core.JSON || !reflect.DeepEqual(files, report.Artifacts) || core.MaxDocumentBytes != 8*1024*1024 {
		t.Fatal("public carrier contract")
	}
	var err error = core.Diagnostics{{Code: "E_SCHEMA_INVALID", Message: "Invalid schema."}}
	if err.Error() != "E_SCHEMA_INVALID: Invalid schema." {
		t.Fatal(err)
	}
}
