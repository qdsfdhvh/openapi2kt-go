package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNestedObjectCollectionDefaultsTypedPresence(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/nested-object-collection-defaults.json")
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
				"public val rows: kotlin.collections.List<RowsInner>? = `openapi-kt model default List org example models models HolderDto RowsInner`(",
				"public val maps: kotlin.collections.Map<kotlin.String, MapsValue>? = `openapi-kt model default Map org example models models HolderDto MapsValue`(",
				"public val mixed: kotlin.collections.List<kotlin.collections.Map<kotlin.String, MixedInnerValue>>? = `openapi-kt model default List Map org example models models HolderDto MixedInnerValue`(",
				"public val requiredRows: kotlin.collections.List<RequiredRowsInner>,",
				"public val nilRows: kotlin.collections.List<NilRowsInner>? = null,",
				// Scalar-only collections keep the existing array helper.
				"`openapi-kt default array String`(",
			} {
				if !strings.Contains(holder, marker) {
					t.Fatal("Lost object collection default API", marker, holder)
				}
			}
			if strings.Contains(holder, "requiredRows: kotlin.collections.List<RequiredRowsInner> =") {
				t.Fatal("required collection default acquired an initializer", holder)
			}
			patch := source(t, got, "PatchHolderInputDto")
			if !strings.Contains(patch, "Patch<kotlin.collections.List<RowsInner>>") || strings.Contains(patch, "model default") {
				t.Fatal("inline request collection default lost Patch suppression", patch)
			}
			required := source(t, got, "RequiredCfgDto")
			if !strings.Contains(required, "public val rows: kotlin.collections.List<RowsInner>,") {
				t.Fatal("required nullable collection default lost presence", required)
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("nondeterministic object collection defaults", err, style, checked)
			}
		}
	}
	// Object leaves inside collections previously failed under the builtin/
	// temporal profiles; they now graduate to this object collection role.
	for _, input := range []string{
		`{"Holder":{"properties":{"value":{"type":"array","items":{"type":"array","items":{"properties":{"id":{"type":"integer"}}}},"default":[]}}}}`,
		`{"Holder":{"properties":{"value":{"type":"array","items":{"type":"array","items":{"properties":{"day":{"type":"string","format":"date"}}}},"default":[]}}}}`,
		`{"Holder":{"properties":{"value":{"type":"array","items":{"type":"object","properties":{"v":{"type":"integer"}}},"default":[{"v":1}]}}}}`,
	} {
		got, err := core.GenerateWithDiagnostics(request(schemas(input), "Dto"))
		if err != nil || !strings.Contains(source(t, got, "HolderDto"), "`openapi-kt model default") {
			t.Fatal("Object collection default graduation", input, got, err)
		}
	}
}
