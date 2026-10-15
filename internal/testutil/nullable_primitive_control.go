package testutil

import (
	"encoding/json"
	"strings"
)

// NullablePrimitiveControl removes only the new intrinsic root feature. Pure
// root aliases are lowered at their use sites with explicit inherited intent;
// the existing default-off scalar-root-ref exclusion stays unchanged.
func NullablePrimitiveControl(input []byte) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(input, &doc); err != nil {
		return nil, err
	}
	catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
	type binding struct {
		root     string
		nullable bool
	}
	bindings := map[string]binding{}
	for n, v := range catalog {
		m := v.(map[string]any)
		if _, alias := m["$ref"]; alias {
			continue
		}
		if value, ok := m["nullable"].(bool); ok && value {
			bindings[n] = binding{n, true}
			delete(m, "nullable")
		}
		if a, ok := m["type"].([]any); ok {
			bindings[n] = binding{n, true}
			m["type"] = a[0]
		}
	}
	var resolve func(string) binding
	resolve = func(n string) binding {
		if b, ok := bindings[n]; ok {
			return b
		}
		m := catalog[n].(map[string]any)
		b := resolve(strings.TrimPrefix(m["$ref"].(string), "#/components/schemas/"))
		if nullable, ok := m["nullable"].(bool); ok {
			b.nullable = nullable
		}
		bindings[n] = b
		return b
	}
	var aliases []string
	for n, v := range catalog {
		if _, ok := v.(map[string]any)["$ref"]; ok {
			resolve(n)
			aliases = append(aliases, n)
		}
	}
	for _, n := range aliases {
		delete(catalog, n)
	}
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			if r, ok := x["$ref"].(string); ok {
				if b, yes := bindings[strings.TrimPrefix(r, "#/components/schemas/")]; yes {
					x["$ref"] = "#/components/schemas/" + b.root
					if _, has := x["nullable"]; !has {
						x["nullable"] = b.nullable
					}
				}
			}
			for _, child := range x {
				walk(child)
			}
		case []any:
			for _, child := range x {
				walk(child)
			}
		}
	}
	walk(doc)
	return json.Marshal(doc)
}
