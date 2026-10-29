package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// Borrowing a metadata enum source may cross independently annotated component
// factories. Only an original marked component gets this permission; the value
// walker still validates the consumer annotation and never inherits a default.
func (a *analyzer) metadataEnumComponentDefault(v *document.Value, pointer string) bool {
	if !a.componentDefaultSchemas[v] || !a.componentRoot(v, pointer) {
		return false
	}
	if has(v, "$ref") {
		target := a.referenceWrapperDefaultTarget(&component{Schema: v, Pointer: pointer})
		return target != nil && metadataEnumDefaultTree(target.Schema)
	}
	return metadataEnumDefaultTree(v)
}

func metadataEnumDefaultTree(v *document.Value) bool {
	for depth := 0; depth <= MaxSchemaDepth; depth++ {
		kind, _, err := schemaType(v, "")
		if err != nil || has(v, "$ref") {
			return false
		}
		if kind == "array" && !has(v, "prefixItems") {
			v = get(v, "items")
		} else if dictionarySchema(v) {
			v = get(v, "additionalProperties")
		} else {
			return metadataStringEnumCandidate(v) && lexicalStringEnumSchema(v)
		}
	}
	return false
}

// Factory attachment is the one permitted change after the vocabulary snapshot.
// Preserve the captured members/name/origin and add only the already validated
// canonical annotation; final tuple validation still checks the full snapshot.
func (a *analyzer) recordMetadataEnumFactory(d *StringEnum, c *component, encoded string) *diagnostic.Diagnostic {
	if a.tupleObjects == nil {
		return nil
	}
	proof, captured := a.tupleObjects.UnionDeclarations[d]
	if !captured {
		return nil
	}
	value := get(c.Schema, "default")
	if value == nil || a.components[c.Wire] != c || !a.metadataEnumComponentDefault(c.Schema, c.Pointer) {
		return schemaReferenceUnsupported(c.Pointer)
	}
	canonical, failure := value.MarshalJSON()
	if failure != nil || string(canonical) != encoded {
		return schemaReferenceUnsupported(c.Pointer)
	}
	snapshot, ok := proof.Snapshot.(*StringEnum)
	if !ok || proof.Source != c.Schema || proof.Pointer != c.Pointer || proof.Origin != a.enumOrigins[d] || !tupleUnionDeclarationMatches(d, snapshot) {
		return schemaReferenceUnsupported(c.Pointer)
	}
	copy := *snapshot
	copy.ComponentDefault = encoded
	proof.Snapshot = &copy
	a.tupleObjects.UnionDeclarations[d] = proof
	return nil
}
