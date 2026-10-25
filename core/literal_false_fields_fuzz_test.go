package core_test

import (
	"bytes"
	"fmt"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func FuzzLiteralFalseFields(f *testing.F) {
	f.Add(uint8(1), true, false)
	f.Add(uint8(240), true, true)
	f.Add(uint8(12), false, true)
	f.Fuzz(func(t *testing.T, raw uint8, version31, required bool) {
		count := int(raw%243) + 1
		properties, keys := []string{}, []string{}
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("p%03d", i)
			properties = append(properties, `"`+name+`":false`)
			keys = append(keys, `"`+name+`"`)
		}
		ownRequired := ""
		if required {
			ownRequired = `"required":[` + strings.Join(keys, ",") + `],`
		}
		input := schemas(`{"Holder":{` + ownRequired + `"properties":{` + strings.Join(properties, ",") + `}}}`)
		if !version31 {
			input = []byte(strings.Replace(string(input), "3.1.0", "3.0.3", 1))
		}
		original := bytes.Clone(input)
		rq := request(input, "Dto")
		rq.Options.Validation = raw&1 != 0
		rq.Options.DeduplicateModels = raw&2 != 0
		rq.Options.HoistSharedShapes = raw&4 != 0
		rq.Options.MergeIdenticalModels = raw&8 != 0
		rq.Options.LockHoistedNames = rq.Options.HoistSharedShapes
		if required {
			rq.Options.ModelStyle = core.PlainClass
		}
		first, e := core.GenerateWithDiagnostics(rq)
		if version31 && count <= 240 && e != nil {
			t.Fatal("bounded literal-false actualmodel refused", e)
		}
		if (!version31 || count > 240) && e == nil {
			t.Fatal("version/slot boundary lost")
		}
		second, retry := core.GenerateWithDiagnostics(rq)
		if !bytes.Equal(original, input) || !reflect.DeepEqual(first, second) || !reflect.DeepEqual(e, retry) {
			t.Fatal("literal-false mutation/nondeterminism", e, retry)
		}
		if e != nil && !reflect.DeepEqual(first, core.GenerationReport{}) {
			t.Fatal("partial literal-false failure output")
		}
	})
}
