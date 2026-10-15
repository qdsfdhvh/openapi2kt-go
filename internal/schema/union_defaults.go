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
	return has(v, "default") && a.primitiveUnionElementCandidate(v)
}

func (a *analyzer) primitiveUnionElementCandidate(v *document.Value) bool {
	keyword := ""
	if has(v, "oneOf") {
		keyword = "oneOf"
	} else if has(v, "anyOf") {
		keyword = "anyOf"
	} else {
		return false
	}
	branches, ok := v.Get(keyword)
	if !ok || branches.Kind() != document.Array || len(branches.Items()) == 0 {
		return false
	}
	for _, branch := range branches.Items() {
		if branch.Kind() != document.Object {
			return false
		}
		resolved, pointer, _, err := a.resolveUnionBranch(unionBranch{Schema: branch, Pointer: naming.Pointer(naming.Pointer("", keyword), strconv.Itoa(0))}, 0)
		if err != nil {
			return false
		}
		if text(get(resolved, "type")) == "null" || inferredNullLiteral(resolved) {
			continue
		}
		kind, nullable, err := schemaType(resolved, pointer)
		if err != nil || nullable {
			return false
		}
		if kind != "string" && kind != "integer" && kind != "number" && kind != "boolean" {
			return false
		}
		if kind == "string" && has(resolved, "format") {
			return false
		}
	}
	return true
}

func (a *analyzer) unionDefaultField(f field, name string, nested *[]Declaration, used map[string]bool) (TypeRef, *diagnostic.Diagnostic) {
	if a.objectDefaultSchemas == nil {
		a.objectDefaultSchemas = map[*document.Value]bool{}
	}
	a.objectDefaultSchemas[f.Schema] = true
	before := len(*nested)
	ty, _, err := a.modelPropertyType(f.Schema, name, f.Pointer, nested, used, f.Depth)
	if err != nil {
		return TypeRef{}, err
	}
	if nested == nil || len(*nested) != before+1 {
		return TypeRef{}, unsupported(naming.Pointer(f.Pointer, "default"))
	}
	validation := *a
	validation.warnings = nil
	if _, err := validation.unionDefaultEncoded(f, (*nested)[before]); err != nil {
		return TypeRef{}, err
	}
	return ty, nil
}

// unionDefaultEncoded dispatches the shared plan's appended union declaration
// so primitive and object unions share one gate and initializer.
func (a *analyzer) unionDefaultEncoded(f field, declaration Declaration) (string, *diagnostic.Diagnostic) {
	switch union := declaration.(type) {
	case *PrimitiveUnion:
		if union.Object != nil || union.NullVariant || !a.primitiveUnionElementCandidate(f.Schema) {
			return "", unsupported(naming.Pointer(f.Pointer, "default"))
		}
		return a.primitiveUnionDefaultEncoded(f, union)
	case *ObjectUnion:
		return a.objectUnionDefaultEncoded(f, union)
	}
	return "", unsupported(naming.Pointer(f.Pointer, "default"))
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

// unionDefaultInitializer revalidates on a scratch plan so it never appends
// declarations twice. An explicit null stays the literal null.
func (a *analyzer) unionDefaultInitializer(f field) (string, *diagnostic.Diagnostic) {
	var scratch []Declaration
	_, _, err := a.modelPropertyType(f.Schema, "Validation", f.Pointer, &scratch, map[string]bool{}, f.Depth)
	if err != nil {
		return "", err
	}
	if len(scratch) != 1 {
		return "", unsupported(naming.Pointer(f.Pointer, "default"))
	}
	encoded, err := a.unionDefaultEncoded(f, scratch[0])
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

// objectUnionDefaultEncoded accepts a tag-dispatched content object union. The
// default must be an object whose discriminator selects a known variant; the
// value is then validated against that variant's original source through the
// shared object walker, so unknown keys, required members and leaf checks keep
// their member Pointers.
func (a *analyzer) objectUnionDefaultEncoded(f field, union *ObjectUnion) (string, *diagnostic.Diagnostic) {
	value := get(f.Schema, "default")
	at := naming.Pointer(f.Pointer, "default")
	if value.Kind() == document.Null {
		if !union.NullVariant {
			return "", invalid("A null default requires a nullable union.", at)
		}
		return "null", nil
	}
	if !union.Content() || union.TagField == "" || len(union.Tags) == 0 || len(union.TagCombinations) != 0 || len(union.Presence) != 0 {
		return "", unsupported(at)
	}
	if value.Kind() != document.Object {
		return "", invalid("A union default must be a JSON object.", at)
	}
	tag, ok := get(value, union.TagField).Text()
	if !ok {
		return "", invalid("A union default must carry its discriminator string.", at)
	}
	index := -1
	for _, rule := range union.Tags {
		if rule.Value == tag {
			index = rule.Index
			break
		}
	}
	if index < 0 || index >= len(union.Variants) {
		return "", invalid("A union default discriminator must select a known variant.", at)
	}
	variant := union.Variants[index]
	source := a.modelOrigins[variant]
	if source == nil {
		return "", unsupported(at)
	}
	node, err := a.objectDefaultNode(source, a.modelPointers[variant], f.Depth, false)
	if err != nil {
		return "", err
	}
	return node.value(value, at)
}

// primitiveUnionDefaultValue mirrors the emitted selector order (String,
// Boolean, Integer, Number), preserves the original JSON token because the
// selector's wrapper choice depends on the token kind, and enforces enum/const
// pins of the matched branch at generation time.
func (a *analyzer) primitiveUnionDefaultValue(f field, union *PrimitiveUnion) (string, *diagnostic.Diagnostic) {
	value := get(f.Schema, "default")
	at := naming.Pointer(f.Pointer, "default")
	kind, encoded, err := unionValueEncoded(value, union.Kinds, at)
	if err != nil {
		return "", err
	}
	if err := a.primitiveUnionPinCheck(f, kind, value, at); err != nil {
		return "", err
	}
	return encoded, nil
}

// primitiveUnionElementNode plans a container element or object property union
// on a scratch so its declaration never appends twice.
func (a *analyzer) primitiveUnionElementNode(v *document.Value, pointer string, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	var scratch []Declaration
	ty, _, err := a.modelPropertyType(v, "Validation", pointer, &scratch, map[string]bool{}, depth)
	if err != nil {
		return nil, err
	}
	union, ok := a.primitiveUnionDeclaration(field{}, &scratch, 0)
	if !ok {
		return nil, unsupported(pointer)
	}
	return &nestedDefaultNode{schema: v, pointer: pointer, ty: ty, union: union, native: a}, nil
}

// unionValue validates one union default member against its kinds and pins.
func (node *nestedDefaultNode) unionValue(v *document.Value, pointer string) (string, *diagnostic.Diagnostic) {
	kind, encoded, err := unionValueEncoded(v, node.union.Kinds, pointer)
	if err != nil {
		return "", err
	}
	if node.native != nil {
		if err := node.native.primitiveUnionPinCheck(field{Schema: node.schema, Pointer: node.pointer}, kind, v, pointer); err != nil {
			return "", err
		}
	}
	return encoded, nil
}

func unionValueEncoded(value *document.Value, kinds []PrimitiveKind, at string) (PrimitiveKind, string, *diagnostic.Diagnostic) {
	kind, ok := unionDefaultKind(value, kinds)
	if !ok {
		return 0, "", invalid("Default member must match an accepted union branch.", at)
	}
	encoded, err := value.MarshalJSON()
	if err != nil {
		return 0, "", invalid("Default member must be valid JSON.", at)
	}
	return kind, string(encoded), nil
}

func unionDefaultKind(value *document.Value, kinds []PrimitiveKind) (PrimitiveKind, bool) {
	has := func(kind PrimitiveKind) bool { return slices.Contains(kinds, kind) }
	switch value.Kind() {
	case document.String:
		if has(PrimitiveString) {
			return PrimitiveString, true
		}
	case document.Boolean:
		if has(PrimitiveBoolean) {
			return PrimitiveBoolean, true
		}
	case document.Number:
		if has(PrimitiveInteger) && unionIntegerToken(value) {
			return PrimitiveInteger, true
		}
		if has(PrimitiveNumber) && unionNumberToken(value) {
			return PrimitiveNumber, true
		}
	}
	return 0, false
}

// primitiveUnionPinCheck finds the branch the runtime selector would choose and
// validates its enum/const vocabulary. Unpinned branches accept any value of
// their kind.
func (a *analyzer) primitiveUnionPinCheck(f field, kind PrimitiveKind, value *document.Value, at string) *diagnostic.Diagnostic {
	view, err := a.readUnion(f.Schema, f.Pointer)
	if err != nil {
		return err
	}
	if view == nil {
		return unsupported(at)
	}
	for _, branch := range view.Branches {
		bv, bp, _, err := a.resolveUnionBranch(branch, f.Depth)
		if err != nil {
			return err
		}
		branchKind, _, err := schemaType(bv, bp)
		if err != nil {
			return err
		}
		if resolved, ok := primitiveKindOfString(branchKind); !ok || resolved != kind {
			continue
		}
		if has(bv, "const") {
			if !primitivePinEqual(get(bv, "const"), value, branchKind, at) {
				return invalid("Default must equal the pinned const value.", at)
			}
			return nil
		}
		if has(bv, "enum") {
			return a.enumDefaultMemberAt(bv, value, branchKind, bp, at)
		}
		return nil
	}
	return invalid("Default must match an accepted union branch.", at)
}

func primitiveKindOfString(kind string) (PrimitiveKind, bool) {
	switch kind {
	case "string":
		return PrimitiveString, true
	case "integer":
		return PrimitiveInteger, true
	case "number":
		return PrimitiveNumber, true
	case "boolean":
		return PrimitiveBoolean, true
	}
	return 0, false
}

func primitivePinEqual(pin, value *document.Value, kind, at string) bool {
	switch kind {
	case "string":
		left, lok := pin.Text()
		right, rok := value.Text()
		return lok && rok && left == right
	case "boolean":
		left, lok := pin.Bool()
		right, rok := value.Bool()
		return lok && rok && left == right
	}
	scalar, err := scalarMapping(kind, pin, at)
	if err != nil {
		return false
	}
	_, left, err := constraintNumber(pin, scalar, at)
	if err != nil {
		return false
	}
	_, right, err := constraintNumber(value, scalar, at)
	return err == nil && left.Cmp(right) == 0
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
