package cli_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestObjectDefaultSourceFanoutPreservesOutput(t *testing.T) {
	var definitions []string
	for i := 0; i < 28; i++ {
		properties := `"s":{"type":"string"}`
		if i < 27 {
			properties = fmt.Sprintf(`"left":{"$ref":"#/components/schemas/N%d"},"right":{"$ref":"#/components/schemas/N%d"}`, i+1, i+1)
		}
		definitions = append(definitions, fmt.Sprintf(`"N%d":{"properties":{%s}}`, i, properties))
	}
	definitions = append(definitions, `"Holder":{"properties":{"v":{"$ref":"#/components/schemas/N0","default":{}}}}`)
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{` + strings.Join(definitions, ",") + `}}}`)
	for _, checked := range []bool{false, true} {
		p := newProject(t)
		put(t, p.Input, rawJSONFieldDocument())
		args := rawJSONFieldArgs(p.Input, p.Output, checked, core.DataClass)
		result(t, invoke(t, p.Root, args...), 0)
		before := snapshot(t, p.Output)
		put(t, p.Input, input)
		for _, suffix := range [][]string{nil, {"--check"}} {
			failure := fails(t, invoke(t, p.Root, append(append([]string{}, args...), suffix...)...), 1, "E_MODEL_LIMIT")
			if len(failure.Files) != 0 || len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Pointer == nil || !strings.HasPrefix(*failure.Diagnostics[0].Pointer, "/components/schemas/N") {
				t.Fatal("source budget lost original Pointer or leaked files", failure)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("source-budget rejection changed output bytes/modes/mtimes")
			}
		}
	}
}
