package kotlin

import (
	"fmt"
	"strings"

	"openapi-kt/internal/naming"
	"openapi-kt/internal/schema"
)

func wireParameter(expr string, p schema.Parameter) string {
	if p.ValueClass {
		if p.ElementNullable {
			return expr + "?.rawValue"
		}
		return expr + ".rawValue"
	}
	if p.Enum {
		if p.ElementNullable {
			return expr + "?.serialName"
		}
		return expr + ".serialName"
	}
	return expr
}

func joinedParameter(expr string, p schema.Parameter) string {
	separator := ","
	switch p.Format {
	case schema.SpaceParameter:
		separator = " "
	case schema.PipeParameter:
		separator = "|"
	}
	joined := expr + ".joinToString(" + naming.Literal(separator) + ")"
	if p.ValueClass {
		if p.ElementNullable {
			joined += " { it?.rawValue?.toString() ?: \"null\" }"
		} else {
			joined += " { it.rawValue.toString() }"
		}
	} else if p.Enum {
		if p.ElementNullable {
			joined += " { it?.serialName ?: \"null\" }"
		} else {
			joined += " { it.serialName }"
		}
	}
	return joined
}

// All parameter wire decisions share this emitter; future streaming requests
// must reuse it instead of reconstructing collection/default/null semantics.
func emitRequestParameter(p schema.Parameter, out *strings.Builder) {
	name := naming.CodeName(p.Name)
	if p.DeepObject != nil {
		receiver := name
		if !p.Required {
			receiver += "?"
		}
		fmt.Fprintf(out, "            %s.let { deepValue ->\n", receiver)
		for _, field := range p.DeepObject.Fields {
			expr := "deepValue." + field.Name
			value := expr
			if field.Nullable {
				value = "it"
			}
			if field.Enum {
				value += ".serialName"
			} else if field.ValueClass {
				value += ".rawValue"
			}
			key := naming.Literal(p.Wire + "[" + field.Wire + "]")
			if field.Nullable {
				fmt.Fprintf(out, "                %s?.let { parameter(%s, %s) }\n", expr, key, value)
			} else {
				fmt.Fprintf(out, "                parameter(%s, %s)\n", key, value)
			}
		}
		out.WriteString("            }\n")
		return
	}
	nullable := p.Type.Kind == schema.Nullable
	receiver := name
	if nullable {
		receiver += "?"
	}
	key := naming.Literal(p.Wire)
	switch p.Format {
	case schema.MultiParameter:
		fmt.Fprintf(out, "            %s.forEach { parameter(%s, %s) }\n", receiver, key, wireParameter("it", p))
	case schema.IndexedParameter:
		fmt.Fprintf(out, "            %s.forEachIndexed { index, item -> parameter(%s + index + %s, %s) }\n", receiver, naming.Literal(p.Wire+"["), naming.Literal("]"), wireParameter("item", p))
	case schema.MapParameter:
		fmt.Fprintf(out, "            %s.forEach { (key, value) -> parameter(key, %s) }\n", receiver, wireParameter("value", p))
	case schema.MapDeepObjectParameter:
		fmt.Fprintf(out, "            %s.forEach { (key, value) -> parameter(%s + key + %s, %s) }\n", receiver, naming.Literal(p.Wire+"["), naming.Literal("]"), wireParameter("value", p))
	case schema.MapCSVParameter:
		expr := name
		if nullable {
			expr = "it"
		}
		value := expr + ".entries.joinToString(\",\") { (key, value) -> key + \",\" + value.toString() }"
		if nullable {
			fmt.Fprintf(out, "            %s?.let { parameter(%s, %s) }\n", name, key, value)
		} else {
			fmt.Fprintf(out, "            parameter(%s, %s)\n", key, value)
		}
	default:
		call := "parameter"
		if p.Location == "header" {
			call = "header"
		}
		expr := name
		if nullable {
			expr = "it"
		}
		value := wireParameter(expr, p)
		if p.Format != schema.ScalarParameter {
			value = joinedParameter(expr, p)
		}
		if nullable {
			fmt.Fprintf(out, "            %s?.let { %s(%s, %s) }\n", name, call, key, value)
		} else {
			fmt.Fprintf(out, "            %s(%s, %s)\n", call, key, value)
		}
	}
}

// Style separators remain syntax; each value is encoded as a path part before
// assembly so slashes and reserved characters cannot alter the route.
func styledPathParameter(p schema.Parameter) string {
	name := naming.CodeName(p.Name)
	value := wireParameter(name, p) + ".toString().encodeURLPathPart()"
	isList := p.Type.WithoutRootNull().Kind == schema.List
	key := naming.Literal(p.Wire) + ".encodeURLPathPart()"
	if p.Type.WithoutRootNull().Kind == schema.Map {
		item := wireParameter("value", p) + ".toString().encodeURLPathPart()"
		entry := "key.encodeURLPathPart() + \",\" + " + item
		separator := ","
		switch p.Format {
		case schema.LabelExplodedParameter:
			entry = "key.encodeURLPathPart() + \"=\" + " + item
			separator = "."
		case schema.MatrixExplodedParameter:
			return name + ".entries.joinToString(\"\") { (key, value) -> \";\" + key.encodeURLPathPart() + \"=\" + " + item + " }"
		}
		value = name + ".entries.joinToString(" + naming.Literal(separator) + ") { (key, value) -> " + entry + " }"
	} else if isList {
		separator := ","
		if p.Format == schema.LabelExplodedParameter {
			separator = "."
		}
		item := wireParameter("it", p) + ".toString().encodeURLPathPart()"
		if p.Format == schema.MatrixExplodedParameter {
			return name + ".joinToString(\"\") { \";\" + " + key + " + \"=\" + " + item + " }"
		}
		value = name + ".joinToString(" + naming.Literal(separator) + ") { " + item + " }"
	}
	if p.Format == schema.MatrixParameter || p.Format == schema.MatrixExplodedParameter {
		return "\";\" + " + key + " + \"=\" + " + value
	}
	return "\".\" + " + value
}
