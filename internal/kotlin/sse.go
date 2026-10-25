package kotlin

import (
	"fmt"
	"strings"

	"openapi-kt/internal/naming"
	"openapi-kt/internal/schema"
)

func groupHasSSE(group schema.APIGroup) bool {
	for _, op := range group.Operations {
		if op.SSE != nil {
			return true
		}
	}
	return false
}
func clientHasSSE(client *schema.Client) bool {
	for _, group := range client.Groups {
		if groupHasSSE(group) {
			return true
		}
	}
	return false
}

func emitSSEEvents(events *schema.SSEEvents, out *strings.Builder) {
	fmt.Fprintf(out, "public sealed class %s {\n    public abstract val event: kotlin.String\n    public abstract val id: kotlin.String?\n\n", events.ClassName)
	for _, v := range events.Variants {
		fmt.Fprintf(out, "    public %s %s(\n", classKeyword(events.Plain), v.Name)
		if v.Data != nil {
			fmt.Fprintf(out, "        public val data: %s,\n", RenderType(*v.Data))
		}
		fmt.Fprintf(out, "        public override val id: kotlin.String? = null,\n    ) : %s() {\n        public override val event: kotlin.String = %s\n", events.ClassName, naming.Literal(v.Event))
		if v.Data != nil && typeHasValidation(*v.Data) {
			out.WriteString("        init {\n")
			t := *v.Data
			if !v.Required {
				t = t.Null()
			}
			emitTypeChecks(t, "this.data", "Field value", 3, out)
			out.WriteString("        }\n")
		}
		if events.Plain {
			props := []schema.Property{}
			if v.Data != nil {
				props = append(props, schema.Property{Name: "data"})
			}
			props = append(props, schema.Property{Name: "id"})
			emitPlainMembers(v.Name, props, []string{"event"}, 2, out)
		}
		out.WriteString("    }\n\n")
	}
	out.WriteString("}\n")
}

func emitSSEOperation(group schema.APIGroup, op schema.Operation, checks operationCheckPlan, namespaces namespacePlan, out *strings.Builder) {
	fmt.Fprintf(out, "    public fun %s(\n", naming.CodeName(op.Name))
	for _, p := range op.Parameters {
		def := ""
		if !p.Required {
			def = " = null"
		}
		fmt.Fprintf(out, "        %s: %s%s,\n", naming.CodeName(p.Name), RenderType(p.Type), def)
	}
	if op.Body != nil {
		def := ""
		if !op.Body.Required {
			def = " = null"
		}
		fmt.Fprintf(out, "        input: %s%s,\n", RenderType(op.Body.Type), def)
	}
	out.WriteString("        requestBuilder: (io.ktor.client.request.HttpRequestBuilder.() -> kotlin.Unit)? = null,\n")
	fmt.Fprintf(out, "    ): kotlinx.coroutines.flow.Flow<%s> = %s {\n", RenderType(op.Response), namespaces.flowBuilderReference(op))
	emitOperationInputChecks(op, checks, 2, out)
	fmt.Fprintf(out, "        this@%s.client.sse(\n            request = {\n                url.takeFrom(%s)\n                this.method = %s(%s)\n", group.Name, urlExpression(group.Name, op), namespaces.httpMethodConstructor(op), naming.Literal(strings.ToUpper(op.Method)))
	if op.Body != nil {
		emitRequestBody(op, namespaces, 4, out)
	}
	for _, p := range op.Parameters {
		if p.Location != "path" {
			var parameters strings.Builder
			emitRequestParameter(p, &parameters)
			out.WriteString(strings.ReplaceAll(parameters.String(), "            ", "                "))
		}
	}
	out.WriteString("                requestBuilder?.invoke(this)\n            },\n            showCommentEvents = false,\n            showRetryEvents = false,\n        ) {\n            incoming.collect { frame ->\n                val value = when (frame.event?.takeIf { it.isNotEmpty() } ?: \"message\") {\n")
	for _, v := range op.SSE.Variants {
		fmt.Fprintf(out, "                    %s -> ", naming.Literal(v.Event))
		if v.Data != nil && v.Required {
			out.WriteString("if (frame.data == null) null else ")
		}
		fmt.Fprintf(out, "%s.%s(", namespaces.sseConstructorReference(op), v.Name)
		if v.Data != nil {
			if !v.Required {
				out.WriteString("if (frame.data == null) null else ")
			}
			fmt.Fprintf(out, "this@%s.json.decodeFromString<%s>(frame.data!!), ", group.Name, RenderType(*v.Data))
		}
		out.WriteString("frame.id)\n")
	}
	out.WriteString("                    else -> null\n                }\n                if (value != null) emit(value)\n            }\n        }\n    }\n\n")
}
