package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestUnformattedNativeEnumRootFactory(t *testing.T) {
	for _, tc := range []struct{ kind, vocab, value, scalar string }{
		{"integer", "[1,2]", "1", "kotlin.Int"},
		{"number", "[1.5,2.5]", "1.5", "kotlin.Double"},
	} {
		for _, own := range []bool{false, true} {
			for _, checked := range []bool{false, true} {
				annotation := ""
				if own {
					annotation = `,"default":` + tc.value
				}
				rq := request(schemas(`{"T":{"type":"`+tc.kind+`","enum":`+tc.vocab+annotation+`}}`), "Dto")
				rq.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal("unformatted native enum root factory rejected", tc.kind, own, checked, err)
				}
				text := source(t, got, "TDto")
				if !strings.Contains(text, "public typealias TDto = "+tc.scalar) || strings.Contains(text, "public fun `TDto default`") != own {
					t.Fatal("shared scalar mapping or opt-in factory drift", text)
				}
			}
		}
	}
}

func TestUnformattedNativeFactoryStrictMembershipAndVocabularyBudget(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ kind, vocab, value string }{
			{"integer", "[1,2]", "9"}, {"integer", "[1,2]", "1.0"},
			{"number", "[9007199254740993,9007199254740995]", "9007199254740992"},
		} {
			rq := request(schemas(`{"T":{"type":"`+tc.kind+`","enum":`+tc.vocab+`,"default":`+tc.value+`}}`), "Dto")
			rq.Options.Validation = checked
			if got, err := core.GenerateWithDiagnostics(rq); err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/T/default" || len(got.Artifacts.Files) != 0 {
				t.Fatal("native default lost strict raw membership/kind", checked, tc, err)
			}
		}
		for _, count := range []int{240, 241} {
			members := make([]string, count)
			for i := range members {
				members[i] = fmt.Sprint(i)
			}
			rq := request(schemas(`{"T":{"type":"integer","enum":[`+strings.Join(members, ",")+`],"default":0}}`), "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if count == 240 && err != nil || count > 240 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0) {
				t.Fatal("raw vocabulary budget drift", checked, count, err)
			}
		}
	}
}
