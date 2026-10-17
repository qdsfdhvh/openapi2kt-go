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
// Validation reuses the property default walk. Every rule that governs a
// property default - declared members, required names, leaf kinds, enum
// vocabularies, counts and bounds, and the value budgets - applies unchanged;
// only the reporting root differs, at the original `.../default/...` Pointer.

// componentDefaultCandidate reports the components whose root may own the
// annotation. Only ordinary object models can carry the factory; every other
// component role (scalar alias, enum, union, collection alias, const, raw, and
// hierarchy) keeps its existing contract and is not extended here.
func (a *analyzer) componentDefaultCandidate(c *component) bool {
	if c == nil || !has(c.Schema, "default") {
		return false
	}
	// A scalar or String-enum component owns its value outright, so the factory
	// returns the alias type directly instead of a constructor.
	if c.Scalar != "" || has(c.Schema, "enum") {
		return a.scalarComponentDefaultCandidate(c)
	}
	if c.InternalModelTarget != nil || c.BindingTarget != nil || c.ValueClass ||
		c.NullableStructuredConst || c.NullableTemporal || c.NullableBigint || c.NullableBytes ||
		c.NullableCallerNative || c.NullablePrimitive || c.NullableNativeEnum || c.NullableEnum ||
		c.NullableRaw || c.RecursiveJSON || c.BooleanRoot || c.NullableHierarchy {
		return false
	}
	if c.Scalar != "" || has(c.Schema, "enum") || has(c.Schema, "$ref") ||
		has(c.Schema, "oneOf") || has(c.Schema, "anyOf") || has(c.Schema, "const") {
		return false
	}
	kind, nullable, err := schemaType(c.Schema, c.Pointer)
	if err != nil || kind != "object" || nullable {
		return false
	}
	return objectDefaultCandidateSchema(c.Schema)
}

// planComponentDefaultFactories attaches the canonical annotation to each
// eligible model and validates it through the shared object-default walk. It
// runs after declaration planning so the shared type plan already accepted the
// annotated node.
func (a *analyzer) planComponentDefaultFactories(declarations []Declaration) *diagnostic.Diagnostic {
	for _, name := range sortedComponentNames(a.components) {
		c := a.components[name]
		if !a.componentDefaultCandidate(c) {
			continue
		}
		if c.Scalar != "" || has(c.Schema, "enum") {
			encoded, err := a.scalarComponentDefaultEncoded(c)
			if err != nil {
				return err
			}
			switch declaration := componentDeclaration(declarations, c.Name).(type) {
			case *StringEnum:
				declaration.ComponentDefault = encoded
			case *ScalarAlias:
				declaration.ComponentDefault = encoded
			}
			continue
		}
		model := componentModel(declarations, c.Name)
		if model == nil {
			// The component planned to a non-model declaration; its own default
			// belongs to that role's separate contract.
			continue
		}
		encoded, err := a.componentDefaultEncoded(c, model)
		if err != nil {
			return err
		}
		model.ComponentDefault = encoded
	}
	return nil
}

// componentDefaultEncoded validates the annotation on its own shallow copy so
// no declaration or warning leaks from the probe, then returns canonical JSON.
func (a *analyzer) componentDefaultEncoded(c *component, model *Model) (string, *diagnostic.Diagnostic) {
	value := get(c.Schema, "default")
	at := naming.Pointer(c.Pointer, "default")
	work, bytes := 0, 0
	if err := nestedDefaultBudget(value, at, 0, &work, &bytes); err != nil {
		return "", err
	}
	validation := *a
	validation.warnings = nil
	validation.objectDefaultWork = nil
	// The shared type plan already ran and saw this original node. Register it so
	// the object walk accepts the annotation it owns, exactly as a field's own
	// default registers its source node.
	if validation.objectDefaultSchemas == nil {
		validation.objectDefaultSchemas = map[*document.Value]bool{}
	}
	validation.objectDefaultSchemas[c.Schema] = true
	field := field{Wire: model.ClassName, Pointer: c.Pointer, Schema: c.Schema, Depth: 0}
	node, err := validation.objectDefaultNode(c.Schema, c.Pointer, 0, true)
	if err != nil {
		return "", err
	}
	encoded, err := node.value(value, at)
	if err != nil {
		return "", err
	}
	if value.Kind() == document.Null {
		// A root-level null keeps its literal form but still requires the
		// component's own explicit nullability, which the walk above enforced.
		return "null", nil
	}
	if len(encoded) > maxNestedDefaultBytes {
		return "", diagnostic.At("E_MODEL_LIMIT", "Canonical component default exceeds the supported 32 KiB JSON byte limit.", at)
	}
	if !json.Valid([]byte(encoded)) {
		return "", diagnostic.At("E_JSON_INVALID", "Component default must encode to valid JSON.", at)
	}
	_ = field
	return encoded, nil
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

// markComponentDefaultSource registers only the components whose root `default`
// this contract admits, so the shared type plan accepts the annotation on the
// same original node it reuses. Anything else is left untouched and keeps its
// existing rejection at the original `.../default` Pointer.
func (a *analyzer) markComponentDefaultSource(v *document.Value, pointer string) *diagnostic.Diagnostic {
	if err := object(v, pointer); err != nil {
		return err
	}
	if has(v, "$ref") || has(v, "allOf") || has(v, "oneOf") || has(v, "anyOf") ||
		has(v, "const") || dictionarySchema(v) {
		return nil
	}
	kind, nullable, err := schemaType(v, pointer)
	if err != nil {
		return nil
	}
	if kind != "object" {
		// A plain scalar or String-enum component root accepts the annotation
		// through its own field walk. Native markers, formats, temporal, bytes
		// and value-class roots keep their existing rejection.
		if nullable || has(v, "format") || has(v, "x-native-type") || has(v, "x-kotlin-value-class") {
			return nil
		}
		if has(v, "enum") {
			if kind != "" && kind != "string" {
				return nil
			}
		} else if kind != "string" && kind != "integer" && kind != "number" && kind != "boolean" {
			return nil
		}
		if a.componentDefaultSchemas == nil {
			a.componentDefaultSchemas = map[*document.Value]bool{}
		}
		a.componentDefaultSchemas[v] = true
		return nil
	}
	// Only a non-empty declared property map becomes an ordinary Model. Empty,
	// raw, const-only and nullable raw projections keep their own role and their
	// existing rejection at the original `.../default` Pointer.
	if nullable || !objectDefaultCandidateSchema(v) {
		return nil
	}
	if a.objectDefaultSchemas == nil {
		a.objectDefaultSchemas = map[*document.Value]bool{}
	}
	a.objectDefaultSchemas[v] = true
	return nil
}

// scalarComponentDefaultCandidate admits the plain scalar and String-enum roots
// whose field default role already exists. Native enum, temporal, bytes,
// caller-native, bigint and value-class components keep their own contracts.
func (a *analyzer) scalarComponentDefaultCandidate(c *component) bool {
	kind, nullable, err := schemaType(c.Schema, c.Pointer)
	if err != nil || nullable {
		return false
	}
	if has(c.Schema, "format") || has(c.Schema, "x-native-type") || has(c.Schema, "x-kotlin-value-class") ||
		has(c.Schema, "const") || has(c.Schema, "oneOf") || has(c.Schema, "anyOf") ||
		has(c.Schema, "allOf") || has(c.Schema, "$ref") {
		return false
	}
	if has(c.Schema, "enum") {
		return (kind == "" || kind == "string") && c.Scalar == ""
	}
	switch kind {
	case "string", "integer", "number", "boolean":
		return c.Scalar != ""
	}
	return false
}

// scalarComponentDefaultEncoded validates the annotation through the same
// field-enum / scalar walk and returns canonical JSON.
func (a *analyzer) scalarComponentDefaultEncoded(c *component) (string, *diagnostic.Diagnostic) {
	at := naming.Pointer(c.Pointer, "default")
	value := get(c.Schema, "default")
	work, bytes := 0, 0
	if err := nestedDefaultBudget(value, at, 0, &work, &bytes); err != nil {
		return "", err
	}
	validation := *a
	validation.warnings = nil
	field := field{Wire: c.Name, Pointer: c.Pointer, Schema: c.Schema, Depth: 0}
	if has(c.Schema, "enum") {
		if _, err := validation.enumDefaultType(field, c.Name, nil, nil); err != nil {
			return "", err
		}
	} else if err := validation.scalarComponentValue(c.Schema, value, at); err != nil {
		return "", err
	}
	encoded, err := value.MarshalJSON()
	if err != nil {
		return "", diagnostic.At("E_JSON_INVALID", "Component default must encode to valid JSON.", at)
	}
	if len(encoded) > maxNestedDefaultBytes {
		return "", diagnostic.At("E_MODEL_LIMIT", "Canonical component default exceeds the supported 32 KiB JSON byte limit.", at)
	}
	return string(encoded), nil
}

// scalarComponentValue validates a plain scalar component default against its
// declared kind, mirroring the field scalar default rules.
func (a *analyzer) scalarComponentValue(schema, value *document.Value, at string) *diagnostic.Diagnostic {
	kind, _, err := schemaType(schema, at)
	if err != nil {
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

func componentDeclaration(declarations []Declaration, name string) Declaration {
	for _, declaration := range declarations {
		if declaration.Name() == name {
			return declaration
		}
	}
	return nil
}
