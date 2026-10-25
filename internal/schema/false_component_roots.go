package schema

import (
	"encoding/json"
	"slices"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

type falseRootSourceRegistry struct {
	Root, Components, Schemas *document.Value
	Nodes                     []borrowedAllOfNode
	Work                      int
}

type falseValueOrigin struct {
	Owner               *component
	Schema              *document.Value
	Wire, Pointer, Name string
	Type                string
}

type falseAliasOrigin struct {
	Owner, Target                         *component
	Schema                                *document.Value
	Wire, Pointer, Name, Reference, Bytes string
	Type                                  string
	Height                                int
	Members                               []document.Member
	Nullable                              int
}

func falseTypeSignature(t TypeRef) string {
	value, _ := json.Marshal(t)
	return string(value)
}

func falseNullableIntent(c *component) int {
	if c.BindingNullable == nil {
		return 0
	}
	if *c.BindingNullable {
		return 1
	}
	return -1
}

func (r *falseRootSourceRegistry) charge(work int, source string) *diagnostic.Diagnostic {
	if work > 16384-r.Work {
		return diagnostic.At("E_MODEL_LIMIT", "False root sources exceed the supported cumulative work budget.", source)
	}
	r.Work += work
	return nil
}

func (a *analyzer) captureFalseRootSource(c *component) *diagnostic.Diagnostic {
	if a.falseRootSources == nil {
		r := &falseRootSourceRegistry{Root: a.root, Components: get(a.root, "components"), Schemas: get(get(a.root, "components"), "schemas")}
		for _, node := range []*document.Value{r.Root, r.Components, r.Schemas} {
			if err := r.charge(1+len(node.Members()), c.Pointer); err != nil {
				return err
			}
			r.Nodes = append(r.Nodes, borrowedAllOfNode{Node: node, Members: append([]document.Member{}, node.Members()...)})
		}
		a.falseRootSources = r
	}
	r := a.falseRootSources
	if r.Root != a.root || get(r.Schemas, c.Wire) != c.Schema {
		return schemaReferenceUnsupported(c.Pointer)
	}
	return r.charge(1+len(c.Schema.Members()), c.Pointer)
}

func (a *analyzer) recordFalseValueOrigin(d *FalseValue, c *component) *diagnostic.Diagnostic {
	d.AbsentSerializer = a.falseAbsentUses[c]
	if err := a.captureFalseRootSource(c); err != nil {
		return err
	}
	if a.falseValueOrigins == nil {
		a.falseValueOrigins = map[*FalseValue]falseValueOrigin{}
	}
	a.falseValueOrigins[d] = falseValueOrigin{c, c.Schema, c.Wire, c.Pointer, c.Name, falseTypeSignature(a.componentType(c))}
	return nil
}

func (a *analyzer) recordFalseAliasOrigin(d *ValueAlias, c *component) *diagnostic.Diagnostic {
	if err := a.captureFalseRootSource(c); err != nil {
		return err
	}
	if a.falseAliasOrigins == nil {
		a.falseAliasOrigins = map[*ValueAlias]falseAliasOrigin{}
	}
	bytes, _ := c.Schema.MarshalJSON()
	a.falseAliasOrigins[d] = falseAliasOrigin{c, c.BindingTarget, c.Schema, c.Wire, c.Pointer, c.Name, c.Reference, string(bytes), falseTypeSignature(d.Type), c.AliasHeight, append([]document.Member{}, c.Schema.Members()...), falseNullableIntent(c)}
	return nil
}

func (a *analyzer) validateFalseRootBindings(ds []Declaration) *diagnostic.Diagnostic {
	r := a.falseRootSources
	if r == nil {
		return nil
	}
	const source = "/components/schemas"
	if r.Root != a.root || get(a.root, "components") != r.Components || get(r.Components, "schemas") != r.Schemas {
		return schemaReferenceUnsupported(source)
	}
	for _, node := range r.Nodes {
		if !slices.Equal(node.Members, node.Node.Members()) {
			return schemaReferenceUnsupported(source)
		}
		for _, m := range node.Members {
			if get(node.Node, m.Name) != m.Value {
				return schemaReferenceUnsupported(source)
			}
		}
	}
	actual := map[Declaration]int{}
	names := map[string]int{}
	for _, d := range ds {
		actual[d]++
		names[d.Name()]++
	}
	roots := make([]*FalseValue, 0, len(a.falseValueOrigins))
	for d := range a.falseValueOrigins {
		roots = append(roots, d)
	}
	slices.SortFunc(roots, func(x, y *FalseValue) int {
		return strings.Compare(a.falseValueOrigins[x].Pointer, a.falseValueOrigins[y].Pointer)
	})
	for _, d := range roots {
		origin := a.falseValueOrigins[d]
		c := origin.Owner
		if a.components[origin.Wire] != c || c.Schema != origin.Schema || get(r.Schemas, origin.Wire) != origin.Schema || c.Wire != origin.Wire || c.Pointer != origin.Pointer || c.Name != origin.Name || !c.FalseRoot || !a.booleanFalseRoot(c.Schema) || c.Reference != "" || c.BindingTarget != nil || falseTypeSignature(a.componentType(c)) != origin.Type || actual[d] != 1 || names[origin.Name] != 1 || d.ClassName != origin.Name || d.AbsentSerializer != a.falseAbsentUses[c] {
			return schemaReferenceUnsupported(origin.Pointer)
		}
	}
	aliases := make([]*ValueAlias, 0, len(a.falseAliasOrigins))
	for d := range a.falseAliasOrigins {
		aliases = append(aliases, d)
	}
	slices.SortFunc(aliases, func(x, y *ValueAlias) int {
		return strings.Compare(a.falseAliasOrigins[x].Pointer, a.falseAliasOrigins[y].Pointer)
	})
	for _, d := range aliases {
		origin := a.falseAliasOrigins[d]
		c := origin.Owner
		bytes, _ := c.Schema.MarshalJSON()
		if a.components[origin.Wire] != c || c.Schema != origin.Schema || get(r.Schemas, origin.Wire) != origin.Schema || c.Wire != origin.Wire || c.Pointer != origin.Pointer || c.Name != origin.Name || c.Reference != origin.Reference || c.BindingTarget != origin.Target || c.AliasHeight != origin.Height || falseNullableIntent(c) != origin.Nullable || string(bytes) != origin.Bytes || !slices.Equal(c.Schema.Members(), origin.Members) || actual[d] != 1 || names[origin.Name] != 1 || d.ClassName != origin.Name || falseTypeSignature(d.Type) != origin.Type || falseTypeSignature(a.componentType(c)) != origin.Type || !a.falseAliasFactoryMatches(d, c) {
			return schemaReferenceUnsupported(origin.Pointer)
		}
		for _, m := range origin.Members {
			if get(c.Schema, m.Name) != m.Value {
				return schemaReferenceUnsupported(origin.Pointer)
			}
		}
	}
	return nil
}
