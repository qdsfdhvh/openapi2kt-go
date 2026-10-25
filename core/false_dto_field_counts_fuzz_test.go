package core_test

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func FuzzFalseDTOFieldCounts(f *testing.F) {
	f.Add(uint8(1), true, false)
	f.Add(uint8(47), false, true)
	f.Fuzz(func(t *testing.T, raw uint8, nullable, tail bool) {
		count := int(raw%50) + 1
		entries := []string{`"Choice":false`}
		previous := "Choice"
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("A%02d", i)
			entries = append(entries, `"`+name+`":{"$ref":"#/components/schemas/`+previous+`","nullable":false}`)
			previous = name
		}
		intent := ""
		if nullable {
			intent = `,"nullable":true`
		}
		entries = append(entries, `"Alias":{"$ref":"#/components/schemas/`+previous+`"`+intent+`}`)
		entries = append(entries, `"Holder":{"minProperties":1,"maxProperties":1,"properties":{"value":{"$ref":"#/components/schemas/Alias"}}},"Consumer":{"properties":{"target":{"$ref":"#/components/schemas/Holder","default":{"value":null}}}}`)
		input := schemas(`{` + strings.Join(entries, ",") + `}`)
		original := bytes.Clone(input)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		if tail {
			rq.Options.DeduplicateModels = true
			rq.Options.HoistSharedShapes = true
			rq.Options.MergeIdenticalModels = true
			rq.Options.LockHoistedNames = true
		}
		if nullable {
			rq.Options.ModelStyle = core.PlainClass
		}
		first, err := core.GenerateWithDiagnostics(rq)
		if nullable && count <= 43 && err != nil {
			t.Fatal("bounded nullable consuming DTO field default refused", err)
		}
		if !nullable && err == nil {
			t.Fatal("nonnullable false member acquired null value")
		}

		second, retry := core.GenerateWithDiagnostics(rq)
		if !bytes.Equal(input, original) || !reflect.DeepEqual(first, second) || !reflect.DeepEqual(err, retry) {
			t.Fatal("false-reference field default source mutation or nondeterminism", err, retry)
		}
		if err != nil && !reflect.DeepEqual(first, core.GenerationReport{}) {
			t.Fatal("partial false-reference field default failure output")
		}
	})
}
