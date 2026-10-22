package schema

import (
	"encoding/json"
	"slices"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Direct inline ordinary DTO object field own defaults. The root uses a plain
// inline object or a separately admitted allOf/collection role. Its properties
// may use the bounded reference, inline allOf, union and scalar-const walks
// below. The shared ordinary property planner owns all declarations and public
// types independently of the annotation.
//
// The default value validates every declared property and requires each
// required wire name. Property values reuse the existing leaf/container
// default planner; nested plain objects recurse through this same walk.

type objectDefaultProperty struct {
	wire     string
	required bool
	node     *nestedDefaultNode
}

// objectDefaultCandidateSchema is the exact source grammar for this role. It
// deliberately excludes constructs owned by separate reference/composition/
// dictionary/const/union roles so existing rejections keep their behavior. It
// returns true when the tree is a plain object candidate or a List/typed Map
// whose element tree contains one, so scalar-only collections stay with the
// existing nested collection profile.
func objectDefaultCandidateSchema(v *document.Value) bool {
	if v == nil {
		return false
	}
	kind, _, err := schemaType(v, "")
	if err != nil {
		return false
	}
	if kind == "array" {
		return objectDefaultCandidateSchema(get(v, "items"))
	}
	if dictionarySchema(v) {
		return objectDefaultCandidateSchema(get(v, "additionalProperties"))
	}
	for _, key := range []string{"$ref", "allOf", "oneOf", "anyOf", "const", "enum", "additionalProperties", "format", "x-native-type", "x-kotlin-value-class"} {
		if has(v, key) {
			return false
		}
	}
	if kind != "object" {
		return false
	}
	properties, exists := v.Get("properties")
	return exists && properties.Kind() == document.Object && len(properties.Members()) > 0
}

// objectDefaultCandidate is the analyzer-aware gate: it accepts a plain object,
// a List/typed Map whose element tree contains one, and (for collection
// elements) a direct local `$ref` that resolves to a plain object target.
func (a *analyzer) objectDefaultCandidate(v *document.Value) bool {
	if v == nil {
		return false
	}
	if inlineAllOfDefaultElement(v) != nil {
		return true
	}
	kind, _, err := schemaType(v, "")
	if err != nil {
		return false
	}
	if kind == "array" {
		return a.objectDefaultCandidate(get(v, "items"))
	}
	if dictionarySchema(v) {
		return a.objectDefaultCandidate(get(v, "additionalProperties"))
	}
	// Follow finite transparent local `$ref` chains to the plain object a
	// container element or field resolves to, mirroring the consuming-field
	// role. Chains ending in a scalar, enum, composition or collection are not
	// object targets and keep their own roles.
	for depth := 0; depth <= MaxSchemaDepth && has(v, "$ref"); depth++ {
		source, _, err := a.defaultSource(text(get(v, "$ref")), "")
		if err != nil {
			return false
		}
		v = source
	}
	return a.primitiveUnionElementCandidate(v) || objectDefaultCandidateSchema(v)
}

// objectDefaultReferenceTarget follows finite transparent local `$ref` chains
// to the plain object a container element or field resolves to, returning that
// original terminal node and Pointer. It reuses the consuming-field source walk
// and never expands references; a chain ending in a scalar, enum, composition
// or collection is not an object target and keeps its own role.
func (a *analyzer) objectDefaultReferenceTarget(v *document.Value) (*document.Value, string, bool) {
	pointer := ""
	for depth := 0; depth <= MaxSchemaDepth && has(v, "$ref"); depth++ {
		source, p, err := a.defaultSource(text(get(v, "$ref")), pointer)
		if err != nil {
			return nil, "", false
		}
		v, pointer = source, p
	}
	if objectDefaultCandidateSchema(v) {
		return v, pointer, true
	}
	return nil, "", false
}

func objectDefaultRequiredNames(v *document.Value, pointer string) (map[string]bool, *diagnostic.Diagnostic) {
	required := map[string]bool{}
	value, ok := v.Get("required")
	if !ok {
		return required, nil
	}
	if value.Kind() != document.Array {
		return nil, invalid("required must be an array.", pointer)
	}
	for _, entry := range value.Items() {
		wire, ok := entry.Text()
		if !ok {
			return nil, invalid("required entries must be strings.", pointer)
		}
		if required[wire] {
			return nil, invalid("required must contain unique declared property names.", pointer)
		}
		required[wire] = true
	}
	return required, nil
}

// objectDefaultStep bounds source planning independently of annotation size.
// Repeated references can otherwise expand a small nominal DAG exponentially,
// including for empty/null values or annotations suppressed after validation.
// Keep the counter on a shallow validation copy, scoped to one source walk.
func (a *analyzer) objectDefaultStep(pointer string) (*analyzer, *diagnostic.Diagnostic) {
	if a.objectDefaultWork == nil {
		validation := *a
		work := 0
		validation.objectDefaultWork = &work
		a = &validation
	}
	*a.objectDefaultWork++
	if *a.objectDefaultWork > maxNestedDefaultNodes {
		return nil, diagnostic.At("E_MODEL_LIMIT", "Object default source planning exceeds the supported 16,384 steps.", pointer)
	}
	return a, nil
}

// objectDefaultNode mirrors the accepted plain-object and List/typed Map source
// grammar. Property and container elements retain the existing leaf/container
// planner, which validates their own grammar, bounds, counts and enum
// vocabularies against original nodes.
func (a *analyzer) objectDefaultNode(v *document.Value, pointer string, depth int, allowOwnDefault bool) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	a, err := a.objectDefaultStep(pointer)
	if err != nil {
		return nil, err
	}
	if err := depthCheck(depth, pointer); err != nil {
		return nil, err
	}
	if err := object(v, pointer); err != nil {
		return nil, err
	}
	kind, nullable, err := schemaType(v, pointer)
	if err != nil {
		return nil, err
	}
	if kind == "array" {
		return a.objectDefaultContainer(v, pointer, depth, TypeRef{Kind: List, Element: &TypeRef{Kind: JSON}}, allowOwnDefault)
	}
	if dictionarySchema(v) {
		if required, ok := v.Get("required"); ok && (required.Kind() != document.Array || len(required.Items()) != 0) {
			return nil, invalid("Dictionary schemas cannot require undeclared fixed properties.", naming.Pointer(pointer, "required"))
		}
		return a.objectDefaultContainer(v, pointer, depth, TypeRef{Kind: Map, Element: &TypeRef{Kind: JSON}}, allowOwnDefault)
	}
	if kind != "object" {
		return nil, unsupported(naming.Pointer(pointer, "type"))
	}
	keys := []string{"type", "nullable", "properties", "required"}
	if a.componentDefaultCounts(v, pointer) {
		keys = append(keys, "minProperties", "maxProperties")
	}
	if allowOwnDefault {
		// A property may declare its own default. Its nested model plan owns that
		// annotation; this walk only validates the outer supplied value.
		keys = append(keys, "default")
	}
	if err := checkKeys(v, keys, pointer); err != nil {
		return nil, err
	}
	properties, exists := v.Get("properties")
	if !exists || properties.Kind() != document.Object || len(properties.Members()) == 0 {
		return nil, unsupported(pointer)
	}
	required, err := objectDefaultRequiredNames(v, pointer)
	if err != nil {
		return nil, err
	}
	node := &nestedDefaultNode{schema: v, pointer: pointer, ty: TypeRef{Kind: JSON}, object: true}
	if a.componentDefaultCounts(v, pointer) {
		checked, err := a.checkedCollection(TypeRef{Kind: JSONObject}, v, pointer)
		if err != nil {
			return nil, err
		}
		node.objectCounts = checked.Collection
	}
	if nullable {
		node.ty = node.ty.Null()
	}
	declared := map[string]bool{}
	for _, member := range properties.Members() {
		declared[member.Name] = true
		childPointer := naming.Pointer(naming.Pointer(pointer, "properties"), member.Name)
		child, err := a.objectDefaultPropertyNode(member.Value, childPointer, depth+1, true)
		if err != nil {
			return nil, err
		}
		node.objectFields = append(node.objectFields, objectDefaultProperty{wire: member.Name, required: required[member.Name], node: child})
	}
	for wire := range required {
		if !declared[wire] {
			return nil, invalid("required must contain unique declared property names.", pointer)
		}
	}
	return node, nil
}

// objectDefaultPropertyNode dispatches one object property, including direct
// inline ordinary allOf constructors. Leaf/container children are validated by
// the existing planner. These walks never create declarations, so identity
// stays with the field type plan.
func (a *analyzer) objectDefaultPropertyNode(v *document.Value, pointer string, depth int, allowOwnDefault bool) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	a, err := a.objectDefaultStep(pointer)
	if err != nil {
		return nil, err
	}
	switch {
	case has(v, "$ref"):
		// A referenced plain object property borrows its resolved target; the
		// consuming DTO serializer still owns the nested codec.
		return a.objectDefaultPropertyReference(v, pointer, depth)
	case has(v, "allOf"):
		return a.inlineAllOfObjectDefaultMember(v, pointer, depth, allowOwnDefault)
	case has(v, "const"):
		return a.objectDefaultConstNode(v, pointer)
	case a.primitiveUnionElementCandidate(v):
		return a.primitiveUnionElementNode(v, pointer, depth)
	case a.allOfDefaultMemberElement(v) != nil:
		// Borrow only this property's original leaf/reference. The enclosing
		// type plan owns its constructor and final nominal binding.
		validation := *a
		validation.objectDefaultAllOfMemberElement = a.allOfDefaultMemberElement(v)
		return validation.objectDefaultNode(v, pointer, depth, allowOwnDefault)
	case a.objectDefaultCandidate(v):
		return a.objectDefaultNode(v, pointer, depth, allowOwnDefault)
	default:
		return a.nestedDefaultSchema(v, "Validation", pointer, nil, nil, depth, true)
	}
}

// objectDefaultShapeNode validates a merged original object shape. Fields keep
// their borrowed source nodes and pointers; no declarations are created. The
// consuming role grants ordinary object members their own default annotations.
func (a *analyzer) objectDefaultShapeNode(s *shape, pointer string, allowOwnDefault bool) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	a, err := a.objectDefaultStep(pointer)
	if err != nil {
		return nil, err
	}
	node := &nestedDefaultNode{pointer: pointer, ty: TypeRef{Kind: JSON}, object: true}
	declared := map[string]bool{}
	for _, f := range s.Fields {
		declared[f.Wire] = true
		child, err := a.objectDefaultPropertyNode(f.Schema, f.Pointer, f.Depth, allowOwnDefault)
		if err != nil {
			return nil, err
		}
		node.objectFields = append(node.objectFields, objectDefaultProperty{wire: f.Wire, required: s.Required[f.Wire], node: child})
	}
	for _, wire := range s.RequiredOrder {
		if !declared[wire] {
			return nil, invalid("required must contain unique declared property names.", pointer)
		}
	}
	return node, nil
}

// objectDefaultContainer builds a List/typed Map node whose element tree may
// contain inline DTO objects. Non-object elements keep the existing planner.
func (a *analyzer) objectDefaultContainer(v *document.Value, pointer string, depth int, kind TypeRef, allowOwnDefault bool) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	edge := "items"
	keys := a.collectionKeys([]string{"type", "nullable", "items"})
	if kind.Kind == Map {
		edge = "additionalProperties"
		keys = a.collectionKeys([]string{"type", "nullable", "properties", "required", "additionalProperties"})
	}
	if allowOwnDefault {
		keys = append(keys, "default")
	}
	if err := checkKeys(v, keys, pointer); err != nil {
		return nil, err
	}
	child, err := a.objectDefaultChild(get(v, edge), naming.Pointer(pointer, edge), depth+1)
	if err != nil {
		return nil, err
	}
	node := &nestedDefaultNode{schema: v, pointer: pointer, element: child}
	kind.Element = &child.ty
	ty, err := a.checkedCollection(kind, v, pointer)
	if err != nil {
		return nil, err
	}
	node.ty = ty
	if _, nullable, _ := schemaType(v, pointer); nullable {
		node.ty = node.ty.Null()
	}
	return node, nil
}

// objectDefaultConstNode validates a scalar const property inside a defaulted
// object. Structured const bodies keep their separate role.
func (a *analyzer) objectDefaultConstNode(v *document.Value, pointer string) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	if err := checkKeys(v, []string{"type", "nullable", "const"}, pointer); err != nil {
		return nil, err
	}
	kind, nullable, err := schemaType(v, pointer)
	if err != nil {
		return nil, err
	}
	if nullable {
		return nil, unsupported(pointer)
	}
	literal := get(v, "const")
	if kind == "" {
		// A typeless scalar const infers its kind from the literal token.
		switch literal.Kind() {
		case document.String:
			kind = "string"
		case document.Boolean:
			kind = "boolean"
		case document.Number:
			if unionIntegerToken(literal) {
				kind = "integer"
			} else if unionNumberToken(literal) {
				kind = "number"
			} else {
				return nil, unsupported(pointer)
			}
		default:
			return nil, unsupported(pointer)
		}
	}
	if kind != "string" && kind != "integer" && kind != "number" && kind != "boolean" {
		return nil, unsupported(pointer)
	}
	return &nestedDefaultNode{schema: v, pointer: pointer, ty: TypeRef{Kind: JSON}, constValue: literal, constKind: kind, native: a}, nil
}

// objectDefaultTransparentLeaf only classifies a source for the member walk.
// The shared field planner still owns reference admission and runtime checks;
// nestedDefaultSchema validates the original leaf grammar and vocabulary.
func objectDefaultTransparentLeaf(v *document.Value, kind string) bool {
	return isBuiltinScalarFactoryRoot(v, kind) || isTemporalScalarRoot(v, kind) || isBytesScalarRoot(v, kind) || enumNominalCandidate(v) || nullableNativeEnumCandidate(v, kind)
}

// objectDefaultTransparentTerminal routes finite transparent `$ref` chains to
// plain objects or supported scalar/enum leaves, without
// consuming the source-work budget. The walk below keeps identity/work checks.
func (a *analyzer) objectDefaultTransparentTerminal(v *document.Value) *document.Value {
	for depth := 0; depth <= MaxSchemaDepth && has(v, "$ref"); depth++ {
		source, _, err := a.defaultSource(text(get(v, "$ref")), "")
		if err != nil {
			return nil
		}
		v = source
	}
	kind, _, _ := schemaType(v, "")
	if objectDefaultCandidateSchema(v) || objectDefaultTransparentLeaf(v, kind) {
		return v
	}
	return nil
}

// objectDefaultTransparentChain follows a finite chain of transparent local
// `$ref`s ending in a plain object or supported scalar/enum
// leaf. It keeps every original hop and nearest explicit nullable intent.
// The caller has already confirmed the terminal with
// objectDefaultTransparentTerminal, so every other role keeps its own walk.
func (a *analyzer) objectDefaultTransparentChain(v *document.Value, pointer, reference string, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	var nullable *bool
	for {
		if err := depthCheck(depth, pointer); err != nil {
			return nil, err
		}
		owner := a.componentsByOriginalSchema(v)
		kind, _, _ := schemaType(v, pointer)
		leaf := objectDefaultTransparentLeaf(v, kind)
		if owner == nil && (has(v, "$ref") || leaf) {
			// An internal hop or leaf keeps its original source identity.
			if err := a.internalDefaultMemberSource(v, pointer, reference); err != nil {
				return nil, err
			}
		} else if owner != nil && (owner.Pointer != pointer || reference != "#"+owner.Pointer || a.hierarchySources[v]) {
			return nil, unsupported(naming.Pointer(pointer, "$ref"))
		}
		if !has(v, "$ref") {
			var node *nestedDefaultNode
			var err *diagnostic.Diagnostic
			if leaf {
				// Validation borrows the original vocabulary; the ordinary field
				// planner remains the sole authority for the actual enum classifier.
				node, err = a.nestedDefaultSchema(v, "Validation", pointer, nil, nil, depth, false)
			} else if objectDefaultCandidateSchema(v) {
				node, err = a.objectDefaultNode(v, pointer, depth, false)
			} else {
				return nil, unsupported(pointer)
			}
			if err != nil {
				return nil, err
			}
			if nullable != nil {
				node.ty = node.ty.WithoutRootNull()
				if *nullable {
					node.ty = node.ty.Null()
				}
			}
			return node, nil
		}
		if nullable == nil {
			nullable = nullableOverride(v)
		}
		stepped, stepErr := a.objectDefaultStep(pointer)
		if stepErr != nil {
			return nil, stepErr
		}
		a = stepped
		if err := checkKeys(v, []string{"$ref", "nullable"}, pointer); err != nil {
			return nil, err
		}
		reference = text(get(v, "$ref"))
		source, sourcePointer, err := a.defaultSource(reference, pointer)
		if err != nil {
			return nil, err
		}
		v, pointer = source, sourcePointer
		depth++
	}
}

// objectDefaultPropertyReference resolves an object property's direct local
// `$ref`. Plain object and named/internal ordinary allOf targets recurse as
// objects; finite named/internal allOf chains preserve original hops and nullable
// intent. Builtin, temporal, bytes and String/native enum chains share that walk;
// other terminal roles remain separate.
func (a *analyzer) objectDefaultPropertyReference(v *document.Value, pointer string, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	if err := checkKeys(v, []string{"$ref", "nullable", "default"}, pointer); err != nil {
		return nil, err
	}
	if a.referencedCollectionAllOfMember(v) {
		return a.referencedCollectionAllOfMemberNode(v, pointer, depth)
	}
	source, sourcePointer, err := a.defaultSource(text(get(v, "$ref")), pointer)
	if err != nil {
		return nil, err
	}
	var node *nestedDefaultNode
	if has(source, "allOf") || has(source, "$ref") {
		// Finite transparent chains reuse the original scalar/plain object
		// terminal. Other chains keep their separate terminal roles.
		if a.objectDefaultTransparentTerminal(source) != nil {
			node, err = a.objectDefaultTransparentChain(source, sourcePointer, text(get(v, "$ref")), depth+1)
		} else {
			node, err = a.referencedAllOfObjectDefaultMember(source, sourcePointer, text(get(v, "$ref")), depth+1)
		}
	} else if a.primitiveUnionElementCandidate(source) {
		// A direct local `$ref` to a primitive union component reuses the
		// object-default property's primitive-union member role.
		node, err = a.primitiveUnionElementNode(source, sourcePointer, depth)
	} else if a.objectDefaultCandidate(source) {
		// The analyzer-aware gate also admits a referenced collection whose
		// element resolves to a plain object target, matching the field role.
		node, err = a.objectDefaultNode(source, sourcePointer, depth, false)
	} else if a.objectDefaultReferencedScalarCollection(source) {
		// A referenced List/typed Map whose elements are scalar/temporal/enum
		// leaves reuses the existing collection leaf planner.
		node, err = a.objectDefaultScalarCollectionNode(source, sourcePointer, depth)
	} else {
		node, err = a.nestedDefaultSchema(source, "Validation", sourcePointer, nil, nil, depth, false)
	}
	if err != nil {
		return nil, err
	}
	if has(v, "nullable") {
		_, nullable, err := schemaType(v, pointer)
		if err != nil {
			return nil, err
		}
		node.ty = node.ty.WithoutRootNull()
		if nullable {
			node.ty = node.ty.Null()
		}
	}
	return node, nil
}

// objectDefaultChild dispatches a container element. Object-containing trees
// recurse here; a direct local `$ref` to a plain object target borrows that
// target; every other element stays with the existing leaf/container planner.
// Element schemas cannot own the default annotation themselves.
func (a *analyzer) objectDefaultChild(v *document.Value, pointer string, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	a, err := a.objectDefaultStep(pointer)
	if err != nil {
		return nil, err
	}
	if v == a.objectDefaultAllOfMemberElement && v != nil {
		if has(v, "$ref") {
			// Element annotations are narrower than property annotations. Keep
			// the complete original chain and terminal's existing source gate.
			if err := checkKeys(v, []string{"$ref", "nullable"}, pointer); err != nil {
				return nil, err
			}
			return a.objectDefaultPropertyReference(v, pointer, depth)
		}
		return a.inlineAllOfObjectDefaultMember(v, pointer, depth, false)
	}
	if v == a.objectDefaultAllOfElement && v != nil {
		return a.ordinaryAllOfDefaultNode(v, pointer, depth)
	}
	if has(v, "$ref") {
		if err := checkKeys(v, []string{"$ref", "nullable"}, pointer); err != nil {
			return nil, err
		}
		source, sourcePointer, err := a.defaultSource(text(get(v, "$ref")), pointer)
		if err != nil {
			return nil, err
		}
		// Follow any finite transparent alias chain to the plain object terminal
		// so an element alias chain matches the consuming-field role. The exact
		// original code below still handles a direct plain object target.
		if target, targetPointer, ok := a.objectDefaultReferenceTarget(v); ok {
			source, sourcePointer = target, targetPointer
		}
		if !a.objectDefaultCandidate(source) {
			return nil, unsupported(naming.Pointer(pointer, "$ref"))
		}
		node, err := a.objectDefaultNode(source, sourcePointer, depth, false)
		if err != nil {
			return nil, err
		}
		// The element's nearest explicit nullability overrides the target intent.
		if has(v, "nullable") {
			_, nullable, err := schemaType(v, pointer)
			if err != nil {
				return nil, err
			}
			node.ty = TypeRef{Kind: JSON}
			if nullable {
				node.ty = node.ty.Null()
			}
		}
		return node, nil
	}
	if a.primitiveUnionElementCandidate(v) {
		return a.primitiveUnionElementNode(v, pointer, depth)
	}
	if objectDefaultCandidateSchema(v) ||
		a.objectDefaultAllOfElement != nil && inlineAllOfDefaultElement(v) == a.objectDefaultAllOfElement ||
		a.objectDefaultAllOfMemberElement != nil && a.allOfDefaultMemberElement(v) == a.objectDefaultAllOfMemberElement ||
		a.objectDefaultPlainDTOMember && a.plainDTOCollectionMember(v) {
		return a.objectDefaultNode(v, pointer, depth, false)
	}
	return a.nestedDefaultSchema(v, "Validation", pointer, nil, nil, depth, false)
}

// objectDefaultField plans the shared field type and fully validates the own
// annotation. Warnings remain owned by the type plan, mirroring the existing
// nested collection default role.
func (a *analyzer) objectDefaultField(f field, ordinary bool, name string, nested *[]Declaration, used map[string]bool) (TypeRef, *diagnostic.Diagnostic) {
	if !ordinary {
		return TypeRef{}, unsupported(naming.Pointer(f.Pointer, "default"))
	}
	if nested == nil {
		scratch := []Declaration{}
		nested = &scratch
	}
	if used == nil {
		used = map[string]bool{}
	}
	// The shared type plan reuses this original node; register it before the
	// nested model shape walk so its own annotation is not rejected as a
	// dangling schema key.
	if a.objectDefaultSchemas == nil {
		a.objectDefaultSchemas = map[*document.Value]bool{}
	}
	a.objectDefaultSchemas[f.Schema] = true
	before := len(*nested)
	ty, _, err := a.modelPropertyType(f.Schema, name, f.Pointer, nested, used, f.Depth)
	if err != nil {
		return TypeRef{}, err
	}
	if has(f.Schema, "allOf") || inlineAllOfDefaultElement(f.Schema) != nil {
		// Only an ordinary constructor DTO acquires this role. Empty/raw and
		// const-only projections, unions and hierarchy declarations stay separate.
		if len(*nested) != before+1 {
			return TypeRef{}, unsupported(naming.Pointer(f.Pointer, "default"))
		}
		model, ok := (*nested)[before].(*Model)
		if !ok || len(model.Properties) == 0 {
			return TypeRef{}, unsupported(naming.Pointer(f.Pointer, "default"))
		}
	}
	validation := *a
	validation.warnings = nil
	if _, err := validation.objectDefaultInitializer(f); err != nil {
		return TypeRef{}, err
	}
	return ty, nil
}

// objectDefaultInitializer revalidates the annotation and returns the decode
// initializer. An explicit null stays the historical literal null.
func (a *analyzer) objectDefaultInitializer(f field) (string, *diagnostic.Diagnostic) {
	validation := *a
	validation.objectDefaultAllOfElement = inlineAllOfDefaultElement(f.Schema)
	a = &validation
	var node *nestedDefaultNode
	var err *diagnostic.Diagnostic
	if has(f.Schema, "allOf") {
		node, err = a.ordinaryAllOfDefaultNode(f.Schema, f.Pointer, f.Depth)
	} else {
		node, err = a.objectDefaultNode(f.Schema, f.Pointer, f.Depth, true)
	}
	if err != nil {
		return "", err
	}
	value := get(f.Schema, "default")
	at := naming.Pointer(f.Pointer, "default")
	work, bytes := 0, 0
	if err := nestedDefaultBudget(value, at, 0, &work, &bytes); err != nil {
		return "", err
	}
	encoded, err := node.value(value, at)
	if err != nil {
		return "", err
	}
	if value.Kind() == document.Null {
		return "null", nil
	}
	if len(encoded) > maxNestedDefaultBytes {
		return "", diagnostic.At("E_MODEL_LIMIT", "Canonical nested default exceeds the supported 32 KiB JSON byte limit.", at)
	}
	return "`openapi-kt default model`(" + naming.Literal(encoded) + ")", nil
}

// objectDefaultValue validates in source member order and sorts only the
// resulting canonical pairs, so diagnostics never move to a sorted key.
func (node *nestedDefaultNode) objectValue(v *document.Value, pointer string) (string, *diagnostic.Diagnostic) {
	if v.Kind() != document.Object {
		return "", invalid("A typed object default must be a JSON object.", pointer)
	}
	if counts := node.objectCounts; counts != nil {
		n := int64(len(v.Members()))
		if counts.Minimum != nil && n < *counts.Minimum || counts.Maximum != nil && n > *counts.Maximum {
			return "", invalid("Object default must satisfy its retained property counts.", pointer)
		}
	}
	index := map[string]*objectDefaultProperty{}
	for i := range node.objectFields {
		index[node.objectFields[i].wire] = &node.objectFields[i]
	}
	type entry struct{ key, value string }
	entries := make([]entry, 0, len(v.Members()))
	present := map[string]bool{}
	for _, member := range v.Members() {
		location := naming.Pointer(pointer, member.Name)
		property, ok := index[member.Name]
		if !ok {
			return "", invalid("A default object key must name a declared property.", location)
		}
		encoded, err := property.node.value(member.Value, location)
		if err != nil {
			return "", err
		}
		present[member.Name] = true
		entries = append(entries, entry{member.Name, encoded})
	}
	for i := range node.objectFields {
		property := &node.objectFields[i]
		if property.required && !present[property.wire] {
			return "", invalid("A default object must provide required property "+property.wire+".", pointer)
		}
	}
	slices.SortFunc(entries, func(x, y entry) int { return strings.Compare(x.key, y.key) })
	members := make([]string, 0, len(entries))
	for _, entry := range entries {
		key, _ := json.Marshal(entry.key)
		members = append(members, string(key)+":"+entry.value)
	}
	return "{" + strings.Join(members, ",") + "}", nil
}

// objectDefaultReferencedScalarCollection reports whether a named/internal
// collection entity resolves its elements to non-object leaves. Such a
// reference reuses the existing nested collection leaf planner, matching the
// consuming-field collection role; inter-container references stay separate.
func (a *analyzer) objectDefaultReferencedScalarCollection(v *document.Value) bool {
	if v == nil || has(v, "$ref") {
		return false
	}
	kind, _, err := schemaType(v, "")
	if err != nil {
		return false
	}
	var element *document.Value
	switch {
	case kind == "array":
		element = get(v, "items")
	case dictionarySchema(v):
		element = get(v, "additionalProperties")
	default:
		return false
	}
	if element == nil || has(element, "oneOf") || has(element, "anyOf") || has(element, "allOf") {
		return false
	}
	if has(element, "$ref") {
		source, _, err := a.defaultSource(text(get(element, "$ref")), "")
		if err != nil {
			return false
		}
		element = source
	}
	elementKind, _, err := schemaType(element, "")
	if err != nil || elementKind == "object" || elementKind == "array" || dictionarySchema(element) {
		return false
	}
	return true
}

// objectDefaultScalarCollectionNode plans the referenced collection's declared
// shape without its own annotation. Element values still follow the existing
// leaf checks, nullable layers and container counts.
func (a *analyzer) objectDefaultScalarCollectionNode(v *document.Value, pointer string, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	kind, _, err := schemaType(v, pointer)
	if err != nil {
		return nil, err
	}
	edge := "items"
	ty := TypeRef{Kind: List, Element: &TypeRef{Kind: JSON}}
	if kind != "array" && dictionarySchema(v) {
		edge = "additionalProperties"
		ty = TypeRef{Kind: Map, Element: &TypeRef{Kind: JSON}}
	}
	keys := a.collectionKeys([]string{"type", "nullable", edge})
	if err := checkKeys(v, keys, pointer); err != nil {
		return nil, err
	}
	edgeSchema := get(v, edge)
	edgePointer := naming.Pointer(pointer, edge)
	if has(edgeSchema, "$ref") {
		// A referenced scalar/enum/temporal element keeps the existing leaf
		// planner against the resolved source.
		if err := checkKeys(edgeSchema, []string{"$ref", "nullable"}, edgePointer); err != nil {
			return nil, err
		}
		source, sourcePointer, err := a.defaultSource(text(get(edgeSchema, "$ref")), edgePointer)
		if err != nil {
			return nil, err
		}
		edgeSchema, edgePointer = source, sourcePointer
	}
	child, err := a.nestedDefaultSchema(edgeSchema, "Validation", edgePointer, nil, nil, depth+1, false)
	if err != nil {
		return nil, err
	}
	ty.Element = &child.ty
	checked, err := a.checkedCollection(ty, v, pointer)
	if err != nil {
		return nil, err
	}
	node := &nestedDefaultNode{schema: v, pointer: pointer, element: child, ty: checked}
	if _, nullable, _ := schemaType(v, pointer); nullable {
		node.ty = node.ty.Null()
	}
	return node, nil
}
