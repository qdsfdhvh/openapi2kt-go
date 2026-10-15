package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func hierarchyIntrinsicNullableInput(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../tests/fixtures/http/hierarchy-intrinsic-nullable.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestIntrinsicNullableHierarchyKeepsPhysicalClassAndConsumerIntent(t *testing.T) {
	for _, suffix := range []string{"", "Dto"} {
		for _, checked := range []bool{false, true} {
			req := request(hierarchyIntrinsicNullableInput(t), suffix)
			req.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(req)
			if err != nil {
				t.Fatal(err)
			}
			root := source(t, got, "Root"+suffix)
			leaf := source(t, got, "First"+suffix)
			if !strings.Contains(root, "public sealed class Root"+suffix+" {") || !strings.Contains(leaf, "public data class First"+suffix+"(") {
				t.Fatal(root, leaf)
			}
			for _, name := range []string{"ERoot", "GRoot"} {
				if !strings.Contains(source(t, got, name+suffix), "= org.example.models.models.Root"+suffix+"?") {
					t.Fatal(name)
				}
			}
			if s := source(t, got, "FRoot"+suffix); strings.Contains(s, "?") || !strings.Contains(s, "= org.example.models.models.Root"+suffix) {
				t.Fatal(s)
			}
			holder := source(t, got, "IntrinsicHolder"+suffix)
			for _, field := range []struct {
				wire, physical string
				nullable       bool
			}{{"root", "Root", true}, {"rootFalse", "Root", false}, {"group", "Group", true}, {"groupFalse", "Group", false}, {"leaf", "First", true}, {"leafFalse", "First", false}} {
				ty := "org.example.models.models." + field.physical + suffix
				if field.nullable {
					ty += "?"
				}
				ending := ","
				if field.nullable {
					ending = " = null," // Ordinary nullable model fields retain their default.
				}
				if !strings.Contains(holder, "public val "+field.wire+": "+ty+ending) {
					t.Fatal(field, holder)
				}
			}
			again, err := core.GenerateWithDiagnostics(req)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("determinism", err)
			}
		}
	}
}
