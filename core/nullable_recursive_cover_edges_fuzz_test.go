package core_test

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func FuzzNullableRecursiveCoverEdges(f *testing.F) {
	f.Add(uint8(1), true, false)
	f.Add(uint8(24), false, true)
	f.Fuzz(func(t *testing.T, raw uint8, nullable, checked bool) {
		n := int(raw%26) + 1
		entries := []string{}
		for i := 0; i < n; i++ {
			name := fmt.Sprint("R", i)
			next := fmt.Sprint("R", (i+1)%n)
			root := coverChainRoot(next, next)
			root = strings.ReplaceAll(root, `"$ref":"#/components/schemas/`+next+`"`, fmt.Sprintf(`"$ref":"#/components/schemas/%s","nullable":%t`, next, nullable))
			entries = append(entries, `"`+name+`":`+root)
		}
		input := schemas(`{` + strings.Join(entries, ",") + `}`)
		original := bytes.Clone(input)
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, e := core.GenerateWithDiagnostics(rq)
		again, retry := core.GenerateWithDiagnostics(rq)
		if !bytes.Equal(input, original) || !reflect.DeepEqual(got, again) || !reflect.DeepEqual(e, retry) {
			t.Fatal("recursive nullable edge mutation or nondeterminism", e, retry)
		}
		if e != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("partial recursive edge artifacts")
		}
	})
}
