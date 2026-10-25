package schema

import (
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

type ObjectTagMatch struct {
	Wire   string
	Values []string
}
type ObjectTagRule struct {
	Matches []ObjectTagMatch
	Index   int
}

type DispatchRule struct {
	Value string
	Index int
}
type ObjectUnion struct {
	Plain           bool
	ClassName       string
	Common          []Property
	Variants        []*Model
	TagField        string
	Tags, Presence  []DispatchRule
	TagCombinations []ObjectTagRule // Composed unions may select on several pins.
	// nil is content dispatch; non-nil contains the closed subtype serial names.
	SerialNames []string
	NullVariant bool
}

func (m *ObjectUnion) Name() string  { return m.ClassName }
func (*ObjectUnion) declaration()    {}
func (m *ObjectUnion) Content() bool { return m.SerialNames == nil }

func shapeField(s *shape, wire string) *field {
	for i := range s.Fields {
		if s.Fields[i].Wire == wire {
			return &s.Fields[i]
		}
	}
	return nil
}
func commonProperty(common []Property, wire string) *Property {
	for i := range common {
		if common[i].WireName == wire {
			return &common[i]
		}
	}
	return nil
}
func bare(t TypeRef) TypeRef {
	for t.Kind == Nullable {
		t = *t.Element
	}
	return t
}

func (a *analyzer) actualField(f *field) (*document.Value, string, *diagnostic.Diagnostic) {
	if err := object(f.Schema, f.Pointer); err != nil {
		return nil, "", err
	}
	if has(f.Schema, "$ref") {
		if err := checkKeys(f.Schema, []string{"$ref", "nullable"}, f.Pointer); err != nil {
			return nil, "", err
		}
		reference, ok := get(f.Schema, "$ref").Text()
		if !ok {
			return nil, "", invalid("$ref must be a string.", f.Pointer)
		}
		// Tag/name probing may inspect an ordinary field reference without
		// borrowing its object shape into a union branch. Newly retained
		// hierarchy nested DTOs keep their source node and full edge budget.
		if target, e := a.lookupSchemaReference(reference, f.Pointer); e == nil && (a.deferredNominals[target.Schema] != nil || target.SourceCatalog != nil && (target.ComposedLexical || target.DirectLexical && (a.hierarchySources[target.SourceCatalog.Schema] || unionNominalCandidate(target.Schema) || enumNominalCandidate(target.Schema))) && (directInternalModel(target.Schema, target.Pointer) || composedModelCandidate(target.Schema) || unionNominalCandidate(target.Schema) || enumNominalCandidate(target.Schema))) {
			bound, e := a.resolveSchema(reference, f.Pointer)
			if e != nil {
				return nil, "", e
			}
			if bound.ModelName != "" || bound.Deferred != nil {
				if e := depthCheck(f.Depth+max(1, bound.Steps), f.Pointer); e != nil {
					return nil, "", e
				}
				return bound.Schema, bound.Pointer, nil
			}
		}
		// A typed field probe may follow an internal chain to an existing
		// primitive wrapper or product union. This inspects tag metadata only;
		// ordinary typed planning rechecks the retained physical declaration.
		if target, e := a.lookupSchemaReference(reference, f.Pointer); e == nil && target.Catalog == nil && has(target.Schema, "$ref") {
			bound, e := a.resolveInternalReferenceChain(target, f.Pointer)
			if e == nil && (bound.PhysicalValueClass != nil || bound.CanonicalTemporal != nil || bound.CanonicalBuiltin != nil || bound.CanonicalBigint != nil || bound.CanonicalBytes != nil || bound.CanonicalCallerNative != nil || bound.CanonicalJSON != nil || bound.CanonicalCover != nil || bound.CanonicalCollection != nil || bound.BorrowedCollection != nil || bound.BorrowedScalar != nil || (bound.ModelName != "" || bound.Deferred != nil) && composedModelCandidate(bound.Schema) && a.containsObjectCompositionUnion(bound.Schema, 0, map[*document.Value]bool{})) {
				if e := depthCheck(f.Depth+max(1, bound.Steps), f.Pointer); e != nil {
					return nil, "", e
				}
				return bound.Schema, bound.Pointer, nil
			}
		}
		if target, e := a.resolveSchema(reference, f.Pointer); e == nil {
			if a.admitBorrowedBuiltin(target, f.Pointer) || a.admitBorrowedTemporal(target, f.Pointer) || a.admitBorrowedBigint(target, f.Pointer) || a.admitBorrowedBytes(target, f.Pointer) || a.admitBorrowedCallerNative(target, f.Pointer) || a.admitBorrowedRawTrue(target, f.Pointer) || a.admitBorrowedRecursiveCover(target, f.Pointer) {
				if e := depthCheck(f.Depth+max(1, target.Steps), f.Pointer); e != nil {
					return nil, "", e
				}
				return target.Schema, target.Pointer, nil
			}
			// Nominal field metadata follows the actual typed declaration. It
			// does not borrow a nullable DTO/union's shape into this variant.
			if a.bindNamedChainModel(target, f.Pointer) || target.ModelName != "" || target.Deferred != nil {
				if e := depthCheck(f.Depth+max(1, target.Steps), f.Pointer); e != nil {
					return nil, "", e
				}
				return target.Schema, target.Pointer, nil
			}
			// A direct internal collection probe uses the same original source
			// authority as a root alias; it adds no catalog or exported type.
			// Typed planning still validates the retained leaf and all checks.
			if target.Catalog == nil && target.BorrowedCollection == nil {
				if e := a.bindBorrowedRootCollection(target, f.Pointer); e != nil {
					return nil, "", e
				}
			}
			if target.CanonicalCollection != nil || target.BorrowedCollection != nil {
				if e := depthCheck(f.Depth+max(1, target.Steps), f.Pointer); e != nil {
					return nil, "", e
				}
				return target.Schema, target.Pointer, nil
			}
		}
		component, _, err := a.resolveNominal(reference, f.Pointer, f.Depth)
		if err != nil {
			return nil, "", err
		}
		return component.Schema, component.Pointer, nil
	}
	return f.Schema, f.Pointer, nil
}
func (a *analyzer) fieldPins(f *field) ([]string, *diagnostic.Diagnostic) {
	if f == nil {
		return []string{}, nil
	}
	v, p, err := a.actualField(f)
	if err != nil {
		return nil, err
	}
	if err := object(v, p); err != nil {
		return nil, err
	}
	if literal, ok := v.Get("const"); ok {
		if structuredLiteral(v) {
			// Validate the original declaration before excluding data from tags.
			if _, err := a.fieldConstant(f.Schema, "validation", f.Wire, f.Pointer, f.Depth, false, true); err != nil {
				return nil, err
			}
			return nil, nil
		}
		return []string{jsonPin(literal)}, nil
	}
	pins := []string{}
	if get(v, "enum").Kind() == document.Array {
		for _, value := range get(v, "enum").Items() {
			if value.Kind() == document.Null && nullableStringEnumNulls(v, p) {
				continue // Nullable codecs own null; enum tags retain string tokens.
			}
			pins = append(pins, jsonPin(value))
		}
	}
	return pins, nil
}

// commonType retains common-value projection. Dictionaries and actual nominal
// references share typed planning; inline owned shapes keep separate policy.
func (a *analyzer) commonType(f *field) (TypeRef, *diagnostic.Diagnostic) {
	if err := depthCheck(f.Depth, f.Pointer); err != nil {
		return TypeRef{}, err
	}
	if has(f.Schema, "default") && has(f.Schema, "enum") {
		return a.enumDefaultType(*f, "Validation", nil, nil)
	}
	if has(f.Schema, "default") {
		ty, _, err := a.modelDefaultType(*f, false, "Validation", nil, nil)
		return ty, err
	}
	if c, err := a.fieldConstant(f.Schema, "validation", f.Wire, f.Pointer, f.Depth, false, true); err != nil {
		return TypeRef{}, err
	} else if c != nil && structuredConstant(*c) {
		return c.Type, nil // Body constants ignore optional/nullable declarations.
	}
	if ty, dictionary, err := a.commonDictionaryType(f); dictionary || err != nil {
		return ty, err
	}
	if ty, nominal, err := a.commonNominalType(f); nominal || err != nil {
		return ty, err
	}
	// A complete cover is an explicitly authorized transparent JSON value,
	// not a borrowed union shape. New chain common fields use the same typed
	// resolver/source-height/nullable decision as ordinary fields.
	if has(f.Schema, "$ref") {
		target, err := a.resolveSchema(text(get(f.Schema, "$ref")), f.Pointer)
		if err != nil {
			return TypeRef{}, err
		}
		if target.CanonicalCover != nil {
			return a.referenceType(target, nullableOverride(f.Schema), f.Depth, f.Pointer)
		}
	}
	v, p, err := a.actualField(f)
	if err != nil {
		return TypeRef{}, err
	}
	_, ownNull, err := schemaType(f.Schema, f.Pointer)
	if err != nil {
		return TypeRef{}, err
	}
	kind, nullable, err := schemaType(v, p)
	if err != nil {
		return TypeRef{}, err
	}
	if has(v, "allOf") || has(v, "oneOf") || has(v, "anyOf") || has(v, "$ref") {
		return TypeRef{}, unsupported(p)
	}
	var ty TypeRef
	if has(v, "const") {
		c, e := constant("validation", f.Wire, v, p, f.Depth)
		if e != nil {
			return TypeRef{}, e
		}
		ty = c.Type
	} else if has(v, "enum") {
		if kind == "" || kind == "string" {
			if _, err := buildStringEnum("Validation", v, p); err != nil {
				return TypeRef{}, err
			}
			ty = TypeRef{Kind: Scalar, Name: "kotlin.String"}
		} else {
			validation := *a
			validation.warnings = nil
			scalar, e := validation.nativeEnum(kind, v, p)
			if e != nil {
				return TypeRef{}, e
			}
			ty = TypeRef{Kind: Scalar, Name: scalar}
		}
	} else {
		switch kind {
		case "string", "integer", "number", "boolean":
			scalar, e := scalarType(kind, v, p)
			if e != nil {
				return TypeRef{}, e
			}
			ty = TypeRef{Kind: Scalar, Name: scalar}
		case "array":
			if err := checkKeys(v, []string{"type", "nullable", "items"}, p); err != nil {
				return TypeRef{}, err
			}
			items, exists := v.Get("items")
			if !exists {
				return TypeRef{}, invalid("Array schemas require items.", p)
			}
			if err := object(items, naming.Pointer(p, "items")); err != nil {
				return TypeRef{}, err
			}
			item, e := a.commonType(&field{Wire: f.Wire, Schema: items, Pointer: naming.Pointer(p, "items"), Depth: f.Depth + 1})
			if e != nil {
				return TypeRef{}, e
			}
			ty = TypeRef{Kind: List, Element: &item}
		case "object":
			if props := get(v, "properties"); props.Kind() == document.Object && len(props.Members()) > 0 {
				return TypeRef{}, unsupported(p)
			}
			if _, err := a.objectShape(v, p, f.Depth); err != nil {
				return TypeRef{}, err
			}
			ty = TypeRef{Kind: JSON}
		case "":
			if err := checkKeys(v, []string{"type", "nullable"}, p); err != nil {
				return TypeRef{}, err
			}
			ty = TypeRef{Kind: JSON}
		default:
			return TypeRef{}, unsupported(p)
		}
	}
	if nullable || ownNull {
		ty = ty.Null()
	}
	return ty, nil
}

func variantName(value string, hasValue bool, index int, used map[string]bool) string {
	var spelling strings.Builder
	if hasValue {
		// Match ASCII lowercase followed by Unicode first-character uppercase,
		// then retain ASCII identifier characters only.
		lower := strings.Map(func(r rune) rune {
			if r >= 'A' && r <= 'Z' {
				return r + 32
			}
			return r
		}, value)
		for _, word := range strings.FieldsFunc(lower, func(r rune) bool { return r == '_' || r == '-' || r == ' ' }) {
			first := true
			for _, r := range word {
				if first {
					first = false
					if expansion := asciiFullUpper(r); expansion != "" {
						spelling.WriteString(expansion)
						continue
					}
					r = unicode.ToUpper(r)
				}
				if r < 128 && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_') {
					spelling.WriteRune(r)
				}
			}
		}
	}
	base := spelling.String()
	if base == "" {
		base = "Variant" + strconv.Itoa(index+1)
	} else if base[0] >= '0' && base[0] <= '9' {
		base = "V" + base
	}
	return uniqueVariantName(base, index, used)
}

// Go's Unicode simple-case table does not expand a first character. These
// full-uppercase expansions are the ones whose ASCII part survives the
// accepted discriminator-name filter; other non-ASCII uppercase parts vanish.
func asciiFullUpper(r rune) string {
	switch r {
	case 'ß':
		return "SS"
	case 'ŉ':
		return "N"
	case 'ǰ':
		return "J"
	case 'ẖ':
		return "H"
	case 'ẗ':
		return "T"
	case 'ẘ':
		return "W"
	case 'ẙ':
		return "Y"
	case 'ẚ':
		return "A"
	case 'ﬀ':
		return "FF"
	case 'ﬁ':
		return "FI"
	case 'ﬂ':
		return "FL"
	case 'ﬃ':
		return "FFI"
	case 'ﬄ':
		return "FFL"
	case 'ﬅ', 'ﬆ':
		return "ST"
	}
	return ""
}

func uniqueVariantName(base string, index int, used map[string]bool) string {
	name := base
	for n := index + 1; used[strings.ToLower(name)]; n++ {
		name = base + strconv.Itoa(n)
	}
	used[strings.ToLower(name)] = true
	return name
}

func (a *analyzer) objectUnion(union *unionView, name string, depth int, standalone bool) (*ObjectUnion, *diagnostic.Diagnostic) {
	return a.objectUnionContext(union, name, depth, standalone, false)
}

// requestRoot is enabled only for an operation's nominal UnionSealed body.
// Primitive/array variants there are empty, unreachable object singletons, not
// the native payload wrappers used in component and nested property contexts.
func (a *analyzer) objectUnionContext(union *unionView, name string, depth int, standalone, requestRoot bool) (*ObjectUnion, *diagnostic.Diagnostic) {
	type resolvedBranch struct {
		schema  *document.Value
		pointer string
		depth   int
		shape   *shape
		empty   bool
	}
	var resolved []resolvedBranch
	for _, branch := range union.Branches {
		if branch.Shape != nil {
			resolved = append(resolved, resolvedBranch{schema: branch.Schema, pointer: branch.Pointer, depth: branch.Depth, shape: branch.Shape})
			continue
		}
		v, p, d, err := a.resolveUnionBranch(branch, depth)
		if err != nil {
			return nil, err
		}
		kind, nullable, err := schemaType(v, p)
		if err != nil {
			return nil, err
		}
		memberNullable := nullable && kind == "object" && !has(v, "allOf")
		if has(v, "allOf") && has(branch.Schema, "$ref") {
			bound, admitted, err := a.nullableObjectReferenceMember(branch, depth)
			if err != nil {
				return nil, err
			}
			memberNullable = admitted && bound.Schema == v && bound.Pointer == p
		}
		if nullable && !memberNullable {
			return nil, unsupported(p)
		}
		if kind != "object" && !(kind == "" && has(v, "allOf")) {
			if !requestRoot {
				return nil, nil
			}
			pinnedScalar := kind == "" && (has(v, "enum") || has(v, "const"))
			if kind != "string" && kind != "boolean" && kind != "integer" && kind != "number" && kind != "array" && !pinnedScalar {
				return nil, unsupported(p)
			}
			// Validate the original borrowed schema before projecting it. In
			// particular, do not drop malformed array items, refs or keywords.
			if err := a.rejectValidationProjection(v, p); err != nil {
				return nil, err
			}
			var scratch []Declaration
			if _, err := a.operationValueType(v, "$RequestVariant", p, &scratch, map[string]bool{}, d); err != nil {
				return nil, err
			}
			resolved = append(resolved, resolvedBranch{schema: v, pointer: p, depth: d, shape: &shape{}, empty: true})
			a.warn("W_HTTP_REQUEST_UNION_PROJECTION", "A non-object root request union variant is an empty singleton encoding an empty object, not its declared payload kind; use Unknown for raw JSON payloads.", branch.Pointer)
			continue
		}
		shape, err := a.objectShape(v, p, d)
		if err != nil {
			return nil, err
		}
		if shape.Nullable != nil && *shape.Nullable && !memberNullable {
			return nil, unsupported(p)
		}
		resolved = append(resolved, resolvedBranch{schema: v, pointer: p, depth: d, shape: shape})
	}
	structuredWires := map[string]bool{}
	for _, branch := range resolved {
		for _, f := range branch.shape.Fields {
			c, err := a.fieldConstant(f.Schema, "validation", f.Wire, f.Pointer, f.Depth, false, true)
			if err != nil {
				return nil, err
			}
			structuredWires[f.Wire] = structuredWires[f.Wire] || c != nil && structuredConstant(*c)
		}
	}
	common := make([]Property, 0)
	for _, first := range resolved[0].shape.Fields {
		present := true
		for _, branch := range resolved {
			f := shapeField(branch.shape, first.Wire)
			present = present && f != nil && !inlineVariantValue(f)
		}
		if !present {
			continue
		}
		var types []TypeRef
		nullable := false
		for _, branch := range resolved {
			f := shapeField(branch.shape, first.Wire)
			ty, err := a.commonType(f)
			if err != nil {
				return nil, err
			}
			types = append(types, ty)
			// A direct const ignores optional presence; ref-owned consts preserve
			// the accepted caller's original-field test before body projection.
			nullable = nullable || !has(f.Schema, "const") && (!branch.shape.Required[first.Wire] || ty.Kind == Nullable)
		}
		ty := bare(types[0])
		same := true
		for _, source := range types {
			same = same && reflect.DeepEqual(ty, bare(source))
		}
		if !same {
			continue
		}
		if nullable {
			ty = ty.Null()
		}
		property, err := naming.PropertyName(first.Wire, first.Pointer)
		if err != nil {
			return nil, err
		}
		if strings.EqualFold(property, "rawElement") {
			return nil, diagnostic.At("E_NAME_COLLISION", "Common property collides with the Unknown payload field.", first.Pointer)
		}
		common = append(common, Property{Name: property, WireName: first.Wire, Type: ty})
	}
	varies := func(wire string) (bool, *diagnostic.Diagnostic) {
		var first []string
		for i, branch := range resolved {
			pins, err := a.fieldPins(shapeField(branch.shape, wire))
			if err != nil {
				return false, err
			}
			if i == 0 {
				first = pins
			} else if !slices.Equal(first, pins) {
				return true, nil
			}
		}
		return false, nil
	}
	candidates := func(hoist bool) ([]string, *diagnostic.Diagnostic) {
		wires := []string{}
		for _, p := range common {
			wires = append(wires, p.WireName)
		}
		if hoist {
			if i := slices.Index(wires, "type"); i >= 0 {
				wires = append([]string{"type"}, append(slices.Clone(wires[:i]), wires[i+1:]...)...)
			}
		}
		for _, branch := range resolved {
			for _, f := range branch.shape.Fields {
				if !slices.Contains(wires, f.Wire) {
					wires = append(wires, f.Wire)
				}
			}
		}
		wires = slices.DeleteFunc(wires, func(wire string) bool { return structuredWires[wire] })
		tiers := map[string]int{}
		for _, wire := range wires {
			any, nonString := false, true
			for _, branch := range resolved {
				if f := shapeField(branch.shape, wire); f != nil {
					v, _, err := a.actualField(f)
					if err != nil {
						return nil, err
					}
					if has(v, "const") || has(v, "enum") {
						any = true
						literal := get(v, "const")
						nonString = nonString && literal != nil && literal.Kind() != document.String
					}
				}
			}
			tier := 2
			if any {
				tier = 0
				if nonString {
					tier = 1
				}
			}
			tiers[wire] = tier
		}
		sort.SliceStable(wires, func(i, j int) bool { return tiers[wires[i]] < tiers[wires[j]] })
		return wires, nil
	}
	dispatchCandidates, err := candidates(true)
	if err != nil {
		return nil, err
	}
	tagField := ""
	hasVarying := false
	for _, wire := range dispatchCandidates {
		varies, err := varies(wire)
		if err != nil {
			return nil, err
		}
		if varies {
			tagField = wire
			hasVarying = true
			break
		}
	}
	if tagField == "" {
		for _, wire := range dispatchCandidates {
			if commonProperty(common, wire) != nil {
				tagField = wire
				break
			}
		}
	}
	if tagField == "" {
		tagField = "type"
	}
	// A JSON data field named type must not conflict with the default
	// sealed class discriminator. Without scalar tags it retains raw Unknown.
	dispatches := !standalone || commonProperty(common, "type") != nil || structuredWires["type"] || hasVarying
	namingCandidates, err := candidates(false)
	if err != nil {
		return nil, err
	}
	model := &ObjectUnion{ClassName: name, Common: common, TagField: tagField, NullVariant: standalone && union.Nullable}
	used := map[string]bool{strings.ToLower(name): true, "companion": true}
	variantPins := [][]string{}
	for index, branch := range resolved {
		var selected, fallback []string
		for _, wire := range namingCandidates {
			pins, err := a.fieldPins(shapeField(branch.shape, wire))
			if err != nil {
				return nil, err
			}
			if len(pins) == 0 {
				continue
			}
			if fallback == nil {
				fallback = pins
			}
			varies, err := varies(wire)
			if err != nil {
				return nil, err
			}
			if varies {
				selected = pins
				break
			}
		}
		values := selected
		if values == nil {
			values = fallback
		}
		reference, isRef := get(union.Branches[index].Schema, "$ref").Text()
		variant := ""
		if standalone && isRef {
			component, err := a.resolve(reference, union.Branches[index].Pointer)
			if err != nil {
				// A newly admitted internal member owns a fresh union variant,
				// independently of the canonical DTO it borrows. Derive this new
				// subtype name from the original terminal Pointer, never a guessed
				// physical classifier or an alias-owned nested namespace.
				target, admitted, failure := a.nullableObjectReferenceMember(union.Branches[index], depth)
				if failure != nil {
					return nil, failure
				}
				if !admitted || target.Schema != branch.schema || target.Pointer != branch.pointer {
					return nil, err
				}
				wire, failure := naming.DecodePointerSegment(target.Pointer[strings.LastIndexByte(target.Pointer, '/')+1:], target.Pointer)
				if failure != nil {
					return nil, failure
				}
				local, failure := naming.ClassName(wire, target.Pointer)
				if failure != nil {
					return nil, failure
				}
				variant = uniqueVariantName(local, index, used)
			} else {
				variant = uniqueVariantName(component.Name, index, used)
			}
		} else {
			value := ""
			if len(values) > 0 {
				value = values[0]
			}
			variant = variantName(value, len(values) > 0, index, used)
		}
		if standalone {
			values = nil
			if dispatches && !structuredWires[tagField] {
				values, err = a.fieldPins(shapeField(branch.shape, tagField))
				if err != nil {
					return nil, err
				}
				if len(values) == 0 && isRef && !structuredWires["type"] {
					values = []string{variant}
				}
			}
		}
		context := *a
		if context.usage != SharedUsage {
			context.usage = RequestVariantUsage
		}
		value := &Model{ClassName: variant}
		if !branch.empty {
			if union.Branches[index].Shape != nil {
				value, err = context.buildModelFromShape(variant, branch.schema, branch.pointer, branch.shape, common, !standalone, nil, false)
			} else {
				value, err = context.buildModelContext(variant, branch.schema, branch.pointer, branch.depth, common, !standalone)
			}
		}
		a.warnings = context.warnings
		if err != nil {
			return nil, err
		}
		if len(value.Properties) == 0 && len(value.Constants) > 0 && strings.EqualFold(value.ClassName, "Serializer") {
			return nil, diagnostic.At("E_NAME_COLLISION", "Const-only variant name conflicts with its singleton-owned serializer.", union.Branches[index].Pointer)
		}
		model.Variants = append(model.Variants, value)
		variantPins = append(variantPins, values)
	}
	claimed, selected := map[string]bool{}, map[int]bool{}
	for index, pins := range variantPins {
		for _, pin := range pins {
			if !claimed[pin] {
				claimed[pin] = true
				selected[index] = true
				model.Tags = append(model.Tags, DispatchRule{pin, index})
			}
		}
	}
	if len(resolved) >= 2 {
		for index, branch := range resolved {
			if selected[index] {
				continue
			}
			for _, wire := range branch.shape.RequiredOrder {
				unique := true
				for j, other := range resolved {
					if j != index && (other.shape.Required[wire] || shapeField(other.shape, wire) != nil) {
						unique = false
						break
					}
				}
				if unique {
					model.Presence = append(model.Presence, DispatchRule{wire, index})
					selected[index] = true
					break
				}
			}
		}
	}
	if union.Branches[0].Shape != nil {
		// A Cartesian composition can have several distinct tag fields. Each
		// rule conjoins all fixed pins, in source order; first full match wins.
		model.Tags = nil
		selected = map[int]bool{}
		for index, branch := range resolved {
			var matches []ObjectTagMatch
			for _, f := range branch.shape.Fields {
				if structuredWires[f.Wire] {
					continue
				}
				pins, err := a.fieldPins(&f)
				if err != nil {
					return nil, err
				}
				if len(pins) > 0 {
					matches = append(matches, ObjectTagMatch{f.Wire, pins})
				}
			}
			if len(matches) > 0 {
				duplicate := slices.ContainsFunc(model.TagCombinations, func(rule ObjectTagRule) bool { return reflect.DeepEqual(rule.Matches, matches) })
				if !duplicate {
					model.TagCombinations = append(model.TagCombinations, ObjectTagRule{matches, index})
					selected[index] = true
				}
			}
		}
	}
	custom := dispatches || len(model.Presence) > 0 || len(model.TagCombinations) > 0
	if !custom {
		model.SerialNames = make([]string, 0, len(model.Variants))
		unique := map[string]bool{}
		for index, variant := range model.Variants {
			serialName := variant.ClassName
			for _, f := range resolved[index].shape.Fields {
				if !structuredWires[f.Wire] && commonProperty(common, f.Wire) != nil {
					v, _, err := a.actualField(&f)
					if err != nil {
						return nil, err
					}
					if has(v, "const") {
						serialName = jsonPin(get(v, "const"))
					}
					break
				}
			}
			if unique[serialName] {
				return nil, diagnostic.At("E_NAME_COLLISION", "Default-polymorphic subtype serial names collide.", union.Pointer)
			}
			unique[serialName] = true
			model.SerialNames = append(model.SerialNames, serialName)
		}
		// This descriptor context remains deferred; do not import pending
		// Rust CLASS-singleton behavior merely because code exists there.
		for _, variant := range model.Variants {
			if len(variant.Properties) == 0 && len(variant.Constants) > 0 && !slices.ContainsFunc(variant.Constants, structuredConstant) {
				return nil, unsupported(union.Pointer)
			}
		}
	}
	for index, branch := range union.Branches {
		if !selected[index] {
			message := "No generated tag or exclusive-required-key branch selects this variant; unrecognized payloads use Unknown."
			if !custom {
				message = "No field-tag or key-presence branch selects this variant; default sealed polymorphism requires a class discriminator instead."
			}
			a.warn("W_UNION_UNREACHABLE_VARIANT", message, branch.Pointer)
		}
	}
	message := "Object unions use tag/key-presence dispatch, not exact oneOf match-count or schema validation."
	if !custom {
		message = "A root union uses default sealed polymorphism and requires a class discriminator, not schema-match validation."
	}
	a.warn("W_OBJECT_UNION_APPROXIMATION", message, union.Pointer)
	return model, nil
}

func reserveObjectUnion(model *ObjectUnion, union *unionView, rootPointer string, standalone bool, used map[string]bool) *diagnostic.Diagnostic {
	type peer struct{ name, pointer string }
	generated := []peer{}
	if model.NullVariant {
		for _, name := range []string{"Null" + model.ClassName, "Null" + model.ClassName + "Serializer"} {
			generated = append(generated, peer{name, union.Pointer})
		}
	}
	pointer := union.Pointer
	if standalone {
		pointer = rootPointer
	}
	if model.Content() || model.NullVariant {
		generated = append(generated, peer{model.ClassName + "Serializer", pointer})
	}
	if !standalone {
		generated = append(generated, peer{model.ClassName, pointer})
	}
	if model.Content() {
		for _, name := range []string{"Unknown" + model.ClassName, "Unknown" + model.ClassName + "Serializer"} {
			generated = append(generated, peer{name, pointer})
		}
	}
	for _, peer := range generated {
		if used[strings.ToLower(peer.name)] {
			return diagnostic.At("E_NAME_COLLISION", "Union-owned declarations collide with another declaration or an owner-reserved name.", peer.pointer)
		}
		used[strings.ToLower(peer.name)] = true
	}
	return nil
}
