package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNestedObjectDefaultsTypedPresence(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/nested-object-defaults.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{"data_class", "plain_class"} {
			r := request(input, "Dto")
			r.Options.Validation = checked
			r.Options.ModelStyle = style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(checked, style, err)
			}
			holder := source(t, got, "HolderDto")
			for _, marker := range []string{
				"public val name: kotlin.String,",
				"public val age: kotlin.Int? = 7,",
				"public val requiredCfg: RequiredCfg,",
				"`openapi-kt model default org example models models HolderDto Cfg`(",
				"`openapi-kt model default org example models models HolderDto Cfg Child`(",
			} {
				if !strings.Contains(holder, marker) {
					t.Fatal("Lost nested object typed API", marker, holder)
				}
			}
			// requiredCfg is required: it keeps a constructor argument and no
			// initializer even though its annotation is valid.
			if strings.Contains(holder, "val requiredCfg: RequiredCfg =") {
				t.Fatal("required object default acquired an initializer", holder)
			}
			// Canonical keys are sorted; source member order only drives
			// diagnostics.
			if !strings.Contains(holder, `{\"age\":9,\"child\":{\"flag\":false}`) {
				t.Fatal("default JSON was not canonically sorted", holder)
			}
			patch := source(t, got, "PatchHolderInputDto")
			if !strings.Contains(patch, "Patch<Cfg>") || !strings.Contains(patch, "Patch.Absent") || strings.Contains(patch, "model default") {
				t.Fatal("inline request object default lost Patch suppression", patch)
			}
			required := source(t, got, "RequiredNullableCfgDto")
			if !strings.Contains(required, "public val value: Value?,") {
				t.Fatal("required nullable object default lost presence", required)
			}
			nativeHolder := source(t, got, "PatchInputDto")
			if !strings.Contains(nativeHolder, "`openapi-kt model default org example models models PatchInputDto Cfg`(") || !strings.Contains(nativeHolder, "public val cfgNullable: CfgNullable? = null,") {
				t.Fatal(nativeHolder)
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("nondeterministic nested object defaults", err, style, checked)
			}
		}
	}
	// This exact formerly rejected typeless object field graduates under the
	// direct object default role; other roles below keep their original guards.
	input = schemas(`{"Record":{"properties":{"value":{"properties":{"id":{"type":"string"}},"default":{}}}}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil || !strings.Contains(source(t, got, "RecordDto"), "`openapi-kt model default org example models models RecordDto Value`(") {
		t.Fatal("Original typeless object default admission", got, err)
	}
}

func TestNestedObjectDefaultsOptionalOverrideAndNull(t *testing.T) {
	input := schemas(`{"Record":{"properties":{"value":{"type":"object","properties":{"a":{"type":"string"}},"nullable":true,"default":{"a":"x"}},"nil":{"type":["object","null"],"properties":{"b":{"type":"integer"}},"default":null}}}}`)
	got := report(t, input, "Dto")
	text := source(t, got, "RecordDto")
	for _, marker := range []string{
		"public val value: Value? = `openapi-kt model default org example models models RecordDto Value`(",
		"public val nil: Nil? = null,",
	} {
		if !strings.Contains(text, marker) {
			t.Fatal(marker, text)
		}
	}
}
