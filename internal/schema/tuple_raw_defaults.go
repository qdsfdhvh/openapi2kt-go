package schema

import (
	"encoding/json"
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Permission comes from the shared tuple decision after every original member
// has validated. The private raw leaf is a value walk, never a synthetic schema
// granting reference, positional-member or ordinary collection permissions.
func (a *analyzer) rawTupleDefaultNode(v *document.Value, pointer string, decision *arrayItemsDecision) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	if decision == nil || decision.Projection == nil || decision.Projection.Kind != JSON || decision.Selected != nil || decision.Pointer != naming.Pointer(pointer, "prefixItems") {
		return nil, unsupported(naming.Pointer(pointer, "default"))
	}
	element := &nestedDefaultNode{schema: v, pointer: decision.Pointer, ty: *decision.Projection, rawJSON: true}
	ty, err := a.checkedCollection(TypeRef{Kind: List, Element: &element.ty}, v, pointer)
	if err != nil {
		return nil, err
	}
	_, nullable, err := schemaType(v, pointer)
	if err != nil {
		return nil, err
	}
	if nullable {
		ty = ty.Null()
	}
	return &nestedDefaultNode{schema: v, pointer: pointer, ty: ty, element: element}, nil
}

func (a *analyzer) rawTupleFieldDefault(v *document.Value, pointer string, decision *arrayItemsDecision) (TypeRef, string, *diagnostic.Diagnostic) {
	node, err := a.rawTupleDefaultNode(v, pointer, decision)
	if err != nil {
		return TypeRef{}, "", err
	}
	value, at := get(v, "default"), naming.Pointer(pointer, "default")
	work, bytes := 0, 0
	if err := nestedDefaultBudget(value, at, 0, &work, &bytes); err != nil {
		return TypeRef{}, "", err
	}
	encoded, err := node.value(value, at)
	if err != nil {
		return TypeRef{}, "", err
	}
	if value.Kind() == document.Null {
		return node.ty, "null", nil
	}
	if len(encoded) > maxNestedDefaultBytes {
		return TypeRef{}, "", diagnostic.At("E_MODEL_LIMIT", "Canonical tuple field default exceeds the supported 32 KiB JSON byte limit.", at)
	}
	return node.ty, "`openapi-kt default array JsonElement`(" + naming.Literal(encoded) + ")", nil
}

// The runtime JSON uniqueness contract compares exact decimal values without
// expanding exponents. Own annotations use the same numeric limits and kind
// distinctions before emission; object member order has no equality meaning.
func rawJSONDefaultKey(v *document.Value, pointer string) (string, *diagnostic.Diagnostic) {
	value, err := rawJSONDefaultKeyValue(v, pointer)
	if err != nil {
		return "", err
	}
	encoded, _ := json.Marshal(value)
	return string(encoded), nil
}

func rawJSONDefaultKeyValue(v *document.Value, pointer string) (any, *diagnostic.Diagnostic) {
	switch v.Kind() {
	case document.Null:
		return []any{"null"}, nil
	case document.Boolean:
		value, _ := v.Bool()
		return []any{"boolean", value}, nil
	case document.String:
		value, _ := v.Text()
		return []any{"string", value}, nil
	case document.Number:
		number, _ := v.Numeric()
		raw := string(number)
		if len(raw) > 256 {
			return nil, invalid("JSON default number exceeds supported uniqueItems precision.", pointer)
		}
		mantissa, exponent := raw, 0
		if index := strings.IndexAny(raw, "eE"); index >= 0 {
			var err error
			exponent, err = strconv.Atoi(raw[index+1:])
			if err != nil || exponent < -512 || exponent > 512 {
				return nil, invalid("JSON default number exceeds supported uniqueItems exponent.", pointer)
			}
			mantissa = raw[:index]
		}
		negative := strings.HasPrefix(mantissa, "-")
		mantissa = strings.TrimPrefix(mantissa, "-")
		fraction := 0
		if index := strings.IndexByte(mantissa, '.'); index >= 0 {
			fraction = len(mantissa) - index - 1
		}
		digits := strings.TrimLeft(strings.ReplaceAll(mantissa, ".", ""), "0")
		normalized := "0"
		if digits != "" {
			significant := strings.TrimRight(digits, "0")
			normalized = significant + "e" + strconv.Itoa(exponent-fraction+len(digits)-len(significant))
			if negative {
				normalized = "-" + normalized
			}
		}
		return []any{"number", normalized}, nil
	case document.Array:
		members := make([]any, 0, len(v.Items()))
		for i, member := range v.Items() {
			value, err := rawJSONDefaultKeyValue(member, naming.Pointer(pointer, strconv.Itoa(i)))
			if err != nil {
				return nil, err
			}
			members = append(members, value)
		}
		return []any{"array", members}, nil
	case document.Object:
		members := make(map[string]any, len(v.Members()))
		for _, member := range v.Members() {
			value, err := rawJSONDefaultKeyValue(member.Value, naming.Pointer(pointer, member.Name))
			if err != nil {
				return nil, err
			}
			members[member.Name] = value
		}
		return []any{"object", members}, nil
	}
	return nil, invalid("A JSON default must be a valid JSON value.", pointer)
}
