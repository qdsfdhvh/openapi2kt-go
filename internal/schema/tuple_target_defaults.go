package schema

import "openapi-kt/internal/document"

// Only the original component's independently admitted factory role permits
// its own annotation. A tuple source/consumer never inherits a target default,
// and a positional annotation cannot borrow permission from its referenced type.
// The ordinary component pass still validates every factory's literal/budgets.
func (a *analyzer) tupleSourceOwnDefault(v *document.Value, pointer string) bool {
	return a.componentDefaultSchemas[v] && a.componentRoot(v, pointer)
}

func (a *analyzer) tupleSourceKeys(v *document.Value, pointer string, keys []string) []string {
	if a.tupleSourceOwnDefault(v, pointer) {
		return append(keys, "default")
	}
	return keys
}
