package schema

import (
	"reflect"
	"slices"
	"sort"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// An internal leaf has source authority, never an invented canonical component
// or standalone declaration. New root bridges share their parsed checks; direct
// historical borrowed uses keep their existing per-use validation objects.
type borrowedScalarSource struct {
	Owner         *component
	OwnerSchema   *document.Value
	Schema        *document.Value
	Pointer       string
	Reference     string
	Type          TypeRef
	Checks        *PrimitiveValidation
	SourceBytes   string
	Vocabulary    *document.Value
	Members       []*document.Value
	SchemaMembers []document.Member
}

type borrowedRootOrigin struct {
	Root     *component
	Schema   *document.Value
	Pointer  string
	Source   *borrowedScalarSource
	Nullable bool
}

func (a *analyzer) bindBorrowedRootScalar(target *resolvedSchema, source string) *diagnostic.Diagnostic {
	if target.BorrowedScalar != nil || target.Catalog != nil || target.SourceCatalog == nil || target.ModelName != "" || target.Deferred != nil {
		return nil
	}
	if err := a.terminalInternalScalar(target.Schema, target.Pointer, source); err != nil {
		return err
	}
	if existing := a.borrowedScalars[target.Schema]; existing != nil {
		target.BorrowedScalar = existing
		return nil
	}
	// Parse exactly the borrowed leaf through the existing typed lowering. Its
	// source checks/hints remain the same; root/path overrides are applied later.
	leaf := *target
	leaf.Nullable, leaf.Steps = nil, 0
	planning := a.aliasPlanning
	a.aliasPlanning = false
	ty, err := a.referenceType(&leaf, nil, 0, source)
	a.aliasPlanning = planning
	if err != nil {
		return err
	}
	if bare(ty).Kind != Scalar {
		return schemaReferenceUnsupported(source)
	}
	lookup := "#" + target.Pointer
	original, err := a.lookupSchemaReference(lookup, source)
	if err != nil || original.Catalog != nil || original.Schema != target.Schema || original.SourceCatalog != target.SourceCatalog {
		return schemaReferenceUnsupported(source)
	}
	bytes, _ := target.Schema.MarshalJSON()
	origin := &borrowedScalarSource{Owner: target.SourceCatalog, OwnerSchema: target.SourceCatalog.Schema, Schema: target.Schema, Pointer: target.Pointer, Reference: lookup, Type: ty, Checks: bare(ty).Validation, SourceBytes: string(bytes)}
	if has(target.Schema, "enum") {
		origin.Vocabulary = get(target.Schema, "enum")
		origin.Members = append([]*document.Value{}, origin.Vocabulary.Items()...)
		origin.SchemaMembers = append([]document.Member{}, target.Schema.Members()...)
	}
	a.borrowedScalars[target.Schema] = origin
	target.BorrowedScalar = origin
	return nil
}

func (a *analyzer) bindNamedBorrowedScalar(target *resolvedSchema) bool {
	return target.Catalog != nil && target.BorrowedScalar != nil && target.Schema == target.BorrowedScalar.Schema && target.Pointer == target.BorrowedScalar.Pointer
}

func (a *analyzer) recordBorrowedRootOrigin(d Declaration, c *component) {
	if c.InternalModelTarget == nil || c.InternalModelTarget.BorrowedScalar == nil {
		return
	}
	a.borrowedRoots[d] = borrowedRootOrigin{c, c.Schema, c.Pointer, c.InternalModelTarget.BorrowedScalar, c.BindingNullable != nil && *c.BindingNullable}
}

func (a *analyzer) validateBorrowedRootBindings(declarations []Declaration) *diagnostic.Diagnostic {
	actual := map[Declaration]int{}
	for _, d := range declarations {
		actual[d]++
	}
	roots := []*component{}
	for _, c := range a.components {
		if c.InternalModelTarget != nil && c.InternalModelTarget.BorrowedScalar != nil {
			roots = append(roots, c)
		}
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i].Pointer < roots[j].Pointer })
	for _, c := range roots {
		origin := c.InternalModelTarget.BorrowedScalar
		lookup, err := a.lookupSchemaReference(origin.Reference, c.Pointer+"/$ref")
		if err != nil || lookup.Catalog != nil || lookup.Schema != origin.Schema || lookup.Pointer != origin.Pointer || lookup.SourceCatalog != origin.Owner || a.components[origin.Owner.Wire] != origin.Owner || origin.Owner.Schema != origin.OwnerSchema || bare(origin.Type).Validation != origin.Checks || a.terminalInternalScalar(origin.Schema, origin.Pointer, c.Pointer+"/$ref") != nil {
			return schemaReferenceUnsupported(c.Pointer + "/$ref")
		}
		bytes, _ := origin.Schema.MarshalJSON()
		if string(bytes) != origin.SourceBytes {
			return schemaReferenceUnsupported(c.Pointer + "/$ref")
		}
		var scalar string
		var checks *PrimitiveValidation
		if origin.Vocabulary != nil {
			if get(origin.Schema, "enum") != origin.Vocabulary || !slices.Equal(origin.Members, origin.Vocabulary.Items()) || !slices.Equal(origin.SchemaMembers, origin.Schema.Members()) {
				return schemaReferenceUnsupported(c.Pointer + "/$ref")
			}
			for _, member := range origin.SchemaMembers {
				if get(origin.Schema, member.Name) != member.Value {
					return schemaReferenceUnsupported(c.Pointer + "/$ref")
				}
			}
			ty, parsed, err := a.nativeEnumFieldType(field{Schema: origin.Schema, Pointer: origin.Pointer}, false)
			if err != nil {
				return err
			}
			scalar, checks = bare(ty).Name, parsed
		} else if a.options.Validation && hasPrimitiveBounds(origin.Schema) {
			ty, parsed, err := a.boundedPrimitiveType(origin.Schema, origin.Pointer, 0)
			if err != nil {
				return err
			}
			scalar, checks = bare(ty).Name, parsed
		} else {
			kind, _, err := schemaType(origin.Schema, origin.Pointer)
			if err != nil {
				return err
			}
			var errMapping *diagnostic.Diagnostic
			scalar, errMapping = scalarType(kind, origin.Schema, origin.Pointer)
			if errMapping != nil {
				return errMapping
			}
		}
		if scalar != bare(origin.Type).Name || !reflect.DeepEqual(checks, origin.Checks) {
			return schemaReferenceUnsupported(c.Pointer + "/$ref")
		}
		if checks := bare(origin.Type).Validation; checks != nil && a.validationSources[checks] != origin.Pointer {
			return schemaReferenceUnsupported(c.Pointer + "/$ref")
		}
		matches := 0
		for d, recorded := range a.borrowedRoots {
			alias, ok := d.(*ValueAlias)
			if actual[d] != 0 && ok && recorded.Root == c && recorded.Schema == c.Schema && recorded.Pointer == c.Pointer && recorded.Source == origin && d.Name() == c.Name && recorded.Nullable == (c.BindingNullable != nil && *c.BindingNullable) && alias.Type.Kind == a.componentType(c).Kind && bare(alias.Type).Kind == Scalar && bare(alias.Type).Name == bare(origin.Type).Name && bare(alias.Type).Validation == bare(origin.Type).Validation {
				matches += actual[d]
			}
		}
		if matches != 1 {
			return schemaReferenceUnsupported(c.Pointer + "/$ref")
		}
	}
	return nil
}
