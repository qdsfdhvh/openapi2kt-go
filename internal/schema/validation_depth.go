package schema

import "openapi-kt/internal/diagnostic"

// Kotlin's metadata parser and nested inline collection checks have lower
// practical limits than the schema graph traversal guard. Bound only emitted
// use-site chains, including DTO getter keys. Other nominal constructor
// validation stays behind its existing DTO/value-class barrier.
const MaxValidationCollectionDepth = 16

func (a *analyzer) validationDepth(plan *Plan) *diagnostic.Diagnostic {
	var check func(TypeRef, int) *diagnostic.Diagnostic
	check = func(ty TypeRef, depth int) *diagnostic.Diagnostic {
		if ty.Kind == List || ty.Kind == Map || ty.Kind == JSONObject {
			depth++
		}
		if ty.Validation != nil && depth > MaxValidationCollectionDepth {
			return diagnostic.At("E_SCHEMA_DEPTH", "Primitive validation exceeds the supported 16 nested collection edges; introduce named object models.", a.validationSources[ty.Validation])
		}
		if ty.Collection != nil && depth > MaxValidationCollectionDepth {
			return diagnostic.At("E_SCHEMA_DEPTH", "Collection validation exceeds the supported 16 nested collection edges; introduce named object models.", a.collectionSources[ty.Collection])
		}
		if ty.Collection != nil && ty.Collection.Unique && !uniqueMapType(*ty.Element) && structuralUniqueType(*ty.Element, 0) && depth+structuralUniqueCollectionDepth(*ty.Element) > MaxValidationCollectionDepth {
			return diagnostic.At("E_SCHEMA_DEPTH", "Collection validation exceeds the supported 16 nested collection edges; introduce named object models.", a.collectionSources[ty.Collection])
		}
		if ty.Collection != nil && ty.Collection.UniqueDTO != nil {
			for _, field := range ty.Collection.UniqueDTO.Fields {
				if depth+structuralUniqueCollectionDepth(field.Type) > MaxValidationCollectionDepth {
					return diagnostic.At("E_SCHEMA_DEPTH", "DTO uniqueness getter validation exceeds the supported 16 nested collection edges.", a.collectionSources[ty.Collection])
				}
			}
		}
		if ty.Element != nil {
			return check(*ty.Element, depth)
		}
		return nil
	}
	var declaration func(Declaration) *diagnostic.Diagnostic
	declaration = func(d Declaration) *diagnostic.Diagnostic {
		switch d := d.(type) {
		case *Model:
			for _, p := range d.Properties {
				if err := check(p.Type, 0); err != nil {
					return err
				}
			}
			for _, child := range d.Nested {
				if err := declaration(child); err != nil {
					return err
				}
			}
		case *ValueAlias:
			return check(d.Type, 0)
		case *ObjectUnion:
			for _, v := range d.Variants {
				if err := declaration(v); err != nil {
					return err
				}
			}
		case *PrimitiveUnion:
			if d.Object != nil {
				return declaration(d.Object)
			}
		case *Hierarchy:
			if d.Leaf != nil {
				return declaration(d.Leaf)
			}
		case *SSEEvents:
			for _, v := range d.Variants {
				if v.Data != nil {
					if err := check(*v.Data, 0); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	for _, d := range plan.Models {
		if err := declaration(d); err != nil {
			return err
		}
	}
	if plan.Client == nil {
		return nil
	}
	for _, d := range plan.Client.Models {
		if err := declaration(d); err != nil {
			return err
		}
	}
	for _, group := range plan.Client.Groups {
		for _, op := range group.Operations {
			for _, p := range op.Parameters {
				if err := check(p.Type, 0); err != nil {
					return err
				}
			}
			if op.Body != nil {
				if err := check(op.Body.Type, 0); err != nil {
					return err
				}
			}
			if err := check(op.Response, 0); err != nil {
				return err
			}
		}
	}
	return nil
}
