package kotlin

import (
	"strconv"
	"strings"

	"openapi-kt/internal/schema"
)

// Value-expression package qualifiers can resolve to method parameters. File
// import aliases preserve types and public call sites and reserve all method,
// parameter and generated local names before allocating ordinary identifiers.
type namespacePlan struct {
	imports      string
	methodAlias  string
	contentAlias string
	flowAlias    string
	events       map[string]string
}

func shadowsNamespace(op schema.Operation, prefix string) bool {
	if prefix == "client" || prefix == "basePath" || prefix == "json" ||
		prefix == "requestBuilder" || prefix == "input" && op.Body != nil {
		return true
	}
	for _, parameter := range op.Parameters {
		if parameter.Name == prefix {
			return true
		}
	}
	return false
}

func (p namespacePlan) httpMethodConstructor(op schema.Operation) string {
	if shadowsNamespace(op, "io") {
		return p.methodAlias
	}
	return "io.ktor.http.HttpMethod"
}

func (p namespacePlan) contentTypeReference(op schema.Operation) string {
	if shadowsNamespace(op, "io") {
		return p.contentAlias
	}
	return "io.ktor.http.ContentType"
}

func (p namespacePlan) flowBuilderReference(op schema.Operation) string {
	if shadowsNamespace(op, "kotlinx") {
		return p.flowAlias
	}
	return "kotlinx.coroutines.flow.flow"
}

func needsSSEConstructorAlias(op schema.Operation) bool {
	prefix := strings.SplitN(RenderType(op.Response), ".", 2)[0]
	return prefix == "frame" || prefix == "value" || shadowsNamespace(op, prefix)
}

func (p namespacePlan) sseConstructorReference(op schema.Operation) string {
	qualified := RenderType(op.Response)
	if needsSSEConstructorAlias(op) {
		return p.events[qualified]
	}
	return qualified
}

func planNamespaceImports(group schema.APIGroup) namespacePlan {
	p := namespacePlan{events: map[string]string{}}
	used := map[string]bool{group.Name: true, "client": true, "basePath": true, "json": true,
		"input": true, "requestBuilder": true, "frame": true, "value": true}
	for _, op := range group.Operations {
		used[op.Name] = true
		for _, parameter := range op.Parameters {
			used[parameter.Name] = true
		}
	}
	allocate := func(base string) string { return allocateNamespaceAlias(used, base) }
	var out strings.Builder
	for _, op := range group.Operations {
		if p.methodAlias == "" && shadowsNamespace(op, "io") {
			p.methodAlias = allocate("OpenapiKtHttpMethodImport")
			out.WriteString("import io.ktor.http.HttpMethod as " + p.methodAlias + "\n")
		}
		if p.contentAlias == "" && op.Body != nil && shadowsNamespace(op, "io") {
			p.contentAlias = allocate("OpenapiKtContentTypeImport")
			out.WriteString("import io.ktor.http.ContentType as " + p.contentAlias + "\n")
		}
		if op.SSE == nil {
			continue
		}
		if p.flowAlias == "" && shadowsNamespace(op, "kotlinx") {
			p.flowAlias = allocate("OpenapiKtFlowImport")
			out.WriteString("import kotlinx.coroutines.flow.flow as " + p.flowAlias + "\n")
		}
		qualified := RenderType(op.Response)
		if needsSSEConstructorAlias(op) && p.events[qualified] == "" {
			alias := allocate("OpenapiKtSSEEventImport" + strconv.Itoa(len(p.events)+1))
			p.events[qualified] = alias
			out.WriteString("import " + qualified + " as " + alias + "\n")
		}
	}
	if out.Len() != 0 {
		out.WriteByte('\n')
	}
	p.imports = out.String()
	return p
}

func allocateNamespaceAlias(used map[string]bool, base string) string {
	name := base
	for suffix := 2; used[name]; suffix++ {
		name = base + strconv.Itoa(suffix)
	}
	used[name] = true
	return name
}

func planClientNamespaceImports(client *schema.Client, apiPackage string) (string, map[string]string) {
	used := map[string]bool{client.Name: true, "httpClient": true, "basePath": true}
	if clientNeedsJSON(client) {
		used["json"] = true
	}
	for _, group := range client.Groups {
		used[group.Property] = true
	}
	shadowed := used[strings.SplitN(apiPackage, ".", 2)[0]]
	refs := map[string]string{}
	var out strings.Builder
	for ordinal, group := range client.Groups {
		qualified := apiPackage + "." + group.Name
		refs[group.Name] = qualified
		if shadowed {
			alias := allocateNamespaceAlias(used, "OpenapiKtAPIImport"+strconv.Itoa(ordinal+1))
			refs[group.Name] = alias
			out.WriteString("import " + qualified + " as " + alias + "\n")
		}
	}
	if out.Len() != 0 {
		out.WriteByte('\n')
	}
	return out.String(), refs
}
