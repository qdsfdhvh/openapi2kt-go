package schema

import (
	"sort"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// An actual original component/factory owns this annotation independently of
// every consuming field and intermediate alias. Internal or copied sources do
// not acquire collection-root default permission.
func (a *analyzer) independentLexicalEnumCollectionDefault(v *document.Value, pointer string) bool {
	if a.independentBorrowedEnumCollectionFieldDefault(v, pointer) {
		return true
	}
	if !a.componentDefaultSchemas[v] || !a.componentRoot(v, pointer) {
		return false
	}
	if !has(v, "$ref") {
		return inlineStringEnumCollectionLeaf(v, 1) != nil
	}
	target := a.referenceWrapperDefaultTarget(&component{Schema: v, Pointer: pointer})
	if target != nil && target.Internal && a.internalEnumCollectionDefaultSource(target.Schema, target.Pointer) {
		return true
	}
	return target != nil && !target.Internal && a.componentRoot(target.Schema, target.Pointer) && inlineStringEnumCollectionLeaf(target.Schema, 1) != nil
}

// New checked roots retain one real enum peer. Prove that vocabulary and the
// original primitive checks as well as the linked collection backing, including
// roots which currently have no consuming reference.
func (a *analyzer) validateBoundedEnumCollectionRoots(declarations []Declaration) *diagnostic.Diagnostic {
	actual := map[Declaration]int{}
	for _, d := range declarations {
		actual[d]++
	}
	var roots []Declaration
	for d, origin := range a.namedCollectionOrigins {
		if origin.BoundedEnum != nil {
			roots = append(roots, d)
		}
	}
	sort.Slice(roots, func(i, j int) bool {
		return a.namedCollectionOrigins[roots[i]].Pointer < a.namedCollectionOrigins[roots[j]].Pointer
	})
	for _, d := range roots {
		origin := a.namedCollectionOrigins[d]
		c, leaf := origin.Owner, origin.BoundedEnum
		if c == nil || a.components[c.Wire] != c || c.Schema != origin.Schema || c.Pointer != origin.Pointer || actual[d] != 1 || inlineStringEnumBoundsLeaf(c.Schema, 1) != leaf {
			return schemaReferenceUnsupported(origin.Pointer)
		}
		bytes, _ := c.Schema.MarshalJSON()
		if string(bytes) != origin.SourceBytes {
			return schemaReferenceUnsupported(origin.Pointer)
		}
		alias, ok := d.(*ValueAlias)
		if !ok || alias.ClassName != c.Name || alias.Type != c.AliasType {
			return schemaReferenceUnsupported(origin.Pointer)
		}
		if origin.Final {
			for _, source := range origin.Types {
				current := *source.Node
				current.SchemaSteps = 0
				if current != source.Value || namedCollectionChecks(current) != source.Checks {
					return schemaReferenceUnsupported(origin.Pointer)
				}
			}
		}
		pointer, source := c.Pointer, c.Schema
		ty := c.AliasType.WithoutRootNull()
		for source != leaf {
			edge, expected := "items", List
			if dictionarySchema(source) {
				edge, expected = "additionalProperties", Map
			}
			if ty.Kind != expected || ty.Element == nil {
				return schemaReferenceUnsupported(origin.Pointer)
			}
			source, pointer = get(source, edge), naming.Pointer(pointer, edge)
			ty = ty.Element.WithoutRootNull()
		}
		if ty.Kind != Named || ty.Element != nil || ty.Serializer != "" || ty.Validation == nil || a.validationSources[ty.Validation] != pointer {
			return schemaReferenceUnsupported(origin.Pointer)
		}
		checks, err := a.primitiveValidation(leaf, "kotlin.String", pointer)
		if err != nil {
			return err
		}
		if namedCollectionChecks(ty) != namedCollectionChecks(TypeRef{Validation: checks}) {
			return schemaReferenceUnsupported(origin.Pointer)
		}
		matches := 0
		for _, peer := range origin.Peers {
			if actual[peer] != 1 || a.declarationOwners[peer] != c {
				return schemaReferenceUnsupported(origin.Pointer)
			}
			enum, ok := peer.(*StringEnum)
			if !ok || a.enumOrigins[enum].Schema != leaf || a.enumOrigins[enum].Pointer != pointer || strings.TrimPrefix(ty.Name, a.options.PackageName+".models.") != enum.ClassName {
				continue
			}
			expected, err := buildStringEnumAllowedKeys(enum.ClassName, leaf, pointer, append([]string{"type", "nullable", "enum", "format", "x-native-type"}, primitiveBoundKeys...))
			if err != nil {
				return err
			}
			if len(enum.Members) != len(expected.Members) {
				return schemaReferenceUnsupported(origin.Pointer)
			}
			for i, member := range enum.Members {
				if member != expected.Members[i] {
					return schemaReferenceUnsupported(origin.Pointer)
				}
			}
			matches++
		}
		if matches != 1 {
			return schemaReferenceUnsupported(origin.Pointer)
		}
	}
	return nil
}
