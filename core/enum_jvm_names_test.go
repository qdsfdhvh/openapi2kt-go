package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestRetainedEnumMemberJVMNamesRejectCertainOverflow(t *testing.T) {
	wire, _ := json.Marshal(strings.Repeat("/", 32766))
	input := schemas(`{"State":{"enum":[null,` + string(wire) + `]},"Alias":{"$ref":"#/components/schemas/State","nullable":false,"default":` + string(wire) + `}}`)
	r := request(input, "Dto")
	got, err := core.GenerateWithDiagnostics(r)
	if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
		t.Fatal("certainly impossible retained enum member generated Kotlin", err)
	}
	diagnostics := err.(core.Diagnostics)
	if diagnostics[0].Code != "E_MODEL_LIMIT" || diagnostics[0].Pointer == nil || *diagnostics[0].Pointer != "/components/schemas/State/enum/1" {
		t.Fatal("lost original null-inclusive source member Pointer", diagnostics)
	}
}

func TestRetainedEnumMemberJVMNamesKeepRolesAndOriginalPointers(t *testing.T) {
	wire, _ := json.Marshal(strings.Repeat("/", 10923)) // 65,537 ASCII name bytes.
	leaf := `{"enum":[null,` + string(wire) + `]}`
	object := `{"properties":{"a/~":` + leaf + `},"required":["a/~"]}`
	response := func(schema string) []byte {
		return []byte(`{"openapi":"3.1.0","paths":{"/x":{"get":{"responses":{"200":{"content":{"application/json":{"schema":` + schema + `}}}}}}}}`)
	}
	for _, tc := range []struct {
		name, pointer string
		document      []byte
	}{
		{"root", "/components/schemas/State/enum/1", schemas(`{"State":` + leaf + `}`)},
		{"nested", "/components/schemas/Holder/properties/a~1~0/enum/1", schemas(`{"Holder":` + object + `}`)},
		{"array", "/components/schemas/State/items/enum/1", schemas(`{"State":{"type":"array","items":` + leaf + `}}`)},
		{"map", "/components/schemas/State/additionalProperties/enum/1", schemas(`{"State":{"type":"object","additionalProperties":` + leaf + `}}`)},
		{"allOf", "/components/schemas/State/allOf/0/properties/a~1~0/enum/1", schemas(`{"State":{"allOf":[` + object + `,{"properties":{"id":{"type":"integer"}}}]}}`)},
		{"object union", "/components/schemas/State/oneOf/0/properties/a~1~0/enum/1", schemas(`{"State":{"oneOf":[` + object + `,{"properties":{"flag":{"type":"boolean"}},"required":["flag"]}]}}`)},
		{"response", "/paths/~1x/get/responses/200/content/application~1json/schema/properties/a~1~0/enum/1", response(object)},
		{"request", "/paths/~1x/post/requestBody/content/application~1json/schema/properties/a~1~0/enum/1", []byte(`{"openapi":"3.1.0","paths":{"/x":{"post":{"requestBody":{"required":true,"content":{"application/json":{"schema":` + object + `}}},"responses":{"204":{}}}}}}`)},
		{"error", "/paths/~1x/get/responses/400/content/application~1json/schema/enum/1", []byte(`{"openapi":"3.1.0","paths":{"/x":{"get":{"responses":{"204":{},"400":{"content":{"application/json":{"schema":` + leaf + `}}}}}}}}`)},
		{"SSE", "/paths/~1x/get/responses/200/content/text~1event-stream/schema/properties/data/properties/a~1~0/enum/1", []byte(`{"openapi":"3.1.0","paths":{"/x":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"properties":{"event":{"const":"item"},"data":` + object + `},"required":["data"]}}}}}}}}}`)},
	} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, checked := range []bool{false, true} {
				for mask := range 8 {
					t.Run(fmt.Sprintf("%s/%s/%t/%d", tc.name, style, checked, mask), func(t *testing.T) {
						r := request(tc.document, "Dto")
						r.Options.ModelStyle, r.Options.Validation = style, checked
						r.Options.DeduplicateModels, r.Options.HoistSharedShapes, r.Options.MergeIdenticalModels = mask&1 != 0, mask&2 != 0, mask&4 != 0
						r.Options.LockHoistedNames = r.Options.HoistSharedShapes
						got, err := core.GenerateWithDiagnostics(r)
						if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
							t.Fatal("retained overflow did not fail atomically", err)
						}
						d := err.(core.Diagnostics)[0]
						if d.Code != "E_MODEL_LIMIT" || d.Pointer == nil || *d.Pointer != tc.pointer {
							t.Fatal("retained member lost its original source", d, tc.pointer)
						}
					})
				}
			}
		}
	}
}

func TestEnumJVMNamesIgnoreNonEmittedVocabulariesAndKeepSafeBytes(t *testing.T) {
	long, _ := json.Marshal(strings.Repeat("/", 32766))
	leaf := `{"enum":[` + string(long) + `]}`
	projected := []byte(`{"openapi":"3.1.0","paths":{"/x":{"get":{"responses":{"200":{"content":{"application/json":{"schema":` + leaf + `}}}}}}}}`)
	discarded := schemas(`{"State":{"allOf":[{"properties":{"value":` + leaf + `}},{"properties":{"value":{"type":"string"}}}]}}`)
	space, _ := json.Marshal(strings.Repeat(" ", 32766))
	safe := schemas(`{"State":{"enum":[null,` + string(space) + `]},"Narrow":{"$ref":"#/components/schemas/State","nullable":false,"default":` + string(space) + `}}`)
	for _, input := range [][]byte{projected, discarded, safe} {
		r := request(input, "Dto")
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal("vocabulary without an overflowing physical member rejected", err)
		}
		control, err := core.GenerateWithDiagnostics(r)
		if err != nil || !reflect.DeepEqual(got, control) {
			t.Fatal("retained enum guard changed deterministic output", err)
		}
	}
}
