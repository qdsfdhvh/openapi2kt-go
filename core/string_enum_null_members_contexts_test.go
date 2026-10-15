package core_test

import (
	"strings"
	"testing"
)

func TestStringEnumNullMembersKeepFoldDefaultsAndSourceExclusions(t *testing.T) {
	for _, leaf := range []string{`{"type":["string","null"],"enum":[null,"ready"],"default":null}`, `{"nullable":true,"enum":[null,"ready"],"default":"ready"}`} {
		got := report(t, fieldDoc(leaf), "Dto")
		if text := source(t, got, "RecordDto"); !strings.Contains(text, "public data object READY") || !strings.Contains(text, "Unrecognized") || strings.Contains(text, `serialName: kotlin.String = "null"`) {
			t.Fatal(text)
		}
	}
	got := report(t, fieldDoc(`{"anyOf":[{"type":["string","null"],"enum":["ready",null]},{"type":"null"}]}`), "Dto")
	if text := source(t, got, "RecordDto"); !strings.Contains(text, "Value?") || !strings.Contains(text, "public data object READY") {
		t.Fatal(text)
	}
	// Existing actual retained owner guards still reject vanished/reprojected
	// declarations; source vocabulary admission is not an ownership exemption.
	input := schemas(`{"Owner":{"allOf":[{"properties":{"state":{"type":["string","null"],"enum":["ready",null]}}},{"properties":{"state":{"type":"string"}}}]} ,"Alias":{"$ref":"#/components/schemas/Owner/allOf/0/properties/state"}}`)
	aliasFailure(t, input, "E_REF_UNSUPPORTED", "/components/schemas/Alias/$ref")
}
