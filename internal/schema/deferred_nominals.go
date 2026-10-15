package schema

import (
	"sort"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// Successful nominal planning retains its original source and emitted typed view.
// Final ownership is checked against the retained declaration tree.
type nominalOrigin struct {
	Schema   *document.Value
	Pointer  string
	Nullable bool
}

func (a *analyzer) recordUnionOrigin(d Declaration, v *document.Value, p string, nullable bool) {
	if a.unionOrigins == nil {
		return
	}
	switch d.(type) {
	case *ObjectUnion, *PrimitiveUnion:
		a.unionOrigins[d] = nominalOrigin{v, p, nullable}
	}
}

func (a *analyzer) recordEnumOrigin(d *StringEnum, v *document.Value, p string, nullable bool) {
	if a.enumOrigins != nil {
		a.enumOrigins[d] = nominalOrigin{v, p, nullable}
	}
}

// This carrier retains lexical catalog/source identity before declarations are
// constructed. Only a unique retained declaration under that catalog may fill Name.
type deferredNominal struct {
	Schema                *document.Value
	Owner                 *component
	Source, Pointer, Name string
	Nullable              bool // Actual emitted typed view; a standalone Null carrier stays non-null.
}

func (a *analyzer) linkDeferredNominals(hierarchy map[string]*Hierarchy) *diagnostic.Diagnostic {
	return a.linkNominalDeclarations(hierarchy, nil, false)
}

func (a *analyzer) linkNominalDeclarations(hierarchy map[string]*Hierarchy, declarations []Declaration, final bool) *diagnostic.Diagnostic {
	type occurrence struct {
		owner       *component
		declaration Declaration
		pointer     string
		nullable    bool
		name        string
	}
	occurrences := map[*document.Value][]occurrence{}
	work := 0
	var collect func(Declaration, string, *component, *document.Value) *diagnostic.Diagnostic
	collect = func(d Declaration, parent string, owner *component, unionSource *document.Value) *diagnostic.Diagnostic {
		work++
		if work > 16384 {
			return hierarchyError("E_SCHEMA_LIMIT", owner.Pointer)
		}
		if origin, ok := a.unionOrigins[d]; ok && (origin.Pointer == owner.Pointer || strings.HasPrefix(origin.Pointer, owner.Pointer+"/")) {
			occurrences[origin.Schema] = append(occurrences[origin.Schema], occurrence{owner: owner, declaration: d, pointer: origin.Pointer, nullable: origin.Nullable, name: parent + "." + d.Name()})
		}
		if origin, ok := a.enumOrigins[d]; ok && (origin.Pointer == owner.Pointer || strings.HasPrefix(origin.Pointer, owner.Pointer+"/")) {
			occurrences[origin.Schema] = append(occurrences[origin.Schema], occurrence{owner: owner, declaration: d, pointer: origin.Pointer, nullable: origin.Nullable, name: parent + "." + d.Name()})
		}
		if origin, ok := a.valueClassOrigins[d]; ok && origin.Pointer == owner.Pointer {
			occurrences[origin.Schema] = append(occurrences[origin.Schema], occurrence{owner: owner, declaration: d, pointer: origin.Pointer, nullable: origin.Nullable, name: parent + "." + d.Name()})
		}
		if m, ok := d.(*Model); ok {
			name := parent + "." + m.ClassName
			if v := a.modelOrigins[m]; v != nil && v != unionSource && len(m.Properties) > 0 {
				p := a.modelPointers[m]
				// A borrowed copy retains the source Pointer, but its declaration
				// scope belongs to the borrowing component. It cannot own that source.
				if p == owner.Pointer && composedModelCandidate(v) || strings.HasPrefix(p, owner.Pointer+"/") {
					occurrences[v] = append(occurrences[v], occurrence{owner: owner, declaration: m, pointer: p, nullable: m.Nullable, name: name})
				}
			}
			for _, child := range m.Nested {
				if err := collect(child, name, owner, nil); err != nil {
					return err
				}
			}
		} else if u, ok := d.(*ObjectUnion); ok {
			// Product variants borrow the union's original schema node. The
			// retained sealed declaration owns that node; variant models do not.
			// Their nested declarations still participate in ambiguity checks.
			for _, v := range u.Variants {
				if err := collect(v, parent+"."+u.ClassName, owner, a.unionOrigins[u].Schema); err != nil {
					return err
				}
			}
		} else if u, ok := d.(*PrimitiveUnion); ok && u.Object != nil {
			if err := collect(u.Object, parent, owner, nil); err != nil {
				return err
			}
		}
		return nil
	}
	keys := []string{}
	for key := range hierarchy {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		h := hierarchy[key]
		if h.Leaf != nil {
			if err := collect(h.Leaf, a.options.PackageName+".models", a.components[key], nil); err != nil {
				return err
			}
		}
	}
	byName := map[string]*component{}
	for _, c := range a.components {
		byName[c.Name] = c
	}
	for _, d := range declarations {
		c := byName[d.Name()]
		if c == nil {
			c = a.declarationOwners[d]
		} // Actual root collection peers retain their lexical catalog.
		if c != nil && hierarchy[c.Wire] == nil {
			if err := collect(d, a.options.PackageName+".models", c, nil); err != nil {
				return err
			}
		}
	}
	// Register actual direct lexical occurrences even before their first
	// typed use. Source-path eligibility is checked independently by lookup.
	for v, all := range occurrences {
		if len(all) == 1 && (directInternalModel(v, "") || composedModelCandidate(v) || unionNominalCandidate(v) || enumNominalCandidate(v) || all[0].owner.ValueClass && all[0].pointer == all[0].owner.Pointer) {
			o := all[0]
			p := o.pointer
			if p != o.owner.Pointer {
				ref := "#" + p
				target, err := a.lookupSchemaReference(ref, p)
				if err != nil || target.Schema != v || target.SourceCatalog != o.owner || !target.ComposedLexical && (!target.DirectLexical || !a.hierarchySources[o.owner.Schema] && !unionNominalCandidate(v) && !enumNominalCandidate(v)) {
					continue
				}
			}
			if a.deferredNominals[v] == nil {
				a.deferredNominals[v] = &deferredNominal{Schema: v, Owner: o.owner, Pointer: p, Name: o.name, Nullable: o.nullable}
			}
		}
	}
	bindings := []*deferredNominal{}
	for _, b := range a.deferredNominals {
		bindings = append(bindings, b)
	}
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].Source < bindings[j].Source })
	for _, b := range bindings {
		matches := []occurrence{}
		for _, o := range occurrences[b.Schema] {
			if o.owner == b.Owner {
				matches = append(matches, o)
			}
		}
		if len(matches) != 1 {
			if !final && hierarchy[b.Owner.Wire] == nil {
				continue
			}
			return schemaReferenceUnsupported(b.Source)
		}
		b.Name = matches[0].name
		b.Nullable = matches[0].nullable
		if b.Source != "" {
			a.internalModelUses[b.Name] = b.Source
		}
	}
	// Resolve transient links on actual retained models, not scratch plans.
	seenTypes := map[*TypeRef]bool{}
	var linkType func(*TypeRef)
	linkType = func(t *TypeRef) {
		// Transparent aliases may still contain cyclic provisional TypeRefs.
		// Their original cycle diagnostic belongs to validateAliases below.
		if seenTypes[t] {
			return
		}
		seenTypes[t] = true
		if t.deferredNominal != nil && t.deferredNominal.Name != "" {
			t.Name = t.deferredNominal.Name
			t.deferredNominal = nil
		}
		if t.Element != nil {
			linkType(t.Element)
		}
	}
	var linkDeclaration func(Declaration)
	linkDeclaration = func(d Declaration) {
		switch d := d.(type) {
		case *Model:
			for i := range d.Properties {
				linkType(&d.Properties[i].Type)
			}
			for i := range d.Constants {
				linkType(&d.Constants[i].Type)
			}
			for _, n := range d.Nested {
				linkDeclaration(n)
			}
		case *ObjectUnion:
			for i := range d.Common {
				linkType(&d.Common[i].Type)
			}
			for _, v := range d.Variants {
				linkDeclaration(v)
			}
		case *PrimitiveUnion:
			if d.Object != nil {
				linkDeclaration(d.Object)
			}
		case *ValueAlias:
			linkType(&d.Type)
		}
	}
	for _, key := range keys {
		if hierarchy[key].Leaf != nil {
			linkDeclaration(hierarchy[key].Leaf)
		}
	}
	for _, d := range declarations {
		linkDeclaration(d)
	}
	for _, c := range a.components {
		linkType(&c.AliasType)
		if t := c.InternalModelTarget; t != nil && t.Deferred != nil && t.Deferred.Name != "" {
			t.ModelName = t.Deferred.Name
			t.Deferred = nil
		}
	}
	if final {
		a.nominalsLinked = true
	}
	return nil
}
