package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func temporalReferenceChainsInput(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../tests/fixtures/http/temporal-reference-chains.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}

func TestTemporalReferenceChainsCanonicalScalarViews(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			r := request(temporalReferenceChainsInput(t), "Dto")
			r.Options.ModelStyle = style
			r.Options.Validation = checked
			got, e := core.GenerateWithDiagnostics(r)
			if e != nil {
				t.Fatal(e)
			}
			for _, tc := range []struct {
				alias, scalar string
				nullable      bool
			}{
				{"StampBridge", "kotlin.time.Instant", true}, {"StampFalseBridge", "kotlin.time.Instant", false}, {"AgainBridge", "kotlin.time.Instant", false},
				{"DayBridge", "kotlinx.datetime.LocalDate", true}, {"DayFalse", "kotlinx.datetime.LocalDate", false}, {"NativeBridge", "kotlin.time.Instant", false},
			} {
				want := "public typealias " + tc.alias + "Dto = " + tc.scalar
				if tc.nullable {
					want += "?"
				}
				if !strings.Contains(source(t, got, tc.alias+"Dto"), want+"\n") {
					t.Fatal(tc, source(t, got, tc.alias+"Dto"))
				}
			}
			patch := source(t, got, "PatchTemporalInputDto")
			if !strings.Contains(patch, "Patch<kotlin.time.Instant>") || !strings.Contains(patch, "Patch<kotlinx.datetime.LocalDate>") {
				t.Fatal(patch)
			}
			api := source(t, got, "DefaultApi")
			if !strings.Contains(api, "stamp: kotlin.time.Instant") || !strings.Contains(api, "day: kotlinx.datetime.LocalDate?") || strings.Contains(api, "stamp.rawValue") || strings.Count(api, "public suspend fun echoTime(") != 1 || strings.Count(api, "public suspend fun echoTime2(") != 1 {
				t.Fatal(api)
			}
			again, e := core.GenerateWithDiagnostics(r)
			if e != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("nondeterministic", e)
			}
		}
	}
}

func TestTemporalReferenceChainOriginalTypedAndRootFailures(t *testing.T) {
	for _, terminal := range []string{`{"type":"string","format":"date"}`, `{"type":"string","format":"date-time"}`, `{"x-native-type":"date"}`} {
		for _, root := range []bool{false, true} {
			body := `"Time":` + terminal + `,"Hops":{"properties":{"time":{"$ref":"#/components/schemas/Time"}}}`
			if root {
				body += `,"Bridge":{"$ref":"#/components/schemas/Hops/properties/time","nullable":false}`
			} else {
				body += `,"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Hops/properties/time"}},"required":["value"]}`
			}
			for _, checked := range []bool{false, true} {
				r := request(schemas(`{`+body+`}`), "Dto")
				r.Options.Validation = checked
				if _, e := core.GenerateWithDiagnostics(r); e != nil {
					t.Fatal(terminal, root, checked, e)
				}
			}
		}
	}
}
