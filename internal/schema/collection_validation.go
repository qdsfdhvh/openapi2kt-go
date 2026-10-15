package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// CollectionValidation checks a typed container, independently of its leaves.
// Provenance stays on the analyzer, not in semantic declaration equality.
type CollectionValidation struct {
	Minimum, Maximum       *int64
	MinimumKey, MaximumKey string
	Unique                 bool
	UniqueJSON             bool // Exact JSON-value keys; distinct from native scalar equality.
}

var collectionBoundKeys = []string{"minItems", "maxItems", "minProperties", "maxProperties", "uniqueItems"}

func (a *analyzer) collectionKeys(base []string) []string {
	if !a.options.Validation {
		return base
	}
	return append(base, collectionBoundKeys...)
}

func (a *analyzer) checkedCollection(ty TypeRef, v *document.Value, pointer string) (TypeRef, *diagnostic.Diagnostic) {
	var checks *CollectionValidation
	for _, key := range collectionBoundKeys {
		if !has(v, key) {
			continue
		}
		if !a.options.Validation {
			return TypeRef{}, unsupported(pointer + "/" + key)
		}
		items := key == "minItems" || key == "maxItems" || key == "uniqueItems"
		if items && ty.Kind != List || !items && ty.Kind != Map && ty.Kind != JSONObject {
			return TypeRef{}, unsupported(pointer + "/" + key)
		}
		if key == "uniqueItems" {
			unique, ok := get(v, key).Bool()
			if !ok {
				return TypeRef{}, invalid("uniqueItems must be Boolean.", pointer+"/"+key)
			}
			if !unique {
				continue // Validated false carries no enforcing metadata.
			}
			jsonValues := uniqueJSONType(*ty.Element) && a.uniqueJSONSource(get(v, "items"), pointer+"/items", 0)
			if !jsonValues && (!uniqueScalarType(*ty.Element) || !a.uniqueScalarSource(get(v, "items"), pointer+"/items", 0)) {
				return TypeRef{}, unsupported(pointer + "/" + key)
			}
			if checks == nil {
				checks = &CollectionValidation{}
			}
			checks.UniqueJSON = jsonValues
			checks.Unique = !jsonValues
			continue
		}
		count, ok := integerLiteral(get(v, key))
		if !ok || count < 0 || count > 2147483647 {
			return TypeRef{}, invalid("Collection count must be a nonnegative Int-range integer literal.", pointer+"/"+key)
		}
		if checks == nil {
			checks = &CollectionValidation{}
		}
		if key == "minItems" || key == "minProperties" {
			checks.Minimum = &count
			checks.MinimumKey = key
		} else {
			checks.Maximum = &count
			checks.MaximumKey = key
		}
	}
	if checks != nil {
		if checks.Minimum != nil && checks.Maximum != nil && *checks.Minimum > *checks.Maximum {
			return TypeRef{}, invalid("Minimum collection count exceeds maximum count.", pointer+"/"+checks.MaximumKey)
		}
		ty.Collection = checks
		a.collectionSources[checks] = pointer
	}
	return ty, nil
}

// Equality eligibility uses both the mapped type and original borrowed source.
// A native/enum/const projection must not acquire primitive equality merely
// because its Kotlin spelling happens to match. Only transparent scalar refs
// and the existing single scalar + null fold can retain this contract.
func uniqueScalarType(ty TypeRef) bool {
	if ty.Kind == Nullable {
		return uniqueScalarType(*ty.Element)
	}
	return ty.Kind == Scalar && ty.Serializer == "" &&
		(ty.Name == "kotlin.String" || ty.Name == "kotlin.Int" || ty.Name == "kotlin.Long" || ty.Name == "kotlin.Boolean")
}

func (a *analyzer) uniqueScalarSource(v *document.Value, pointer string, depth int) bool {
	if v == nil || v.Kind() != document.Object || depth > MaxSchemaDepth {
		return false
	}
	for _, key := range []string{"enum", "const", "x-native-type", "x-kotlin-value-class", "allOf"} {
		if has(v, key) {
			return false
		}
	}
	if has(v, "$ref") {
		c, err := a.resolveSchema(text(get(v, "$ref")), pointer+"/$ref")
		return err == nil && a.uniqueScalarSource(c.Schema, c.Pointer, depth+1)
	}
	if union, err := readUnion(v, pointer); err != nil {
		return false
	} else if union != nil {
		return len(union.Branches) == 1 && union.HasNull &&
			a.uniqueScalarSource(union.Branches[0].Schema, union.Branches[0].Pointer, depth+1)
	}
	kind, _, err := schemaType(v, pointer)
	if err != nil {
		return false
	}
	format := text(get(v, "format"))
	switch kind {
	case "string", "boolean":
		return format == ""
	case "integer":
		return format == "" || format == "int32" || format == "int64"
	}
	return false
}
