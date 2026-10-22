package schema

import (
	"strconv"
	"time"

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
			enumValues := uniqueStringEnumType(*ty.Element) && a.uniqueStringEnumSource(get(v, "items"), pointer+"/items", 0)
			nativeEnumValues := uniqueNativeEnumType(*ty.Element) && a.uniqueNativeEnumSource(get(v, "items"), pointer+"/items", 0)
			mapValues := uniqueMapType(*ty.Element) && a.uniqueMapSource(get(v, "items"), pointer+"/items")
			if !jsonValues && !enumValues && !nativeEnumValues && !mapValues && (!uniqueScalarType(*ty.Element) || !a.uniqueScalarSource(get(v, "items"), pointer+"/items", 0)) {
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

// Only actual nominal String enum leaves use serialName equality. The original
// vocabulary, not a coincident Named spelling or projected String type,
// authorizes this role. Folded vocabularies remain separate.
func uniqueStringEnumType(ty TypeRef) bool {
	ty = bare(ty)
	return ty.Kind == Named && ty.Element == nil && ty.Serializer == ""
}

func (a *analyzer) uniqueStringEnumSource(v *document.Value, pointer string, depth int) bool {
	if v == nil || v.Kind() != document.Object || depth > MaxSchemaDepth {
		return false
	}
	if has(v, "$ref") {
		// The typed planner already admitted this path. Retain the same
		// original vocabulary/declaration authority as borrowed enum leaves;
		// neither a Named spelling nor a flattened copy grants eligibility.
		if checkKeys(v, []string{"$ref", "nullable"}, pointer) != nil {
			return false
		}
		target, err := a.resolveSchema(text(get(v, "$ref")), pointer+"/$ref")
		if err != nil {
			return false
		}
		if has(target.Schema, "$ref") {
			return a.uniqueStringEnumSource(target.Schema, target.Pointer, depth+1)
		}
		return enumNominalCandidate(target.Schema) && a.admitBorrowedNominalLeaf(target, pointer+"/$ref")
	}
	if !plainStringEnumSchema(v) {
		return false
	}
	kind, _, err := schemaType(v, pointer)
	return err == nil && (kind == "" || kind == "string")
}

// Native enum uniqueness keeps the existing mapped membership plan. Scalar
// spelling alone cannot authorize enum equality on an erased projection.
func uniqueNativeEnumType(ty TypeRef) bool {
	ty = bare(ty)
	return ty.Kind == Scalar && ty.Serializer == "" && ty.Validation != nil && len(ty.Validation.Enum) != 0 &&
		(ty.Name == "kotlin.Int" || ty.Name == "kotlin.Long" || ty.Name == "kotlin.Boolean" || ty.Name == "kotlin.Float" || ty.Name == "kotlin.Double")
}

func (a *analyzer) uniqueNativeEnumSource(v *document.Value, pointer string, depth int) bool {
	if v == nil || v.Kind() != document.Object || depth > MaxSchemaDepth {
		return false
	}
	for _, key := range []string{"const", "x-native-type", "x-kotlin-value-class", "allOf"} {
		if has(v, key) {
			return false
		}
	}
	if has(v, "$ref") {
		target, err := a.resolveSchema(text(get(v, "$ref")), pointer+"/$ref")
		return err == nil && a.uniqueNativeEnumSource(target.Schema, target.Pointer, depth+1)
	}
	if union, err := a.readUnion(v, pointer); err != nil {
		return false
	} else if union != nil {
		return len(union.Branches) == 1 && union.HasNull &&
			a.uniqueNativeEnumSource(union.Branches[0].Schema, union.Branches[0].Pointer, depth+1)
	}
	kind, _, err := schemaType(v, pointer)
	format := text(get(v, "format"))
	return err == nil && has(v, "enum") && (kind == "boolean" && format == "" ||
		kind == "integer" && (format == "" || format == "int32" || format == "int64") ||
		kind == "number" && (format == "" || format == "float" || format == "double"))
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
		(ty.Name == "kotlin.String" || ty.Name == "kotlin.uuid.Uuid" || ty.Name == "kotlin.Int" || ty.Name == "kotlin.Long" || ty.Name == "kotlin.Boolean" || ty.Name == "kotlin.Float" || ty.Name == "kotlin.Double" || ty.Name == "kotlin.time.Instant" || ty.Name == "kotlinx.datetime.LocalDate")
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
	if union, err := a.readUnion(v, pointer); err != nil {
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
	case "string":
		return format == "" || format == "uuid" || format == "date" || format == "date-time"
	case "boolean":
		return format == ""
	case "integer":
		return format == "" || format == "int32" || format == "int64"
	case "number":
		return format == "" || format == "float" || format == "double"
	}
	return false
}

// Default equality follows mapped values while emitted JSON keeps its original
// spelling. UUID spelling variants and equivalent Instant offsets/fractions
// cannot disguise duplicates. Callers validate each original literal separately.
func uniqueScalarDefaultKey(ty TypeRef, v *document.Value) string {
	encoded, _ := v.MarshalJSON()
	switch bare(ty).Name {
	case "kotlin.uuid.Uuid":
		if value, ok := canonicalUUID(text(v)); ok {
			return value
		}
	case "kotlin.Float", "kotlin.Double":
		if number, ok := floatLiteral(v); ok {
			bits := 64
			if bare(ty).Name == "kotlin.Float" {
				number, bits = float64(float32(number)), 32
			}
			if number == 0 {
				number = 0 // Signed zeros compare equally in typed floating consumers.
			}
			return strconv.FormatFloat(number, 'g', -1, bits)
		}
	case "kotlin.Int", "kotlin.Long":
		if number, ok := integerLiteral(v); ok {
			return strconv.FormatInt(number, 10)
		}
	case "kotlin.time.Instant":
		if value, err := time.Parse(time.RFC3339Nano, text(v)); err == nil {
			return value.UTC().Format(time.RFC3339Nano)
		}
	}
	return string(encoded)
}
