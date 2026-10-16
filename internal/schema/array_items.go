package schema

import (
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// arrayItemsDecision borrows every original positional schema. A mixed prefix
// has no synthetic schema node: its explicit raw projection is a typed choice.
// Non-prefix arrays retain their historical element planner and source path.
type arrayItemsDecision struct {
	Selected   *document.Value
	Pointer    string
	Prefix     []unionBranch
	Projection *TypeRef
}

type tupleScalarSignature struct {
	Kind, Format        string
	HasFormat, Nullable bool
}

func tupleScalar(v *document.Value, pointer string, depth int) (tupleScalarSignature, *diagnostic.Diagnostic) {
	if err := depthCheck(depth, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	if err := object(v, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	if err := checkKeys(v, []string{"type", "format", "nullable"}, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	kind, nullable, err := schemaType(v, pointer)
	if err != nil {
		return tupleScalarSignature{}, err
	}
	format := text(get(v, "format"))
	ordinary := kind == "string" && (format == "" || format == "uuid") || kind == "boolean" && format == "" || kind == "integer" && (format == "" || format == "int32" || format == "int64") || kind == "number" && (format == "" || format == "float" || format == "double")
	if !ordinary {
		return tupleScalarSignature{}, unsupported(pointer)
	}
	if _, err := scalarMapping(kind, v, pointer); err != nil {
		return tupleScalarSignature{}, err
	}
	return tupleScalarSignature{Kind: kind, Format: format, HasFormat: has(v, "format"), Nullable: nullable}, nil
}

func (a *analyzer) arrayItems(v *document.Value, pointer string, depth int) (*arrayItemsDecision, *diagnostic.Diagnostic) {
	keys := a.collectionKeys([]string{"type", "nullable", "items", "prefixItems"})
	if a.objectDefaultSchemas[v] {
		keys = append(keys, "default")
	}
	if err := checkKeys(v, keys, pointer); err != nil {
		return nil, err
	}
	decision := &arrayItemsDecision{Selected: get(v, "items"), Pointer: naming.Pointer(pointer, "items")}
	prefix, exists := v.Get("prefixItems")
	if !exists {
		if decision.Selected == nil {
			return nil, invalid("Arrays require an items schema.", pointer)
		}
		return decision, nil
	}
	location := naming.Pointer(pointer, "prefixItems")
	if !strings.HasPrefix(text(get(a.root, "openapi")), "3.1.") {
		return nil, unsupported(location)
	}
	if prefix.Kind() != document.Array || len(prefix.Items()) == 0 {
		return nil, invalid("prefixItems must be a non-empty schema array.", location)
	}
	uniform := true
	var first tupleScalarSignature
	for i, member := range prefix.Items() {
		source := naming.Pointer(location, strconv.Itoa(i))
		signature, err := tupleScalar(member, source, depth+1)
		if err != nil {
			return nil, err
		}
		decision.Prefix = append(decision.Prefix, unionBranch{Schema: member, Pointer: source})
		if i == 0 {
			first = signature
		} else if signature != first {
			uniform = false
		}
	}
	if decision.Selected != nil {
		// Preserve the historical tuple+boolean-tail exclusion at prefixItems;
		// source members above are still validated before this unsupported tail.
		if decision.Selected.Kind() == document.Boolean {
			return nil, unsupported(location)
		}
		if _, err := tupleScalar(decision.Selected, decision.Pointer, depth+1); err != nil {
			return nil, err
		}
	} else if uniform {
		decision.Selected = decision.Prefix[0].Schema
		decision.Pointer = decision.Prefix[0].Pointer
	} else {
		raw := TypeRef{Kind: JSON}
		decision.Projection = &raw
		decision.Pointer = location
	}
	a.warn("W_TUPLE_APPROXIMATION", "A positional array uses a List approximation; positions, exact length and unrestricted tail values are not enforced and a typed List may narrow the tail.", location)
	return decision, nil
}
