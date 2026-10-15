package schema

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// These limits apply only to the newly admitted nested annotation role. The
// byte cap also keeps the helper argument below JVM modified-UTF8 constant limits.
const maxNestedDefaultNodes = 16384
const maxNestedDefaultBytes = 32 * 1024

// Keep original schema nodes and constraint provenance alongside the actual
// typed plan. Empty/null annotations must validate the entire source schema.
type nestedDefaultNode struct {
	schema  *document.Value
	pointer string
	ty      TypeRef
	element *nestedDefaultNode
	enum    *StringEnum
	// Referenced collection own-default terminals keep the original source
	// schema as the strict membership authority; the actual Named TypeRef
	// stays the only type identity.
	stringEnum bool
	// Native annotations reuse the original collection-leaf planner and exact
	// membership checker; the mapped scalar TypeRef remains the type authority.
	native *analyzer
	// Plain inline ordinary DTO objects recurse through their declared
	// properties. The ty JSON/nullable view carries only null intent; the actual
	// nominal type binding lives with the consuming field plan.
	object       bool
	objectFields []objectDefaultProperty
}

func nestedCollectionDefaultSchema(v *document.Value) bool {
	item := get(v, "items")
	if dictionarySchema(v) {
		item = get(v, "additionalProperties")
	}
	return bareArrayDefaultSchema(item) || dictionarySchema(item)
}

func (a *analyzer) nestedCollectionDefault(v *document.Value, pointer string, depth int) (TypeRef, string, *diagnostic.Diagnostic) {
	return a.nestedCollectionDefaultNamed(v, "Validation", pointer, nil, nil, depth)
}

func (a *analyzer) nestedCollectionDefaultNamed(v *document.Value, name, pointer string, nested *[]Declaration, used map[string]bool, depth int) (TypeRef, string, *diagnostic.Diagnostic) {
	node, err := a.nestedDefaultSchema(v, name, pointer, nested, used, depth, true)
	if err != nil {
		return TypeRef{}, "", err
	}
	value := get(v, "default")
	at := naming.Pointer(pointer, "default")
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
		return TypeRef{}, "", diagnostic.At("E_MODEL_LIMIT", "Canonical nested default exceeds the supported 32 KiB JSON byte limit.", at)
	}
	helper := "`openapi-kt default tree " + nestedDefaultTypeKey(node.ty.WithoutRootNull()) + "`"
	if node.hasEnum() {
		// The emitter binds the final actual lexical TypeRef after relocation.
		// No classifier spelling is stored in the annotation initializer.
		helper = "`openapi-kt default enum tree`"
	}
	return node.ty, helper + "(" + naming.Literal(encoded) + ")", nil
}

func (node *nestedDefaultNode) hasEnum() bool {
	return node.enum != nil || node.element != nil && node.element.hasEnum()
}

func (a *analyzer) nestedDefaultSchema(v *document.Value, name, pointer string, nested *[]Declaration, used map[string]bool, depth int, root bool) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, pointer); err != nil {
		return nil, err
	}
	if err := object(v, pointer); err != nil {
		return nil, err
	}
	kind, nullable, typeErr := schemaType(v, pointer)
	container := kind == "array" || dictionarySchema(v)
	keys := []string{"type", "nullable", "format"}
	if container {
		keys = []string{"type", "nullable", "items"}
		if dictionarySchema(v) {
			keys = []string{"type", "nullable", "properties", "required", "additionalProperties"}
		}
		keys = a.collectionKeys(keys)
	} else if has(v, "enum") && typeErr == nil && (kind == "" || kind == "string") {
		// Nested enum leaves retain the original unbounded enum grammar.
		// Direct single-layer String bounds do not authorize this role.
		keys = append(keys, "enum")
	} else if has(v, "enum") && typeErr == nil && slices.Contains([]string{"boolean", "integer", "number"}, kind) {
		keys = append(keys, "enum")
		if a.options.Validation {
			keys = append(keys, primitiveBoundKeys...)
		}
	} else if a.options.Validation {
		keys = append(keys, primitiveBoundKeys...)
	}
	if root {
		keys = append(keys, "default")
	}
	if err := checkKeys(v, keys, pointer); err != nil {
		return nil, err
	}
	if typeErr != nil {
		return nil, typeErr
	}
	node := &nestedDefaultNode{schema: v, pointer: pointer}
	if container {
		edge, suffix := "items", "Inner"
		typeKind := List
		if dictionarySchema(v) {
			if kind != "object" {
				return nil, unsupported(pointer)
			}
			if required, ok := v.Get("required"); ok && (required.Kind() != document.Array || len(required.Items()) != 0) {
				return nil, invalid("Dictionary schemas cannot require undeclared fixed properties.", naming.Pointer(pointer, "required"))
			}
			edge, suffix = "additionalProperties", "Value"
			typeKind = Map
		}
		element, err := a.nestedDefaultSchema(get(v, edge), name+suffix, naming.Pointer(pointer, edge), nested, used, depth+1, false)
		if err != nil {
			return nil, err
		}
		node.element = element
		ty, err := a.checkedCollection(TypeRef{Kind: typeKind, Element: &element.ty}, v, pointer)
		if err != nil {
			return nil, err
		}
		node.ty = ty
	} else if has(v, "enum") && (kind == "" || kind == "string") {
		if has(v, "format") {
			return nil, unsupported(naming.Pointer(pointer, "enum"))
		}
		enum, err := buildStringEnum(name, v, pointer)
		if err != nil {
			return nil, err
		}
		if nested != nil {
			if err := reserveNested(used, name, pointer); err != nil {
				return nil, err
			}
			a.recordEnumOrigin(enum, v, pointer, nullable)
			*nested = append(*nested, enum)
		}
		node.enum = enum
		node.ty = TypeRef{Kind: Named, Name: name}
	} else if has(v, "enum") {
		ty, err := a.collectionDefaultElement(v, name, pointer, nested, used, depth)
		if err != nil {
			return nil, err
		}
		node.ty, node.native = ty, a
		return node, nil // The existing leaf planner owns nullable/check metadata.
	} else {
		if !slices.Contains([]string{"string", "boolean", "integer", "number"}, kind) {
			return nil, unsupported(pointer)
		}
		scalar, err := scalarMapping(kind, v, pointer)
		if err != nil {
			return nil, err
		}
		if !slices.Contains([]string{"kotlin.String", "kotlin.Boolean", "kotlin.Int", "kotlin.Long", "kotlin.Float", "kotlin.Double", "kotlin.time.Instant", "kotlinx.datetime.LocalDate"}, scalar) {
			return nil, unsupported(pointer)
		}
		checks, err := a.primitiveValidation(v, scalar, pointer)
		if err != nil {
			return nil, err
		}
		if checks != nil && checks.MultipleOf != nil && checks.Floating {
			return nil, unsupported(naming.Pointer(pointer, "multipleOf"))
		}
		if checks != nil {
			a.validationSources[checks] = pointer
		}
		node.ty = TypeRef{Kind: Scalar, Name: scalar, Validation: checks}
	}
	if nullable {
		node.ty = node.ty.Null()
	}
	return node, nil
}

// Preflight the whole original JSON tree before allocating canonical containers.
// Sorting occurs only after validation, so neither resource nor value diagnostics
// move to a sorted key/index. No schema key inside default data is interpreted.
func nestedDefaultBudget(v *document.Value, pointer string, depth int, work, bytes *int) *diagnostic.Diagnostic {
	if err := depthCheck(depth, pointer); err != nil {
		return err
	}
	*work++
	if *work > maxNestedDefaultNodes {
		return diagnostic.At("E_MODEL_LIMIT", "Nested collection default exceeds the supported 16,384 value nodes.", pointer)
	}
	switch v.Kind() {
	case document.Array:
		*bytes += 2
		for i, member := range v.Items() {
			if i > 0 {
				*bytes++
			}
			if err := nestedDefaultBudget(member, naming.Pointer(pointer, strconv.Itoa(i)), depth+1, work, bytes); err != nil {
				return err
			}
		}
	case document.Object:
		*bytes += 2
		for i, member := range v.Members() {
			key, _ := json.Marshal(member.Name)
			*bytes += len(key) + 1
			if i > 0 {
				*bytes++
			}
			location := naming.Pointer(pointer, member.Name)
			if *bytes > maxNestedDefaultBytes {
				return diagnostic.At("E_MODEL_LIMIT", "Nested collection default exceeds the supported 32 KiB JSON byte limit.", location)
			}
			if err := nestedDefaultBudget(member.Value, location, depth+1, work, bytes); err != nil {
				return err
			}
		}
	default:
		encoded, _ := v.MarshalJSON()
		*bytes += len(encoded)
	}
	if *bytes > maxNestedDefaultBytes {
		return diagnostic.At("E_MODEL_LIMIT", "Nested collection default exceeds the supported 32 KiB JSON byte limit.", pointer)
	}
	return nil
}

func (node *nestedDefaultNode) value(v *document.Value, pointer string) (string, *diagnostic.Diagnostic) {
	if v.Kind() == document.Null {
		if node.ty.Kind != Nullable {
			return "", invalid("A null default member requires explicit schema nullability.", pointer)
		}
		return "null", nil
	}
	if node.object {
		return node.objectValue(v, pointer)
	}
	ty := node.ty.WithoutRootNull()
	if node.enum != nil || node.stringEnum {
		// The schema was fully checked before even an empty/null tree is
		// visited. Annotation membership is strict; runtime stays open.
		text, ok := v.Text()
		if ok {
			for _, member := range get(node.schema, "enum").Items() {
				if wire, stringMember := member.Text(); stringMember && wire == text {
					encoded, _ := v.MarshalJSON()
					return string(encoded), nil
				}
			}
		}
		return "", invalid("Default must be a member of the declared enum vocabulary.", pointer)
	}
	if ty.Kind == Scalar {
		if _, err := scalarLiteral(ty.Name, v, pointer); err != nil {
			return "", invalid("Default member must fit its mapped Kotlin scalar type.", pointer)
		}
		if ty.Name == "kotlin.time.Instant" && !collectionDefaultInstantPattern.MatchString(text(v)) {
			return "", invalid("Timestamp default must fit the Kotlin Instant format and precision.", pointer)
		}
		if node.native != nil {
			if err := node.native.collectionDefaultMemberChecks(v, node.schema, ty, node.pointer, pointer); err != nil {
				return "", err
			}
		} else if err := validateDefaultChecksAt(v, node.schema, ty.Validation, ty.Name, node.pointer, pointer); err != nil {
			return "", err
		}
		switch ty.Name {
		case "kotlin.Float", "kotlin.Double":
			number, _ := floatLiteral(v)
			bits := 64
			if ty.Name == "kotlin.Float" {
				number = float64(float32(number))
				bits = 32
			}
			return strconv.FormatFloat(number, 'g', -1, bits), nil
		case "kotlin.Int", "kotlin.Long":
			number, _ := integerLiteral(v)
			return strconv.FormatInt(number, 10), nil
		default:
			encoded, _ := v.MarshalJSON()
			return string(encoded), nil
		}
	}
	count := len(v.Items())
	if ty.Kind == Map {
		if v.Kind() != document.Object {
			return "", invalid("A typed dictionary default must be a JSON object.", pointer)
		}
		count = len(v.Members())
	} else if v.Kind() != document.Array {
		return "", invalid("A typed list default must be a JSON array.", pointer)
	}
	if c := ty.Collection; c != nil {
		if c.Minimum != nil && int64(count) < *c.Minimum || c.Maximum != nil && int64(count) > *c.Maximum {
			return "", invalid("Default must satisfy its retained collection counts.", pointer)
		}
	}
	if ty.Kind == List {
		members := make([]string, 0, count)
		var seen map[string]bool
		if ty.Collection != nil && ty.Collection.Unique {
			seen = map[string]bool{}
		}
		for i, member := range v.Items() {
			location := naming.Pointer(pointer, strconv.Itoa(i))
			encoded, err := node.element.value(member, location)
			if err != nil {
				return "", err
			}
			if seen != nil {
				if seen[encoded] {
					return "", invalid("Default members must satisfy uniqueItems.", location)
				}
				seen[encoded] = true
			}
			members = append(members, encoded)
		}
		return "[" + strings.Join(members, ",") + "]", nil
	}
	type entry struct{ key, value string }
	entries := make([]entry, 0, count)
	for _, member := range v.Members() {
		encoded, err := node.element.value(member.Value, naming.Pointer(pointer, member.Name))
		if err != nil {
			return "", err
		}
		entries = append(entries, entry{member.Name, encoded})
	}
	slices.SortFunc(entries, func(x, y entry) int { return strings.Compare(x.key, y.key) })
	members := make([]string, 0, count)
	for _, entry := range entries {
		key, _ := json.Marshal(entry.key)
		members = append(members, string(key)+":"+entry.value)
	}
	return "{" + strings.Join(members, ",") + "}", nil
}

func nestedDefaultTypeKey(ty TypeRef) string {
	switch ty.Kind {
	case Nullable:
		return "Nullable " + nestedDefaultTypeKey(*ty.Element)
	case List:
		return "List " + nestedDefaultTypeKey(*ty.Element)
	case Map:
		return "Map " + nestedDefaultTypeKey(*ty.Element)
	case Scalar:
		// Backtick identifiers still obey JVM method-name restrictions. Use
		// unambiguous short keys only for these admitted builtin temporal types;
		// helpers bind the final actual TypeRef in the existing JSON emitter.
		switch ty.Name {
		case "kotlin.time.Instant":
			return "Scalar Instant"
		case "kotlinx.datetime.LocalDate":
			return "Scalar LocalDate"
		}
		fallthrough
	default:
		return fmt.Sprintf("Scalar %s", strings.TrimPrefix(ty.Name, "kotlin."))
	}
}
