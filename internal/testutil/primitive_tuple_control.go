package testutil

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// PrimitiveTupleControl lowers only the frozen synthetic prefix tuple profile
// for before-binary controls. It is not production schema normalization or a
// source-depth/Pointer equivalence guarantee.
func PrimitiveTupleControl(input []byte) ([]byte, error) {
	var root any
	if err := json.Unmarshal(input, &root); err != nil {
		return nil, err
	}
	var lower func(any) error
	lower = func(value any) error {
		switch value := value.(type) {
		case map[string]any:
			if prefix, exists := value["prefixItems"]; exists {
				members, ok := prefix.([]any)
				if !ok || len(members) == 0 {
					return fmt.Errorf("control requires nonempty prefixItems")
				}
				if _, exists := value["items"]; !exists {
					uniform := true
					for _, member := range members[1:] {
						uniform = uniform && reflect.DeepEqual(member, members[0])
					}
					if uniform {
						value["items"] = members[0]
					} else {
						value["items"] = map[string]any{}
					}
				}
				delete(value, "prefixItems")
			}
			for _, child := range value {
				if err := lower(child); err != nil {
					return err
				}
			}
		case []any:
			for _, child := range value {
				if err := lower(child); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := lower(root); err != nil {
		return nil, err
	}
	return json.Marshal(root)
}
