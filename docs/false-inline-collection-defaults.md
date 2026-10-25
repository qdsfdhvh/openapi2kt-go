# Own defaults on inline false collection fields

The [ordinary named shared inline collection-field profile](false-inline-collection-fields.md) may own an outer field default. Its finite inline List/typed Map tree still ends at an actual named false component or pure alias; no synthetic component or enum conversion grants this role. The owning DTO keeps no component root annotation in this profile.

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Denied":false,
  "Maybe":{"$ref":"#/components/schemas/Denied","nullable":true},
  "Holder":{"properties":{
    "empty":{"type":"array","items":{"$ref":"#/components/schemas/Denied"},"default":[]},
    "values":{"type":"array","items":{"$ref":"#/components/schemas/Maybe"},"default":[null]}
  }}
}}}
```

Empty containers are valid. Supplied non-null false leaves reject every value, including JSON null; logically nullable leaves admit null. Each container's own nullable intent remains independent, so an optional presence carrier alone cannot authorize a null annotation. Nested inline containers, strict false aliases, and existing validation-on per-layer counts keep their original checks.

Optional fields decode their own canonical annotation through the actual collection/leaf serializers. Each construction and missing-field decode creates fresh container state. Required annotations validate completely before suppressing their initializer and default-null flag; the constructor and descriptor still require the argument. Existing explicitNulls behavior for missing required nullable parameters remains. Target or child annotations are not inherited.

The planner binds the actual retained model/property/type, required/default/initializer/omission policy, original field/container/reference nodes, canonical value and complete literal member/item/Get identities. JSON map keys resembling schema metadata remain ordinary data. Private literal records share the original false-source work budget and cannot be discarded to bypass validation. Helper names use existing nominal-type keys that remain legal JVM names.

Source depth48, cumulative false-source work16,384, default16,384 nodes/32 KiB, checked depth16, JVM240 and enum limits remain. Resource probes are independent inputs: the 240-field/15-container and near32 KiB cases do not promise a combined maximum. Enforcing uniqueness, owner component defaults, child/wrapper/container-reference annotations, tuple projection, literal-false syntax and inline/internal/composed/directional owners retain their separate boundaries.

Previously rejected eligible own annotations now generate collection initializers. Existing accepted output, JSON/HTTP codecs, Data/Plain policies, naming-lock and ownership formats require no migration.

The [fixture](../tests/fixtures/http/false-inline-collection-defaults.json), [value/options/budget contracts](../core/false_inline_collection_defaults_test.go), [actual initializer/literal guards](../internal/schema/false_inline_collection_defaults_test.go), [native lifecycle](../internal/cli/false_inline_collection_defaults_process_test.go) and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FalseInlineCollectionDefaultsContracts.kt) define the profile. `bash scripts/test.sh feature false-inline-collection-defaults` runs Data/Plain × validation off/on, baseline/optimized naming locks, four JSON settings, fresh state, controlled HTTP and the independent resource probes. Broader Boolean/default graphs, platforms and final acceptance remain open.
