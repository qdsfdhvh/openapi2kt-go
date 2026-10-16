package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestReferencedObjectDefaultsTypedPresence(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/referenced-object-defaults.json")
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
				"public val cfg: org.example.models.models.InnerDto? = `openapi-kt model default org example models models InnerDto`(",
				"public val aliasCfg: org.example.models.models.InnerDto? = `openapi-kt model default org example models models InnerDto`(",
				"public val requiredCfg: org.example.models.models.InnerDto,",
				"public val nilCfg: org.example.models.models.InnerDto? = null,",
			} {
				if !strings.Contains(holder, marker) {
					t.Fatal("Lost referenced object default API", marker, holder)
				}
			}
			if strings.Contains(holder, "requiredCfg: org.example.models.models.InnerDto =") {
				t.Fatal("required referenced default acquired an initializer", holder)
			}
			// One helper serves both fields whose actual type is the same DTO.
			if strings.Count(holder, "private fun `openapi-kt model default org example models models InnerDto`(") != 1 {
				t.Fatal("helper was not deduplicated", holder)
			}
			patch := source(t, got, "PatchHolderInputDto")
			if !strings.Contains(patch, "Patch<org.example.models.models.InnerDto>") || !strings.Contains(patch, "Patch.Absent") || strings.Contains(patch, "model default") {
				t.Fatal("inline request referenced default lost Patch suppression", patch)
			}
			required := source(t, got, "RequiredCfgDto")
			if !strings.Contains(required, "public val value: org.example.models.models.InnerDto?,") {
				t.Fatal("required nullable referenced default lost presence", required)
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("nondeterministic referenced object defaults", err, style, checked)
			}
		}
	}
	// The exact formerly rejected referenced DTO default graduates.
	input = schemas(`{"Inner":{"type":"object","properties":{"a":{"type":"string"}}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Inner","default":{"a":"x"}}}}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil || !strings.Contains(source(t, got, "HolderDto"), "`openapi-kt model default org example models models InnerDto`(") {
		t.Fatal("Original referenced object default admission", got, err)
	}
}

func TestReferencedObjectDefaultsNullablePrecedence(t *testing.T) {
	// Transparent aliases collapse to the canonical DTO type and share its helper.
	input := schemas(`{"Inner":{"type":"object","properties":{"a":{"type":"string"}}},"Alias":{"$ref":"#/components/schemas/Inner"},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Alias","default":{"a":"x"}}}}}`)
	got := report(t, input, "Dto")
	text := source(t, got, "HolderDto")
	if !strings.Contains(text, "org.example.models.models.InnerDto? = `openapi-kt model default org example models models InnerDto`(") {
		t.Fatal(text)
	}
	// Explicit nullable target with a null default stays literal null.
	nullable := schemas(`{"Inner":{"type":["object","null"],"properties":{"a":{"type":"string"}}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Inner","default":null}}}}`)
	got = report(t, nullable, "Dto")
	if !strings.Contains(source(t, got, "HolderDto"), "val value: org.example.models.models.InnerDto? = null,") {
		t.Fatal(source(t, got, "HolderDto"))
	}
}
