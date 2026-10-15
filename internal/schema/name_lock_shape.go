package schema

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// V1 is a fixed positional protocol, independent of Go field/type spellings.
// Changing any slot/tag requires a new fingerprint/lock version. Nil pointers
// and explicit zero rules remain distinct, as in conservative equivalence.
func lockPrimitiveV1(p *PrimitiveValidation) any {
	if p == nil {
		return nil
	}
	bound := func(b *NumericBound) any {
		if b == nil {
			return nil
		}
		return []any{b.Literal, b.Zero, b.Exclusive}
	}
	view := []any{p.MinLength, p.MaxLength, bound(p.Minimum), bound(p.Maximum), bound(p.ExclusiveMinimum), bound(p.ExclusiveMaximum), bound(p.MultipleOf), p.Floating, p.Pattern}
	if len(p.Enum) != 0 {
		return []any{"native-enum-membership-v1", view, p.Enum}
	}
	return view
}
func lockCollectionV1(c *CollectionValidation) any {
	if c == nil {
		return nil
	}
	view := []any{c.Minimum, c.Maximum, c.MinimumKey, c.MaximumKey, c.Unique}
	if c.UniqueJSON {
		return []any{"json-unique-v1", view}
	}
	return view
}
func lockTypeV1(t TypeRef, names map[string]string) any {
	tag := ""
	switch t.Kind {
	case Scalar:
		tag = "scalar"
	case Named:
		tag = "named"
	case List:
		tag = "list"
	case Nullable:
		tag = "nullable"
	case JSON:
		tag = "json"
	case Map:
		tag = "map"
	case JSONObject:
		tag = "json-object"
	case PatchValue:
		tag = "patch"
	case InlineValue:
		tag = "inline"
	default:
		panic("unknown type in v1 name-lock fingerprint")
	}
	name := t.Name
	if t.Kind == Named {
		if normal, ok := names[name]; ok {
			name = normal
		}
	}
	var element any
	if t.Element != nil {
		element = lockTypeV1(*t.Element, names)
	}
	view := ""
	switch t.ArrayAliasView {
	case OrdinaryAlias:
		view = "ordinary"
	case NullableArrayAlias:
		view = "nullable-root"
	case NonNullArrayAlias:
		view = "nonnull-root"
	default:
		panic("unknown array view in v1 name-lock fingerprint")
	}
	return []any{tag, name, element, t.Serializer, lockPrimitiveV1(t.Validation), lockCollectionV1(t.Collection), view}
}
func lockDeclarationV1(d Declaration, names map[string]string, top bool) any {
	name := d.Name()
	if top {
		name = "$root"
	}
	switch d := d.(type) {
	case *Model:
		properties, constants, nested := []any{}, []any{}, []any{}
		for _, p := range d.Properties {
			properties = append(properties, []any{p.Name, p.WireName, lockTypeV1(p.Type, names), p.DefaultNull, lockPrimitiveV1(p.Validation)})
		}
		for _, c := range d.Constants {
			constants = append(constants, []any{c.Name, c.WireName, lockTypeV1(c.Type, names), c.Initializer})
		}
		for _, n := range d.Nested {
			nested = append(nested, lockDeclarationV1(n, names, false))
		}
		return []any{"model", name, properties, constants, nested, d.Nullable, lockCollectionV1(d.PropertyCount)}
	case *StringEnum:
		members := []any{}
		for _, m := range d.Members {
			members = append(members, []any{m.Name, m.Wire})
		}
		return []any{"string-enum", name, members}
	default:
		panic("ineligible declaration in v1 name-lock fingerprint")
	}
}
func nameLockShapeV1(d Declaration, pkg string) string {
	names := map[string]string{}
	dedupOwnedNames(d, d.Name(), "$root", pkg+".api.models", names)
	view := []any{"openapi-kt-owned-shape-v1", lockDeclarationV1(d, names, true)}
	// New default-bearing declarations were previously rejected. Keep the
	// frozen v1 positional view intact and use a separate fingerprint domain
	// for these new shapes; the persisted opaque-hash format remains v1.
	var defaults []any
	var visit func(Declaration, string)
	visit = func(declaration Declaration, path string) {
		if model, ok := declaration.(*Model); ok {
			for _, p := range model.Properties {
				if p.Default != "" {
					defaults = append(defaults, []any{path, p.WireName, p.Default})
				}
			}
			for _, child := range model.Nested {
				visit(child, path+"/"+child.Name())
			}
		}
	}
	visit(d, "$root")
	if len(defaults) != 0 {
		view = []any{"openapi-kt-owned-shape-defaults-v1", view, defaults}
	}
	data, e := json.Marshal(view)
	if e != nil {
		panic("invalid v1 name-lock semantic view")
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
