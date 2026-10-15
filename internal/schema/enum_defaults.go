package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Field-only permission preserves strict root/collection/reference grammar.
func (a *analyzer) enumDefaultValue(v *document.Value, pointer string) *diagnostic.Diagnostic {
	at := naming.Pointer(pointer, "default")
	kind, nullable, err := schemaType(v, pointer)
	if err != nil {
		return err
	}
	for _, key := range []string{"$ref", "allOf", "oneOf", "anyOf", "const", "x-native-type", "x-kotlin-value-class"} {
		if has(v, key) {
			return unsupported(at)
		}
	}
	if (kind == "" || kind == "string") && has(v, "format") {
		return unsupported(at)
	}
	value := get(v, "default")
	if kind == "null" {
		if _, err := nullOnlyEnumProfile(v, pointer, true); err != nil {
			return err
		}
		if value.Kind() != document.Null {
			return invalid("A null-only enum default must be JSON null.", at)
		}
		return nil
	}
	if value.Kind() == document.Null {
		if !nullable {
			return invalid("A null default requires a nullable schema.", at)
		}
		return nil
	}
	return a.enumDefaultMemberAt(v, value, kind, pointer, at)
}

// Match each non-null default against the original vocabulary before mapped
// Float/Double rounding. Collection annotations supply their own member Pointer;
// this does not grant a default role to the element schema itself.
func (a *analyzer) enumDefaultMemberAt(v, value *document.Value, kind, pointer, at string) *diagnostic.Diagnostic {
	if kind != "" && kind != "string" {
		scalar, err := scalarMapping(kind, v, pointer)
		if err != nil {
			return err
		}
		if _, err := scalarLiteral(scalar, value, at); err != nil {
			return invalid("Default must be representable by the enum's mapped Kotlin scalar.", at)
		}
	}
	values := get(v, "enum")
	if values.Kind() != document.Array {
		return invalid("enum must be a non-empty array.", naming.Pointer(pointer, "enum"))
	}
	allowNull := a.checkedNativeEnumNulls(v, pointer)
	for _, member := range values.Items() {
		if member.Kind() == document.Null && allowNull {
			continue // Non-null defaults match the original scalar vocabulary.
		}
		if kind == "" || kind == "string" {
			left, lok := value.Text()
			right, rok := member.Text()
			if lok && rok && left == right {
				return nil
			}
		} else if kind == "boolean" {
			left, lok := value.Bool()
			right, rok := member.Bool()
			if lok && rok && left == right {
				return nil
			}
		} else {
			scalar, err := scalarMapping(kind, v, pointer)
			if err != nil {
				return err
			}
			_, left, err := constraintNumber(value, scalar, at)
			if err != nil {
				return err
			}
			_, right, err := constraintNumber(member, scalar, naming.Pointer(pointer, "enum"))
			if err != nil {
				return err
			}
			if left.Cmp(right) == 0 {
				return nil
			}
		}
	}
	return invalid("Default must be a member of the declared enum vocabulary.", at)
}

func (a *analyzer) enumDefaultType(f field, name string, nested *[]Declaration, used map[string]bool) (TypeRef, *diagnostic.Diagnostic) {
	if err := depthCheck(f.Depth, f.Pointer); err != nil {
		return TypeRef{}, err
	}
	if err := a.enumDefaultValue(f.Schema, f.Pointer); err != nil {
		return TypeRef{}, err
	}
	kind, nullable, err := schemaType(f.Schema, f.Pointer)
	if err != nil {
		return TypeRef{}, err
	}
	var ty TypeRef
	if kind == "null" {
		scalar, e := nullOnlyEnumProfile(f.Schema, f.Pointer, true)
		if e != nil {
			return TypeRef{}, e
		}
		ty = TypeRef{Kind: Scalar, Name: scalar}
	} else if kind == "" || kind == "string" {
		enum, e := buildStringEnumKeys(name, f.Schema, f.Pointer, true)
		if e != nil {
			return TypeRef{}, e
		}
		if nested != nil {
			if e := reserveNested(used, name, f.Pointer); e != nil {
				return TypeRef{}, e
			}
			a.recordEnumOrigin(enum, f.Schema, f.Pointer, nullable)
			*nested = append(*nested, enum)
		}
		ty = TypeRef{Kind: Named, Name: name}
	} else {
		scalar, e := a.nativeEnumKeys(kind, f.Schema, f.Pointer, true)
		if e != nil {
			return TypeRef{}, e
		}
		ty = TypeRef{Kind: Scalar, Name: scalar}
	}
	if nullable {
		ty = ty.Null()
	}
	return ty, nil
}
