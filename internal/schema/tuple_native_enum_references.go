package schema

import (
	"slices"
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Native vocabularies keep their scalar wire codec and original checks. Their
// original source still distinguishes otherwise equal scalar List projections.
func (a *analyzer) tupleNativeEnum(original *document.Value, source string, terminal *document.Value, pointer string, depth int) (tupleScalarSignature, *diagnostic.Diagnostic) {
	if err := checkKeys(terminal, a.tupleSourceKeys(terminal, pointer, []string{"type", "format", "enum", "nullable"}), pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	reference := text(get(original, "$ref"))
	a.recordBorrowedDTOAliasEdges(reference, source)
	if err := a.visitBorrowedSourceAlias(reference, source, depth, false); err != nil {
		return tupleScalarSignature{}, err
	}
	target, err := a.resolveSchema(reference, source)
	if err != nil {
		return tupleScalarSignature{}, err
	}
	if target.Catalog != nil && !a.admitBorrowedNominalLeaf(target, source) {
		return tupleScalarSignature{}, schemaReferenceUnsupported(source)
	}
	ty, err := a.referenceType(target, nullableOverride(original), depth, source)
	if err != nil {
		return tupleScalarSignature{}, err
	}
	ty = defaultBacking(ty)
	if !uniqueNativeEnumType(ty) || target.Schema != terminal || target.Pointer != pointer {
		return tupleScalarSignature{}, schemaReferenceUnsupported(source)
	}
	if err := a.captureTupleNativeSource(original, source, terminal, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	kind, _, err := schemaType(terminal, pointer)
	if err != nil {
		return tupleScalarSignature{}, err
	}
	return tupleScalarSignature{Kind: kind, Format: text(get(terminal, "format")), HasFormat: has(terminal, "format"), Nullable: ty.Kind == Nullable, EnumSource: terminal}, nil
}

type tupleNativeSourceRegistry struct {
	Root        *document.Value
	Nodes       map[*document.Value]borrowedAllOfNode
	Terminals   map[*document.Value]string
	Catalogs    map[*component]borrowedAllOfShapeRef
	Sources     []tupleNativeSourcePath
	Work, Bytes int
}

type tupleNativeSourcePath struct {
	Node    *document.Value
	Pointer string
}

func (r *tupleNativeSourceRegistry) charge(work, bytes int, source string) *diagnostic.Diagnostic {
	if work > 16384-r.Work || bytes > (8<<20)-r.Bytes {
		return diagnostic.At("E_MODEL_LIMIT", "Tuple native enum sources exceed the cumulative work or byte budget.", source)
	}
	r.Work += work
	r.Bytes += bytes
	return nil
}

func tupleNativeNodeUnchanged(n borrowedAllOfNode) bool {
	if !slices.Equal(n.Members, n.Node.Members()) || !slices.Equal(n.Items, n.Node.Items()) {
		return false
	}
	for _, m := range n.Members {
		if get(n.Node, m.Name) != m.Value {
			return false
		}
	}
	return true
}

func (r *tupleNativeSourceRegistry) node(v *document.Value, source string) *diagnostic.Diagnostic {
	if old, ok := r.Nodes[v]; ok {
		if !tupleNativeNodeUnchanged(old) {
			return schemaReferenceUnsupported(source)
		}
		return nil
	}
	if err := r.charge(1+len(v.Members())+len(v.Items()), 0, source); err != nil {
		return err
	}
	if r.Nodes == nil {
		r.Nodes = map[*document.Value]borrowedAllOfNode{}
	}
	r.Nodes[v] = borrowedAllOfNode{v, append([]document.Member{}, v.Members()...), append([]*document.Value{}, v.Items()...)}
	r.Sources = append(r.Sources, tupleNativeSourcePath{v, source})
	return nil
}

// Retain the physical path, including original container nodes, so a same-byte
// copied ancestor or alias cannot silently substitute a different source tree.
func (a *analyzer) captureTupleNativePath(v *document.Value, pointer, source string) *diagnostic.Diagnostic {
	r := a.tupleNativeSources
	if !strings.HasPrefix(pointer, "/") {
		return schemaReferenceUnsupported(source)
	}
	current := a.root
	for _, raw := range strings.Split(pointer[1:], "/") {
		if current == nil {
			return schemaReferenceUnsupported(source)
		}
		if err := r.node(current, source); err != nil {
			return err
		}
		key, err := naming.DecodePointerSegment(raw, source)
		if err != nil {
			return err
		}
		switch current.Kind() {
		case document.Object:
			current = get(current, key)
		case document.Array:
			i, e := strconv.Atoi(key)
			if e != nil || i < 0 || i >= len(current.Items()) || strconv.Itoa(i) != key {
				return schemaReferenceUnsupported(source)
			}
			current = current.Items()[i]
		default:
			return schemaReferenceUnsupported(source)
		}
	}
	if current != v {
		return schemaReferenceUnsupported(source)
	}
	return r.node(v, source)
}

func (a *analyzer) captureTupleNativeSource(original *document.Value, source string, terminal *document.Value, pointer string) *diagnostic.Diagnostic {
	if a.tupleNativeSources == nil {
		a.tupleNativeSources = &tupleNativeSourceRegistry{}
	}
	r := a.tupleNativeSources
	if r.Root == nil {
		r.Root = a.root
	} else if r.Root != a.root {
		return schemaReferenceUnsupported(source)
	}
	if r.Catalogs == nil {
		r.Catalogs = map[*component]borrowedAllOfShapeRef{}
	}
	v, p := original, source
	for steps := 0; steps <= MaxSchemaDepth; steps++ {
		if err := a.captureTupleNativePath(v, p, source); err != nil {
			return err
		}
		if v == terminal {
			break
		}
		if !has(v, "$ref") {
			return schemaReferenceUnsupported(source)
		}
		ref := text(get(v, "$ref"))
		target, location, err := a.defaultSource(ref, p)
		if err != nil {
			return err
		}
		lookup, err := a.lookupSchemaReference(ref, source)
		if err != nil {
			return err
		}
		owner := lookup.SourceCatalog
		if owner == nil {
			owner = lookup.Catalog
		}
		if owner == nil {
			return schemaReferenceUnsupported(source)
		}
		if old, ok := r.Catalogs[owner]; ok {
			if old.Schema != owner.Schema || old.Pointer != owner.Pointer || old.Reference != owner.Reference {
				return schemaReferenceUnsupported(source)
			}
		} else {
			if err := r.charge(1, 0, source); err != nil {
				return err
			}
			r.Catalogs[owner] = borrowedAllOfShapeRef{Owner: owner, Schema: owner.Schema, Pointer: owner.Pointer, Reference: owner.Reference}
		}
		v, p = target, location
	}
	if v != terminal || p != pointer {
		return schemaReferenceUnsupported(source)
	}
	if r.Terminals == nil {
		r.Terminals = map[*document.Value]string{}
	}
	bytes, _ := terminal.MarshalJSON()
	if old, ok := r.Terminals[terminal]; ok {
		if old != string(bytes) {
			return schemaReferenceUnsupported(source)
		}
		return nil
	}
	if err := r.charge(0, len(bytes), source); err != nil {
		return err
	}
	nodes, _, err := a.borrowedAllOfSourceBudget(terminal, pointer, source, nil)
	if err != nil {
		return err
	}
	for _, n := range nodes {
		if err := r.node(n.Node, source); err != nil {
			return err
		}
	}
	r.Terminals[terminal] = string(bytes)
	return nil
}

func (a *analyzer) validateTupleNativeSources() *diagnostic.Diagnostic {
	r := a.tupleNativeSources
	if r == nil || len(r.Sources) == 0 {
		return nil
	}
	if r.Root != a.root {
		return schemaReferenceUnsupported(r.Sources[0].Pointer)
	}
	for _, source := range r.Sources {
		if !tupleNativeNodeUnchanged(r.Nodes[source.Node]) {
			return schemaReferenceUnsupported(source.Pointer)
		}
	}
	for c, old := range r.Catalogs {
		if a.components[c.Wire] != c || old.Schema != c.Schema || old.Pointer != c.Pointer || old.Reference != c.Reference {
			return schemaReferenceUnsupported(r.Sources[0].Pointer)
		}
	}
	for v, old := range r.Terminals {
		bytes, _ := v.MarshalJSON()
		if string(bytes) != old {
			return schemaReferenceUnsupported(r.Sources[0].Pointer)
		}
	}
	return nil
}
