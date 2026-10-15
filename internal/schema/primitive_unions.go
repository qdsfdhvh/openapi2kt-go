package schema

import (
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

type PrimitiveKind uint8

const (
	PrimitiveString PrimitiveKind = iota
	PrimitiveInteger
	PrimitiveNumber
	PrimitiveBoolean
)

func (k PrimitiveKind) Prefix() string { return [...]string{"String", "Int", "Number", "Boolean"}[k] }
func (k PrimitiveKind) Scalar() string {
	return [...]string{"kotlin.String", "kotlin.Long", "kotlin.Double", "kotlin.Boolean"}[k]
}
func (k PrimitiveKind) Descriptor() string {
	return [...]string{"STRING", "LONG", "DOUBLE", "BOOLEAN"}[k]
}
func (k PrimitiveKind) Method() string { return [...]string{"String", "Long", "Double", "Boolean"}[k] }
func (k PrimitiveKind) Reader() string {
	return [...]string{"content", "longOrNull", "doubleOrNull", "booleanOrNull"}[k]
}

type PrimitiveUnion struct {
	Plain       bool
	ClassName   string
	Kinds       []PrimitiveKind
	Object      *Model
	NullVariant bool
}

func (m *PrimitiveUnion) Name() string { return m.ClassName }
func (*PrimitiveUnion) declaration()   {}

func (a *analyzer) resolveUnionBranch(branch unionBranch, depth int) (*document.Value, string, int, *diagnostic.Diagnostic) {
	v, p := branch.Schema, branch.Pointer
	if target, admitted, err := a.nullableObjectReferenceMember(branch, depth); admitted || err != nil {
		if err != nil {
			return nil, "", 0, err
		}
		return target.Schema, target.Pointer, depth + max(1, target.Steps), nil
	}
	kind, nullable, err := schemaType(v, p)
	if err != nil {
		return nil, "", 0, err
	}
	if nullable && !(kind == "object" && !has(v, "allOf") && !has(v, "oneOf") && !has(v, "anyOf") && !has(v, "$ref")) {
		key := "type"
		if b, ok := get(v, "nullable").Bool(); ok && b {
			key = "nullable"
		}
		return nil, "", 0, unsupported(naming.Pointer(p, key))
	}
	if target, ok := v.Get("$ref"); ok {
		if err := checkKeys(v, []string{"$ref", "nullable"}, p); err != nil {
			return nil, "", 0, err
		}
		reference, ok := target.Text()
		if !ok {
			return nil, "", 0, invalid("$ref must be a string.", p)
		}
		component, steps, err := a.resolveNominal(reference, p, depth)
		if err != nil {
			return nil, "", 0, err
		}
		v, p, depth = component.Schema, component.Pointer, depth+steps
	}
	if err := depthCheck(depth, p); err != nil {
		return nil, "", 0, err
	}
	return v, p, depth, nil
}

func (a *analyzer) primitiveProjection(kind string, v *document.Value, pointer string, depth int) (string, *diagnostic.Diagnostic) {
	pin := ""
	if has(v, "const") {
		if _, err := constant("validation", "value", v, pointer, depth); err != nil {
			return "", err
		}
		pin = "const"
	} else if has(v, "enum") {
		if kind == "string" {
			if _, err := buildStringEnum("Validation", v, pointer); err != nil {
				return "", err
			}
		} else {
			validation := *a
			validation.warnings = nil
			if _, err := validation.nativeEnum(kind, v, pointer); err != nil {
				return "", err
			}
		}
		pin = "enum"
	} else {
		return scalarType(kind, v, pointer)
	}
	scalar, err := scalarMapping(kind, v, pointer)
	if err != nil {
		return "", err
	}
	a.warn("W_PRIMITIVE_UNION_PIN_APPROXIMATION", "A primitive union pin is validated but its native payload serializer does not enforce membership.", naming.Pointer(pointer, pin))
	return scalar, nil
}

func (a *analyzer) primitiveUnion(union *unionView, name string, depth int, standalone bool, nested *[]Declaration, used map[string]bool) (TypeRef, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, union.Pointer); err != nil {
		return TypeRef{}, err
	}
	var kinds []PrimitiveKind
	var scalarNames, formats []string
	var formatPresent []bool
	hasPins := false
	var objectModel *Model
	for _, branch := range union.Branches {
		v, p, d, err := a.resolveUnionBranch(branch, depth)
		if err != nil {
			return TypeRef{}, err
		}
		kind, nullable, err := schemaType(v, p)
		if err != nil {
			return TypeRef{}, err
		}
		if nullable {
			return TypeRef{}, unsupported(p)
		}
		if kind == "" && has(v, "allOf") {
			kind = "object"
		}
		var primitive PrimitiveKind
		native := true
		switch kind {
		case "string":
			primitive = PrimitiveString
		case "integer":
			primitive = PrimitiveInteger
		case "number":
			primitive = PrimitiveNumber
		case "boolean":
			primitive = PrimitiveBoolean
		default:
			native = false
		}
		if native {
			scalar, err := a.primitiveProjection(kind, v, p, d)
			if err != nil {
				return TypeRef{}, err
			}
			hasPins = hasPins || has(v, "enum") || has(v, "const")
			scalarNames = append(scalarNames, scalar)
			formats = append(formats, text(get(v, "format")))
			formatPresent = append(formatPresent, has(v, "format"))
			kinds = append(kinds, primitive)
		} else if kind == "object" {
			if objectModel != nil {
				return TypeRef{}, unsupported(union.Pointer)
			}
			if err := rejectObjectCounts(v, p); err != nil {
				return TypeRef{}, err
			}
			objectModel, err = a.buildModel("Object"+name, v, p, d)
			if err != nil {
				return TypeRef{}, err
			}
			if objectModel.Nullable || len(objectModel.Properties) == 0 && len(objectModel.Constants) == 0 {
				return TypeRef{}, unsupported(union.Pointer)
			}
		} else {
			return TypeRef{}, unsupported(union.Pointer)
		}
	}
	if len(kinds) == 0 {
		return TypeRef{}, unsupported(union.Pointer)
	}
	same := true
	for i := range kinds {
		same = same && kinds[i] == kinds[0] && scalarNames[i] == scalarNames[0] && formats[i] == formats[0] && formatPresent[i] == formatPresent[0]
	}
	if !standalone && objectModel == nil && !hasPins && same {
		a.warn("W_PRIMITIVE_UNION_APPROXIMATION", "A same-kind union folds to its native scalar, not exact oneOf match-count validation.", union.Pointer)
		return TypeRef{Kind: Scalar, Name: scalarNames[0]}, nil
	}
	a.warn("W_PRIMITIVE_UNION_APPROXIMATION", "A primitive union uses JSON-kind dispatch (integer Long and number Double wrappers), not exact oneOf match-count or constraint validation.", union.Pointer)
	seen := map[PrimitiveKind]bool{}
	for _, kind := range kinds {
		if seen[kind] {
			return TypeRef{}, unsupported(union.Pointer)
		}
		seen[kind] = true
	}
	nullVariant := standalone && union.Nullable
	generated := []string{name + "Serializer"}
	if !standalone {
		generated = append(generated, name)
	}
	if nullVariant {
		generated = append(generated, "Null"+name, "Null"+name+"Serializer")
	}
	for _, kind := range kinds {
		variant := kind.Prefix() + name
		generated = append(generated, variant+"Serializer", variant)
	}
	if objectModel != nil {
		generated = append(generated, objectModel.ClassName)
	}
	for _, peer := range generated {
		if used[strings.ToLower(peer)] {
			return TypeRef{}, diagnostic.At("E_NAME_COLLISION", "Union-owned declarations collide with another declaration or an owner-reserved name.", union.Pointer)
		}
		used[strings.ToLower(peer)] = true
	}
	*nested = append(*nested, &PrimitiveUnion{ClassName: name, Kinds: kinds, Object: objectModel, NullVariant: nullVariant})
	return TypeRef{Kind: Named, Name: name}, nil
}

func (a *analyzer) rootUnion(v *document.Value, name, pointer string, used map[string]bool) (Declaration, *diagnostic.Diagnostic) {
	if has(v, "discriminator") {
		return nil, unsupported(pointer)
	}
	union, err := readUnion(v, pointer)
	if err != nil {
		return nil, err
	}
	if err := a.deriveObjectReferenceNullability(union, 1); err != nil {
		return nil, err
	}
	count := len(union.Branches)
	if union.Nullable {
		count++
	}
	if count > MaxMembers {
		return nil, diagnostic.At("E_UNION_LIMIT", "Root union exceeds the supported 240 branch limit, including its null variant.", union.Pointer)
	}
	if union.OverlappingNulls {
		a.warn("W_ONEOF_NULL_OVERLAP", "Root null dispatch does not enforce exact oneOf match count for overlapping null branches.", union.Pointer)
	}
	model, err := a.objectUnion(union, name, 1, true)
	if err != nil {
		return nil, err
	}
	if model != nil {
		if err := reserveObjectUnion(model, union, pointer, true, used); err != nil {
			return nil, err
		}
		a.recordUnionOrigin(model, v, pointer, false)
		return model, nil
	}
	var declarations []Declaration
	if _, err := a.primitiveUnion(union, name, 2, true, &declarations, used); err != nil {
		return nil, err
	}
	if len(declarations) != 1 {
		panic("primitive declaration invariant")
	}
	a.recordUnionOrigin(declarations[0], v, pointer, false)
	return declarations[0], nil
}

// jsonPin returns the original decoder's canonical scalar spelling for a wire
// dispatch tag. It is not Kotlin literal escaping or a membership validator.
func jsonPin(v *document.Value) string {
	if value, ok := v.Text(); ok {
		return value
	}
	if value, ok := v.Bool(); ok {
		return strconv.FormatBool(value)
	}
	if number, ok := v.Numeric(); ok {
		identity := nativeIdentity(v)
		if strings.HasPrefix(identity, "integer:") {
			return string(number)
		}
		n, _ := number.Float64()
		// The accepted decoder displays f64 in decimal for exponents -5..15,
		// retains .0 on integral floats, and spells scientific exponents
		// with an explicit sign and no leading zero. Go's default 'g'
		// threshold is different and would change tag/variant names.
		scientific := strconv.FormatFloat(n, 'e', -1, 64)
		parts := strings.Split(scientific, "e")
		exponent, _ := strconv.Atoi(parts[1])
		if exponent >= -5 && exponent <= 15 {
			out := strconv.FormatFloat(n, 'f', -1, 64)
			if !strings.Contains(out, ".") {
				out += ".0"
			}
			return out
		}
		sign := "+"
		if exponent < 0 {
			sign = "-"
			exponent = -exponent
		}
		return parts[0] + "e" + sign + strconv.Itoa(exponent)
	}
	if v.Kind() == document.Null {
		return "null"
	}
	data, _ := v.MarshalJSON()
	return string(data)
}
