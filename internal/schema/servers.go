package schema

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// serverPlanner shares an expansion budget across every server source, including
// overridden and unselected entries. Literal-only accepted URLs fit the same
// input budget; variable repetition cannot multiply it per server or operation.
type serverPlanner struct{ remaining int }

// serverBasePath validates every declared server before choosing the first.
// A present empty array explicitly selects the default empty prefix; absence
// is handled by the caller so operation/path/root inheritance stays distinct.
func (planner *serverPlanner) serverBasePath(servers *document.Value, pointer string) (string, *diagnostic.Diagnostic) {
	if servers.Kind() != document.Array {
		return "", invalid("servers must be an array.", pointer)
	}
	var selected string
	for i, server := range servers.Items() {
		p := naming.Pointer(pointer, strconv.Itoa(i))
		if err := operationKeys(server, []string{"url", "description", "variables"}, p); err != nil {
			return "", err
		}
		template, ok := get(server, "url").Text()
		if !ok {
			return "", invalid("Server URL must be a string.", p+"/url")
		}
		defaults, err := serverVariableDefaults(get(server, "variables"), has(server, "variables"), p+"/variables")
		if err != nil {
			return "", err
		}
		url, err := planner.expandServerURL(template, defaults, p+"/url")
		if err != nil {
			return "", err
		}
		if i == 0 {
			// Keep the established root-server normalization, including the
			// additional slash when a nonempty URL already ends in slash.
			selected = strings.TrimLeft(url, "/")
			if selected != "" {
				selected += "/"
			}
		}
	}
	return selected, nil
}

func serverVariableDefaults(variables *document.Value, present bool, pointer string) (map[string]string, *diagnostic.Diagnostic) {
	if !present {
		return nil, nil
	}
	if err := object(variables, pointer); err != nil {
		return nil, err
	}
	defaults := make(map[string]string, len(variables.Members()))
	for _, variable := range httpMembers(variables) {
		p := naming.Pointer(pointer, variable.Name)
		if variable.Name == "" || strings.ContainsAny(variable.Name, "{}\r\n") {
			return nil, invalid("Server variable name must be nonempty and contain no template delimiters.", p)
		}
		if err := operationKeys(variable.Value, []string{"default", "enum", "description"}, p); err != nil {
			return nil, err
		}
		value, ok := get(variable.Value, "default").Text()
		if !ok {
			return nil, invalid("Server variable default must be a string.", p+"/default")
		}
		if strings.ContainsAny(value, "{}\r\n") {
			return nil, operationUnsupported(p + "/default")
		}
		if has(variable.Value, "enum") {
			vocabulary := get(variable.Value, "enum")
			if vocabulary.Kind() != document.Array || len(vocabulary.Items()) == 0 {
				return nil, invalid("Server variable enum must be a nonempty string array.", p+"/enum")
			}
			members := make([]string, 0, len(vocabulary.Items()))
			for i, member := range vocabulary.Items() {
				text, ok := member.Text()
				if !ok {
					return nil, invalid("Server variable enum member must be a string.", naming.Pointer(p+"/enum", strconv.Itoa(i)))
				}
				members = append(members, text)
			}
			if !slices.Contains(members, value) {
				return nil, invalid("Server variable default must belong to its enum.", p+"/default")
			}
		}
		defaults[variable.Name] = value
	}
	return defaults, nil
}

// Plan only retained operations after exclusions. Shared helper identities are
// independent of client naming and do not change the existing root-only API.
func planServerDefaults(client *Client) {
	for _, group := range client.Groups {
		for _, op := range group.Operations {
			client.HasServerOverrides = client.HasServerOverrides || op.BasePath != nil
		}
	}
	if !client.HasServerOverrides {
		return
	}
	// Reuse source pointers before content lookup: a long shared root URL
	// must not be rehashed or emitted once per inheriting operation.
	bySource := map[*string]string{}
	byValue := map[string]string{}
	for g := range client.Groups {
		for i := range client.Groups[g].Operations {
			op := &client.Groups[g].Operations[i]
			if op.BasePath == nil {
				op.BasePath = &client.BasePath
			}
			identity := bySource[op.BasePath]
			if identity == "" {
				identity = byValue[*op.BasePath]
				if identity == "" {
					client.ServerDefaults = append(client.ServerDefaults, *op.BasePath)
					identity = fmt.Sprintf("%x", sha256.Sum256([]byte(*op.BasePath)))
					byValue[*op.BasePath] = identity
				}
				bySource[op.BasePath] = identity
			}
			op.ServerDefault = identity
		}
	}
}

// Expansion is one pass over original template bytes. Defaults are opaque
// strings, never recursively interpreted, and cannot amplify output beyond the
// existing document byte budget.
func (planner *serverPlanner) expandServerURL(template string, defaults map[string]string, pointer string) (string, *diagnostic.Diagnostic) {
	if strings.ContainsAny(template, "\r\n") {
		return "", operationUnsupported(pointer)
	}
	var out strings.Builder
	appendPart := func(part string) *diagnostic.Diagnostic {
		if len(part) > planner.remaining {
			return diagnostic.At("E_INPUT_LIMIT", "Server URL expansion exceeds the document byte budget.", pointer)
		}
		planner.remaining -= len(part)
		out.WriteString(part)
		return nil
	}
	for rest := template; rest != ""; {
		start := strings.IndexAny(rest, "{}")
		if start < 0 {
			if err := appendPart(rest); err != nil {
				return "", err
			}
			break
		}
		if rest[start] != '{' {
			return "", operationUnsupported(pointer)
		}
		end := strings.IndexByte(rest[start+1:], '}')
		if end < 0 {
			return "", operationUnsupported(pointer)
		}
		name := rest[start+1 : start+1+end]
		value, found := defaults[name]
		if !found || strings.ContainsRune(name, '{') {
			return "", operationUnsupported(pointer)
		}
		if err := appendPart(rest[:start]); err != nil {
			return "", err
		}
		if err := appendPart(value); err != nil {
			return "", err
		}
		rest = rest[start+end+2:]
	}
	return out.String(), nil
}
