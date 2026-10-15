package schema

import "openapi-kt/internal/document"

// Null vocabulary members use already admitted nullable scalar codecs. They do
// not infer nullability or introduce a scalar literal or a copied vocabulary.
func (a *analyzer) checkedNativeEnumNulls(v *document.Value, pointer string) bool {
	if !a.options.Validation {
		return false
	}
	kind, nullable, err := schemaType(v, pointer)
	return err == nil && nullable && nullableNativeEnumCandidate(v, kind)
}
