package schema

import (
	"encoding/json"
	"slices"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Direct inline ordinary DTO object field own defaults. This role is narrower
// than the inline object type planner: it accepts only plain inline objects
// without references, composition, dictionaries, consts or nominal leaves. The
// field type is still planned by the shared ordinary property planner, so the
// declaration set and public type do not depend on the annotation.
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
	if has(v, "$ref") {
		source, _, err := a.defaultSource(text(get(v, "$ref")), "")
		if err != nil {
			return false
		}
		return objectDefaultCandidateSchema(source)
	}
	return objectDefaultCandidateSchema(v)
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

// objectDefaultNode mirrors the accepted plain-object and List/typed Map source
// grammar. Property and container elements retain the existing leaf/container
// planner, which validates their own grammar, bounds, counts and enum
// vocabularies against original nodes.
func (a *analyzer) objectDefaultNode(v *document.Value, pointer string, depth int, allowOwnDefault bool) (*nestedDefaultNode, *diagnostic.Diagnostic) {
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
	if nullable {
		node.ty = node.ty.Null()
	}
	declared := map[string]bool{}
	for _, member := range properties.Members() {
		declared[member.Name] = true
		childPointer := naming.Pointer(naming.Pointer(pointer, "properties"), member.Name)
		var child *nestedDefaultNode
		if objectDefaultCandidateSchema(member.Value) {
			child, err = a.objectDefaultNode(member.Value, childPointer, depth+1, true)
		} else {
			// Leaf/container children are validated by the existing planner. It
			// never creates declarations here, so identity stays with the field
			// type plan performed before this walk.
			child, err = a.nestedDefaultSchema(member.Value, "Validation", childPointer, nil, nil, depth+1, true)
		}
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

// objectDefaultChild dispatches a container element. Object-containing trees
// recurse here; a direct local `$ref` to a plain object target borrows that
// target; every other element stays with the existing leaf/container planner.
// Element schemas cannot own the default annotation themselves.
func (a *analyzer) objectDefaultChild(v *document.Value, pointer string, depth int) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	if has(v, "$ref") {
		if err := checkKeys(v, []string{"$ref", "nullable"}, pointer); err != nil {
			return nil, err
		}
		source, sourcePointer, err := a.defaultSource(text(get(v, "$ref")), pointer)
		if err != nil {
			return nil, err
		}
		if !objectDefaultCandidateSchema(source) {
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
	if objectDefaultCandidateSchema(v) {
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
	ty, _, err := a.modelPropertyType(f.Schema, name, f.Pointer, nested, used, f.Depth)
	if err != nil {
		return TypeRef{}, err
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
	node, err := a.objectDefaultNode(f.Schema, f.Pointer, f.Depth, true)
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
