package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func unionAllOfInput(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../tests/fixtures/http/union-allof.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestUnionAllOfBorrowedObjectShapes(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			r := request(unionAllOfInput(t), "Dto")
			r.Options.ModelStyle, r.Options.Validation = style, checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(style, checked, err)
			}
			composed := source(t, got, "models/ComposedDto")
			for _, part := range []string{"public sealed class ComposedDto", "public abstract val id: kotlin.String", "public val body: kotlin.String", "public val count: kotlin.Int", "UnknownComposedDto"} {
				if !strings.Contains(composed, part) {
					t.Fatal(part, composed)
				}
			}
			product := source(t, got, "models/ProductDto")
			keyword := "public data class "
			if style == core.PlainClass {
				keyword = "public class "
			}
			for _, part := range []string{keyword + "Note(", keyword + "Note2(", keyword + "Count(", keyword + "Count4("} {
				if !strings.Contains(product, part) {
					t.Fatal(part, product)
				}
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("non-deterministic", err)
			}
		}
	}
}
