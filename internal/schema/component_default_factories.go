package schema

import (
	"encoding/json"
	"slices"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Component schemas may declare their own root `default`. There is no call site
// for a component, so the annotation never becomes a constructor initializer:
// the class keeps its normal signature and presence classification, and the
// generator adds an opt-in `default()` factory that decodes the canonical JSON
// through the component's own serializer.
//
// Validation reuses the property default walks. Every rule that governs a field
// default - declared members, required names, leaf kinds, enum vocabularies,
// counts and bounds, and the value budgets - applies unchanged; only the
// reporting root differs, at the original `.../default` Pointer. This contract
// adds no new shape grammar: a component root is admitted exactly when the
// equivalent field default would be.

// componentDefaultCandidate reports the components whose root may own the
// annotation. The dispatch mirrors the field default dispatch, so a shape that a
// field would reject stays rejected here too.
func (a *analyzer) componentDefaultCandidate(c *component) bool {
	if c == nil || !has(c.Schema, "default") {
		return false
	}
	kind, nullable, err := schemaType(c.Schema, c.Pointer)
	if err != nil || nullable {
		return false
	}
	// A Scalar only exists after the declaration loop, and markComponentDefaultSource
	// runs before it. Key the dispatch off the schema itself so both callers agree.
	if has(c.Schema, "enum") || isPlainScalarRoot(c.Schema, kind) || isTemporalScalarRoot(c.Schema, kind) {
		return a.scalarComponentDefaultCandidate(c)
	}
	switch {
	case has(c.Schema, "allOf"):
		// A disjoint ordinary allOf constructor reuses the field allOf walk.
		return !has(c.Schema, "oneOf") && !has(c.Schema, "anyOf") && !has(c.Schema, "$ref")
	case kind == "array" || dictionarySchema(c.Schema):
		// A List/typed Map root reuses the nested collection default walk, which
		// already scopes the element/value leaves it supports.
		return a.collectionComponentDefaultCandidate(c)
	case kind == "object":
		return objectDefaultCandidateSchema(c.Schema)
	}
	return false
}

// markComponentDefaultSource registers only the components whose root `default`
// this contract admits, so the shared type plan accepts the annotation on the
// same original node it reuses. Anything else keeps its existing rejection at
// the original `.../default` Pointer.
func (a *analyzer) markComponentDefaultSource(v *document.Value, pointer string) *diagnostic.Diagnostic {
	if err := object(v, pointer); err != nil {
		return err
	}
	if has(v, "$ref") || has(v, "oneOf") || has(v, "anyOf") || has(v, "const") {
		return nil
	}
	c := &component{Wire: pointer, Name: pointer, Pointer: pointer, Schema: v}
	if !a.componentDefaultCandidate(c) {
		return nil
	}
	// Both markers are set: the object walk consults objectDefaultSchemas, while
	// the scalar/enum and collection type planners consult componentDefaultSchemas.
	if a.objectDefaultSchemas == nil {
		a.objectDefaultSchemas = map[*document.Value]bool{}
	}
	a.objectDefaultSchemas[v] = true
	if a.componentDefaultSchemas == nil {
		a.componentDefaultSchemas = map[*document.Value]bool{}
	}
	a.componentDefaultSchemas[v] = true
	return nil
}

// planComponentDefaultFactories attaches the canonical annotation to each
// eligible declaration and validates it through the shared default walk. It runs
// after declaration planning so the shared type plan already accepted the node.
func (a *analyzer) planComponentDefaultFactories(declarations []Declaration) *diagnostic.Diagnostic {
	for _, name := range sortedComponentNames(a.components) {
		c := a.components[name]
		if !a.componentDefaultCandidate(c) {
			continue
		}
		encoded, err := a.componentDefaultEncoded(c)
		if err != nil {
			return err
		}
		switch declaration := componentDeclaration(declarations, c.Name).(type) {
		case *Model:
			declaration.ComponentDefault = encoded
		case *StringEnum:
			declaration.ComponentDefault = encoded
		case *ScalarAlias:
			declaration.ComponentDefault = encoded
		case *ValueAlias:
			declaration.ComponentDefault = encoded
		}
	}
	return nil
}

// componentDefaultEncoded validates the annotation on a shallow copy so no
// declaration or warning leaks from the probe, then returns canonical JSON.
func (a *analyzer) componentDefaultEncoded(c *component) (string, *diagnostic.Diagnostic) {
	at := naming.Pointer(c.Pointer, "default")
	value := get(c.Schema, "default")
	work, bytes := 0, 0
	if err := nestedDefaultBudget(value, at, 0, &work, &bytes); err != nil {
		return "", err
	}
	// Do not clear warnings here: the shared type plan owns source warnings such
	// as W_ENUM_NATIVE_APPROXIMATION and they must reach the caller once.
	validation := *a
	validation.objectDefaultWork = nil
	if validation.objectDefaultSchemas == nil {
		validation.objectDefaultSchemas = map[*document.Value]bool{}
	}
	validation.objectDefaultSchemas[c.Schema] = true

	kind, _, _ := schemaType(c.Schema, c.Pointer)
	if has(c.Schema, "enum") || isPlainScalarRoot(c.Schema, kind) || isTemporalScalarRoot(c.Schema, kind) {
		// The enum/scalar roles own their own validation above; the canonical
		// value is the literal JSON of the annotation itself.
		if _, err := validation.scalarComponentNode(c); err != nil {
			return "", err
		}
		encoded, marshalErr := value.MarshalJSON()
		if marshalErr != nil {
			return "", diagnostic.At("E_JSON_INVALID", "Component default must encode to valid JSON.", at)
		}
		if len(encoded) > maxNestedDefaultBytes {
			return "", diagnostic.At("E_MODEL_LIMIT", "Canonical component default exceeds the supported 32 KiB JSON byte limit.", at)
		}
		a.warnings = mergeWarnings(a.warnings, validation.warnings)
		return string(encoded), nil
	}
	node, err := validation.componentDefaultNode(c)
	if err != nil {
		return "", err
	}
	encoded, err := node.value(value, at)
	if err != nil {
		return "", err
	}
	a.warnings = mergeWarnings(a.warnings, validation.warnings)
	if value.Kind() == document.Null {
		return "null", nil
	}
	if len(encoded) > maxNestedDefaultBytes {
		return "", diagnostic.At("E_MODEL_LIMIT", "Canonical component default exceeds the supported 32 KiB JSON byte limit.", at)
	}
	if !json.Valid([]byte(encoded)) {
		return "", diagnostic.At("E_JSON_INVALID", "Component default must encode to valid JSON.", at)
	}
	return encoded, nil
}

// componentDefaultNode dispatches to the walk that owns this root shape.
func (a *analyzer) componentDefaultNode(c *component) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	if has(c.Schema, "allOf") {
		return a.ordinaryAllOfDefaultNode(c.Schema, c.Pointer, 0)
	}
	kind, _, err := schemaType(c.Schema, c.Pointer)
	if err != nil {
		return nil, err
	}
	if has(c.Schema, "enum") || isPlainScalarRoot(c.Schema, kind) || isTemporalScalarRoot(c.Schema, kind) {
		return a.scalarComponentNode(c)
	}
	if kind == "array" || dictionarySchema(c.Schema) {
		return a.nestedDefaultSchema(c.Schema, c.Name, c.Pointer, nil, nil, 0, true)
	}
	return a.objectDefaultNode(c.Schema, c.Pointer, 0, true)
}

// scalarComponentDefaultCandidate admits the plain scalar and String-enum roots
// whose field default role exists. Native enum, temporal, bytes, caller-native,
// bigint and value-class components keep their own contracts.
func (a *analyzer) scalarComponentDefaultCandidate(c *component) bool {
	kind, nullable, err := schemaType(c.Schema, c.Pointer)
	if err != nil || nullable {
		return false
	}
	if has(c.Schema, "x-native-type") || has(c.Schema, "x-kotlin-value-class") ||
		has(c.Schema, "const") || has(c.Schema, "oneOf") || has(c.Schema, "anyOf") ||
		has(c.Schema, "allOf") || has(c.Schema, "$ref") {
		return false
	}
	// An enum vocabulary owns its own format handling: a plain String vocabulary
	// has no format restriction here, and a numeric/Boolean native vocabulary
	// keeps its scalar wire type under its own role.
	if has(c.Schema, "enum") {
		if kind == "" || kind == "string" {
			// A String enum with a format keeps the separate temporal/bytes roles.
			return !has(c.Schema, "format")
		}
		// A Boolean vocabulary has no format of its own; a numeric one needs the
		// explicit native format that identifies its wire type. An unformatted
		// numeric enum component keeps its existing rejection.
		if kind == "boolean" {
			return true
		}
		if !has(c.Schema, "format") {
			return false
		}
		return kind == "integer" || kind == "number"
	}
	// A non-enum `format` is admitted only for the standard temporal scalars,
	// whose field default role already exists. Bytes and other formats stay
	// excluded.
	if has(c.Schema, "format") && !isTemporalScalarRoot(c.Schema, kind) {
		return false
	}
	return isPlainScalarRoot(c.Schema, kind) || isTemporalScalarRoot(c.Schema, kind)
}

// collectionComponentDefaultCandidate admits a List/typed Map root when the
// nested collection default walk accepts the whole tree. Nested containers,
// allOf-wrapped containers, union variants, reference elements, counts and
// uniqueItems keep exactly the boundaries that walk already enforces.
func (a *analyzer) collectionComponentDefaultCandidate(c *component) bool {
	// A component root is a *single* container layer over a leaf element. Nested
	// containers, allOf-wrapped containers and union variants keep exactly the
	// exclusions the field role already enforces.
	kind, nullable, err := schemaType(c.Schema, c.Pointer)
	if err != nil || nullable {
		return false
	}
	edge := ""
	switch {
	case kind == "array":
		edge = "items"
	case dictionarySchema(c.Schema):
		edge = "additionalProperties"
	default:
		return false
	}
	if has(c.Schema, "oneOf") || has(c.Schema, "anyOf") || has(c.Schema, "allOf") {
		return false
	}
	element := get(c.Schema, edge)
	if element == nil {
		return false
	}
	elementKind, elementNullable, err := schemaType(element, naming.Pointer(c.Pointer, edge))
	if err != nil || elementNullable {
		return false
	}
	if elementKind == "array" || dictionarySchema(element) ||
		has(element, "oneOf") || has(element, "anyOf") || has(element, "allOf") || has(element, "$ref") {
		return false
	}
	probe := *a
	probe.warnings = nil
	probe.objectDefaultWork = nil
	if probe.objectDefaultSchemas == nil {
		probe.objectDefaultSchemas = map[*document.Value]bool{}
	}
	probe.objectDefaultSchemas[c.Schema] = true
	_, failure := probe.nestedDefaultSchema(c.Schema, c.Name, c.Pointer, nil, nil, 0, true)
	return failure == nil
}

func componentDeclaration(declarations []Declaration, name string) Declaration {
	for _, declaration := range declarations {
		if declaration.Name() == name {
			return declaration
		}
	}
	return nil
}

func componentModel(declarations []Declaration, name string) *Model {
	for _, declaration := range declarations {
		model, ok := declaration.(*Model)
		if ok && model.ClassName == name {
			return model
		}
	}
	return nil
}

func sortedComponentNames(components map[string]*component) []string {
	names := make([]string, 0, len(components))
	for name := range components {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// isPlainScalarRoot reports a plain scalar schema root without markers, formats
// or composition, which is the shape the scalar component factory admits.
func isPlainScalarRoot(v *document.Value, kind string) bool {
	if kind != "string" && kind != "integer" && kind != "number" && kind != "boolean" {
		return false
	}
	if has(v, "format") || has(v, "x-native-type") || has(v, "x-kotlin-value-class") ||
		has(v, "const") || has(v, "oneOf") || has(v, "anyOf") || has(v, "allOf") ||
		has(v, "$ref") || has(v, "enum") || dictionarySchema(v) {
		return false
	}
	return true
}

// scalarComponentNode validates a scalar or String-enum component default with
// the same leaf grammar and vocabulary membership the field role uses. The
// shared nested walk already owns both shapes, so it is reused directly.
func (a *analyzer) scalarComponentNode(c *component) (*nestedDefaultNode, *diagnostic.Diagnostic) {
	at := naming.Pointer(c.Pointer, "default")
	value := get(c.Schema, "default")
	if has(c.Schema, "enum") {
		// The enum role owns its own vocabulary check and node kind.
		field := field{Wire: c.Name, Pointer: c.Pointer, Schema: c.Schema, Depth: 0}
		kind, _, kindErr := schemaType(c.Schema, c.Pointer)
		if kindErr != nil {
			return nil, kindErr
		}
		if kind == "integer" || kind == "number" || kind == "boolean" {
			// Native numeric/Boolean vocabularies keep their scalar wire type and
			// exact pre-rounding membership check.
			if _, _, err := a.nativeEnumFieldType(field, true); err != nil {
				return nil, err
			}
			return &nestedDefaultNode{schema: c.Schema, pointer: c.Pointer, ty: TypeRef{Kind: JSON}}, nil
		}
		if _, err := a.enumDefaultType(field, c.Name, nil, nil); err != nil {
			return nil, err
		}
		return &nestedDefaultNode{schema: c.Schema, pointer: c.Pointer, ty: TypeRef{Kind: JSON}}, nil
	}
	if err := a.scalarComponentValue(c.Schema, value, at); err != nil {
		return nil, err
	}
	return a.nestedDefaultSchema(c.Schema, c.Name, c.Pointer, nil, nil, 0, true)
}

// scalarComponentValue validates a plain scalar component default against its
// declared kind, mirroring the field scalar default rules.
func (a *analyzer) scalarComponentValue(schema, value *document.Value, at string) *diagnostic.Diagnostic {
	kind, _, err := schemaType(schema, "")
	if err != nil {
		return err
	}
	if isTemporalScalarRoot(schema, kind) {
		// The temporal role owns its own lexeme validation and canonical form.
		_, err := scalarSchemaDefault(schema, "")
		return err
	}
	switch kind {
	case "string":
		if _, ok := value.Text(); !ok {
			return invalid("A String default must be a JSON string.", at)
		}
	case "integer":
		if !unionIntegerToken(value) {
			return invalid("An integer default must be an integral JSON number.", at)
		}
	case "number":
		if !unionNumberToken(value) {
			return invalid("A number default must be a JSON number.", at)
		}
	case "boolean":
		if _, ok := value.Bool(); !ok {
			return invalid("A Boolean default must be true or false.", at)
		}
	default:
		return unsupported(at)
	}
	return nil
}

// isTemporalScalarRoot reports a standard date/date-time scalar root, the two
// formats whose field default role already exists.
func isTemporalScalarRoot(v *document.Value, kind string) bool {
	if kind != "string" {
		return false
	}
	switch text(get(v, "format")) {
	case "date", "date-time":
		return true
	}
	return false
}

// mergeWarnings appends only warnings the caller does not already have, so a
// probe that re-runs the type plan cannot duplicate a source warning.
func mergeWarnings(existing, extra []diagnostic.Diagnostic) []diagnostic.Diagnostic {
	seen := make(map[string]bool, len(existing))
	for _, w := range existing {
		key := w.Code
		if w.Pointer != nil {
			key += "|" + *w.Pointer
		}
		seen[key] = true
	}
	for _, w := range extra {
		key := w.Code
		if w.Pointer != nil {
			key += "|" + *w.Pointer
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		existing = append(existing, w)
	}
	return existing
}
