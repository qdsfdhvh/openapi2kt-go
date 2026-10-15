# Null-only defaults on DTO fields

Direct ordinary DTO fields with `type:"null"`, or an explicit/inferred null-only enum, accept `default:null`. The default is representable by the existing JsonNull domain even when the schema has no `nullable:true` annotation. It does not infer Kotlin nullability or create a new enum, wrapper or serializer.

```json
{"Holder":{"properties":{
  "required":{"type":"null","default":null},
  "optional":{"enum":[null],"default":null},
  "requiredNullable":{"type":"null","nullable":true,"default":null}
},"required":["required","requiredNullable"]}}
```

`required` is a JsonNull constructor argument; its annotation never supplies a missing required argument. `requiredNullable` is `JsonNull?` with no initializer. `optional` uses the existing `JsonNull? = null` initializer and decodes an omitted key to Kotlin null. There is no default JsonNull object or new body property. Supplied JsonNull values and Kotlin null retain the existing nullable-codec collapse on the wire.

Optional explicitly nullable inline request fields keep `Patch<JsonNull> = Patch.Absent`. A default annotation must not turn a missing request key into an update. Patch.Null and Patch.Value(JsonNull) retain their distinct Kotlin states and shared wire null. Other request/union presence policies remain as documented in the [scalar defaults](model-defaults.md), [inferred null](inferred-null-literals.md) and [explicit null](explicit-null-literals.md) contracts.

## Validation and supported contexts

Only an actual JSON null is a valid default for this profile. String `"null"`, numbers, booleans, arrays and objects fail at the original `/default` Pointer. The original enum vocabulary is validated first, including duplicate/non-null indices and the raw 240-member cap; a valid default cannot hide a malformed enum. Original discarded fields also retain their source/default diagnostics.

The field profile is available in shared/nested DTOs, supported ordinary allOf shapes, object-union fields and DTO payloads used by HTTP or SSE. Non-null required arguments and wrong-value JsonNull codec rejection are retained. With `explicitNulls=false`, the existing serializer may infer Kotlin null for a missing required nullable key; this adds no independent required-key validator.

Defaults on component roots, `$ref`/composition wrappers, List/Map leaves, fixed-object extra schemas and direct SSE data roots remain outside this field-only permission. Const+default combinations remain rejected, as do native/value-class/format enum extensions and unsupported constraints. A direct non-enum null field follows its existing scalar-format grammar. General structured/default/const support is still separate work.

## Ownership and compatibility

Existing accepted sources keep their default initializers, constructors, warnings and wire behavior. Previously rejected direct null-only default annotations become accepted; the two former enum-default rejection cases retain their original inputs in positive tests. A required nullable null-only field with a default follows the existing required-default rule and has no constructor initializer, even where the corresponding historical no-default shared nullable field had one.

The existing default-sensitive shape fingerprint domain and lock protocol remain unchanged. Equivalent null-only enum and plain null defaults have the same emitted type/default identity; lock replay and optimizer ownership remain deterministic. Source depth 48, serialized slots 240 and all other compiler/work limits remain in force. The tests do not establish every wide-by-deep combination or additional Kotlin platforms.

The [fixture](../tests/fixtures/http/null-only-defaults.json), [core tests](../core/null_only_defaults_test.go), [grammar/presence guards](../core/null_only_defaults_guards_test.go), [source/resource/lock tests](../core/null_only_defaults_resources_test.go), [native preservation tests](../internal/cli/null_only_defaults_process_test.go) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullOnlyDefaultsContracts.kt) cover this profile. Run `bash scripts/test.sh feature null-only-defaults` for fresh Data/Plain × validation projects, all four Json settings, required/optional decoding, wrong-value rejection, typed HTTP and fragmented live SSE closure/client reuse. The feature also reuses the existing null/Patch consumer against default-bearing fields; the ordinary HTTP suite also generates this fixture in the canonical null package and executes both consumers; the explicit-null standalone route keeps its original fixture.
