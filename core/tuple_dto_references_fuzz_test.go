package core_test

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func FuzzTupleDTOReferences(f *testing.F) {
	f.Add(uint8(1), true, false)
	f.Add(uint8(47), false, true)
	f.Fuzz(func(t *testing.T, raw uint8, nullable, tail bool) {
		count := int(raw%50) + 1
		entries := []string{`"Item":{"properties":{"value":{"type":"string"},"next":{"$ref":"#/components/schemas/Item"}},"required":["value"]}`}
		previous := "Item"
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("A%02d", i)
			entries = append(entries, `"`+name+`":{"$ref":"#/components/schemas/`+previous+`","nullable":false}`)
			previous = name
		}
		intent := ""
		if nullable {
			intent = `,"nullable":true`
		}
		member := `{"$ref":"#/components/schemas/` + previous + `"` + intent + `}`
		ending := ""
		if tail {
			ending = `,"items":{"type":"integer"}`
		}
		entries = append(entries, `"Probe":{"type":"array","prefixItems":[`+member+`,`+member+`]`+ending+`}`)
		input := schemas(`{` + strings.Join(entries, ",") + `}`)
		original := bytes.Clone(input)
		rq := request(input, "Dto")
		first, err := core.GenerateWithDiagnostics(rq)
		second, retry := core.GenerateWithDiagnostics(rq)
		if !bytes.Equal(input, original) || !reflect.DeepEqual(first, second) || !reflect.DeepEqual(err, retry) {
			t.Fatal("DTO tuple source mutation or nondeterminism", err, retry)
		}
		if err != nil && !reflect.DeepEqual(first, core.GenerationReport{}) {
			t.Fatal("partial DTO tuple failure output")
		}
	})
}
