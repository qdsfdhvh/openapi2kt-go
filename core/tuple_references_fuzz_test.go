package core_test

import (
	"bytes"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func FuzzTupleScalarReferences(f *testing.F) {
	f.Add(uint8(1), true, false)
	f.Add(uint8(47), false, true)
	f.Fuzz(func(t *testing.T, raw uint8, nullable, tail bool) {
		n := int(raw%50) + 1
		aliases := []string{`"Text":{"type":"string","nullable":true}`}
		previous := "Text"
		for i := 0; i < n; i++ {
			name := strings.Repeat("A", i+1)
			aliases = append(aliases, `"`+name+`":{"$ref":"#/components/schemas/`+previous+`","nullable":false}`)
			previous = name
		}
		member := `{"$ref":"#/components/schemas/` + previous + `"}`
		if nullable {
			member = `{"$ref":"#/components/schemas/` + previous + `","nullable":true}`
		}
		suffix := ""
		if tail {
			suffix = `,"items":{"type":"integer"}`
		}
		aliases = append(aliases, `"Probe":{"type":"array","prefixItems":[`+member+`,{"type":"string"}]`+suffix+`}`)
		input := schemas(`{` + strings.Join(aliases, ",") + `}`)
		original := bytes.Clone(input)
		rq := request(input, "Dto")
		first, err := core.GenerateWithDiagnostics(rq)
		second, again := core.GenerateWithDiagnostics(rq)
		if !bytes.Equal(input, original) || (err == nil) != (again == nil) || !reflect.DeepEqual(first, second) {
			t.Fatal("source mutation or nondeterminism", err, again)
		}
		if err != nil && len(first.Artifacts.Files) != 0 {
			t.Fatal("partial failure output")
		}
	})
}
