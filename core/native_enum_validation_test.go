package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNativeEnumDirectDTOFieldMembershipIsOptInAndMapped(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"properties":{"count":{"type":"integer","enum":[1,2]},"wide":{"type":"integer","format":"int64","enum":[-9223372036854775808,3]},"ratio":{"type":"number","format":"float","enum":[1,1.0,2.5]},"flag":{"type":"boolean","enum":[true]},"optional":{"type":["integer","null"],"enum":[1,2],"default":2}},"required":["count","wide","ratio","flag"]}}}}`)
	req := request(input, "Dto")
	off, err := core.GenerateWithDiagnostics(req)
	if err != nil {
		t.Fatal(err)
	}
	req.Options.Validation = true
	on, err := core.GenerateWithDiagnostics(req)
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"value == 1 || value == 2", "value == (-9223372036854775807L - 1L)", "value == 1.0f || value == 2.5f", "value == true", "does not belong to enum."} {
		if !strings.Contains(source(t, on, "RecordDto"), part) {
			t.Fatal(part, source(t, on, "RecordDto"))
		}
	}
	if len(on.Warnings) != 0 || len(off.Warnings) != 5 {
		t.Fatal(on.Warnings, off.Warnings)
	}
	if !strings.Contains(source(t, on, "RecordDto"), "= 2") {
		t.Fatal("default was discarded")
	}
	again, err := core.GenerateWithDiagnostics(req)
	if err != nil || !reflect.DeepEqual(on, again) {
		t.Fatal(err)
	}
}
