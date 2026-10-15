package testutil

import (
	"encoding/json"
	"fmt"
)

// BooleanRootControl lowers only direct true named roots for the frozen whole
// fixture. It is test tooling, not production normalization or depth equivalence.
func BooleanRootControl(input []byte) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(input, &doc); err != nil {
		return nil, err
	}
	components, ok := doc["components"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("control needs components")
	}
	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("control needs schemas")
	}
	n := 0
	for name, schema := range schemas {
		if flag, ok := schema.(bool); ok && flag {
			schemas[name] = map[string]any{"type": "object"}
			n++
		}
	}
	if n != 1 {
		return nil, fmt.Errorf("frozen control expects one true root, got %d", n)
	}
	return json.Marshal(doc)
}
