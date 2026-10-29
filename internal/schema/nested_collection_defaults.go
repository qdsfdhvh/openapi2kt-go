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
	dtoUniqueKeys *dtoUniqueDefaultKeys
	uninhabited   bool
	// Granted by an original tuple projection or admitted JSON codec source.
	// JsonNull remains a JSON value independently of Kotlin nullable intent.
	rawJSON bool
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
	// A repeated ordinary allOf numeric field keeps only selected original
	// endpoints for literal checks. The public TypeRef owns emitted validation.
	numericAggregate *numericDefaultAggregate
	// Plain inline ordinary DTO objects recurse through their declared
	// properties. The ty JSON/nullable view carries only null intent; the actual
	// nominal type binding lives with the consuming field plan.
	object       bool
	objectFields []objectDefaultProperty
	objectCounts *CollectionValidation
	// Inline primitive union leaves retain the shared declaration and validate
	// each value against its kinds and enum/const pins.
	union *PrimitiveUnion
	// Tuple unions borrow the actual selected declaration, including root
	// subtype dispatch and null carriers, rather than guessing a nominal type.
	tupleUnion        Declaration
	unionDepth        int
	hierarchyTag      *StringEnum
	hierarchyTagValue *string
	// Scalar const properties validate the supplied value for exact equality.
	constValue *document.Value
	constKind  string
	// nullOnly is granted only by the original explicit/inferred JsonNull grammar.
	nullOnly      bool
	literalSource *tupleLiteralSource
	// native carries the analyzer for pin/const checks on union and const nodes.
}

func nestedCollectionDefaultSchema(v *document.Value) bool {
	item := get(v, "items")
	if dictionarySchema(v) {
		item = get(v, "additionalProperties")
	}
	leafKind, _, _ := schemaType(item, "")
	format := text(get(item, "format"))
	return leafKind == "null" || bareArrayDefaultSchema(item) || dictionarySchema(item) ||
		leafKind == "string" && (format == "byte" || format == "binary")
}

func (a *analyzer) nestedCollectionDefault(v *document.Value, pointer string, depth int) (TypeRef, string, *diagnostic.Diagnostic) {
	return a.nestedCollectionDefaultNamed(v, "Validation", pointer, nil, nil, depth)
}

func (a *analyzer) nestedCollectionDefaultNamed(v *document.Value, name, pointer string, nested *[]Declaration, used map[string]bool, depth int) (TypeRef, string, *diagnostic.Diagnostic) {
	return a.nestedCollectionDefaultNamedWithEnumBounds(v, name, pointer, nested, used, depth, nil)
}

func (a *analyzer) nestedCollectionDefaultNamedWithEnumBounds(v *document.Value, name, pointer string, nested *[]Declaration, used map[string]bool, depth int, boundedEnum *document.Value) (TypeRef, string, *diagnostic.Diagnostic) {
	node, err := a.nestedDefaultSchemaWithEnumBounds(v, name, pointer, nested, used, depth, true, boundedEnum)
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
	return node.enum != nil || node.stringEnum || node.element != nil && node.element.hasEnum()
}

func (a *analyzer) nestedDefaultSchema(v *document.Value, name, pointer string, nested *[]Declaration, used map[string]bool, depth int, root bool) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	return a.nestedDefaultSchemaWithEnumBounds(v, name, pointer, nested, used, depth, root, nil)
}

// An ordinary field/property may retain one original checked enum source at
// the end of its admitted inline container path. The exact node grants no
// permission to a copied leaf or a reference.
func (a *analyzer) nestedDefaultSchemaWithEnumBounds(v *document.Value, name, pointer string, nested *[]Declaration, used map[string]bool, depth int, root bool, boundedEnum *document.Value) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	if v != nil && v == boundedEnum && a.options.Validation && directStringEnumFieldBounds(v) {
		ty, err := a.checkedStringEnumLeaf(v, name, pointer, nested, used, depth, root && has(v, "default"))
		if err != nil {
			return nil, err
		}
		return &nestedDefaultNode{schema: v, pointer: pointer, ty: ty, stringEnum: true}, nil
	}
	if node, err := a.literalFalseCollectionDefaultLeaf(v, pointer, depth); err != nil {
		return nil, err
	} else if node != nil {
		return node, nil
	}
	if node, err := a.falseCollectionDefaultLeaf(v, pointer, depth); err != nil {
		return nil, err
	} else if node != nil {
		return node, nil
	}
	if err := depthCheck(depth, pointer); err != nil {
		return nil, err
	}
	if err := object(v, pointer); err != nil {
		return nil, err
	}
	kind, nullable, typeErr := schemaType(v, pointer)
	container := kind == "array" || dictionarySchema(v)
	keys := []string{"type", "nullable", "format"}
	if kind == "null" && typeErr == nil {
		keys = []string{"type", "nullable"}
		if has(v, "enum") {
			keys = append(keys, "enum")
		}
	} else if container {
		keys = []string{"type", "nullable", "items"}
		if dictionarySchema(v) {
			keys = []string{"type", "nullable", "properties", "required", "additionalProperties"}
		}
		keys = a.collectionKeys(keys)
	} else if has(v, "enum") && typeErr == nil && (kind == "" || kind == "string") {
		// Nested enum leaves retain the original unbounded enum grammar.
		// Direct single-layer String bounds do not authorize this role.
		keys = append(keys, "enum")
		if lexicalStringEnumSchema(v) {
			keys = append(keys, "x-native-type")
		}
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
	if kind == "null" {
		if has(v, "enum") {
			if _, err := nullOnlyEnumProfile(v, pointer, root && has(v, "default")); err != nil {
				return nil, err
			}
		} else if _, err := scalarMapping(kind, v, pointer); err != nil {
			return nil, err
		}
		if root && has(v, "default") {
			if has(v, "enum") {
				if err := a.enumDefaultValue(v, pointer); err != nil {
					return nil, err
				}
			} else if _, err := scalarSchemaDefault(v, pointer); err != nil {
				return nil, err
			}
		}
		node.ty = TypeRef{Kind: Scalar, Name: "kotlinx.serialization.json.JsonNull"}
		node.nullOnly = true
	} else if container {
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
		element, err := a.nestedDefaultSchemaWithEnumBounds(get(v, edge), name+suffix, naming.Pointer(pointer, edge), nested, used, depth+1, false, boundedEnum)
		if err != nil {
			return nil, err
		}
		node.element = element
		ty, err := a.checkedCollectionRole(TypeRef{Kind: typeKind, Element: &element.ty}, v, pointer, element.object)
		if err != nil {
			return nil, err
		}
		node.ty = ty
		node.dtoUniqueKeys = a.dtoUniqueKeys
	} else if has(v, "enum") && (kind == "" || kind == "string") {
		// root grants this node its own annotation role (for example an
		// ordinary object property). Container elements still pass false.
		if root && has(v, "default") {
			if err := a.enumDefaultValue(v, pointer); err != nil {
				return nil, err
			}
		}
		enum, err := buildStringEnumKeys(name, v, pointer, root)
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
		var ty TypeRef
		var err *diagnostic.Diagnostic
		if root && has(v, "default") {
			// Only an already permitted property's own annotation gains this
			// role. Collection children retain root=false and their old grammar.
			if values := get(v, "enum"); values.Kind() == document.Array && len(values.Items()) > MaxMembers {
				return nil, diagnostic.At("E_MODEL_LIMIT", "Native enum object default exceeds the supported 240-member limit.", naming.Pointer(pointer, "enum"))
			}
			if a.options.Validation {
				ty, err = a.nativeEnumValueTypeOwnDefault(v, pointer, depth, true)
			} else {
				ty, err = a.enumDefaultType(field{Schema: v, Pointer: pointer, Depth: depth}, name, nil, nil)
			}
		} else {
			ty, err = a.collectionDefaultElement(v, name, pointer, nested, used, depth)
		}
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
		if !slices.Contains([]string{"kotlin.String", "kotlin.uuid.Uuid", "kotlin.Boolean", "kotlin.Int", "kotlin.Long", "kotlin.Float", "kotlin.Double", "kotlin.time.Instant", "kotlinx.datetime.LocalDate", "kotlin.ByteArray"}, scalar) {
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
	if node.hierarchyTag != nil {
		wire, ok := v.Text()
		found := false
		if ok {
			for _, member := range node.hierarchyTag.Members {
				found = found || member.Wire == wire
			}
		}
		if !found {
			return "", invalid("A hierarchy discriminator default must name a declared tag.", pointer)
		}
		if node.hierarchyTagValue != nil && wire != *node.hierarchyTagValue {
			return "", invalid("A hierarchy discriminator default must equal its actual typed body tag.", pointer)
		}
	}
	if node.rawJSON {
		if node.ty.WithoutRootNull().Kind == JSONObject && v.Kind() != document.Object && !(v.Kind() == document.Null && node.ty.Kind == Nullable) {
			return "", invalid("A JsonObject default member must be an object or an explicitly nullable null.", pointer)
		}
		if checks := node.ty.WithoutRootNull().Collection; node.ty.WithoutRootNull().Kind == JSONObject && v.Kind() == document.Object && checks != nil {
			count := int64(len(v.Members()))
			if checks.Minimum != nil && count < *checks.Minimum || checks.Maximum != nil && count > *checks.Maximum {
				return "", invalid("A JsonObject default member must satisfy its retained property counts.", pointer)
			}
		}
		encoded, err := v.MarshalJSON()
		if err != nil {
			return "", invalid("A raw tuple default member must be valid JSON.", pointer)
		}
		return string(encoded), nil
	}
	if node.tupleUnion != nil {
		if v.Kind() == document.Null && node.ty.Kind == Nullable {
			return "null", nil
		}
		if hierarchy, ok := node.tupleUnion.(*Hierarchy); ok {
			return node.native.tupleHierarchyDefaultEncoded(v, pointer, hierarchy, node.unionDepth)
		}
		if vocabulary, ok := node.tupleUnion.(*StringEnum); ok {
			if wire, ok := v.Text(); ok {
				for _, member := range vocabulary.Members {
					if member.Wire == wire {
						encoded, _ := v.MarshalJSON()
						return string(encoded), nil
					}
				}
			}
			return "", invalid("Default must be a member of the normalized enum vocabulary.", pointer)
		}
		return node.native.unionDeclarationEncoded(v, pointer, node.schema, node.pointer, node.unionDepth, node.tupleUnion)
	}
	if node.constValue != nil && node.constKind == "null" {
		if node.constValue.Kind() != document.Null || v.Kind() != document.Null {
			return "", invalid("Default member must equal the field const value.", pointer)
		}
		return "null", nil
	}
	if node.nullOnly {
		if v.Kind() != document.Null {
			return "", invalid("Default member must be JSON null for its JsonNull codec.", pointer)
		}
		return "null", nil
	}
	if v.Kind() == document.Null {
		if node.ty.Kind != Nullable {
			return "", invalid("A null default member requires explicit schema nullability.", pointer)
		}
		return "null", nil
	}
	if node.uninhabited {
		return "", invalid("A false default leaf admits no non-null value.", pointer)
	}
	if node.literalSource != nil {
		return node.literalSource.defaultValue(v, pointer)
	}
	if node.union != nil {
		return node.unionValue(v, pointer)
	}
	if node.constValue != nil {
		if !primitivePinEqual(node.constValue, v, node.constKind, pointer) {
			return "", invalid("Default member must equal the field const value.", pointer)
		}
		encoded, _ := v.MarshalJSON()
		return string(encoded), nil
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
					if err := validateDefaultChecksAt(v, node.schema, ty.Validation, "kotlin.String", node.pointer, pointer); err != nil {
						return "", err
					}
					encoded, _ := v.MarshalJSON()
					return string(encoded), nil
				}
			}
		}
		return "", invalid("Default must be a member of the declared enum vocabulary.", pointer)
	}
	if ty.Kind == Scalar {
		if ty.Name == "kotlin.ByteArray" {
			// The existing bytes codec uses a signed-integer JSON array. Keep
			// each byte's original Pointer rather than coarsening leaf errors.
			if _, err := bytesDefaultLiteral(v, pointer); err != nil {
				return "", err
			}
			encoded, _ := v.MarshalJSON()
			return string(encoded), nil
		}
		if _, err := scalarLiteral(ty.Name, v, pointer); err != nil {
			return "", invalid("Default member must fit its mapped Kotlin scalar type.", pointer)
		}
		if ty.Name == "kotlin.time.Instant" && !collectionDefaultInstantPattern.MatchString(text(v)) {
			return "", invalid("Timestamp default must fit the Kotlin Instant format and precision.", pointer)
		}
		if node.numericAggregate != nil {
			if err := node.numericAggregate.validate(v, ty.Name, pointer); err != nil {
				return "", err
			}
		} else if node.native != nil {
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
		if ty.Collection != nil && (ty.Collection.Unique || ty.Collection.UniqueJSON) {
			seen = map[string]bool{}
		}
		for i, member := range v.Items() {
			location := naming.Pointer(pointer, strconv.Itoa(i))
			encoded, err := node.element.value(member, location)
			if err != nil {
				return "", err
			}
			if seen != nil {
				key := structuralUniqueDefaultKey(node.element.ty, member)
				if ty.Collection.UniqueJSON {
					var failure *diagnostic.Diagnostic
					key, failure = rawJSONDefaultKey(member, location)
					if failure != nil {
						return "", failure
					}
				}
				if ty.Collection.UniqueDTO != nil {
					var failure *diagnostic.Diagnostic
					key, failure = node.dtoUniqueKeys.key(ty.Collection.UniqueDTO, member, location)
					if failure != nil {
						return "", failure
					}
				}
				if seen[key] {
					return "", invalid("Default members must satisfy uniqueItems.", location)
				}
				seen[key] = true
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
		case "kotlinx.serialization.json.JsonNull":
			return "Scalar JsonNull"
		case "kotlinx.serialization.json.JsonObject":
			return "Scalar JsonObject"
		case "kotlinx.serialization.json.JsonArray":
			return "Scalar JsonArray"
		case "kotlin.uuid.Uuid":
			return "Scalar Uuid"
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
