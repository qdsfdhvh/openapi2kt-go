package core_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func FuzzRetainedEnumJVMNameLowerBound(f *testing.F) {
	for _, n := range []uint16{0, 1, 10922, 10923, 11999} {
		for mode := range uint8(8) {
			f.Add(n, mode)
		}
	}
	f.Fuzz(func(t *testing.T, size uint16, mode uint8) {
		n := int(size) % 12000
		wire, _ := json.Marshal(strings.Repeat("/", n))
		leaf := `{"enum":[null,` + string(wire) + `]}`
		var input []byte
		pointer := "/components/schemas/State/enum/1"
		switch mode % 3 {
		case 0:
			input = schemas(`{"State":` + leaf + `}`)
		case 1:
			input = schemas(`{"State":{"properties":{"a/~":` + leaf + `}}}`)
			pointer = "/components/schemas/State/properties/a~1~0/enum/1"
		case 2:
			// The response uses a scalar projection, not an emitted enum class.
			input = []byte(`{"openapi":"3.1.0","paths":{"/x":{"get":{"responses":{"200":{"content":{"application/json":{"schema":` + leaf + `}}}}}}}}`)
		}
		r := request(input, "Dto")
		r.Options.Validation = mode&1 != 0
		if mode&2 != 0 {
			r.Options.ModelStyle = core.PlainClass
		}
		r.Options.DeduplicateModels, r.Options.HoistSharedShapes, r.Options.MergeIdenticalModels = mode&4 != 0, mode&4 != 0, mode&4 != 0
		r.Options.LockHoistedNames = r.Options.HoistSharedShapes
		got, err := core.GenerateWithDiagnostics(r)
		control, replayErr := core.GenerateWithDiagnostics(r)
		if !reflect.DeepEqual(got, control) || !reflect.DeepEqual(err, replayErr) {
			t.Fatal("enum lower-bound check was not deterministic")
		}
		// EnumMemberName expands each slash to SLASH with separators (6n-1).
		// Acceptance here asserts only this lower bound, not JVM feasibility.
		if mode%3 != 2 && n >= 10923 {
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("certain physical class-name overflow generated output", n, mode, err)
			}
			d := err.(core.Diagnostics)[0]
			if d.Code != "E_MODEL_LIMIT" || d.Pointer == nil || *d.Pointer != pointer {
				t.Fatal("overflow lost source Pointer", d, pointer)
			}
		} else if err != nil {
			t.Fatal("lower-bound proof overclaimed a physical overflow", n, mode, err)
		}
		// Equal-size whitespace maps to VALUE, without changing any input cap.
		space, _ := json.Marshal(strings.Repeat(" ", n))
		r.Document = schemas(`{"State":{"enum":[null,` + string(space) + `]}}`)
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("safe vocabulary was refused by source token size", n, err)
		}
	})
}
