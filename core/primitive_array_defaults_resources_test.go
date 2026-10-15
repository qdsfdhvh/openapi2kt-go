package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestPrimitiveArrayDefaultsOriginalDepthMembersAndLargeLiteralBudgets(t *testing.T) {
	for _, count := range []int{240, 241} {
		fields := make([]string, count)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"v%d":{"type":"array","items":{"type":"integer"},"default":[%d]}`, i, i)
		}
		report, err := core.GenerateWithDiagnostics(request(schemas(`{"Holder":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto"))
		if count == 240 && err != nil {
			t.Fatal(err)
		}
		if count == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(report.Artifacts.Files) != 0) {
			t.Fatal("Array defaults bypassed constructor slots", err)
		}
	}
	for _, length := range []int{47, 48} {
		field := `{"type":"array","items":{"type":"string"},"default":["ok"]}`
		for i := 0; i < length; i++ {
			field = fmt.Sprintf(`{"properties":{"n%d":%s}}`, i, field)
		}
		report, err := core.GenerateWithDiagnostics(request(schemas(`{"Holder":`+field+`}`), "Dto"))
		if length == 47 && err != nil {
			t.Fatal("Source48 array default rejected", err)
		}
		if length == 48 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(report.Artifacts.Files) != 0) {
			t.Fatal("Original array item source49 disappeared", err)
		}
	}
	for _, literal := range []string{strings.Repeat("a", 70000), strings.Repeat("🙂", 18000)} {
		input := schemas(`{"Holder":{"properties":{"kotlin":{"type":"string"},"value":{"type":"array","items":{"type":"string"},"default":["` + literal + `"]}}}}`)
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if err != nil || !strings.Contains(source(t, report, "HolderDto"), literal) {
			t.Fatal("Large array literal rejected/truncated", err)
		}
	}
	items := strings.Repeat("1,", 99999) + "1"
	report, err := core.GenerateWithDiagnostics(request(schemas(`{"Holder":{"properties":{"value":{"type":"array","items":{"type":"integer"},"default":[`+items+`]}}}}`), "Dto"))
	if err != nil || !strings.Contains(source(t, report, "HolderDto"), "["+items+"]") {
		t.Fatal("Large array data consumed schema work/slots", err)
	}
}
