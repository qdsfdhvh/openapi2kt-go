# Builtin scalar defaults on direct collection elements

An ordinary named DTO without `$ref` or composition may declare an independent
`default` on the direct `items` of one of its array properties or the direct
`additionalProperties` of one of its typed Map properties. The original child
must be an inline builtin String, Int, Long, Float, Double, Boolean, UUID,
LocalDate or Instant schema, and the enclosing collection and its consumers
must already be supported. A DTO may also own an already admitted
component-root default factory.

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Holder":{"properties":{
    "names":{"type":"array","items":{"type":"string","default":"guest"}},
    "days":{"type":"array","items":{"type":"string","format":"date",
                                    "default":"2000-01-01"}},
    "scores":{"type":"object",
              "additionalProperties":{"type":"integer","default":7},
              "default":{"first":3}}
  },"default":{"scores":{"chosen":9}}}
}}}
```

The child annotation validates independently of the property annotation.
`names` does not acquire a list initializer or a `"guest"` element; `scores`
uses its own map default when the field is absent and does not add a `7` value.
The explicit `Holder.default()` factory instead produces the owner annotation's
`scores` value, `{"chosen":9}`. A missing optional collection follows its
existing presence behavior. Required and Patch
suppression do not turn a child annotation into an initializer. Existing
borrowed-collection and DTO uniqueness uses may retain this permission only
when their original source is such an eligible direct field. An independently
validated owner factory retains its own value: if it omits a collection field,
the child annotation does not fill that field; if it supplies a collection,
its literal remains authoritative.

The child's own nullable intent controls whether `null` is valid. Non-null
values use the existing scalar literal grammar, including integer ranges and
finite Float/Double mapping. Supported scalar checks apply when validation is
enabled; constrained inputs retain their existing validation admission rules.
UUID values follow the existing compact or hyphenated grammar, and supported
length and pattern checks inspect the canonical UUID text. Date values must be
valid ISO dates; date-time values use the existing collection grammar with at
most nine fractional digits and an offset no greater than 18:00. Temporal
length and pattern checks remain unsupported.
An empty or null outer default does not hide an invalid child annotation.
Invalid values fail at the child's original escaped `/default` Pointer before
output changes.

The permission is tied to the original component property and physical child
node. It does not extend to component-root collections, nested container
children, tuple members or tails, fixed-property extras, unadmitted owner
factories, ordinary `$ref` children, bytes, enum, const, native or value-class
children, or hierarchy-owned DTOs. Enum children have their own
[bounded contract](collection-reference-child-defaults.md). Other source
roles, reference graphs and general defaults keep their separate eligibility.

The existing input 8 MiB, source depth 48, cumulative source work 16,384,
default value 16,384 and 32 KiB, checked collection depth 16 and JVM slot 240
limits remain. Original source and value nodes, final element types and
validation checks remain authoritative. Previously rejected eligible
annotations now generate; for otherwise identical input, a valid child
annotation does not change generated Kotlin, wire behavior, name-lock or
output ownership formats.

[Core tests](../core/inline_builtin_collection_child_defaults_test.go),
[UUID and temporal tests](../core/inline_special_collection_child_defaults_test.go),
[owner factory tests](../core/inline_builtin_collection_child_owner_defaults_test.go),
[source and type authority tests](../internal/schema/inline_builtin_collection_child_defaults_test.go)
and [CLI preservation tests](../internal/cli/inline_builtin_collection_child_defaults_process_test.go)
cover this role. The existing `primitive-array-defaults`,
`primitive-map-defaults` and `uuid-unique` feature routes exercise fresh Kotlin consumers;
`nullable-component-defaults` exercises owner factories and fresh containers.
These consumers include missing and supplied values; runtime and platform results must be
reported for the exact candidate and target. The
[platform probe](platform-enum-field-bounds.md) also exercises builtin6 child
annotations in its common consumer; UUID and temporal child annotations
require their own platform evidence.
