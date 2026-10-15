package schema

import (
	"sort"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// jsonMedia ranks exact application/json before parameterized/case variants,
// then structured syntax +json. Source order cannot choose the wire type.
func jsonMedia(content *document.Value) string {
	keys := []string{}
	for _, m := range content.Members() {
		keys = append(keys, m.Name)
	}
	sort.Strings(keys)
	rank := func(s string) int {
		if s == "application/json" {
			return 0
		}
		base := strings.ToLower(strings.TrimSpace(strings.SplitN(s, ";", 2)[0]))
		if base == "application/json" {
			return 1
		}
		if strings.HasPrefix(base, "application/") && strings.HasSuffix(base, "+json") {
			return 2
		}
		return 3
	}
	best := ""
	priority := 3
	for _, key := range keys {
		if r := rank(key); r < priority {
			best = key
			priority = r
		}
	}
	return best
}

func successRank(status string) int {
	switch status {
	case "200":
		return 0
	case "201":
		return 1
	case "202":
		return 2
	case "203":
		return 3
	case "206":
		return 4
	}
	if len(status) == 3 && status[0] == '2' && status[1] >= '0' && status[1] <= '9' && status[2] >= '0' && status[2] <= '9' {
		return 5
	}
	if strings.EqualFold(status, "2XX") {
		return 6
	}
	return 7
}

type ErrorResponse struct {
	Status  string
	HasBody bool
	Type    *TypeRef
}

func errorStatus(status string) bool {
	return status == "default" || strings.HasPrefix(status, "4") || strings.HasPrefix(status, "5")
}

func errorRank(status string) int {
	if status == "default" {
		return 2
	}
	if strings.ContainsAny(status, "xX") {
		return 1
	}
	return 0
}

func (a *analyzer) responseType(responses *document.Value, pointer, owner string, models *[]Declaration, errors *[]ErrorResponse, stream **SSEEvents) (TypeRef, *diagnostic.Diagnostic) {
	unit := TypeRef{Kind: Scalar, Name: "kotlin.Unit"}
	if responses.Kind() != document.Object || len(responses.Members()) == 0 {
		return TypeRef{}, invalid("responses must be a nonempty object.", pointer)
	}
	types := map[string]TypeRef{}
	bodies := map[string]bool{}
	planned := map[string][]Declaration{}
	streams := map[string]*SSEEvents{}
	declaredErrors := map[string]*ErrorResponse{}
	for _, entry := range httpMembers(responses) {
		p := naming.Pointer(pointer, entry.Name)
		status := entry.Name
		valid := status == "default"
		if len(status) == 3 && status[0] >= '1' && status[0] <= '5' {
			valid = (status[1] >= '0' && status[1] <= '9' && status[2] >= '0' && status[2] <= '9') || strings.EqualFold(status[1:], "XX")
		}
		if !valid {
			return TypeRef{}, invalid("Invalid response status.", p)
		}
		isError := errorStatus(status)
		if isError {
			declaredErrors[status] = &ErrorResponse{Status: status}
		}
		response, sourcePointer, err := a.operationReference(entry.Value, p)
		if err != nil {
			return TypeRef{}, err
		}
		p = sourcePointer
		if err := operationKeys(response, []string{"description", "content"}, p); err != nil {
			return TypeRef{}, err
		}
		types[status] = unit
		if !has(response, "content") {
			continue
		}
		content := get(response, "content")
		if err := object(content, p+"/content"); err != nil {
			return TypeRef{}, err
		}
		if len(content.Members()) == 0 {
			continue
		}
		media := jsonMedia(content)
		if media == "" {
			media = sseMedia(content)
		}
		if !isError && (media == "" || successRank(status) == 7) {
			return TypeRef{}, operationUnsupported(p + "/content")
		}
		// Validate every media entry, even when preference selects a different one.
		for _, entry := range httpMembers(content) {
			mp := naming.Pointer(p+"/content", entry.Name)
			if err := operationKeys(entry.Value, []string{"schema", "example", "examples"}, mp); err != nil {
				return TypeRef{}, err
			}
			if strings.EqualFold(strings.TrimSpace(strings.SplitN(entry.Name, ";", 2)[0]), "text/event-stream") {
				// Ktor's SSE handshake accepts HTTP 200 (and treats 204 as
				// termination). Do not plan typed streams for other statuses.
				if isError || status != "200" {
					return TypeRef{}, operationUnsupported(mp)
				}
				events, declarations, err := a.sseSchema(get(entry.Value, "schema"), owner, mp+"/schema")
				if err != nil {
					return TypeRef{}, err
				}
				if entry.Name == media {
					types[status] = TypeRef{Kind: Named, Name: a.options.PackageName + ".api.models." + events.ClassName}
					planned[status], streams[status], bodies[status] = declarations, events, true
				}
				continue
			}
			if isError {
				schema := get(entry.Value, "schema")
				if schema != nil {
					context := *a
					context.usage = SharedUsage
					// Allocate shared provenance maps before copying the analyzer so
					// selected inline error roots participate in later relocation.
					if a.operationRoles == nil {
						a.operationRoles = map[string]string{}
					}
					if a.operationOrigins == nil {
						a.operationOrigins = map[string][]string{}
					}
					context.operationRoles, context.operationOrigins = a.operationRoles, a.operationOrigins
					var scratch []Declaration
					var ty TypeRef
					var err *diagnostic.Diagnostic
					inline := entry.Name == media && inlineErrorModelCandidate(schema, mp+"/schema")
					if inline {
						ty, scratch, err = context.inlineErrorType(schema, owner, status, mp+"/schema")
					} else {
						ty, err = context.propertyType(schema, "$Error", mp+"/schema", &scratch, map[string]bool{}, 0)
					}
					if err != nil {
						return TypeRef{}, err
					}
					a.warnings = context.warnings
					if entry.Name == media {
						declaredErrors[status].HasBody = true
						if has(schema, "$ref") || inline {
							declaredErrors[status].Type = &ty
						}
						if inline {
							*models = append(*models, scratch...)
						}
					}
				}
				continue
			}
			ty, declarations, err := a.operationResponseSchema(get(entry.Value, "schema"), owner, mp+"/schema")
			if err != nil {
				return TypeRef{}, err
			}
			if entry.Name == media {
				types[status] = ty
				planned[status] = declarations
				bodies[status] = true
			}
		}
	}
	errorKeys := sortedKeys(declaredErrors)
	sort.SliceStable(errorKeys, func(i, j int) bool { return errorRank(errorKeys[i]) < errorRank(errorKeys[j]) })
	for _, status := range errorKeys {
		*errors = append(*errors, *declaredErrors[status])
	}
	keys := sortedKeys(types)
	sort.SliceStable(keys, func(i, j int) bool { return successRank(keys[i]) < successRank(keys[j]) })
	for _, key := range keys {
		if successRank(key) < 7 && bodies[key] {
			*models = append(*models, planned[key]...)
			*stream = streams[key]
			return types[key], nil
		}
	}
	for _, key := range keys {
		if successRank(key) < 7 {
			return types[key], nil
		}
	}
	// No declared success is still a Unit operation. default is never decoded
	// as success; its schema is only used by the declared-error contract.
	return unit, nil
}
