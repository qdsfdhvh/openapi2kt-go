package schema

import "openapi-kt/internal/document"

// Detect only the existing ordinary DTO/allOf leaf role along a finite inline
// container path. The broader object-default detector also admits union leaves;
// those component factories keep their independent source contract.
func (a *analyzer) objectCollectionComponentRoot(v *document.Value) bool {
	kind, _, err := schemaType(v, "")
	if err != nil || kind != "array" && !dictionarySchema(v) {
		return false
	}
	if inlineAllOfDefaultElement(v) != nil {
		return true
	}
	for edges := 0; edges <= MaxSchemaDepth && v != nil; edges++ {
		if has(v, "$ref") {
			// Initial marking precedes the catalog. Borrow only a direct
			// original named DTO source; the full typed walk validates it later.
			_, target, known := a.componentSchemaAtReference(text(get(v, "$ref")))
			return known && objectDefaultCandidateSchema(target)
		}
		kind, _, err := schemaType(v, "")
		if err != nil {
			return false
		}
		switch {
		case kind == "array":
			v = get(v, "items")
		case dictionarySchema(v):
			v = get(v, "additionalProperties")
		default:
			return objectDefaultCandidateSchema(v)
		}
	}
	return false
}
