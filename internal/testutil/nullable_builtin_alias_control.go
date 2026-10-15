package testutil

import (
	"encoding/json"
	"fmt"
	"strings"
)

// NullableBuiltinAliasControl lowers the synthetic builtin-alias envelope to
// existing canonical scalar uses with explicit nearest nullable intent. It is
// test tooling, not admission or reference expansion in the generator.
func NullableBuiltinAliasControl(input []byte) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(input, &doc); err != nil {
		return nil, err
	}
	catalog := doc["components"].(map[string]any)["schemas"].(map[string]any)
	type binding struct {
		root     string
		nullable bool
	}
	cache := map[string]binding{}
	active := map[string]bool{}
	aliases := map[string]binding{}
	var resolve func(string) (binding, error)
	resolve = func(n string) (binding, error) {
		if b, ok := cache[n]; ok {
			return b, nil
		}
		if active[n] {
			return binding{}, fmt.Errorf("control alias cycle")
		}
		active[n] = true
		m, ok := catalog[n].(map[string]any)
		if !ok {
			return binding{}, fmt.Errorf("control missing target")
		}
		b := binding{root: n}
		if r, ok := m["$ref"].(string); ok {
			var err error
			b, err = resolve(strings.TrimPrefix(r, "#/components/schemas/"))
			if err != nil {
				return b, err
			}
		}
		if nullable, ok := m["nullable"].(bool); ok {
			b.nullable = nullable
		}
		active[n] = false
		cache[n] = b
		return b, nil
	}
	for n, v := range catalog {
		if _, ok := v.(map[string]any)["$ref"]; ok {
			b, err := resolve(n)
			if err != nil {
				return nil, err
			}
			aliases[n] = b
		}
	}
	for n := range aliases {
		delete(catalog, n)
	}
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			if r, ok := x["$ref"].(string); ok {
				if b, yes := aliases[strings.TrimPrefix(r, "#/components/schemas/")]; yes {
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
