package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

type tupleBooleanReferenceSource struct {
	Original, Terminal       *document.Value
	Pointer, TerminalPointer string
	Root                     *component
	Declared                 *component
	Steps                    int
	False, Nullable          bool
	Type                     TypeRef
	Signature                string
}

type tupleBooleanReferenceUse struct {
	Source    *tupleBooleanReferenceSource
	Snapshots []namedCollectionSourceType
}

type tupleBooleanReferenceSelection struct {
	Array   *document.Value
	Pointer string
	Source  *tupleBooleanReferenceSource
}

type tupleBooleanReferenceRegistry struct {
	Sources    map[*document.Value]*tupleBooleanReferenceSource
	Order      []*tupleBooleanReferenceSource
	Uses       map[*TypeRef]tupleBooleanReferenceUse
	Selections []tupleBooleanReferenceSelection
}

func (a *analyzer) initTupleBooleanReferences() {
	if a.tupleBooleanRefs == nil {
		a.tupleBooleanRefs = &tupleBooleanReferenceRegistry{Sources: map[*document.Value]*tupleBooleanReferenceSource{}, Uses: map[*TypeRef]tupleBooleanReferenceUse{}}
	}
}

// This role borrows an actual existing named Boolean root and its pure aliases.
// It does not authorize false references in the general resolver or synthesize
// a nominal classifier from a positional source.
func (a *analyzer) tupleBooleanReference(original *document.Value, source string, terminal *document.Value, pointer string, depth int) (tupleScalarSignature, *diagnostic.Diagnostic) {
	declared, err := a.resolveDeclared(text(get(original, "$ref")), source)
	if err != nil {
		return tupleScalarSignature{}, err
	}
	root := declared
	if root.BindingTarget != nil {
		root = root.BindingTarget
	}
	if root.Schema != terminal || root.Pointer != pointer || a.components[root.Wire] != root || !root.FalseRoot && !root.BooleanRoot || root.FalseRoot != a.booleanFalseRoot(terminal) || root.BooleanRoot != a.booleanTrueValue(terminal) {
		return tupleScalarSignature{}, schemaReferenceUnsupported(source)
	}
	target, err := a.resolveSchema(text(get(original, "$ref")), source)
	if err != nil {
		return tupleScalarSignature{}, err
	}
	ty, err := a.referenceTypeRole(target, nullableOverride(original), depth, source, true)
	if err != nil {
		return tupleScalarSignature{}, err
	}
	ty = tupleBooleanReferenceType(ty, root.BooleanRoot)
	if err := a.captureTupleNativeSource(original, source, terminal, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	a.initTupleBooleanReferences()
	r := a.tupleBooleanRefs
	signature := falsePropertySignature(Property{Type: ty})
	if previous := r.Sources[original]; previous != nil {
		if previous.Pointer != source || previous.Terminal != terminal || previous.TerminalPointer != pointer || previous.Root != root || previous.Declared != declared || previous.Steps != 1+declared.AliasHeight || previous.False != root.FalseRoot || previous.Signature != signature {
			return tupleScalarSignature{}, schemaReferenceUnsupported(source)
		}
	} else {
		use := &tupleBooleanReferenceSource{Original: original, Terminal: terminal, Pointer: source, TerminalPointer: pointer, Root: root, Declared: declared, Steps: 1 + declared.AliasHeight, False: root.FalseRoot, Nullable: ty.Kind == Nullable, Type: ty, Signature: signature}
		r.Sources[original] = use
		r.Order = append(r.Order, use)
	}
	kind := "tuple_true"
	var nominal *document.Value
	if root.FalseRoot {
		kind, nominal = "tuple_false_ref", terminal
	}
	return tupleScalarSignature{Kind: kind, Nullable: ty.Kind == Nullable, ModelSource: nominal}, nil
}

func (a *analyzer) tupleBooleanReferenceElement(v *document.Value, pointer string, depth int) (*TypeRef, *diagnostic.Diagnostic) {
	if a.tupleBooleanRefs == nil || a.tupleBooleanRefs.Sources[v] == nil {
		return nil, nil
	}
	source := a.tupleBooleanRefs.Sources[v]
	if source.Pointer != pointer {
		return nil, schemaReferenceUnsupported(pointer)
	}
	target, err := a.resolveSchema(text(get(v, "$ref")), pointer)
	if err != nil {
		return nil, err
	}
	ty, err := a.referenceTypeRole(target, nullableOverride(v), depth, pointer, true)
	if err != nil {
		return nil, err
	}
	ty = tupleBooleanReferenceType(ty, !source.False)
	if falsePropertySignature(Property{Type: ty}) != source.Signature {
		return nil, schemaReferenceUnsupported(pointer)
	}
	return &ty, nil
}

// Cache original hops even if a raw projection or explicit tail erases the
// prefix's reference type. Incoming collection aliases must still pay them.
func (a *analyzer) tupleBooleanReferenceWork(element TypeRef, decision *arrayItemsDecision) TypeRef {
	if !a.aliasPlanning || a.tupleBooleanRefs == nil || len(a.tupleBooleanRefs.Order) == 0 || decision == nil {
		return element
	}
	for _, prefix := range decision.Prefix {
		if source := a.tupleBooleanRefs.Sources[prefix.Schema]; source != nil {
			element.SchemaSteps = max(element.SchemaSteps, source.Steps)
		}
	}
	return element
}

func tupleBooleanReferenceType(t TypeRef, raw bool) TypeRef {
	if !raw {
		return t
	}
	nullable, steps := t.Kind == Nullable, t.SchemaSteps
	t = TypeRef{Kind: JSON}
	if nullable {
		t = t.Null()
	}
	t.SchemaSteps = steps
	return t
}

func (a *analyzer) recordTupleBooleanReferenceUse(t *TypeRef, source *tupleBooleanReferenceSource) {
	a.tupleBooleanRefs.Uses[t] = tupleBooleanReferenceUse{source, dtoOriginalGetterTypes(t)}
}

func (a *analyzer) selectTupleBooleanReference(array *document.Value, pointer string, decision *arrayItemsDecision) {
	if a.tupleBooleanRefs == nil || decision.Projection != nil {
		return
	}
	if source := a.tupleBooleanRefs.Sources[decision.Selected]; source != nil {
		for _, selected := range a.tupleBooleanRefs.Selections {
			if selected.Array == array {
				return
			}
		}
		a.tupleBooleanRefs.Selections = append(a.tupleBooleanRefs.Selections, tupleBooleanReferenceSelection{array, pointer, source})
	}
}

// A nullable false alias admits null; the non-null root admits no values. This
// domain check is only for truthful projection warnings, not type equivalence.
func (a *analyzer) tupleFalseDomain(v *document.Value) (bool, bool) {
	if a.booleanFalseRoot(v) {
		return false, true
	}
	if a.tupleBooleanRefs != nil {
		if source := a.tupleBooleanRefs.Sources[v]; source != nil && source.False {
			return source.Nullable, true
		}
	}
	return false, false
}

func tupleBooleanReferenceUseMatches(t *TypeRef, use tupleBooleanReferenceUse) bool {
	if len(use.Snapshots) == 0 || use.Snapshots[0].Node != t || falsePropertySignature(Property{Type: *t}) != use.Source.Signature {
		return false
	}
	for _, snapshot := range use.Snapshots {
		value := *snapshot.Node
		value.SchemaSteps = 0
		if value != snapshot.Value || namedCollectionChecks(value) != snapshot.Checks {
			return false
		}
	}
	return true
}

func (a *analyzer) validateTupleBooleanReferenceSources(declarations []Declaration, used map[*tupleBooleanReferenceSource]int) *diagnostic.Diagnostic {
	r := a.tupleBooleanRefs
	if r == nil || len(r.Order) == 0 {
		return nil
	}
	if err := a.validateFalseRootBindings(declarations); err != nil {
		return err
	}
	for _, source := range r.Order {
		root := source.Root
		if r.Sources[source.Original] != source || a.components[root.Wire] != root || a.components[source.Declared.Wire] != source.Declared || source.Steps != 1+source.Declared.AliasHeight || root.Schema != source.Terminal || root.Pointer != source.TerminalPointer || root.FalseRoot != source.False || root.BooleanRoot == source.False || falsePropertySignature(Property{Type: source.Type}) != source.Signature {
			return schemaReferenceUnsupported(source.Pointer)
		}
		matches := 0
		for _, d := range declarations {
			if source.False {
				if d, ok := d.(*FalseValue); ok {
					if origin, ok := a.falseValueOrigins[d]; ok && origin.Owner == root {
						matches++
					}
				}
			} else if origin, ok := a.rawTrueOrigins[d]; ok && origin.Owner == root && a.rawTrueSourceMatches(origin) && a.rawTrueDeclaration(d, origin) {
				matches++
			}
		}
		if matches != 1 {
			return schemaReferenceUnsupported(source.Pointer)
		}
	}
	for _, selection := range r.Selections {
		if r.Sources[selection.Source.Original] != selection.Source || used[selection.Source] == 0 {
			return schemaReferenceUnsupported(selection.Pointer)
		}
	}
	return nil
}
