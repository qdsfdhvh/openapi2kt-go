# Structured const values

Plain non-null object and array `const` declarations can project to `JsonObject` and `JsonArray` in value positions. Explicit matching `type` and omitted `type` retain the same classifier. Component roots export transparent Kotlin typealiases; unmarked or explicitly false nullable component aliases retain the canonical backing. Literal keys such as `$ref`, `type`, `properties`, `default` and `enum` remain JSON data.

```json
{
  "Object": {"const": {"version": 1}},
  "Alias": {"$ref": "#/components/schemas/Object"},
  "Values": {
    "properties": {
      "fixed": {"$ref": "#/components/schemas/Alias"},
      "list": {"type": "array", "items": {"const": {"example": true}}}
    }
  }
}
```

`Object` and `Alias` expose the kotlinx.serialization JsonObject classifier. Root alias assignment and serialization have no literal-membership barrier. List/Map values and typed HTTP inputs/results/SSE data retain their respective JSON kinds; object values reject arrays, arrays reject objects, and non-null values reject JSON null. Other objects/arrays of the same kind are accepted and preserved. Query parameters retain Ktor `toString` projection: these JSON classifiers become one JSON-text parameter value, with URL escaping, rather than recursive query expansion or use of the client's JSON configuration. There is no value-position default: callers supply HTTP/root values and list/map elements. `--validation` does not enforce equality with the declared literal.

Each original value declaration emits `W_CONST_STRUCTURED_APPROXIMATION` at its `/const`, describing kind projection without literal membership or a value-position default. Component aliases and uses retain original definition warnings; an ordinary DTO reference does not invent a `/const` pointer on the reference node. The backing JSON codec retains its normal numeric spelling/precision policy, without arbitrary-precision guarantees.

Ordinary DTO fields that reference a structured const declaration use the separate [body constant contract](structured-const-fields.md): no constructor/copy parameter, an always-encoded body default, default value on missing JSON, same-kind alternatives retained on decoding and a non-null field even when the consumer is marked nullable. Direct const fields keep their original behavior. Collection/reference consumer nullability remains independent: a list item referencing a non-null root with `nullable: true` permits null values.

Intrinsic nullable roots and own-nullable component aliases have the separate [nullable structured const contract](nullable-structured-const.md). Effective `nullable: false` keeps the non-null backing. `const` plus `default`, sibling items/properties/additional schemas, formats/native mappings, value-class extensions and unsupported constraints remain diagnosed at their original schema pointers. Union/hierarchy data bodies use the [pinned body contract](pinned-structured-const.md). Discarded extra-key and overridden declarations use their [validation-only contract](discarded-structured-const.md); general internal-reference/composition graphs retain their separate exclusions. Source checks run before dropping overridden field contributions.

Structured const arrays are JSON classifiers, rather than transparent List aliases. Their complete alias/source hop budget remains 48; consumers and fields do not reset cached work. JSON literal depth uses the document loader budget, independent of schema edges. The 8 MiB input, composition work 16,384, checked collection edges 16 and 240 serialized/method slots remain unchanged. Literal data is never traversed as schema keywords. Overloads use the actual JsonObject/JsonArray classifier, including matching caller-provided native classifiers.

Migration: these value profiles and ordinary DTO references were previously rejected. The original field-only negative inputs are preserved verbatim as explicit root/leaf graduations, with malformed sources still covered by negative checks. Existing accepted schemas retain generated APIs, warnings, bytes and name-lock identities. A newly admitted schema can be rejected under warnings-as-errors. Independent boundary evidence does not establish Cartesian maxima, minimum Kotlin/Gradle versions, Android or KMP runtime support.

Run `bash scripts/test.sh feature structured-const-values` for fresh Data/Plain × validation projects with all four JSON settings and controlled HTTP/fragmented live SSE assertions. See the [fixture](../tests/fixtures/http/structured-const-values.json), [core contract](../core/structured_const_values_test.go), [grammar/source guards](../core/structured_const_values_guards_test.go), [source/erasure limits](../core/structured_const_values_resources_test.go), [native preservation](../internal/cli/structured_const_values_process_test.go) and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/StructuredConstValuesContracts.kt).

[Structured const tuple sources](tuple-structured-const.md) separately admit original JsonObject/JsonArray positions and existing references, with strict complete-literal outer annotations and data-only capture of const/default/enum payloads. Original identity, nullable/Patch and cumulative budgets remain; broader union/recursive/default graphs stay independent.
