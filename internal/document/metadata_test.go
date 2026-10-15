package document

import (
	"bytes"
	"testing"
)

func TestMetadataParserDoesNotRaiseSchemaInputLimit(t *testing.T) {
	data := append([]byte(`{}`), bytes.Repeat([]byte(" "), MaxDocumentBytes-1)...)
	if _, err := Load(data, JSON, "input.json"); err == nil || err.Code != "E_INPUT_LIMIT" {
		t.Fatal(err)
	}
	if value, err := ParseObject(data, 64*1024*1024); err != nil || value.Kind() != Object {
		t.Fatal(value, err)
	}
	if _, err := ParseObject(data, len(data)-1); err == nil || err.Code != "E_INPUT_LIMIT" {
		t.Fatal("caller bound", err)
	}
}
