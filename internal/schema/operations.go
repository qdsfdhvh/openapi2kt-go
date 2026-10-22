package schema

import (
	"slices"
	"sort"
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// Plan keeps model and HTTP decisions in the same validated, pure pipeline.
// A nil Client means model-only output, preserving its artifact set exactly.
type Plan struct {
	Models        []Declaration
	Client        *Client
	PatchPackage  string
	BigintPackage string
	EmissionHints EmissionHints
	// Recorded role+suffix provenance; ambiguous same-name allocations are empty.
	operationRoles   map[string]string
	operationOrigins map[string][]string
	// Generated optimizer aliases are typed edges, unlike caller code spellings.
	ownedAliases map[*ScalarAlias]string
}

func (p *Plan) ownedAlias(name, target string) *ScalarAlias {
	alias := &ScalarAlias{ClassName: name, Scalar: target}
	if p.ownedAliases == nil {
		p.ownedAliases = map[*ScalarAlias]string{}
	}
	p.ownedAliases[alias] = target
	return alias
}

type Client struct {
	Name, BasePath     string
	HasServerOverrides bool
	ServerDefaults     []string
	Groups             []APIGroup
	Models             []Declaration
}

type APIGroup struct {
	Name, Property, Tag string
	Operations          []Operation
}

type Operation struct {
	Name, Method, Path, Summary string
	// BasePath is the resolved per-operation default when server scope differs
	// from the document. A supplied client basePath still overrides every scope.
	BasePath            *string
	ServerDefault       string // Stable shared helper identity; empty keeps the established root-only API.
	Parameters          []Parameter
	Response            TypeRef
	Errors              []ErrorResponse
	Body                *RequestBody
	SSE                 *SSEEvents
	Models              []Declaration
	Pointer, ModelOwner string
	FallbackModels      bool
}

type Parameter struct {
	Name, Wire, Location       string
	Type                       TypeRef
	Required, Enum, ValueClass bool
	Format                     CollectionFormat
	InlineEnum                 *StringEnum
	ElementNullable            bool
	DeepObject                 *DeepObject
}

// CollectionFormat is a wire decision, independent of Kotlin type rendering.
// Enum describes the scalar or supported container leaf, never the container.
type CollectionFormat uint8

const (
	ScalarParameter CollectionFormat = iota
	MultiParameter
	CSVParameter
	SpaceParameter
	PipeParameter
	IndexedParameter
	MapParameter
	MapCSVParameter
	LabelParameter
	LabelExplodedParameter
	MatrixParameter
	MatrixExplodedParameter
	MapDeepObjectParameter
)

func operationUnsupported(pointer string) *diagnostic.Diagnostic {
	return diagnostic.At("E_OPERATIONS_UNSUPPORTED", "This HTTP construct is not supported.", pointer)
}

// Operation objects have different keyword sets from schemas. Never use the
// schema annotation allowlist to silently admit an HTTP semantic extension.
func operationKeys(v *document.Value, allowed []string, pointer string) *diagnostic.Diagnostic {
	if err := object(v, pointer); err != nil {
		return err
	}
	for _, m := range v.Members() {
		if !slices.Contains(allowed, m.Name) {
			return operationUnsupported(naming.Pointer(pointer, m.Name))
		}
	}
	for _, key := range []string{"summary", "description"} {
		if has(v, key) && get(v, key).Kind() != document.String {
			return invalid("Expected a text annotation.", naming.Pointer(pointer, key))
		}
	}
	if has(v, "deprecated") && get(v, "deprecated").Kind() != document.Boolean {
		return invalid("Expected a boolean annotation.", naming.Pointer(pointer, "deprecated"))
	}
	return nil
}

func (a *analyzer) clientPlan(root *document.Value) (*Client, *diagnostic.Diagnostic) {
	paths := get(root, "paths")
	if paths == nil || len(paths.Members()) == 0 {
		return nil, nil
	}
	if has(root, "security") {
		return nil, operationUnsupported("/security")
	}
	client := &Client{Name: "ApiClient"}
	if has(root, "info") {
		info := get(root, "info")
		if err := object(info, "/info"); err != nil {
			return nil, err
		}
		if has(info, "title") {
			title, ok := get(info, "title").Text()
			if !ok {
				return nil, invalid("Title must be a string.", "/info/title")
			}
			if a.options.ClientName == "" && strings.TrimSpace(title) != "" {
				name, err := naming.APIClassName(title, "/info/title")
				if err != nil {
					return nil, err
				}
				client.Name = strings.TrimSuffix(strings.TrimSuffix(name, "API"), "Api") + "ApiClient"
			}
		}
	}
	if a.options.ClientName != "" {
		client.Name = a.options.ClientName
	}
	servers := &serverPlanner{remaining: document.MaxDocumentBytes}
	if has(root, "servers") {
		var err *diagnostic.Diagnostic
		client.BasePath, err = servers.serverBasePath(get(root, "servers"), "/servers")
		if err != nil {
			return nil, err
		}
	}
	groups := map[string]*APIGroup{}
	operationSources := map[string]*document.Value{}
	count := 0
	for _, entry := range httpMembers(paths) {
		p := naming.Pointer("/paths", entry.Name)
		if !strings.HasPrefix(entry.Name, "/") || strings.ContainsAny(entry.Name, "?#\r\n") {
			return nil, invalid("Expected a path template beginning with slash.", p)
		}
		item := entry.Value
		if err := operationKeys(item, []string{"summary", "description", "servers", "parameters", "get", "put", "post", "delete", "options", "head", "patch", "trace"}, p); err != nil {
			return nil, err
		}
		var pathBase *string
		if has(item, "servers") {
			value, err := servers.serverBasePath(get(item, "servers"), p+"/servers")
			if err != nil {
				return nil, err
			}
			pathBase = &value
		}
		base, err := a.parameters(get(item, "parameters"), has(item, "parameters"), p+"/parameters")
		if err != nil {
			return nil, err
		}
		for _, method := range []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"} {
			if !has(item, method) {
				continue
			}
			count++
			if count > 16384 {
				return nil, diagnostic.At("E_OPERATION_LIMIT", "Too many HTTP operations.", p)
			}
			opPointer := p + "/" + method
			v := get(item, method)
			if err := operationKeys(v, []string{"operationId", "tags", "summary", "description", "deprecated", "servers", "parameters", "responses", "requestBody"}, opPointer); err != nil {
				return nil, err
			}
			opBase := pathBase
			if has(v, "servers") {
				value, err := servers.serverBasePath(get(v, "servers"), opPointer+"/servers")
				if err != nil {
					return nil, err
				}
				opBase = &value
			}
			tag := "Default"
			if has(v, "tags") {
				tags := get(v, "tags")
				if tags.Kind() != document.Array {
					return nil, invalid("tags must be an array.", opPointer+"/tags")
				}
				for i, value := range tags.Items() {
					s, ok := value.Text()
					if !ok {
						return nil, invalid("Tag must be a string.", naming.Pointer(opPointer+"/tags", strconv.Itoa(i)))
					}
					if i == 0 {
						tag = s
					}
				}
			}
			name, err := naming.APIClassName(tag, opPointer+"/tags")
			if err != nil {
				return nil, err
			}
			name += "Api"
			var id *string
			if has(v, "operationId") {
				s, ok := get(v, "operationId").Text()
				if !ok || s == "" {
					return nil, invalid("operationId must be nonempty text.", opPointer+"/operationId")
				}
				id = &s
			}
			params, err := a.parameters(get(v, "parameters"), has(v, "parameters"), opPointer+"/parameters")
			if err != nil {
				return nil, err
			}
			params = mergeParameters(base, params)
			if err := pathParameters(entry.Name, params, opPointer); err != nil {
				return nil, err
			}
			methodName, err := naming.MethodName(id, method, entry.Name, opPointer+"/operationId")
			if err != nil {
				return nil, err
			}
			owner := methodName
			if id != nil {
				owner = *id
			}
			body, err := a.requestBody(get(v, "requestBody"), opPointer+"/requestBody", owner)
			if err != nil {
				return nil, err
			}
			if err := nameParameters(params, opPointer, body); err != nil {
				return nil, err
			}
			models, err := a.bindParameterModels(params, owner, opPointer)
			if err != nil {
				return nil, err
			}
			if body != nil {
				models = append(models, body.Models...)
			}
			var errors []ErrorResponse
			var stream *SSEEvents
			response, err := a.responseType(get(v, "responses"), opPointer+"/responses", owner, &models, &errors, &stream)
			if err != nil {
				return nil, err
			}
			if stream != nil {
				params, err = sseParameters(params, opPointer, body)
				if err != nil {
					return nil, err
				}
			}
			group := groups[name]
			if group == nil {
				group = &APIGroup{Name: name, Tag: tag}
				groups[name] = group
			}
			if tag < group.Tag {
				group.Tag = tag
			}
			operationSources[opPointer] = v
			group.Operations = append(group.Operations, Operation{Name: methodName, Method: method, Path: entry.Name, BasePath: opBase, Summary: text(get(v, "summary")), Parameters: params, Response: response, Errors: errors, Body: body, SSE: stream, Models: models, Pointer: opPointer, ModelOwner: owner, FallbackModels: id == nil})
		}
	}
	if count == 0 {
		return nil, operationUnsupported("/paths")
	}
	used := map[string]bool{}
	excluded := make(map[string]bool, len(a.options.ExcludeAPIs))
	for _, name := range a.options.ExcludeAPIs {
		excluded[name] = true
	}
	for _, name := range sortedKeys(groups) {
		if excluded[name] {
			continue
		}
		group := groups[name]
		if used[strings.ToLower(name)] {
			return nil, diagnostic.At("E_NAME_COLLISION", "API filenames collide case-insensitively.", "/paths")
		}
		used[strings.ToLower(name)] = true
		property, err := naming.APICamelName(group.Tag, "/paths")
		if err != nil {
			return nil, err
		}
		if slices.Contains([]string{"httpClient", "basePath", "json"}, property) {
			property += "Api"
		}
		group.Property = property
		renameMethods(group)
		for i := range group.Operations {
			op := &group.Operations[i]
			if !op.FallbackModels || op.ModelOwner == op.Name {
				continue
			}
			// Re-plan fallback identities under the final method name. Initial
			// analysis already owns warnings; do not duplicate them on this pass.
			bound := *a
			bound.warnings = nil
			models, err := bound.bindParameterModels(op.Parameters, op.Name, op.Pointer)
			if err != nil {
				return nil, err
			}
			source := operationSources[op.Pointer]
			body, err := bound.requestBody(get(source, "requestBody"), op.Pointer+"/requestBody", op.Name)
			if err != nil {
				return nil, err
			}
			if body != nil {
				models = append(models, body.Models...)
			}
			var errors []ErrorResponse
			var stream *SSEEvents
			response, err := bound.responseType(get(source, "responses"), op.Pointer+"/responses", op.Name, &models, &errors, &stream)
			if err != nil {
				return nil, err
			}
			op.Models, op.Response, op.ModelOwner, op.Body, op.Errors, op.SSE = models, response, op.Name, body, errors, stream
		}
		client.Groups = append(client.Groups, *group)
	}
	used = map[string]bool{}
	for _, group := range client.Groups {
		if used[group.Property] {
			return nil, diagnostic.At("E_NAME_COLLISION", "Client accessors collide.", "/paths")
		}
		used[group.Property] = true
	}
	planServerDefaults(client)
	if err := a.operationDeclarations(client); err != nil {
		return nil, err
	}
	return client, nil
}

// HTTP path/status/media objects are unordered inventories. Sort borrowed
// member views before semantic analysis so diagnostics cannot depend on JSON
// object spelling order; parameter/tag arrays retain their meaningful order.
func httpMembers(v *document.Value) []document.Member {
	members := slices.Clone(v.Members())
	sort.Slice(members, func(i, j int) bool { return members[i].Name < members[j].Name })
	return members
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// Transparent aliases have their backing type's erased signature. Nominal
// DTO/enum types have no Element; InlineValue identity must not be unboxed.
// Collection alias graphs are already validated before operation allocation.
func erasedMethodType(t TypeRef) string {
	for t.Element != nil && (t.Kind == Nullable || t.Kind == Named) {
		t = *t.Element
	}
	// These builtin projections render exactly the same classifiers as accepted
	// caller-native scalars. Source kind is not a Kotlin signature distinction.
	// Other kinds (especially nominal/value-class identities) remain barriers.
	switch t.Kind {
	case JSON:
		t.Kind, t.Name = Scalar, "kotlinx.serialization.json.JsonElement"
	case JSONObject:
		t.Kind, t.Name = Scalar, "kotlinx.serialization.json.JsonObject"
	}
	return strconv.Itoa(int(t.Kind)) + ":" + t.Name
}

func renameMethods(group *APIGroup) {
	sort.Slice(group.Operations, func(i, j int) bool {
		a, b := group.Operations[i], group.Operations[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Method < b.Method
	})
	taken, seen := map[string]bool{}, map[string]bool{}
	for _, op := range group.Operations {
		taken[op.Name] = true
	}
	for i := range group.Operations {
		op := &group.Operations[i]
		signature := op.Name
		for _, p := range op.Parameters {
			signature += "|" + erasedMethodType(p.Type)
		}
		if op.Body != nil {
			signature += "|" + erasedMethodType(op.Body.Type)
		}
		if seen[signature] {
			for n := 2; ; n++ {
				candidate := op.Name + strconv.Itoa(n)
				if !taken[candidate] {
					op.Name = candidate
					taken[candidate] = true
					break
				}
			}
		}
		seen[signature] = true
	}
}
