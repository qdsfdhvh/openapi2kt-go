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
	// Discarded allOf contributions validate annotations without owning any
	// declarations. Their scratch plan still needs a local declaration sink.
	if nested == nil {
		scratch := []Declaration{}
		nested = &scratch
	}
	if used == nil {
		used = map[string]bool{}
	}
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
		// A request-context union projects to a raw JsonElement carrier.
		if carrier := ty.WithoutRootNull(); carrier.Kind == JSON || carrier.Kind == JSONObject {
			validation := *a
			validation.warnings = nil
			if _, err := validation.rawJSONDefaultEncoded(f, ty); err != nil {
				return TypeRef{}, err
			}
			return ty, nil
		}
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
	ty, _, err := a.modelPropertyType(f.Schema, "Validation", f.Pointer, &scratch, map[string]bool{}, f.Depth)
	if err != nil {
		return "", err
	}
	if len(scratch) != 1 {
		if carrier := ty.WithoutRootNull(); carrier.Kind == JSON || carrier.Kind == JSONObject {
			return a.rawJSONDefaultEncoded(f, ty)
		}
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

// composedUnionDefaultField plans the shared field type for a union-in-allOf
// (Cartesian) field and validates its own default against the merged variant
// shape. Ordinary allOf merges keep their existing default rejection.
func (a *analyzer) composedUnionDefaultField(f field, name string, nested *[]Declaration, used map[string]bool) (TypeRef, *diagnostic.Diagnostic) {
	// Discarded allOf contributions validate annotations without owning any
	// declarations. Their scratch plan still needs a local declaration sink.
	if nested == nil {
		scratch := []Declaration{}
		nested = &scratch
	}
	if used == nil {
		used = map[string]bool{}
	}
	if a.objectDefaultSchemas == nil {
		a.objectDefaultSchemas = map[*document.Value]bool{}
	}
	a.objectDefaultSchemas[f.Schema] = true
	validation := *a
	validation.warnings = nil
	view, err := validation.objectCompositionUnion(f.Schema, f.Pointer, f.Depth)
	if err != nil {
		return TypeRef{}, err
	}
	if view == nil {
		return TypeRef{}, unsupported(naming.Pointer(f.Pointer, "default"))
	}
	before := 0
	if nested != nil {
		before = len(*nested)
	}
	ty, _, err := a.modelPropertyType(f.Schema, name, f.Pointer, nested, used, f.Depth)
	if err != nil {
		return TypeRef{}, err
	}
	if nested == nil || len(*nested) != before+1 {
		return TypeRef{}, unsupported(naming.Pointer(f.Pointer, "default"))
	}
	union, ok := (*nested)[before].(*ObjectUnion)
	if !ok {
		return TypeRef{}, unsupported(naming.Pointer(f.Pointer, "default"))
	}
	validation = *a
	validation.warnings = nil
	if _, err := validation.objectUnionEncoded(get(f.Schema, "default"), naming.Pointer(f.Pointer, "default"), f.Schema, f.Pointer, f.Depth, union); err != nil {
		return TypeRef{}, err
	}
	return ty, nil
}

// composedUnionDefaultInitializer revalidates on a scratch plan so it never
// appends declarations twice.
func (a *analyzer) composedUnionDefaultInitializer(f field) (string, *diagnostic.Diagnostic) {
	var scratch []Declaration
	if _, _, err := a.modelPropertyType(f.Schema, "Validation", f.Pointer, &scratch, map[string]bool{}, f.Depth); err != nil {
		return "", err
	}
	if len(scratch) != 1 {
		return "", unsupported(naming.Pointer(f.Pointer, "default"))
	}
	union, ok := scratch[0].(*ObjectUnion)
	if !ok {
		return "", unsupported(naming.Pointer(f.Pointer, "default"))
	}
	encoded, err := a.objectUnionEncoded(get(f.Schema, "default"), naming.Pointer(f.Pointer, "default"), f.Schema, f.Pointer, f.Depth, union)
	if err != nil {
		return "", err
	}
	if encoded == "null" {
		return "null", nil
	}
	return "`openapi-kt default model`(" + naming.Literal(encoded) + ")", nil
}

// planReferencedUnionDefault handles an ordinary direct local `$ref` field
// whose resolved component is a root union. The union declaration is replanned
// on a silenced copy so the component's own plan is untouched; the initializer
// binds the actual component type through the emitter.
func (a *analyzer) planReferencedUnionDefault(plan referencedFieldDefault) (bool, *diagnostic.Diagnostic) {
	ref, ok := get(plan.Field.Schema, "$ref").Text()
	if !ok {
		return false, nil
	}
	source, sourcePointer, err := a.defaultSource(ref, plan.Field.Pointer)
	if err != nil {
		return false, err
	}
	if !has(source, "oneOf") && !has(source, "anyOf") && !has(source, "allOf") {
		return false, nil
	}
	validation := *a
	validation.warnings = nil
	var declaration Declaration
	if has(source, "allOf") && !has(source, "oneOf") && !has(source, "anyOf") {
		// A composed component mirrors its standalone root plan, including the
		// standalone null variant for an explicitly nullable root.
		view, err := validation.objectCompositionUnion(source, sourcePointer, 0)
		if err != nil {
			return false, err
		}
		if view == nil {
			return false, nil
		}
		model, err := validation.objectUnion(view, "Validation", 1, true)
		if err != nil {
			return false, err
		}
		if model == nil {
			return false, unsupported(sourcePointer)
		}
		declaration = model
	} else {
		declaration, err = validation.rootUnion(source, "Validation", sourcePointer, map[string]bool{})
		if err != nil {
			return false, err
		}
	}
	value := get(plan.Field.Schema, "default")
	at := naming.Pointer(plan.Field.Pointer, "default")
	encoded, err := validation.unionDeclarationEncoded(value, at, source, sourcePointer, plan.Field.Depth, declaration)
	if err != nil {
		return false, err
	}
	if plan.Model == nil {
		return true, nil
	}
	property := &plan.Model.Properties[plan.Index]
	if !property.DefaultNull || property.Type.Kind == PatchValue {
		return true, nil
	}
	if value.Kind() == document.Null {
		property.Default = "null"
		return true, nil
	}
	property.Default = "`openapi-kt default model`(" + naming.Literal(encoded) + ")"
	return true, nil
}

// unionDeclarationEncoded dispatches an already planned union declaration for
// the referenced role.
func (a *analyzer) unionDeclarationEncoded(value *document.Value, at string, source *document.Value, sourcePointer string, depth int, declaration Declaration) (string, *diagnostic.Diagnostic) {
	switch union := declaration.(type) {
	case *PrimitiveUnion:
		if union.Object != nil || !a.primitiveUnionElementCandidate(source) {
			return "", unsupported(at)
		}
		return a.primitiveUnionEncoded(value, at, source, sourcePointer, depth, union)
	case *ObjectUnion:
		return a.objectUnionEncoded(value, at, source, sourcePointer, depth, union)
	}
	return "", unsupported(at)
}

// composedObjectUnionEncoded accepts a Cartesian composed union (union-in-allOf).
// The composed branches are recomputed so the selected variant validates against
// its merged original shape; selection mirrors the emitted conjunction rules.
func (a *analyzer) composedObjectUnionEncoded(value *document.Value, at string, source *document.Value, sourcePointer string, depth int, union *ObjectUnion) (string, *diagnostic.Diagnostic) {
	if !union.Content() {
		return "", unsupported(at)
	}
	if value.Kind() == document.Null {
		if !union.NullVariant {
			return "", invalid("A null default requires a nullable union.", at)
		}
		return "null", nil
	}
	composition, err := a.objectCompositionUnion(source, sourcePointer, depth)
	if err != nil {
		return "", err
	}
	if composition == nil || len(composition.Branches) != len(union.Variants) {
		return "", unsupported(at)
	}
	if value.Kind() != document.Object {
		return "", invalid("A union default must be a JSON object.", at)
	}
	index := -1
	for _, rule := range union.TagCombinations {
		matched := true
		for _, match := range rule.Matches {
			content, ok := unionTagContent(get(value, match.Wire))
			if !ok || !slices.Contains(match.Values, content) {
				matched = false
				break
			}
		}
		if matched {
			index = rule.Index
			break
		}
	}
	if index < 0 {
		for _, rule := range union.Presence {
			if has(value, rule.Value) {
				index = rule.Index
				break
			}
		}
	}
	if index < 0 || index >= len(union.Variants) {
		return "", invalid("A union default must select a known composed variant.", at)
	}
	variant := union.Variants[index]
	if a.modelOrigins[variant] == nil || composition.Branches[index].Shape == nil {
		return "", unsupported(at)
	}
	node, err := a.objectDefaultShapeNode(composition.Branches[index].Shape, a.modelPointers[variant], false)
	if err != nil {
		return "", err
	}
	return node.value(value, at)
}

// unionTagContent mirrors the emitted `obj[key]?.jsonPrimitive?.content`
// comparison, including the canonical spellings of number and boolean tokens.
func unionTagContent(v *document.Value) (string, bool) {
	if v == nil {
		return "", false
	}
	switch v.Kind() {
	case document.String:
		return text(v), true
	case document.Number:
		number, ok := v.Numeric()
		return string(number), ok
	case document.Boolean:
		flag, _ := v.Bool()
		return strconv.FormatBool(flag), true
	case document.Null:
		return "null", true
	}
	return "", false
}

func (a *analyzer) primitiveUnionDefaultEncoded(f field, union *PrimitiveUnion) (string, *diagnostic.Diagnostic) {
	return a.primitiveUnionEncoded(get(f.Schema, "default"), naming.Pointer(f.Pointer, "default"), f.Schema, f.Pointer, f.Depth, union)
}

func (a *analyzer) primitiveUnionEncoded(value *document.Value, at string, source *document.Value, sourcePointer string, depth int, union *PrimitiveUnion) (string, *diagnostic.Diagnostic) {
	work, bytes := 0, 0
	if err := nestedDefaultBudget(value, at, 0, &work, &bytes); err != nil {
		return "", err
	}
	if value.Kind() == document.Null {
		view, err := a.readUnion(source, sourcePointer)
		if err != nil {
			return "", err
		}
		if view == nil || !view.Nullable {
			return "", invalid("A null default requires a nullable union.", at)
		}
		return "null", nil
	}
	kind, encoded, err := unionValueEncoded(value, union.Kinds, at)
	if err != nil {
		return "", err
	}
	if err := a.primitiveUnionPinCheck(source, sourcePointer, depth, kind, value, at); err != nil {
		return "", err
	}
	return encoded, nil
}

// objectUnionDefaultEncoded accepts a tag-dispatched content object union. The
// default must be an object whose discriminator selects a known variant; the
// value is then validated against that variant's original source through the
// shared object walker, so unknown keys, required members and leaf checks keep
// their member Pointers.
func (a *analyzer) objectUnionDefaultEncoded(f field, union *ObjectUnion) (string, *diagnostic.Diagnostic) {
	return a.objectUnionEncoded(get(f.Schema, "default"), naming.Pointer(f.Pointer, "default"), f.Schema, f.Pointer, f.Depth, union)
}

func (a *analyzer) objectUnionEncoded(value *document.Value, at string, source *document.Value, sourcePointer string, depth int, union *ObjectUnion) (string, *diagnostic.Diagnostic) {
	work, bytes := 0, 0
	if err := nestedDefaultBudget(value, at, 0, &work, &bytes); err != nil {
		return "", err
	}
	if value.Kind() == document.Null {
		if !union.NullVariant {
			return "", invalid("A null default requires a nullable union.", at)
		}
		return "null", nil
	}
	if len(union.TagCombinations) != 0 {
		// Cartesian composed variants select by conjoined tag pins; presence-only
		// rules may coexist for pin-less variants.
		return a.composedObjectUnionEncoded(value, at, source, sourcePointer, depth, union)
	}
	if !union.Content() || len(union.Tags) == 0 && len(union.Presence) == 0 {
		return "", unsupported(at)
	}
	if value.Kind() != document.Object {
		return "", invalid("A union default must be a JSON object.", at)
	}
	// Mirror the emitted selector: discriminator tags first, then presence.
	index := -1
	if union.TagField != "" {
		if tag, ok := get(value, union.TagField).Text(); ok {
			for _, rule := range union.Tags {
				if rule.Value == tag {
					index = rule.Index
					break
				}
			}
		}
	}
	if index < 0 {
		for _, rule := range union.Presence {
			if has(value, rule.Value) {
				index = rule.Index
				break
			}
		}
	}
	if index < 0 || index >= len(union.Variants) {
		return "", invalid("A union default must select a known variant.", at)
	}
	variant := union.Variants[index]
	variantSource := a.modelOrigins[variant]
	if variantSource == nil {
		return "", unsupported(at)
	}
	node, err := a.objectDefaultNode(variantSource, a.modelPointers[variant], depth, false)
	if err != nil {
		return "", err
	}
	return node.value(value, at)
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
		if err := node.native.primitiveUnionPinCheck(node.schema, node.pointer, 0, kind, v, pointer); err != nil {
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
func (a *analyzer) primitiveUnionPinCheck(source *document.Value, sourcePointer string, depth int, kind PrimitiveKind, value *document.Value, at string) *diagnostic.Diagnostic {
	view, err := a.readUnion(source, sourcePointer)
	if err != nil {
		return err
	}
	if view == nil {
		return unsupported(at)
	}
	for _, branch := range view.Branches {
		bv, bp, _, err := a.resolveUnionBranch(branch, depth)
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
