package schema

import (
	"strconv"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/naming"
)

// CheckRetainedEnumJVMNameLimits rejects a certain class-name overflow, not
// arbitrary identifier feasibility. An emitted enum member is a nested class;
// its member name alone is a lower bound of its JVM binary-name UTF8 constant.
// Final retained declarations distinguish physical classes from temporary
// scalar vocabulary checks and discarded source validators.
func CheckRetainedEnumJVMNameLimits(plan *Plan) *diagnostic.Diagnostic {
	seen := map[Declaration]bool{}
	var visit func(Declaration) *diagnostic.Diagnostic
	visit = func(declaration Declaration) *diagnostic.Diagnostic {
		if seen[declaration] {
			return nil
		}
		seen[declaration] = true
		switch d := declaration.(type) {
		case *StringEnum:
			origin, ok := plan.enumDiagnosticOrigins[d]
			if !ok {
				return nil
			}
			for _, member := range d.Members {
				if len(member.Name) <= maxJVMUTF8ConstantBytes {
					continue
				}
				// Original indices include null entries omitted from the physical
				// String member list. Provenance is diagnostic-only.
				for i, value := range get(origin.Schema, "enum").Items() {
					if wire, ok := value.Text(); ok && wire == member.Wire {
						return diagnostic.At("E_MODEL_LIMIT", "An enum member's minimum JVM class name exceeds the UTF-8 limit; shorten the source token.", naming.Pointer(naming.Pointer(origin.Pointer, "enum"), strconv.Itoa(i)))
					}
				}
			}
		case *Model:
			for _, nested := range d.Nested {
				if err := visit(nested); err != nil {
					return err
				}
			}
		case *ObjectUnion:
			for _, variant := range d.Variants {
				if err := visit(variant); err != nil {
					return err
				}
			}
		case *PrimitiveUnion:
			if d.Object != nil {
				return visit(d.Object)
			}
		case *Hierarchy:
			for _, enum := range d.Enums {
				if err := visit(enum); err != nil {
					return err
				}
			}
			if d.Leaf != nil {
				return visit(d.Leaf)
			}
		}
		return nil
	}
	for _, declaration := range plan.Models {
		if err := visit(declaration); err != nil {
			return err
		}
	}
	if plan.Client != nil {
		for _, declaration := range plan.Client.Models {
			if err := visit(declaration); err != nil {
				return err
			}
		}
	}
	return nil
}
