package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// Reference candidacy is separate from the unbounded vocabulary grammar.
// Final retained declarations still authorize every provisional lexical source.
func (a *analyzer) enumReferenceCandidate(v *document.Value) bool {
	return enumNominalCandidate(v) || a.checkedEnumReferenceCandidate(v)
}

func (a *analyzer) checkedEnumReferenceCandidate(v *document.Value) bool {
	return a.options.Validation && directStringEnumFieldBounds(v) && lexicalStringEnumSchema(v)
}

type checkedEnumSource struct {
	Root           *component
	Schema         *document.Value
	Pointer, Bytes string
	Checks         *PrimitiveValidation
	CheckKey       string
	Declarations   map[*StringEnum]bool
	Failure        *diagnostic.Diagnostic
	Getters        []checkedEnumGetter
	RootFactories  []*checkedEnumRootFactory
	GettersFinal   bool
}

type checkedEnumGetter struct {
	Model       *Model
	Index       int
	Name, Wire  string
	Validation  *PrimitiveValidation
	CheckKey    string
	Types       []namedCollectionSourceType
	Provisional bool
}

func (a *analyzer) recordCheckedEnumGetters(model *Model, fields []field) {
	sources := map[string]*document.Value{}
	for _, field := range fields {
		if leaf := inlineStringEnumBoundsLeaf(field.Schema, 0); leaf != nil {
			sources[field.Wire] = leaf
		} else if leaf := a.checkedEnumRootFieldSource(field.Schema, field.Pointer); leaf != nil {
			sources[field.Wire] = leaf
		}
	}
	for i := range model.Properties {
		property := &model.Properties[i]
		checked := a.checkedEnumSources[sources[property.WireName]]
		if checked == nil {
			continue
		}
		checked.Getters = append(checked.Getters, checkedEnumGetter{Model: model, Index: i, Name: property.Name, Wire: property.WireName, Validation: property.Validation, CheckKey: namedCollectionChecks(TypeRef{Validation: property.Validation}), Types: dtoOriginalGetterTypes(&property.Type), Provisional: checked.Root != nil && !checked.GettersFinal})
	}
}

// Follow only the already admitted transparent source path of a retained
// property. Declaration planning has resolved aliases and rejected cycles;
// this lookup grants no reference, composition or child-annotation role.
func (a *analyzer) checkedEnumRootFieldSource(v *document.Value, pointer string) *document.Value {
	for depth := 0; depth <= MaxSchemaDepth; depth++ {
		if a.checkedStringEnumRoot(v, pointer) {
			return v
		}
		if has(v, "$ref") {
			var err *diagnostic.Diagnostic
			v, pointer, err = a.defaultSource(text(get(v, "$ref")), pointer)
			if err != nil {
				return nil
			}
		} else if dictionarySchema(v) {
			v, pointer = get(v, "additionalProperties"), pointer+"/additionalProperties"
		} else if kind, _, _ := schemaType(v, pointer); kind == "array" && !has(v, "prefixItems") {
			v, pointer = get(v, "items"), pointer+"/items"
		} else {
			return nil
		}
	}
	return nil
}

func checkedEnumGetterMatches(getter checkedEnumGetter) bool {
	if getter.Model == nil || getter.Index < 0 || getter.Index >= len(getter.Model.Properties) || len(getter.Types) == 0 {
		return false
	}
	property := &getter.Model.Properties[getter.Index]
	if property.Name != getter.Name || property.WireName != getter.Wire || property.Validation != getter.Validation || namedCollectionChecks(TypeRef{Validation: property.Validation}) != getter.CheckKey {
		return false
	}
	current := dtoOriginalGetterTypes(&property.Type)
	if len(current) != len(getter.Types) {
		return false
	}
	for i, snapshot := range getter.Types {
		if current[i] != snapshot {
			return false
		}
	}
	return true
}

// A forward reference and its original ordinary field share one check object.
// Vocabulary/classifier identity remains with the original retained enum.
func (a *analyzer) checkedEnumValidation(v *document.Value, pointer string) (*PrimitiveValidation, *diagnostic.Diagnostic) {
	if a.checkedEnumSources == nil {
		a.checkedEnumSources = map[*document.Value]*checkedEnumSource{}
	}
	bytes, _ := v.MarshalJSON()
	if source := a.checkedEnumSources[v]; source != nil {
		if source.Schema != v || source.Pointer != pointer || source.Bytes != string(bytes) || namedCollectionChecks(TypeRef{Validation: source.Checks}) != source.CheckKey {
			return nil, schemaReferenceUnsupported(pointer)
		}
		return source.Checks, source.Failure
	}
	checks, err := a.primitiveValidation(v, "kotlin.String", pointer)
	a.checkedEnumSources[v] = &checkedEnumSource{Schema: v, Pointer: pointer, Bytes: string(bytes), Checks: checks, CheckKey: namedCollectionChecks(TypeRef{Validation: checks}), Declarations: map[*StringEnum]bool{}, Failure: err}
	if checks != nil {
		a.validationSources[checks] = pointer
		a.emissionHints[checks] = true
	}
	return checks, err
}

// A real original field owns its annotation independently of its borrowers.
// The final borrowed-enum barrier still requires one retained checked enum.
func (a *analyzer) independentCheckedEnumFieldDefault(v *document.Value, pointer string) bool {
	if !a.checkedEnumReferenceCandidate(v) || !has(v, "default") || a.componentRoot(v, pointer) {
		return false
	}
	checked := a.checkedEnumSources[v]
	if checked == nil || checked.Pointer != pointer || checked.Schema != v || len(checked.Declarations) == 0 {
		return false
	}
	source, err := a.lookupSchemaReference("#"+pointer, pointer)
	return err == nil && source.Schema == v && source.Pointer == pointer && source.SourceCatalog != nil && (source.DirectLexical || source.ComposedLexical)
}
