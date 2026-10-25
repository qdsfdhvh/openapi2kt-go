package schema

import (
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// DTO and non-discriminator pinned body planning enable this profile. The literal's object
// keys are data, never schema keywords or references; its original tree stays
// borrowed and the JSON loader's independent container budget remains in force.
func structuredFieldConstant(name, wire string, schema, value *document.Value, kind, pointer string) (*Constant, *diagnostic.Diagnostic) {
	scalar, helper, failure := structuredConstantShape(schema, value, kind, pointer)
	if failure != nil {
		return nil, failure
	}
	encoded, err := value.MarshalJSON()
	if err != nil {
		return nil, invalid("Invalid structured const literal.", naming.Pointer(pointer, "const"))
	}
	return &Constant{Name: name, WireName: wire, Type: TypeRef{Kind: Scalar, Name: scalar}, Initializer: helper + "(" + naming.Literal(string(encoded)) + ")"}, nil
}

func structuredLiteral(v *document.Value) bool {
	value := get(v, "const")
	return has(v, "const") && (value.Kind() == document.Object || value.Kind() == document.Array)
}

// Value positions use typed JSON projection, with no initializer or membership
// barrier. Intrinsic nullable declarations retain a transparent nullable view;
// ordinary body constants still plan their separate non-null default.
func valueConstant(name, wire string, v *document.Value, pointer string, depth int) (*Constant, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, pointer); err != nil {
		return nil, err
	}
	if structuredLiteral(v) {
		_, nullable, err := schemaType(v, pointer)
		if err != nil {
			return nil, err
		}
		value, err := constantProfile(name, wire, v, pointer, depth, true)
		if err != nil {
			return nil, err
		}
		if nullable {
			value.Type = value.Type.Null()
		}
		return value, nil
	}
	return constant(name, wire, v, pointer, depth)
}

func (a *analyzer) warnValueConst(v *document.Value, pointer string) {
	if structuredLiteral(v) {
		a.warn("W_CONST_STRUCTURED_APPROXIMATION", "A structured const value retains its JSON kind but does not enforce literal membership or supply a value-position default.", naming.Pointer(pointer, "const"))
	} else {
		a.warnConst(pointer)
	}
}

func structuredConstBinding(c *component) bool {
	return c != nil && structuredLiteral(c.Schema) && (c.Scalar == "kotlinx.serialization.json.JsonObject" || c.Scalar == "kotlinx.serialization.json.JsonArray")
}

func structuredConstant(c Constant) bool {
	return c.Type.Kind == Scalar && (c.Type.Name == "kotlinx.serialization.json.JsonObject" && strings.HasPrefix(c.Initializer, "`openapi-kt object const`(") || c.Type.Name == "kotlinx.serialization.json.JsonArray" && strings.HasPrefix(c.Initializer, "`openapi-kt array const`("))
}

// The loader already validates the complete literal JSON tree and its budgets.
// This checks schema/type compatibility without encoding an unused expression.
func structuredConstantShape(schema, value *document.Value, kind, pointer string) (string, string, *diagnostic.Diagnostic) {
	for _, key := range []string{"format", "x-native-type"} {
		if has(schema, key) {
			return "", "", unsupported(naming.Pointer(pointer, key))
		}
	}
	actual, scalar, helper := "object", "kotlinx.serialization.json.JsonObject", "`openapi-kt object const`"
	if value.Kind() == document.Array {
		actual, scalar, helper = "array", "kotlinx.serialization.json.JsonArray", "`openapi-kt array const`"
	}
	if kind != "" && kind != actual {
		return "", "", invalid("A structured const must match its declared JSON type.", naming.Pointer(pointer, "const"))
	}
	return scalar, helper, nil
}

func validateDiscardedConstant(v *document.Value, pointer string, depth int) *diagnostic.Diagnostic {
	if !structuredLiteral(v) {
		_, err := constantProfile("validation", "field", v, pointer, depth, true)
		return err
	}
	kind, value, err := constantDeclaration(v, pointer, depth)
	if err != nil {
		return err
	}
	_, _, err = structuredConstantShape(v, value, kind, pointer)
	return err
}

func validateDiscardedStructuredValue(v *document.Value, pointer string, depth int) *diagnostic.Diagnostic {
	if err := depthCheck(depth, pointer); err != nil {
		return err
	}
	// Value projection checks nullable/type before const keys. Preserve its
	// diagnostic precedence independently of the body contribution grammar.
	if _, _, err := schemaType(v, pointer); err != nil {
		return err
	}
	return validateDiscardedConstant(v, pointer, depth)
}
