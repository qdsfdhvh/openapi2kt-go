package schema

import (
	"math"
	"strconv"
	"strings"
	"time"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

func integerLiteral(v *document.Value) (int64, bool) {
	n, ok := v.Numeric()
	if !ok || strings.ContainsAny(string(n), ".eE") || string(n) == "-0" {
		return 0, false
	}
	value, err := n.Int64()
	return value, err == nil
}

func floatLiteral(v *document.Value) (float64, bool) {
	n, ok := v.Numeric()
	if !ok {
		return 0, false
	}
	value, err := n.Float64()
	return value, err == nil && !math.IsInf(value, 0) && !math.IsNaN(value)
}

func scalarLiteral(scalar string, value *document.Value, pointer string) (string, *diagnostic.Diagnostic) {
	bad := func() (string, *diagnostic.Diagnostic) {
		return "", invalid("const must be representable by its mapped Kotlin scalar type.", pointer)
	}
	switch scalar {
	case "kotlin.String":
		literal, ok := value.Text()
		if !ok {
			return bad()
		}
		return naming.Literal(literal), nil
	case "kotlin.uuid.Uuid":
		literal, ok := value.Text()
		if !ok {
			return bad()
		}
		canonical, ok := canonicalUUID(literal)
		if !ok {
			return bad()
		}
		return scalar + ".parse(" + naming.Literal(canonical) + ")", nil
	case "kotlin.time.Instant", "kotlinx.datetime.LocalDate":
		literal, ok := value.Text()
		if !ok {
			return bad()
		}
		layout := time.RFC3339Nano
		if scalar == "kotlinx.datetime.LocalDate" {
			layout = "2006-01-02"
		}
		if _, err := time.Parse(layout, literal); err != nil {
			return bad()
		}
		return scalar + ".parse(" + naming.Literal(literal) + ")", nil
	case "kotlin.Boolean":
		literal, ok := value.Bool()
		if !ok {
			return bad()
		}
		return strconv.FormatBool(literal), nil
	case "kotlin.Int", "kotlin.Long":
		literal, ok := integerLiteral(value)
		if !ok {
			return bad()
		}
		if scalar == "kotlin.Int" {
			if literal < math.MinInt32 || literal > math.MaxInt32 {
				return bad()
			}
			if literal == math.MinInt32 {
				return "kotlin.Int.MIN_VALUE", nil
			}
			return strconv.FormatInt(literal, 10), nil
		}
		if literal == math.MinInt64 {
			return "kotlin.Long.MIN_VALUE", nil
		}
		return strconv.FormatInt(literal, 10) + "L", nil
	case "kotlin.Double", "kotlin.Float":
		literal, ok := floatLiteral(value)
		if !ok {
			return bad()
		}
		bits := 64
		if scalar == "kotlin.Float" {
			literal = float64(float32(literal))
			if math.IsInf(literal, 0) {
				return bad()
			}
			bits = 32
		}
		out := strconv.FormatFloat(literal, 'f', -1, bits)
		if !strings.ContainsAny(out, ".eE") {
			out += ".0"
		}
		if bits == 32 {
			out += "f"
		}
		return out, nil
	default:
		return "", unsupported(pointer)
	}
}

func constant(name, wire string, v *document.Value, pointer string, depth int) (*Constant, *diagnostic.Diagnostic) {
	return constantProfile(name, wire, v, pointer, depth, false)
}

// constantDeclaration validates source syntax before any initializer is built.
// Its ordering also governs discarded body contributions.
func constantDeclaration(v *document.Value, pointer string, depth int) (string, *document.Value, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, pointer); err != nil {
		return "", nil, err
	}
	if err := checkKeys(v, []string{"type", "nullable", "format", "const", "x-native-type"}, pointer); err != nil {
		return "", nil, err
	}
	kind, _, err := schemaType(v, pointer)
	if err != nil {
		return "", nil, err
	}
	return kind, get(v, "const"), nil
}

func constantProfile(name, wire string, v *document.Value, pointer string, depth int, structuredField bool) (*Constant, *diagnostic.Diagnostic) {
	kind, value, err := constantDeclaration(v, pointer, depth)
	if err != nil {
		return nil, err
	}
	location := naming.Pointer(pointer, "const")
	if structuredField && (value.Kind() == document.Object || value.Kind() == document.Array) {
		return structuredFieldConstant(name, wire, v, value, kind, pointer)
	}
	if kind == "" && has(v, "x-native-type") {
		kind = "native"
	}
	if kind == "" {
		if has(v, "format") {
			return nil, unsupported(naming.Pointer(pointer, "format"))
		}
		switch value.Kind() {
		case document.String:
			token := strings.TrimSpace(text(value))
			_, numeric := strconv.ParseFloat(token, 64)
			_, withSuffix := strconv.ParseFloat(strings.TrimRight(token, "dDfF"), 64)
			if token == "true" || token == "false" || numeric == nil || withSuffix == nil || strings.HasPrefix(strings.ToLower(strings.TrimLeft(token, "+-")), "0x") {
				return nil, unsupported(location)
			}
			kind = "string"
		case document.Boolean:
			kind = "boolean"
		case document.Number:
			if n, ok := integerLiteral(value); ok && n >= math.MinInt32 && n <= math.MaxInt32 {
				kind = "integer"
			} else if token, _ := value.Numeric(); strings.ContainsAny(string(token), ".eE") || string(token) == "-0" {
				kind = "number"
			} else {
				// Values outside u64/i64 are floating in the original decoder.
				if _, e := strconv.ParseUint(string(token), 10, 64); e == nil {
					return nil, unsupported(location)
				}
				if _, e := strconv.ParseInt(string(token), 10, 64); e == nil {
					return nil, unsupported(location)
				}
				kind = "number"
			}
		default:
			return nil, unsupported(location)
		}
	}
	if kind == "null" && value.Kind() == document.Null && !has(v, "format") && !has(v, "x-native-type") {
		return &Constant{Name: name, WireName: wire, Type: TypeRef{Kind: Scalar, Name: "kotlinx.serialization.json.JsonNull"}, Initializer: "kotlinx.serialization.json.JsonNull"}, nil
	}
	if kind != "string" && kind != "integer" && kind != "number" && kind != "boolean" && kind != "native" {
		return nil, unsupported(naming.Pointer(pointer, "type"))
	}
	if value.Kind() == document.Null || value.Kind() == document.Array || value.Kind() == document.Object {
		return nil, unsupported(location)
	}
	scalar, err := scalarMapping(kind, v, pointer)
	if err != nil {
		return nil, err
	}
	var initializer string
	if scalar == "kotlin.Long" && text(get(v, "x-native-type")) == "bigint" && value.Kind() == document.String {
		literal, parseErr := strconv.ParseInt(text(value), 10, 64)
		if parseErr != nil {
			return nil, invalid("const must be representable by its mapped Kotlin scalar type.", location)
		}
		if literal == math.MinInt64 {
			initializer = "kotlin.Long.MIN_VALUE"
		} else {
			initializer = strconv.FormatInt(literal, 10) + "L"
		}
	} else {
		initializer, err = scalarLiteral(scalar, value, location)
	}
	if err != nil {
		return nil, err
	}
	if scalar == "kotlin.uuid.Uuid" {
		initializer = scalarDefaultInitializer(scalar, initializer)
	}
	return &Constant{Name: name, WireName: wire, Type: TypeRef{Kind: Scalar, Name: scalar}, Initializer: initializer}, nil
}

func (a *analyzer) warnConst(pointer string) {
	a.warn("W_CONST_SCALAR_APPROXIMATION", "A scalar const in a value position retains its scalar wire type but does not enforce literal membership.", naming.Pointer(pointer, "const"))
}

// nativeIdentity mirrors the original decoder's number variant distinction:
// integer 1 and floating 1.0 remain distinct literals, while 1e0 and 1.0 share
// the same floating value. Do not compare raw input tokens for duplicate pins.
func nativeIdentity(v *document.Value) string {
	if n, ok := v.Numeric(); ok {
		if !strings.ContainsAny(string(n), ".eE") && string(n) != "-0" {
			if _, err := strconv.ParseInt(string(n), 10, 64); err == nil {
				return "integer:" + string(n)
			}
			if _, err := strconv.ParseUint(string(n), 10, 64); err == nil {
				return "integer:" + string(n)
			}
		}
		f, _ := n.Float64()
		return "float:" + strconv.FormatFloat(f, 'g', -1, 64)
	}
	b, _ := v.Bool()
	return "boolean:" + strconv.FormatBool(b)
}

func (a *analyzer) nativeEnum(kind string, v *document.Value, pointer string) (string, *diagnostic.Diagnostic) {
	if kind == "null" {
		return nullOnlyEnumProfile(v, pointer, a != nil && a.componentDefaultSchemas[v])
	}
	// A component root's own `default` is validated by the factory planner; the
	// shared type plan borrows the same node and must not reject the annotation.
	return a.nativeEnumKeys(kind, v, pointer, a != nil && a.componentDefaultSchemas[v])
}

func (a *analyzer) nativeEnumKeys(kind string, v *document.Value, pointer string, fieldDefault bool) (string, *diagnostic.Diagnostic) {
	return a.nativeEnumProfile(kind, v, pointer, fieldDefault, false)
}

func (a *analyzer) nativeEnumProfile(kind string, v *document.Value, pointer string, fieldDefault, checkedBounds bool) (string, *diagnostic.Diagnostic) {
	keys := []string{"type", "nullable", "format", "enum"}
	// A component root's own `default` is validated by the factory planner; the
	// shared type plan borrows the same node and must accept the key without
	// taking on the field role's membership semantics.
	if fieldDefault || a != nil && a.componentDefaultSchemas[v] {
		keys = append(keys, "default")
	}
	if checkedBounds {
		keys = append(keys, primitiveBoundKeys...)
	}
	if err := checkKeys(v, keys, pointer); err != nil {
		return "", err
	}
	scalar, err := scalarMapping(kind, v, pointer)
	if err != nil {
		return "", err
	}
	location := naming.Pointer(pointer, "enum")
	values := get(v, "enum")
	if values.Kind() != document.Array || len(values.Items()) == 0 {
		return "", invalid("enum must be a non-empty array of native scalar literals.", location)
	}
	allowNull := a.checkedNativeEnumNulls(v, pointer)
	seen := map[string]bool{}
	nonNull := 0
	for i, value := range values.Items() {
		p := naming.Pointer(location, strconv.Itoa(i))
		if value.Kind() == document.Null && allowNull {
			if seen["null"] {
				return "", invalid("Native enum literal entries must be unique.", p)
			}
			seen["null"] = true
			continue
		}
		if _, err := scalarLiteral(scalar, value, p); err != nil {
			return "", invalid("Enum member must be a literal representable by its mapped Kotlin scalar.", p)
		}
		key := nativeIdentity(value)
		if seen[key] {
			return "", invalid("Native enum literal entries must be unique.", p)
		}
		seen[key] = true
		nonNull++
	}
	if nonNull == 0 {
		return "", invalid("A nullable native enum must retain a non-null scalar member.", location)
	}
	a.warn("W_ENUM_NATIVE_APPROXIMATION", "A native scalar enum retains its scalar wire type but does not enforce membership.", location)
	return scalar, nil
}

func buildStringEnum(name string, v *document.Value, pointer string) (*StringEnum, *diagnostic.Diagnostic) {
	return buildStringEnumKeys(name, v, pointer, false)
}

func buildStringEnumKeys(name string, v *document.Value, pointer string, fieldDefault bool) (*StringEnum, *diagnostic.Diagnostic) {
	return buildStringEnumKeysFor(name, v, pointer, fieldDefault, nil)
}

// buildStringEnumKeysFor additionally admits a component root's own `default`
// when the caller's analyzer marked that original node.
func buildStringEnumKeysFor(name string, v *document.Value, pointer string, fieldDefault bool, a *analyzer) (*StringEnum, *diagnostic.Diagnostic) {
	keys := []string{"type", "nullable", "format", "enum", "x-native-type"}
	if fieldDefault || a != nil && a.componentDefaultSchemas[v] {
		keys = append(keys, "default")
	}
	if a != nil && a.checkedStringEnumRoot(v, pointer) {
		keys = append(keys, primitiveBoundKeys...)
	}
	return buildStringEnumAllowedKeys(name, v, pointer, keys)
}

// Callers grant only their own schema role; vocabulary parsing always borrows
// the original array/member nodes and retains the same names and wire strings.
func buildStringEnumAllowedKeys(name string, v *document.Value, pointer string, keys []string) (*StringEnum, *diagnostic.Diagnostic) {
	kind, _, err := schemaType(v, pointer)
	if err != nil {
		return nil, err
	}
	if kind != "" && kind != "string" {
		return nil, unsupported(naming.Pointer(pointer, "enum"))
	}
	if err := checkKeys(v, keys, pointer); err != nil {
		return nil, err
	}
	// Vocabulary callers ignore the mapped native type, but must still reject
	// malformed or unsafe extension metadata. Simple HTTP callers separately
	// use scalarMapping after validating this same vocabulary.
	if has(v, "x-native-type") {
		if _, err := scalarMapping("string", v, pointer); err != nil {
			return nil, err
		}
	}
	if format, ok := v.Get("format"); ok {
		location := naming.Pointer(pointer, "format")
		spelling, valid := format.Text()
		if !valid {
			return nil, invalid("format must be a string.", location)
		}
		if spelling != "" && spelling != "uuid" && spelling != "date" && spelling != "date-time" && spelling != "byte" && spelling != "binary" {
			return nil, unsupported(location)
		}
	}
	location := naming.Pointer(pointer, "enum")
	values := get(v, "enum")
	if values.Kind() != document.Array || len(values.Items()) == 0 {
		return nil, invalid("enum must be a non-empty array of unique strings.", location)
	}
	if len(values.Items()) > MaxMembers {
		return nil, diagnostic.At("E_ENUM_LIMIT", "Enum exceeds the supported 240-member limit; split it into smaller types.", location)
	}
	allowNull := nullableStringEnumNulls(v, pointer)
	hasNull := false
	for _, value := range values.Items() {
		hasNull = hasNull || value.Kind() == document.Null
	}
	strings := make([]string, 0, len(values.Items()))
	seen := map[string]bool{}
	nullSeen := false
	for i, value := range values.Items() {
		at := naming.Pointer(location, strconv.Itoa(i))
		if value.Kind() == document.Null && allowNull {
			if nullSeen {
				return nil, invalid("String enum entries must be unique.", at)
			}
			nullSeen = true
			continue
		}
		s, ok := value.Text()
		if !ok {
			return nil, invalid("String enum entries must be strings.", at)
		}
		if hasNull && allowNull && seen[s] {
			return nil, invalid("String enum entries must be unique.", at)
		}
		seen[s] = true
		strings = append(strings, s)
	}
	if len(strings) == 0 {
		return nil, invalid("A nullable String enum must retain a string member.", location)
	}
	return stringEnumValues(name, strings, location)
}

func stringEnumValues(name string, values []string, pointer string) (*StringEnum, *diagnostic.Diagnostic) {
	if len(values) > MaxMembers {
		return nil, diagnostic.At("E_ENUM_LIMIT", "Enum exceeds the supported 240-member limit.", pointer)
	}
	used := map[string]bool{"companion": true, "serializer": true, "unrecognized": true}
	if used[strings.ToLower(name)] {
		return nil, diagnostic.At("E_NAME_COLLISION", "Enum name collides with a serializer-owned declaration.", pointer)
	}
	seen := map[string]bool{}
	result := &StringEnum{ClassName: name}
	for i, wire := range values {
		if seen[wire] {
			return nil, invalid("String enum entries must be unique.", naming.Pointer(pointer, strconv.Itoa(i)))
		}
		seen[wire] = true
		base := naming.EnumMemberName(wire)
		member := base
		for attempt := 0; used[strings.ToLower(member)]; {
			attempt++
			member = base + "_VALUE"
			if attempt > 1 {
				member += strconv.Itoa(attempt)
			}
		}
		used[strings.ToLower(member)] = true
		result.Members = append(result.Members, EnumMember{Name: member, Wire: wire})
	}
	return result, nil
}

// nativeEnumWarn emits the native-enum approximation warning for a component
// root whose default is validated by the factory planner, without re-running
// the vocabulary parse.
func (a *analyzer) nativeEnumWarn(kind string, v *document.Value, pointer string) {
	a.warn("W_ENUM_NATIVE_APPROXIMATION", "A native scalar enum retains its scalar wire type but does not enforce membership.", naming.Pointer(pointer, "enum"))
}
