package schema

import (
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// dictionarySchema identifies only direct dictionaries. Object compositions and
// union common-value projections have separate contracts and cannot silently
// discard their additionalProperties semantics through object flattening.
func dictionarySchema(v *document.Value) bool {
	if !has(v, "additionalProperties") || has(v, "allOf") || has(v, "oneOf") || has(v, "anyOf") {
		return false
	}
	properties, exists := v.Get("properties")
	return !exists || properties.Kind() == document.Object && len(properties.Members()) == 0
}

// Catalog schemaType grammar is validated before planning. Both direct arrays
// and dictionaries use transparent aliases; their underlying type is linked by
// address so forward consumers retain checks after declaration planning.
func collectionAliasSchema(v *document.Value) bool {
	// Const values retain their JSON classifier, not List/Map alias backing.
	if has(v, "const") {
		return false
	}
	if dictionarySchema(v) {
		return true
	}
	kind, _, _ := schemaType(v, "")
	return kind == "array"
}

// The new array-root profile does not alter historical nullable dictionaries.
func nullableArraySchema(v *document.Value) bool {
	kind, nullable, _ := schemaType(v, "")
	return kind == "array" && nullable
}

// dictionaryValue validates the container independently of its value projection.
// A nil value denotes the open/closed JsonObject representation.
func (a *analyzer) dictionaryValue(v *document.Value, pointer string) (*document.Value, *diagnostic.Diagnostic) {
	keys := a.collectionKeys([]string{"type", "nullable", "properties", "required", "additionalProperties"})
	if a.objectDefaultSchemas[v] || a.independentBorrowedEnumCollectionFieldDefault(v, pointer) {
		keys = append(keys, "default")
	}
	if err := checkKeys(v, keys, pointer); err != nil {
		return nil, err
	}
	if required, ok := v.Get("required"); ok && (required.Kind() != document.Array || len(required.Items()) != 0) {
		return nil, invalid("Dictionary schemas cannot require undeclared fixed properties.", naming.Pointer(pointer, "required"))
	}
	ap := get(v, "additionalProperties")
	location := naming.Pointer(pointer, "additionalProperties")
	if flag, ok := ap.Bool(); ok {
		if !flag {
			a.warnAdditionalProperties(location)
		}
		return nil, nil
	}
	if err := object(ap, location); err != nil {
		return nil, err
	}
	return ap, nil
}

func (a *analyzer) dictionaryType(v *document.Value, name, pointer string, nested *[]Declaration, used map[string]bool, depth int) (TypeRef, *diagnostic.Diagnostic) {
	ap, err := a.dictionaryValue(v, pointer)
	if err != nil {
		return TypeRef{}, err
	}
	if ap == nil {
		return a.checkedCollection(TypeRef{Kind: JSONObject}, v, pointer)
	}
	location := naming.Pointer(pointer, "additionalProperties")
	item, err := a.collectionElementType(ap, name+"Value", location, nested, used, depth+1)
	if err != nil {
		return TypeRef{}, err
	}
	// An unconstrained value is still an object-only dictionary, not arbitrary
	// JSON. Annotated/nullable JsonElement values carry JSON null intrinsically.
	if bare(item).Kind == JSON {
		return a.checkedCollection(TypeRef{Kind: JSONObject}, v, pointer)
	}
	return a.checkedCollection(TypeRef{Kind: Map, Element: &item}, v, pointer)
}

func (a *analyzer) warnAdditionalProperties(pointer string) {
	a.warn("W_ADDITIONAL_PROPERTIES_APPROXIMATION", "Fixed-field models do not collect additional keys, and additionalProperties restrictions are not enforced by generated serializers.", pointer)
}

// Validate the ignored value schema, even when a fixed-field DTO cannot store
// extra keys. Scratch declarations must not leak into that DTO's namespace.
func (a *analyzer) validateAdditionalProperties(v *document.Value, pointer string, depth int) *diagnostic.Diagnostic {
	ap, ok := v.Get("additionalProperties")
	if !ok {
		return nil
	}
	location := naming.Pointer(pointer, "additionalProperties")
	// Validate the complete original literal, then retain only the existing
	// extras projection warning: no body/default/helper/value role is exported.
	if structuredLiteral(ap) {
		if err := validateDiscardedStructuredValue(ap, location, depth+1); err != nil {
			return err
		}
		a.warnAdditionalProperties(location)
		return nil
	}
	if _, ok := ap.Bool(); !ok {
		if err := a.rejectValidationProjection(ap, location); err != nil {
			return err
		}
		var nested []Declaration
		if _, err := a.propertyType(ap, "AdditionalValue", location, &nested, map[string]bool{}, depth+1); err != nil {
			return err
		}
	}
	a.warnAdditionalProperties(location)
	return nil
}

// Kotlin cannot compile transparent recursive aliases (including cycles through
// Map/List/nullable). A nominal class/enum/union is a recursion barrier. Cached
// heights keep alias DAG checking linear without hiding a too-deep ancestor.
func (a *analyzer) validateAliases(declarations []Declaration) *diagnostic.Diagnostic {
	aliases := map[string]*ValueAlias{}
	pointers := map[string]string{}
	prefix := a.options.PackageName + ".models."
	for _, d := range declarations {
		if alias, ok := d.(*ValueAlias); ok {
			aliases[prefix+alias.ClassName] = alias
		}
	}
	for _, c := range a.components {
		pointers[prefix+c.Name] = c.Pointer
	}
	visiting := map[string]bool{}
	heights := map[string]int{}
	var visit func(string, int) (int, *diagnostic.Diagnostic)
	var typeHeight func(TypeRef, int) (int, *diagnostic.Diagnostic)
	typeHeight = func(ty TypeRef, depth int) (int, *diagnostic.Diagnostic) {
		if ty.SchemaSteps != 0 {
			steps := ty.SchemaSteps
			ty.SchemaSteps = 0
			h, err := typeHeight(ty, depth+steps)
			return h + steps, err
		}
		switch ty.Kind {
		case Nullable:
			return typeHeight(*ty.Element, depth)
		case Map, List:
			h, err := typeHeight(*ty.Element, depth+1)
			return h + 1, err
		case Named:
			name := ty.Name
			if !strings.Contains(name, ".") {
				name = prefix + name
			}
			if aliases[name] != nil {
				h, err := visit(name, depth+1)
				return h + 1, err
			}
		}
		return 0, nil
	}
	visit = func(name string, depth int) (int, *diagnostic.Diagnostic) {
		pointer := pointers[name]
		if visiting[name] {
			return 0, diagnostic.At("E_REF_CYCLE", "Dictionary aliases cannot form a transparent recursive type; introduce a named object model.", pointer)
		}
		if err := depthCheck(depth, pointer); err != nil {
			return 0, err
		}
		if h, ok := heights[name]; ok {
			if err := depthCheck(depth+h, pointer); err != nil {
				return 0, err
			}
			return h, nil
		}
		visiting[name] = true
		h, err := typeHeight(aliases[name].Type, depth)
		delete(visiting, name)
		if err != nil {
			return 0, err
		}
		if err := depthCheck(depth+h, pointer); err != nil {
			return 0, err
		}
		heights[name] = h
		return h, nil
	}
	for _, d := range declarations {
		if _, ok := d.(*ValueAlias); ok {
			if _, err := visit(prefix+d.Name(), 0); err != nil {
				return err
			}
		}
	}
	for _, c := range a.components {
		if _, ok := aliases[prefix+c.Name]; ok {
			c.CollectionHeight = heights[prefix+c.Name]
		}
	}
	return nil
}
