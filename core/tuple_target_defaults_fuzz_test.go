package core_test

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func FuzzTupleTargetDefaults(f *testing.F) {
	f.Add(uint8(1), uint8(0), true)
	f.Add(uint8(47), uint8(3), false)
	f.Fuzz(func(t *testing.T, raw, kind uint8, checked bool) {
		target, value := `{"type":"string","default":"target"}`, `"alias"`
		switch kind % 4 {
		case 1:
			target = `{"type":"string","enum":["target","alias"],"default":"target"}`
		case 2:
			target = `{"properties":{"value":{"type":"string"}},"required":["value"],"default":{"value":"target"}}`
			value = `{"value":"alias"}`
		case 3:
			target = `{"allOf":[{"properties":{"value":{"type":"string"}},"required":["value"]}],"default":{"value":"target"}}`
			value = `{"value":"alias"}`
		}
		entries := []string{`"Target":` + target}
		previous := "Target"
		for i := 0; i < int(raw%50)+1; i++ {
			name := fmt.Sprintf("A%02d", i)
			entries = append(entries, `"`+name+`":{"$ref":"#/components/schemas/`+previous+`","default":`+value+`}`)
			previous = name
		}
		entries = append(entries, `"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/`+previous+`"}],"items":{"type":"integer"}}`)
		input := schemas(`{` + strings.Join(entries, ",") + `}`)
		original := bytes.Clone(input)
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		first, err := core.GenerateWithDiagnostics(rq)
		second, retry := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(err, retry) || !bytes.Equal(input, original) {
			t.Fatal("independent factory source mutation or nondeterminism")
		}
		if err != nil && len(first.Artifacts.Files) != 0 {
			t.Fatal("partial factory error artifacts")
		}
	})
}
