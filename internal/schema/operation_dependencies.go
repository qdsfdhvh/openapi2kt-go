package schema

import (
	"reflect"
	"strings"
	"unicode"
)

// Opaque incoming code/classifier references cannot be rebound by an owned-tree
// pass. Preserve their referenced API owners instead of erasing nested access.
// Reflection visits only typed IR values, never JSON/schema graphs or source text.
// Only TypeRef/Constant/ScalarAlias code-bearing fields contribute identifiers;
// names, wire literals, descriptions and validator patterns are not code edges.
func operationOpaqueOwners(plan *Plan, pkg string) map[string]bool {
	roots, rootTypes := map[string]bool{}, map[string]bool{}
	for _, d := range plan.Client.Models {
		roots[d.Name()] = true
		rootTypes[d.Name()] = true
		rootTypes[pkg+".api.models."+d.Name()] = true
	}
	protected := map[string]bool{}
	add := func(code string) {
		for _, token := range strings.FieldsFunc(code, func(r rune) bool { return r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
			if roots[token] {
				protected[token] = true
			}
		}
	}
	collect := func(value any, owned map[string]string) {
		seen := map[uintptr]bool{}
		var walk func(reflect.Value)
		walk = func(v reflect.Value) {
			if !v.IsValid() {
				return
			}
			if v.Kind() == reflect.Interface {
				if !v.IsNil() {
					walk(v.Elem())
				}
				return
			}
			if v.Kind() == reflect.Pointer {
				if v.IsNil() || seen[v.Pointer()] {
					return
				}
				seen[v.Pointer()] = true
				walk(v.Elem())
				return
			}
			if v.CanInterface() {
				switch item := v.Interface().(type) {
				case TypeRef:
					add(item.Serializer)
					if item.Kind == Scalar || item.Kind == InlineValue || (item.Kind == Named && owned[item.Name] == "" && !rootTypes[item.Name]) {
						add(item.Name)
					}
					if item.Element != nil {
						walk(reflect.ValueOf(item.Element))
					}
					return
				case Constant:
					add(item.Initializer)
					walk(reflect.ValueOf(item.Type))
					return
				case ScalarAlias:
					add(item.Scalar)
					return
				}
			}
			switch v.Kind() {
			case reflect.Struct:
				for i := 0; i < v.NumField(); i++ {
					if v.Field(i).CanInterface() {
						walk(v.Field(i))
					}
				}
			case reflect.Slice, reflect.Array:
				for i := 0; i < v.Len(); i++ {
					walk(v.Index(i))
				}
			}
		}
		walk(reflect.ValueOf(value))
	}
	for index, inventory := range [][]Declaration{plan.Models, plan.Client.Models} {
		for _, d := range inventory {
			owned := map[string]string{}
			// Existing models emit in different packages; only their own declared names
			// can be excluded. API references in a component remain incoming edges.
			modelPkg := pkg + ".models"
			if index == 1 {
				modelPkg = pkg + ".api.models"
			}
			dedupOwnedNames(d, d.Name(), "$owner", modelPkg, owned)
			collect(d, owned)
		}
	}
	for _, g := range plan.Client.Groups {
		for _, op := range g.Operations {
			collect(op.Parameters, nil)
			collect(op.Response, nil)
			collect(op.Body, nil)
			collect(op.Errors, nil)
			collect(op.SSE, nil)
		}
	}
	return protected
}
