package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestIntrinsicNullableRawMethodSlots(t *testing.T) {
	for _, stream := range []bool{false, true} {
		limit := 240
		if stream {
			limit--
		}
		for _, count := range []int{limit, limit + 1} {
			var parameters []string
			for i := range count {
				parameters = append(parameters, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":{"$ref":"#/components/schemas/A"}}`, i))
			}
			responses := `{"204":{}}`
			if stream {
				responses = `{"200":{"content":{"text/event-stream":{"schema":{"properties":{}}}}}}`
			}
			input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"A":{"$ref":"#/components/schemas/Z"},"Z":{"type":"object","nullable":true}}},"paths":{"/wide":{"get":{"parameters":[` + strings.Join(parameters, ",") + `],"responses":` + responses + `}}}}`)
			got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
			if count == limit {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_OPERATION_LIMIT" || len(got.Artifacts.Files) != 0 || len(got.Warnings) != 0 {
				t.Fatal(count, stream, got, err)
			}
		}
	}
}
