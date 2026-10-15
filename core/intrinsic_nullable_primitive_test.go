package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/testutil"
)

func intrinsicPrimitiveInput(t testing.TB) []byte {
	t.Helper()
	b, err := os.ReadFile("../tests/fixtures/http/intrinsic-nullable-primitives.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func TestIntrinsicNullablePrimitiveWholeControl(t *testing.T) {
	b, err := testutil.NullablePrimitiveControl(intrinsicPrimitiveInput(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []bool{false, true} {
		r := request(b, "Dto")
		r.Options.Validation = flag
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal(err)
		}
	}
}
func TestIntrinsicNullablePrimitiveDesiredSuccess(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			r := request(intrinsicPrimitiveInput(t), suffix)
			r.Options.Validation = flag
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(err)
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("determinism", err)
			}
			if len(got.Warnings) != 0 {
				t.Fatal(got.Warnings)
			}
			for _, n := range []string{"String", "Boolean", "Int", "Long", "Float", "Double"} {
				if s := source(t, got, "Z"+n+suffix); !strings.Contains(s, "= kotlin."+n+"?") {
					t.Fatal(s)
				}
			}
			for _, n := range []string{"AFalse", "CFalseInherited"} {
				if s := source(t, got, n+suffix); !strings.Contains(s, "= kotlin.String\n") {
					t.Fatal(s)
				}
			}
			for _, n := range []string{"AInherited", "BNullable", "ATrue"} {
				if s := source(t, got, n+suffix); !strings.Contains(s, "= kotlin.String?") {
					t.Fatal(s)
				}
			}
			h := source(t, got, "Holder"+suffix)
			if !strings.Contains(h, "val long: kotlin.Long? = null") || !strings.Contains(h, "val narrow: kotlin.String,") || !strings.Contains(h, "kotlin.collections.Map<kotlin.String, kotlin.Long?>") {
				t.Fatal(h)
			}
			p := source(t, got, "SubmitInput"+suffix)
			if !strings.Contains(p, "val requiredNull: kotlin.Long?,") || !strings.Contains(p, "Patch<kotlin.Long>") || !strings.Contains(p, "val requiredFalse: kotlin.Long,") {
				t.Fatal(p)
			}
		}
	}
}
