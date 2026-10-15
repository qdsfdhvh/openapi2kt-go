package testutil

import (
	"encoding/json"
	"fmt"
)

// RecursiveJSONControl lowers only the frozen fixture's ZAny root to true.
// Raw messages preserve property/branch ordering in every untouched subtree.
// This is whole-envelope test tooling, not production schema normalization.
func RecursiveJSONControl(input []byte) ([]byte, error) {
	var doc, components, schemas map[string]json.RawMessage
	if err := json.Unmarshal(input, &doc); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(doc["components"], &components); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(components["schemas"], &schemas); err != nil {
		return nil, err
	}
	if len(schemas["ZAny"]) == 0 || doc["openapi"] == nil {
		return nil, fmt.Errorf("control needs the frozen ZAny fixture")
	}
	schemas["ZAny"] = json.RawMessage("true")
	var err error
	components["schemas"], err = json.Marshal(schemas)
	if err != nil {
		return nil, err
	}
	doc["components"], err = json.Marshal(components)
	if err != nil {
		return nil, err
	}
	return json.Marshal(doc)
}
