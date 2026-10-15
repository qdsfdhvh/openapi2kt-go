package testutil

import "encoding/json"

// BooleanValueControl lowers the synthetic direct-true value profile to its
// existing unconstrained-object counterpart. Non-schema booleans and Boolean
// additionalProperties are deliberately untouched; this is test tooling only.
func BooleanValueControl(input []byte) ([]byte, error) {
	var root map[string]any
	if err := json.Unmarshal(input, &root); err != nil {
		return nil, err
	}
	var schema func(any) any
	schema = func(v any) any {
		if b, ok := v.(bool); ok && b {
			return map[string]any{}
		}
		m, ok := v.(map[string]any)
		if !ok {
			return v
		}
		if props, ok := m["properties"].(map[string]any); ok {
			for key, child := range props {
				props[key] = schema(child)
			}
		}
		if child, ok := m["items"]; ok {
			m["items"] = schema(child)
		}
		if child, ok := m["additionalProperties"].(map[string]any); ok {
			m["additionalProperties"] = schema(child)
		}
		for _, key := range []string{"oneOf", "anyOf", "allOf"} {
			if children, ok := m[key].([]any); ok {
				for i, child := range children {
					children[i] = schema(child)
				}
			}
		}
		return m
	}
	if components, ok := root["components"].(map[string]any); ok {
		if schemas, ok := components["schemas"].(map[string]any); ok {
			for name, v := range schemas {
				schemas[name] = schema(v)
			}
		}
	}
	var envelopes func(any)
	envelopes = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			for key, child := range v {
				if key == "schema" {
					v[key] = schema(child)
				} else {
					envelopes(child)
				}
			}
		case []any:
			for _, child := range v {
				envelopes(child)
			}
		}
	}
	if paths, exists := root["paths"]; exists {
		envelopes(paths)
	}
	return json.Marshal(root)
}
