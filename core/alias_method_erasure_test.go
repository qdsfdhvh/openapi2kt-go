package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTransparentAliasMethodErasure(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/alias-method-erasure.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", "Dto"} {
		t.Run("suffix="+suffix, func(t *testing.T) {
			rq := request(input, suffix)
			r, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(err)
			}
			again, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("alias method allocation is not deterministic", err)
			}
			s := source(t, r, "DefaultApi")
			for _, name := range []string{"send", "send2", "send3", "send4", "send5", "map", "map2"} {
				if n := strings.Count(s, "public suspend fun "+name+"("); n != 1 {
					t.Errorf("%s count %d, want one erased-signature-safe method", name, n)
				}
			}
			for _, name := range []string{"model", "wrapped"} {
				if n := strings.Count(s, "public suspend fun "+name+"("); n != 2 {
					t.Errorf("%s count %d, want two distinct nominal overloads", name, n)
				}
			}
			if strings.Contains(s, "public suspend fun model2(") || strings.Contains(s, "public suspend fun wrapped2(") {
				t.Error("nominal DTO/valueclass identities must remain signature barriers")
			}
		})
	}
}
