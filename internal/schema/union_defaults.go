package schema

import (
	"math"
	"slices"
	"strconv"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Inline primitive union field own defaults. The role is deliberately narrow:
// only a field whose source `oneOf`/`anyOf` is composed of plain String,
// Integer, Number or Boolean branches (no pins, formats, objects, references or
// per-branch nullability) and whose plan produces the native primitive-union
// declaration. The initializer decodes through the final actual union
// serializer, so the emitted Kotlin call site and wire behavior are unchanged.
func (a *analyzer) primitiveUnionDefaultCandidate(v *document.Value) bool {
	keyword := ""
	if has(v, "oneOf") {
		keyword = "oneOf"
	} else if has(v, "anyOf") {
		keyword = "anyOf"
	} else {
		return false
	}
	if !has(v, "default") {
		return false
	}
	branches, ok := v.Get(keyword)
	if !ok || branches.Kind() != document.Array || len(branches.Items()) == 0 {
		return false
	}
	for i, branch := range branches.Items() {
		if branch.Kind() != document.Object {
			return false
		}
		p := naming.Pointer(naming.Pointer("", keyword), strconv.Itoa(i))
		if text(get(branch, "type")) == "null" || inferredNullLiteral(branch) {
			continue
		}
		kind, nullable, err := schemaType(branch, p)
		if err != nil || nullable {
			return false
		}
		if kind != "string" && kind != "integer" && kind != "number" && kind != "boolean" {
			return false
		}
		if has(branch, "enum") || has(branch, "const") || kind == "string" && has(branch, "format") {
			return false
		}
	}
	return true
}

func (a *analyzer) primitiveUnionDefaultField(f field, name string, nested *[]Declaration, used map[string]bool) (TypeRef, *diagnostic.Diagnostic) {
	if a.objectDefaultSchemas == nil {
		a.objectDefaultSchemas = map[*document.Value]bool{}
	}
	a.objectDefaultSchemas[f.Schema] = true
	before := len(*nested)
	ty, _, err := a.modelPropertyType(f.Schema, name, f.Pointer, nested, used, f.Depth)
	if err != nil {
		return TypeRef{}, err
	}
	union, ok := a.primitiveUnionDeclaration(f, nested, before)
	if !ok {
		return TypeRef{}, unsupported(naming.Pointer(f.Pointer, "default"))
	}
	validation := *a
	validation.warnings = nil
	if _, err := validation.primitiveUnionDefaultEncoded(f, union); err != nil {
		return TypeRef{}, err
	}
	return ty, nil
}

// primitiveUnionDeclaration extracts exactly the declaration the shared plan
// appended for this field and rejects folded scalars, object unions and nested
// owners.
func (a *analyzer) primitiveUnionDeclaration(f field, nested *[]Declaration, before int) (*PrimitiveUnion, bool) {
	if nested == nil || len(*nested) != before+1 {
		return nil, false
	}
	union, ok := (*nested)[before].(*PrimitiveUnion)
	if !ok || union.Object != nil || union.NullVariant {
		return nil, false
	}
	return union, true
}

// primitiveUnionDefaultInitializer revalidates on a scratch plan so it never
// appends declarations twice. An explicit null stays the literal null.
func (a *analyzer) primitiveUnionDefaultInitializer(f field) (string, *diagnostic.Diagnostic) {
	var scratch []Declaration
	_, _, err := a.modelPropertyType(f.Schema, "Validation", f.Pointer, &scratch, map[string]bool{}, f.Depth)
	if err != nil {
		return "", err
	}
	union, ok := a.primitiveUnionDeclaration(f, &scratch, 0)
	if !ok {
		return "", unsupported(naming.Pointer(f.Pointer, "default"))
	}
	encoded, err := a.primitiveUnionDefaultEncoded(f, union)
	if err != nil {
		return "", err
	}
	if encoded == "null" {
		return "null", nil
	}
	return "`openapi-kt default model`(" + naming.Literal(encoded) + ")", nil
}

func (a *analyzer) primitiveUnionDefaultEncoded(f field, union *PrimitiveUnion) (string, *diagnostic.Diagnostic) {
	value := get(f.Schema, "default")
	at := naming.Pointer(f.Pointer, "default")
	if value.Kind() == document.Null {
		view, err := a.readUnion(f.Schema, f.Pointer)
		if err != nil {
			return "", err
		}
		if view == nil || !view.Nullable {
			return "", invalid("A null default requires a nullable union.", at)
		}
		return "null", nil
	}
	return a.primitiveUnionDefaultValue(f, union)
}

// primitiveUnionDefaultValue mirrors the emitted selector order (String,
// Boolean, Integer, Number) and preserves the original JSON token, because the
// selector's wrapper choice depends on the token kind.
func (a *analyzer) primitiveUnionDefaultValue(f field, union *PrimitiveUnion) (string, *diagnostic.Diagnostic) {
	value := get(f.Schema, "default")
	at := naming.Pointer(f.Pointer, "default")
	has := func(kind PrimitiveKind) bool { return slices.Contains(union.Kinds, kind) }
	switch value.Kind() {
	case document.String:
		if !has(PrimitiveString) {
			return "", invalid("Default must match an accepted union branch.", at)
		}
	case document.Boolean:
		if !has(PrimitiveBoolean) {
			return "", invalid("Default must match an accepted union branch.", at)
		}
	case document.Number:
		if has(PrimitiveInteger) && unionIntegerToken(value) {
			break
		}
		if has(PrimitiveNumber) && unionNumberToken(value) {
			break
		}
		return "", invalid("Default must match an accepted union branch.", at)
	default:
		return "", invalid("Default must be a JSON primitive matching an accepted union branch.", at)
	}
	encoded, marshalErr := value.MarshalJSON()
	if marshalErr != nil {
		return "", invalid("Default must be valid JSON.", at)
	}
	return string(encoded), nil
}

func unionIntegerToken(value *document.Value) bool {
	n, ok := value.Numeric()
	if !ok {
		return false
	}
	_, err := n.Int64()
	return err == nil
}

func unionNumberToken(value *document.Value) bool {
	n, ok := value.Numeric()
	if !ok {
		return false
	}
	f, err := n.Float64()
	return err == nil && !math.IsInf(f, 0) && !math.IsNaN(f)
}
