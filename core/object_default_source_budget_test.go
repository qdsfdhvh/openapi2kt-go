package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

// Each level is tiny and nominally shared, but a default source walk used to
// build both complete successor trees even when its supplied value was {}.
func objectDefaultFanoutDocument(levels int, field string) []byte {
	var parts []string
	for i := 0; i < levels; i++ {
		properties := `"value":{"type":"string"}`
		if i+1 < levels {
			properties = fmt.Sprintf(`"left":{"$ref":"#/components/schemas/Node%02d"},"right":{"$ref":"#/components/schemas/Node%02d"}`, i+1, i+1)
		}
		parts = append(parts, fmt.Sprintf(`"Node%02d":{"properties":{%s}}`, i, properties))
	}
	parts = append(parts, `"Holder":{"properties":{"v":`+field+`}}`)
	return schemas("{" + strings.Join(parts, ",") + "}")
}

func TestObjectDefaultSourceFanoutBudget(t *testing.T) {
	field := `{"$ref":"#/components/schemas/Node00","default":{}}`
	for _, levels := range []int{14, 28} {
		input := objectDefaultFanoutDocument(levels, field)
		for _, checked := range []bool{false, true} {
			for formIndex, form := range []string{
				string(input),
				strings.Replace(string(input), `"Holder":{"properties"`, `"Holder":{"required":["v"],"properties"`, 1),
				strings.Replace(string(input), field, strings.Replace(field, `"default":{}`, `"nullable":true,"default":null`, 1), 1),
				strings.Replace(string(input), `"Holder":{"properties":{"v":`+field+`}}`, `"Holder":{"allOf":[{"properties":{"v":`+field+`}},{"properties":{"v":{"type":"string"}}}]}`, 1),
			} {
				// The larger original repro covers the ordinary consumer. Keep the
				// suppression matrix at14: checked discarded shapes have their own
				// earlier source-depth limit at the larger size.
				if levels == 28 && formIndex > 0 {
					continue
				}
				r := request([]byte(form), "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				ds, ok := err.(core.Diagnostics)
				if !ok || len(ds) == 0 || ds[0].Code != "E_MODEL_LIMIT" || ds[0].Pointer == nil || !strings.HasPrefix(*ds[0].Pointer, "/components/schemas/Node") || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatalf("levels=%d checked=%v form=%d must stop before suppression with original-source diagnostic: %v", levels, checked, formIndex, err)
				}
			}
			requestDoc := strings.TrimSuffix(string(carrierRequestDocument(field, false)), "}") + `,"components":` + strings.SplitN(string(input), `"components":`, 2)[1]
			r := request([]byte(requestDoc), "Dto")
			r.Options.Validation = checked
			if got, err := core.GenerateWithDiagnostics(r); err == nil || !strings.Contains(err.Error(), "E_MODEL_LIMIT") || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("Patch annotation bypassed source work budget", err)
			}
		}
	}
	// Typed reference admission and bounded object defaults remain available.
	for _, input := range [][]byte{
		objectDefaultFanoutDocument(28, strings.Replace(field, `,"default":{}`, "", 1)),
		objectDefaultFanoutDocument(8, field),
	} {
		if _, err := core.GenerateWithDiagnostics(request(input, "Dto")); err != nil {
			t.Fatal("bounded source or no-default nominal graph regressed", err)
		}
	}
}
