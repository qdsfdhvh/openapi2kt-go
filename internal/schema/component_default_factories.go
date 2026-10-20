package schema

import (
	"encoding/json"
	"slices"
	"strings"

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
	return a.componentDefaultShape(c)
}

// componentDefaultShape reports whether this root shape owns a default role.
// A reference wrapper asks this of its resolved target, because the target need
// not carry a default itself - only be a shape whose role already exists.
func (a *analyzer) componentDefaultShape(c *component) bool {
	if c == nil {
		return false
	}
	kind, nullable, err := schemaType(c.Schema, c.Pointer)
	if err != nil {
		return false
	}
	if has(c.Schema, "$ref") {
		return a.referenceWrapperDefaultCandidate(c)
	}
	// Null uses the same direct source roles as a non-null annotation, including
	// ordinary allOf constructors validated by the shared object walk below.
	// Other nullable compositions keep their separate roles.
	if nullable && get(c.Schema, "default").Kind() == document.Null &&
		(has(c.Schema, "oneOf") || has(c.Schema, "anyOf")) {
		return false
	}
	if nullable && hasObjectCounts(c.Schema) {
		// A counted nullable root is a separate role regardless of its annotation.
		return false
	}
	// A Scalar only exists after the declaration loop, and markComponentDefaultSource
	// runs before it. Key the dispatch off the schema itself so both callers agree.
	if has(c.Schema, "enum") || isPlainScalarRoot(c.Schema, kind) || isTemporalScalarRoot(c.Schema, kind) || isBytesScalarRoot(c.Schema, kind) {
		return a.scalarComponentDefaultCandidate(c)
	}
	switch {
	case has(c.Schema, "allOf"):
		// A disjoint ordinary allOf constructor reuses the field allOf walk.
		return !has(c.Schema, "oneOf") && !has(c.Schema, "anyOf") && !has(c.Schema, "$ref")
	case kind == "array" || dictionarySchema(c.Schema):
		// A List/typed Map root reuses the nested collection default walk, which
		// already scopes the element/value leaves it supports. An *open* dictionary
		// is a raw JsonObject carrier instead, and reuses the field role's raw JSON
		// grammar below.
		if rawJSONComponentRoot(c.Schema) {
			return true
		}
		return a.collectionComponentDefaultCandidate(c)
	case kind == "object":
		// An unconstrained/empty object is a raw JsonElement carrier, matching the
		// field role's raw JSON grammar rather than a DTO projection.
		if rawJSONComponentRoot(c.Schema) {
			return true
		}
		return objectDefaultCandidateSchema(c.Schema)
	}
	return false
}

// rawJSONComponentRoot reports the two raw JSON carrier roots whose field default
// role already exists: an *open* dictionary (`additionalProperties: true`) and an
// unconstrained/empty object. The grammar is borrowed from the field role; this
// adds a component entry point, not a new shape.
func rawJSONComponentRoot(v *document.Value) bool {
	if v == nil {
		return false
	}
	for _, key := range []string{"$ref", "allOf", "oneOf", "anyOf", "enum", "const", "format", "items", "x-native-type", "x-kotlin-value-class"} {
		if has(v, key) {
			return false
		}
	}
	if dictionarySchema(v) {
		flag, ok := get(v, "additionalProperties").Bool()
		return ok && flag
	}
	if properties, ok := v.Get("properties"); ok {
		if properties.Kind() != document.Object || len(properties.Members()) != 0 {
			return false
		}
	}
	kind, _, err := schemaType(v, "")
	if err != nil {
		return false
	}
	return kind == "" || kind == "object"
}

// markComponentDefaultSource registers only the components whose root `default`
// this contract admits, so the shared type plan accepts the annotation on the
// same original node it reuses. Anything else keeps its existing rejection at
// the original `.../default` Pointer.
func (a *analyzer) markComponentDefaultSource(v *document.Value, pointer string) *diagnostic.Diagnostic {
	if err := object(v, pointer); err != nil {
		return err
	}
	if has(v, "oneOf") || has(v, "anyOf") || has(v, "const") {
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

	if has(c.Schema, "$ref") {
		return a.referenceWrapperDefaultEncoded(c, &validation)
	}

	kind, _, _ := schemaType(c.Schema, c.Pointer)
	if has(c.Schema, "enum") || isPlainScalarRoot(c.Schema, kind) || isTemporalScalarRoot(c.Schema, kind) || isBytesScalarRoot(c.Schema, kind) {
		// The enum/scalar roles own their own validation above; the canonical
		// value is the literal JSON of the annotation itself.
		node, err := validation.scalarComponentNode(c)
		if err != nil {
			return "", err
		}
		if value.Kind() != document.Null && isPlainScalarRoot(c.Schema, kind) && hasPrimitiveBounds(c.Schema) {
			if _, err := node.value(value, at); err != nil {
				return "", err
			}
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
	// A raw JSON carrier root validates like the field role: a JsonObject default
	// must be a JSON object and a JsonElement default accepts any JSON value.
	if rawJSONComponentRoot(c.Schema) {
		return a.rawJSONComponentEncoded(c, at, value)
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
	if has(c.Schema, "enum") || isPlainScalarRoot(c.Schema, kind) || isTemporalScalarRoot(c.Schema, kind) || isBytesScalarRoot(c.Schema, kind) {
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
	kind, _, err := schemaType(c.Schema, c.Pointer)
	if err != nil {
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
	// A non-enum `format` is admitted only for the formats whose field default
	// role already exists: the standard temporal scalars and byte/binary. Other
	// formats stay excluded.
	if has(c.Schema, "format") && !isTemporalScalarRoot(c.Schema, kind) && !isBytesScalarRoot(c.Schema, kind) {
		return false
	}
	return isPlainScalarRoot(c.Schema, kind) || isTemporalScalarRoot(c.Schema, kind) || isBytesScalarRoot(c.Schema, kind)
}

// collectionComponentDefaultCandidate admits finite inline List/typed Map
// trees through the shared nested default walk. That walk validates every
// source node, including for empty/null annotations, and retains its leaf,
// child-default, composition, reference and resource boundaries.
func (a *analyzer) collectionComponentDefaultCandidate(c *component) bool {
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
	if value.Kind() == document.Null {
		// Only the direct component role gains null. Reuse explicit schema
		// nullability without widening scalarTargetEncoded's wrapper role.
		if _, err := scalarSchemaDefault(c.Schema, c.Pointer); err != nil {
			return nil, err
		}
	} else if err := a.scalarComponentValue(c.Schema, value, c.Pointer, at); err != nil {
		return nil, err
	}
	rootKind, _, rootErr := schemaType(c.Schema, c.Pointer)
	if rootErr != nil {
		return nil, rootErr
	}
	if isBytesScalarRoot(c.Schema, rootKind) {
		// The shared nested walk deliberately excludes ByteArray, so that
		// collection elements and object-default leaves stay out of the bytes
		// role. Build the root node directly instead of widening that walk.
		return &nestedDefaultNode{schema: c.Schema, pointer: c.Pointer, ty: TypeRef{Kind: Scalar, Name: "kotlin.ByteArray"}}, nil
	}
	return a.nestedDefaultSchema(c.Schema, c.Name, c.Pointer, nil, nil, 0, true)
}

// scalarComponentValue validates a plain scalar component default against its
// declared kind, mirroring the field scalar default rules.
func (a *analyzer) scalarComponentValue(schema, value *document.Value, rootPointer, at string) *diagnostic.Diagnostic {
	kind, _, err := schemaType(schema, "")
	if err != nil {
		return err
	}
	if isTemporalScalarRoot(schema, kind) || isBytesScalarRoot(schema, kind) {
		// The temporal and bytes roles own their own lexeme validation and
		// canonical form; reuse the field grammar so both stay identical. The
		// schema pointer must be the component's own root, otherwise the reported
		// location collapses to a bare `/default`.
		_, err := scalarSchemaDefault(schema, rootPointer)
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

// isBytesScalarRoot reports a byte/binary string root, the shape whose field
// default role the bytes contract already defines. Only the root is admitted:
// collection elements and object-default leaves keep their own exclusions.
func isBytesScalarRoot(v *document.Value, kind string) bool {
	if kind != "string" {
		return false
	}
	switch text(get(v, "format")) {
	case "byte", "binary":
		return true
	}
	return false
}

// referenceWrapperDefaultCandidate admits a `$ref`-only component when the
// reference names a finite local target whose own root default role exists and
// whose alias role is itself admitted. The wrapper keeps its alias identity and
// its own Pointer; the target gains nothing.
//
// This runs during markComponentDefaultSource, before a.components exists, so the
// target is read from the source document instead of the catalog.
func (a *analyzer) referenceWrapperDefaultCandidate(c *component) bool {
	if c == nil ||
		has(c.Schema, "oneOf") || has(c.Schema, "anyOf") || has(c.Schema, "const") {
		return false
	}
	target := a.referenceWrapperDefaultTarget(c)
	if target == nil {
		return false
	}
	targetKind, _, kindErr := schemaType(target.Schema, target.Pointer)
	if kindErr != nil {
		return false
	}
	// A `$ref` to a plain scalar is itself a validation-opt-in alias, so a wrapper
	// must not widen that role; leaving it out keeps the original rejection.
	if !a.options.Validation && isPlainScalarRoot(target.Schema, targetKind) {
		return false
	}
	return a.componentDefaultShape(&component{Wire: target.Pointer, Name: target.Pointer, Pointer: target.Pointer, Schema: target.Schema})
}

// wrapperDefaultSource retains the original terminal and the nearest explicit
// alias intent separately. A nil Nullable keeps the terminal's existing grammar.
type wrapperDefaultSource struct {
	Pointer  string
	Schema   *document.Value
	Nullable *bool
}

// referenceWrapperDefaultTarget follows finite component aliases without copying
// or expanding sources. The nearest explicit nullable wins, just as in the shared
// alias resolver. Defaults are validated independently on their original owners;
// walking through an annotation neither inherits it nor grants its owner a role.
func (a *analyzer) referenceWrapperDefaultTarget(c *component) *wrapperDefaultSource {
	if c == nil {
		return nil
	}
	reference, ok := get(c.Schema, "$ref").Text()
	if !ok {
		return nil
	}
	intent := nullableOverride(c.Schema)
	seen := map[*document.Value]bool{c.Schema: true}
	for steps := 1; steps <= MaxSchemaDepth; steps++ {
		pointer, target, ok := a.componentSchemaAtReference(reference)
		if !ok || seen[target] {
			return nil
		}
		seen[target] = true
		if !has(target, "$ref") {
			return &wrapperDefaultSource{Pointer: pointer, Schema: target, Nullable: intent}
		}
		if checkKeys(target, []string{"$ref", "nullable", "default"}, pointer) != nil {
			return nil
		}
		if _, _, err := schemaType(target, pointer); err != nil {
			return nil
		}
		if intent == nil {
			intent = nullableOverride(target)
		}
		reference, ok = get(target, "$ref").Text()
		if !ok {
			return nil
		}
	}
	return nil
}

// componentSchemaAtReference returns the original schema node a local component
// reference names, without requiring catalog state.
func (a *analyzer) componentSchemaAtReference(reference string) (string, *document.Value, bool) {
	const prefix = "#/components/schemas/"
	if a.componentSchemas == nil || !strings.HasPrefix(reference, prefix) {
		return "", nil, false
	}
	encoded := strings.TrimPrefix(reference, prefix)
	if encoded == "" || strings.ContainsAny(encoded, "/%") {
		return "", nil, false
	}
	wire, err := naming.DecodePointerSegment(encoded, "")
	if err != nil {
		return "", nil, false
	}
	schema, ok := a.componentSchemas.Get(wire)
	return naming.Pointer("/components/schemas", wire), schema, ok
}

// wrapperValidationView builds a throwaway node carrying the target's shape plus
// the wrapper's annotation. The shared default planners read the annotation from
// the schema node, so this lets them validate the wrapper's value without writing
// a key into the target's original node.
func (a *analyzer) wrapperValidationView(target *wrapperDefaultSource, wrapper *document.Value, at string) (*document.Value, *diagnostic.Diagnostic) {
	targetJSON, err := target.Schema.MarshalJSON()
	if err != nil {
		return nil, unsupported(at)
	}
	valueJSON, err := get(wrapper, "default").MarshalJSON()
	if err != nil {
		return nil, diagnostic.At("E_JSON_INVALID", "Component default must encode to valid JSON.", at)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(targetJSON, &fields); err != nil || fields == nil {
		return nil, unsupported(at)
	}
	// Each component owns its annotation. Replace only the temporary view's
	// root default, leaving the target's independent factory and validation on
	// its original source node. Child annotations keep their existing grammar.
	fields["default"] = valueJSON
	// Override only this temporary root view. For OpenAPI 3.1 the type array's
	// null member must also be removed on narrowing; nullable:false alone cannot
	// override schemaType's union of both source forms. Child nullability stays.
	if target.Nullable != nil {
		fields["nullable"], _ = json.Marshal(*target.Nullable)
		if !*target.Nullable {
			var kinds []string
			if json.Unmarshal(fields["type"], &kinds) == nil && len(kinds) > 0 {
				kinds = slices.DeleteFunc(kinds, func(kind string) bool { return kind == "null" })
				if len(kinds) != 1 {
					return nil, unsupported(at)
				}
				fields["type"], _ = json.Marshal(kinds[0])
			}
		}
	}
	combined, err := json.Marshal(fields)
	if err != nil {
		return nil, unsupported(at)
	}
	view, parseErr := document.ParseObject(combined, document.MaxDocumentBytes)
	if parseErr != nil {
		return nil, parseErr
	}
	return view, nil
}

// referenceWrapperDefaultEncoded validates a wrapper's annotation against the
// resolved target's own default role and returns canonical JSON. Every diagnostic
// is reported at the wrapper's original `.../default` Pointer.
func (a *analyzer) referenceWrapperDefaultEncoded(c *component, validation *analyzer) (string, *diagnostic.Diagnostic) {
	at := naming.Pointer(c.Pointer, "default")
	value := get(c.Schema, "default")
	work, bytes := 0, 0
	if err := nestedDefaultBudget(value, at, 0, &work, &bytes); err != nil {
		return "", err
	}
	target := a.referenceWrapperDefaultTarget(c)
	if target == nil {
		return "", unsupported(at)
	}
	view, err := a.wrapperValidationView(target, c.Schema, at)
	if err != nil {
		return "", err
	}
	shadow := &component{Wire: target.Pointer, Name: target.Pointer, Pointer: c.Pointer, Schema: view}
	validation.objectDefaultSchemas[view] = true
	validation.componentDefaultSchemas[view] = true
	// Raw carrier targets retain their own value/null grammar. The ordinary
	// object walk cannot validate JsonElement/JsonObject alias projections.
	if rawJSONComponentRoot(view) {
		return validation.rawJSONComponentEncoded(shadow, at, value)
	}
	// A scalar, enum, temporal or bytes target owns its own leaf grammar; the
	// nested collection walk would misclassify the value for those shapes.
	if validation.scalarLeafTarget(view) {
		kind, _, _ := schemaType(view, c.Pointer)
		if target.Nullable != nil || isPlainScalarRoot(view, kind) && hasPrimitiveBounds(view) {
			// Explicit alias intent and bounded plain scalars use the component
			// leaf role, including source validation for null and annotation checks.
			// Other chains retain their existing scalarTargetEncoded policy.
			node, err := validation.scalarComponentNode(shadow)
			if err != nil {
				return "", err
			}
			if value.Kind() != document.Null && isPlainScalarRoot(view, kind) {
				// Planning retains primitive checks; visiting the annotation is
				// what enforces them. Keep the original JSON token for emission.
				if _, err := node.value(value, at); err != nil {
					return "", err
				}
			}
			encoded, marshalErr := value.MarshalJSON()
			if marshalErr != nil {
				return "", invalid("A component default must encode to valid JSON.", at)
			}
			if len(encoded) > maxNestedDefaultBytes {
				return "", diagnostic.At("E_MODEL_LIMIT", "Canonical component default exceeds the supported 32 KiB JSON byte limit.", at)
			}
			a.warnings = mergeWarnings(a.warnings, validation.warnings)
			return string(encoded), nil
		}
		encoded, scalarErr := validation.scalarTargetEncoded(view, value, c.Pointer, at)
		if scalarErr != nil {
			return "", scalarErr
		}
		a.warnings = mergeWarnings(a.warnings, validation.warnings)
		return encoded, nil
	}
	node, nodeErr := validation.componentDefaultNode(shadow)
	if nodeErr != nil {
		return "", nodeErr
	}
	if target.Nullable != nil {
		node.ty = node.ty.WithoutRootNull()
		if *target.Nullable {
			node.ty = node.ty.Null()
		}
	}
	encoded, valErr := node.value(value, at)
	if valErr != nil {
		return "", valErr
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

// scalarLeafTarget reports a target whose default role is a single leaf value
// rather than a container or object tree.
func (a *analyzer) scalarLeafTarget(view *document.Value) bool {
	kind, _, err := schemaType(view, "")
	if err != nil {
		return false
	}
	return has(view, "enum") || isPlainScalarRoot(view, kind) ||
		isTemporalScalarRoot(view, kind) || isBytesScalarRoot(view, kind)
}

// scalarTargetEncoded validates a leaf target's value with that role's own
// grammar and returns the canonical JSON of the annotation. root is the schema
// location the leaf planners append "default" to; at is the full value Pointer.
func (a *analyzer) scalarTargetEncoded(view, value *document.Value, root, at string) (string, *diagnostic.Diagnostic) {
	if has(view, "enum") {
		field := field{Pointer: root, Schema: view, Depth: 0}
		if err := a.enumDefaultValue(view, root); err != nil {
			return "", err
		}
		kind, _, kindErr := schemaType(view, root)
		if kindErr != nil {
			return "", kindErr
		}
		if kind == "integer" || kind == "number" || kind == "boolean" {
			if _, _, err := a.nativeEnumFieldType(field, true); err != nil {
				return "", err
			}
		}
	} else if err := a.scalarComponentValue(view, value, root, at); err != nil {
		return "", err
	}
	encoded, marshalErr := value.MarshalJSON()
	if marshalErr != nil {
		return "", diagnostic.At("E_JSON_INVALID", "Component default must encode to valid JSON.", at)
	}
	if len(encoded) > maxNestedDefaultBytes {
		return "", diagnostic.At("E_MODEL_LIMIT", "Canonical component default exceeds the supported 32 KiB JSON byte limit.", at)
	}
	return string(encoded), nil
}

// rawJSONComponentEncoded validates a raw JSON carrier component root. The
// grammar is the field role's: an open dictionary maps to JsonObject and needs a
// JSON object annotation, while an unconstrained/empty object maps to JsonElement
// and accepts any JSON value. The canonical JSON is the annotation itself.
func (a *analyzer) rawJSONComponentEncoded(c *component, at string, value *document.Value) (string, *diagnostic.Diagnostic) {
	work, bytes := 0, 0
	if err := nestedDefaultBudget(value, at, 0, &work, &bytes); err != nil {
		return "", err
	}
	// The field role keeps its approximation warning for these carriers; the
	// shared type plan already emitted it once for this node.
	open := dictionarySchema(c.Schema)
	if value.Kind() == document.Null {
		// Only the component's explicit nullable source authorizes Kotlin null;
		// optional consuming properties and raw JSON's JsonNull value do not.
		if _, nullable, err := schemaType(c.Schema, c.Pointer); err == nil && nullable {
			return "null", nil
		}
		return "", unsupported(at)
	}
	if open && value.Kind() != document.Object {
		return "", invalid("A raw object default must be a JSON object.", at)
	}
	encoded, err := value.MarshalJSON()
	if err != nil {
		return "", invalid("A raw JSON default must be valid JSON.", at)
	}
	if len(encoded) > maxNestedDefaultBytes {
		return "", diagnostic.At("E_MODEL_LIMIT", "Canonical component default exceeds the supported 32 KiB JSON byte limit.", at)
	}
	if !json.Valid(encoded) {
		return "", diagnostic.At("E_JSON_INVALID", "Component default must encode to valid JSON.", at)
	}
	return string(encoded), nil
}
