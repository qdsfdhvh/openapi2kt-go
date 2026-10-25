package kotlin

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"openapi-kt/internal/schema"
)

// Constructor parameters can shadow a package in annotation arguments. Import
// only the affected classifiers, using names reserved against this whole file.
// The aliases never alter the semantic IR or user default/documentation text.
type constructorModelImports struct {
	mode        string
	serializers map[string]string
}
type constructorImports struct {
	models map[*schema.Model]constructorModelImports
	names  map[string]string
}

func newConstructorImports(root schema.Declaration) *constructorImports {
	result := &constructorImports{models: map[*schema.Model]constructorModelImports{}, names: map[string]string{}}
	used := map[string]bool{}
	models := []*schema.Model{}
	var collect func(schema.Declaration)
	collect = func(d schema.Declaration) {
		used[strings.ToLower(strings.Trim(d.Name(), "`"))] = true
		if m, ok := d.(*schema.Model); ok {
			models = append(models, m)
			for _, p := range m.Properties {
				used[strings.ToLower(strings.Trim(p.Name, "`"))] = true
			}
			for _, c := range m.Constants {
				used[strings.ToLower(strings.Trim(c.Name, "`"))] = true
			}
			for _, n := range m.Nested {
				collect(n)
			}
		}
	}
	collect(root)
	reserve := func(target, base string) string {
		if existing := result.names[target]; existing != "" {
			return existing
		}
		name := base
		for index := 2; used[strings.ToLower(name)]; index++ {
			name = base + strconv.Itoa(index)
		}
		used[strings.ToLower(name)] = true
		result.names[target] = name
		return name
	}
	for _, m := range models {
		scope := map[string]bool{}
		for _, p := range m.Properties {
			scope[p.Name] = true
		}
		entry := constructorModelImports{mode: "kotlinx.serialization.EncodeDefault.Mode", serializers: map[string]string{}}
		for _, p := range m.Properties {
			if scope["kotlinx"] && (p.OmitDefault || p.Type.Kind == schema.PatchValue) {
				entry.mode = reserve("kotlinx.serialization.EncodeDefault.Mode", "OpenapiKtEncodeDefaultMode")
			}
			var visit func(schema.TypeRef)
			visit = func(t schema.TypeRef) {
				if prefix, _, qualified := strings.Cut(t.Serializer, "."); qualified && scope[prefix] {
					entry.serializers[t.Serializer] = reserve(t.Serializer, "OpenapiKtTypeUseSerializer")
				}
				if t.Element != nil {
					visit(*t.Element)
				}
			}
			visit(p.Type)
		}
		result.models[m] = entry
	}
	return result
}
func (c *constructorImports) emit(out *strings.Builder) {
	names := make([]string, 0, len(c.names))
	for name := range c.names {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(out, "import %s as %s\n", name, c.names[name])
	}
	if len(names) > 0 {
		out.WriteString("\n")
	}
}
func constructorModeName(model *schema.Model, contexts []*constructorImports) string {
	if len(contexts) > 0 && contexts[0] != nil {
		if value, ok := contexts[0].models[model]; ok {
			return value.mode
		}
	}
	return "kotlinx.serialization.EncodeDefault.Mode"
}
func constructorType(t schema.TypeRef, model *schema.Model, contexts []*constructorImports) string {
	if len(contexts) == 0 || contexts[0] == nil || len(contexts[0].models[model].serializers) == 0 {
		return RenderType(t)
	}
	names := contexts[0].models[model].serializers
	var rebind func(schema.TypeRef) schema.TypeRef
	rebind = func(ty schema.TypeRef) schema.TypeRef {
		if name := names[ty.Serializer]; name != "" {
			ty.Serializer = name
		}
		if ty.Element != nil {
			child := rebind(*ty.Element)
			ty.Element = &child
		}
		return ty
	}
	return RenderType(rebind(t))
}
