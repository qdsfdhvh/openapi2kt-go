package schema

import (
	"slices"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

var metadata = []string{"title", "description", "example", "examples", "externalDocs", "$comment", "deprecated"}

func object(value *document.Value, pointer string) *diagnostic.Diagnostic {
	if value.Kind() != document.Object {
		return invalid("Schema must be an object.", pointer)
	}
	return nil
}

func get(v *document.Value, key string) *document.Value {
	value, _ := v.Get(key)
	return value
}

func has(v *document.Value, key string) bool {
	_, ok := v.Get(key)
	return ok
}

func text(v *document.Value) string {
	value, _ := v.Text()
	return value
}

func checkKeys(v *document.Value, allowed []string, pointer string) *diagnostic.Diagnostic {
	for _, key := range []string{"title", "description", "$comment"} {
		if value, ok := v.Get(key); ok && value.Kind() != document.String {
			return invalid("Schema text metadata must be a string.", naming.Pointer(pointer, key))
		}
	}
	if value, ok := v.Get("deprecated"); ok && value.Kind() != document.Boolean {
		return invalid("deprecated must be a boolean.", naming.Pointer(pointer, "deprecated"))
	}
	for _, member := range v.Members() {
		if !slices.Contains(metadata, member.Name) && !slices.Contains(allowed, member.Name) {
			return unsupported(naming.Pointer(pointer, member.Name))
		}
	}
	return nil
}

func schemaType(v *document.Value, pointer string) (string, bool, *diagnostic.Diagnostic) {
	nullable := false
	if value, ok := v.Get("nullable"); ok {
		flag, isBool := value.Bool()
		if !isBool {
			return "", false, invalid("nullable must be a boolean.", naming.Pointer(pointer, "nullable"))
		}
		nullable = flag
	}
	kind, typeNull := "", false
	if value, ok := v.Get("type"); ok {
		switch value.Kind() {
		case document.String:
			kind = text(value)
		case document.Array:
			names := map[string]bool{}
			for _, entry := range value.Items() {
				name, ok := entry.Text()
				if !ok {
					return "", false, invalid("type entries must be strings.", pointer)
				}
				if names[name] {
					return "", false, invalid("type entries must be unique.", pointer)
				}
				names[name] = true
			}
			typeNull = names["null"]
			delete(names, "null")
			if len(names) != 1 {
				return "", false, unsupported(naming.Pointer(pointer, "type"))
			}
			for name := range names { // exactly one element; no ordering decision
				kind = name
			}
		default:
			return "", false, invalid("type must be a string or a single-type-plus-null array.", pointer)
		}
	} else if has(v, "properties") || has(v, "additionalProperties") {
		kind = "object"
	} else if inferredNullLiteral(v) {
		kind = "null"
	}
	return kind, nullable || typeNull, nil
}

func scalarMapping(kind string, v *document.Value, pointer string) (string, *diagnostic.Diagnostic) {
	format := ""
	if value, ok := v.Get("format"); ok {
		var isText bool
		format, isText = value.Text()
		if !isText {
			return "", invalid("format must be a string.", pointer)
		}
	}
	if value, ok := v.Get("x-native-type"); ok {
		native, valid := value.Text()
		location := naming.Pointer(pointer, "x-native-type")
		if !valid || native == "" {
			return "", invalid("x-native-type must be nonempty text.", location)
		}
		switch native {
		case "date":
			return "kotlin.time.Instant", nil
		case "bigint":
			return "kotlin.Long", nil
		default:
			if !naming.ValidateNativeType(native) {
				return "", invalid("x-native-type must be a safe Kotlin type name.", location)
			}
			return native, nil
		}
	}
	switch {
	case (kind == "null" || kind == "x-void") && format == "":
		return "kotlinx.serialization.json.JsonNull", nil
	case kind == "string" && format == "date-time":
		return "kotlin.time.Instant", nil
	case kind == "string" && format == "date":
		return "kotlinx.datetime.LocalDate", nil
	case kind == "string" && (format == "byte" || format == "binary"):
		return "kotlin.ByteArray", nil
	case kind == "string" && (format == "" || format == "uuid"):
		return "kotlin.String", nil
	case kind == "integer" && (format == "" || format == "int32"):
		return "kotlin.Int", nil
	case kind == "integer" && format == "int64":
		return "kotlin.Long", nil
	case kind == "number" && (format == "" || format == "double"):
		return "kotlin.Double", nil
	case kind == "number" && format == "float":
		return "kotlin.Float", nil
	case kind == "boolean" && format == "":
		return "kotlin.Boolean", nil
	default:
		return "", unsupported(naming.Pointer(pointer, "format"))
	}
}

func scalarType(kind string, v *document.Value, pointer string) (string, *diagnostic.Diagnostic) {
	return scalarTypeFor(nil, kind, v, pointer)
}

// scalarTypeFor additionally admits a component root's own `default` when the
// caller's analyzer marked that original node for the factory planner.
func scalarTypeFor(a *analyzer, kind string, v *document.Value, pointer string) (string, *diagnostic.Diagnostic) {
	keys := []string{"type", "nullable", "format", "x-native-type"}
	if a != nil && a.componentDefaultSchemas[v] {
		keys = append(keys, "default")
	}
	if err := checkKeys(v, keys, pointer); err != nil {
		return "", err
	}
	return scalarMapping(kind, v, pointer)
}

// Semantic planners use one nominal identity/serializer for root object aliases.
// Binding graph construction and physical schema Pointer lookup use the original
// declared node instead, so neither aliases nor source locations are copied.
func (a *analyzer) resolve(reference, pointer string) (*component, *diagnostic.Diagnostic) {
	c, err := a.resolveDeclared(reference, pointer)
	if err != nil {
		return nil, err
	}
	if c.BindingTarget != nil {
		return c.BindingTarget, nil
	}
	return c, nil
}

func (a *analyzer) resolveDeclared(reference, pointer string) (*component, *diagnostic.Diagnostic) {
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(reference, prefix) {
		return nil, diagnostic.At("E_REF_UNSUPPORTED", "Only local named component references are supported.", pointer)
	}
	encoded := strings.TrimPrefix(reference, prefix)
	if encoded == "" || strings.ContainsAny(encoded, "/%") {
		return nil, diagnostic.At("E_REF_UNSUPPORTED", "Only local named component references are supported.", pointer)
	}
	wire, err := naming.DecodePointerSegment(encoded, pointer)
	if err != nil {
		return nil, err
	}
	component, ok := a.components[wire]
	if !ok {
		return nil, diagnostic.At("E_REF_MISSING", "Referenced component does not exist.", pointer)
	}
	return component, nil
}
