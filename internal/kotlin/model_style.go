package kotlin

import (
	"fmt"
	"strings"

	"openapi-kt/internal/naming"
	"openapi-kt/internal/schema"
)

func classKeyword(plain bool) string {
	if plain {
		return "class"
	}
	return "data class"
}

// Equality boxes constructor fields to preserve data-class Float/Double
// semantics (including nullable floats), and array reference equality. Hash
// overload selection is STATIC: a caller-owned typealias to an array selects
// contentHashCode, while an Any holding an array retains ordinary identity hash.
func emitPlainMembers(name string, properties []schema.Property, reserved []string, indent int, out *strings.Builder) {
	if len(properties) == 0 {
		return
	}
	pad := strings.Repeat("    ", indent)
	used := map[string]bool{name: true}
	for _, p := range properties {
		used[p.Name] = true
	}
	for _, n := range reserved {
		used[n] = true
	}
	helper := "_openapiKtPlainHash"
	for i := 2; used[helper]; i++ {
		helper = fmt.Sprintf("_openapiKtPlainHash%d", i)
	}
	fmt.Fprintf(out, "\n%soverride fun equals(other: kotlin.Any?): kotlin.Boolean {\n%s    if (this === other) return true\n%s    if (other !is %s) return false\n", pad, pad, pad, name)
	for _, p := range properties {
		fmt.Fprintf(out, "%s    if ((this.%s as kotlin.Any?) != (other.%s as kotlin.Any?)) return false\n", pad, p.Name, p.Name)
	}
	fmt.Fprintf(out, "%s    return true\n%s}\n\n%soverride fun hashCode(): kotlin.Int {\n%s    var result = this.%s(this.%s)\n", pad, pad, pad, pad, helper, properties[0].Name)
	for _, p := range properties[1:] {
		fmt.Fprintf(out, "%s    result = 31 * result + this.%s(this.%s)\n", pad, helper, p.Name)
	}
	fmt.Fprintf(out, "%s    return result\n%s}\n\n%soverride fun toString(): kotlin.String = %s\n", pad, pad, pad, naming.Literal(name))
	fmt.Fprintf(out, "\n%sprivate fun %s(value: kotlin.Any?): kotlin.Int = value?.hashCode() ?: 0\n", pad, helper)
	for _, ty := range []string{"ByteArray", "ShortArray", "IntArray", "LongArray", "FloatArray", "DoubleArray", "CharArray", "BooleanArray", "Array<*>"} {
		fmt.Fprintf(out, "%sprivate fun %s(value: kotlin.%s?): kotlin.Int = value?.contentHashCode() ?: 0\n", pad, helper, ty)
	}
}
