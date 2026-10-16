package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestReferencedObjectCollectionDefaultsTypedPresence(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/referenced-object-collection-defaults.json")
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
				"public val inlineRows: kotlin.collections.List<org.example.models.models.ItemDto>? = `openapi-kt model default List org example models models ItemDto`(",
				"public val inlineMap: kotlin.collections.Map<kotlin.String, org.example.models.models.ItemDto>? = `openapi-kt model default Map org example models models ItemDto`(",
				"public val namedRows: org.example.models.models.RowsDto? = `openapi-kt default tree Alias org example models models RowsDto`(",
				"public val namedMap: org.example.models.models.MapRowsDto? = `openapi-kt default tree Alias org example models models MapRowsDto`(",
				"public val requiredRows: kotlin.collections.List<org.example.models.models.ItemDto>,",
				"public val nilRows: kotlin.collections.List<org.example.models.models.ItemDto>? = null,",
			} {
				if !strings.Contains(holder, marker) {
					t.Fatal("Lost referenced object collection API", marker, holder)
				}
			}
			if strings.Contains(holder, "requiredRows: kotlin.collections.List<org.example.models.models.ItemDto> =") {
				t.Fatal("required referenced collection default acquired an initializer", holder)
			}
			patch := source(t, got, "PatchHolderInputDto")
			if !strings.Contains(patch, "Patch<kotlin.collections.List<org.example.models.models.ItemDto>>") || strings.Contains(patch, "model default") || strings.Contains(patch, "default tree") {
				t.Fatal("inline request referenced collection default lost Patch suppression", patch)
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("nondeterministic referenced object collection defaults", err, style, checked)
			}
		}
	}
	// A referenced DTO element inside an inline collection previously failed at
	// the `$ref` element; it now graduates to this role.
	for _, input := range []string{
		`{"Item":{"type":"object","properties":{"a":{"type":"string"}}},"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Item"},"default":[{"a":"x"}]}}}}`,
		`{"Item":{"type":"object","properties":{"a":{"type":"string"}}},"Holder":{"properties":{"value":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Item"},"default":{"k":{"a":"x"}}}}}}`,
	} {
		got, err := core.GenerateWithDiagnostics(request(schemas(input), "Dto"))
		if err != nil || !strings.Contains(source(t, got, "HolderDto"), "`openapi-kt model default") {
			t.Fatal("Referenced object collection graduation", input, got, err)
		}
	}
}
