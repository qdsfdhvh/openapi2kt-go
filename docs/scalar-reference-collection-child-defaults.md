# Scalar reference defaults on collection elements

An already supported physical List `items` or typed Map
`additionalProperties` schema may declare its own `default` alongside a local
`$ref` to an already admitted ordinary scalar. The target may be builtin
String, Boolean, Int, Long, Float, Double or UUID, standard `date` or
`date-time`, or `byte`/`binary` with the existing signed-integer-array wire
representation. Existing named aliases, accepted internal references and
their nullable rules continue to govern which paths are admitted.

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Count":{"type":"integer","default":1},
  "Day":{"type":"string","format":"date"},
  "Holder":{"properties":{
    "counts":{"type":"array",
              "items":{"$ref":"#/components/schemas/Count","default":2}},
    "days":{"type":"object",
            "additionalProperties":{"$ref":"#/components/schemas/Day",
                                    "default":"2024-02-29"},
            "default":{"today":"2024-03-01"}}
  }}
}}}
```

Each child annotation is validated on its own. The `counts` annotation does
not supply an element or a list initializer; `days` uses its field default
without adding a value from the child annotation. Null requires the actual
child's effective nullable view, including the nearest admitted alias or
use-site override. Values use the existing scalar literal grammar and any
retained checks when validation is enabled. Empty or null outer defaults and
required/Patch initializer suppression do not hide an invalid child.

This role reuses the existing resolver: a default does not admit a new
reference graph or change the target's default rules. An admitted scalar
component and its admitted named aliases may each have an independent
component-default factory. Their own values are validated separately, and
neither their factories nor the child annotation populate the container.
These factories can coexist with already admitted collection field and object
defaults. A finite List/typed Map path ending at an admitted physical scalar
reference may also own a collection factory, as may an admitted collection
alias. The path may cross existing named collection roots and their finite
named aliases. Each collection root, alias and outer value is validated
independently against its actual type; no annotation supplies another
annotation's value. The scalar child need not have a default for an outer
factory to work. Internal inter-container references and other collection
factory roles retain their separate restrictions. DTO, const,
caller-native/provider and value-class targets do not gain this permission.
Tuple positions or tails, fixed-object discarded extra properties and other
child roles retain their separate contracts. Existing enum children use the
[enum child contract](collection-reference-child-defaults.md); direct inline
builtin children use their [own bounded contract](inline-builtin-collection-child-defaults.md).

## Collection-valued reference children

A physical List element or typed Map value may also have its own default
when its `$ref` targets an already admitted named finite List/typed Map or its
named aliases, and that collection path ends either in an admitted physical
scalar reference or in a direct inline builtin, UUID, date/date-time or
byte/binary scalar leaf. The target collection and aliases may have their own
independent factories. For example:

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Number":{"type":"integer","default":1},
  "Values":{"additionalProperties":{"$ref":"#/components/schemas/Number","default":2},
            "default":{"target":3}},
  "ValuesAlias":{"$ref":"#/components/schemas/Values","default":{"alias":4}},
  "Rows":{"type":"array",
          "items":{"$ref":"#/components/schemas/ValuesAlias","default":{"child":5}},
          "default":[{"outer":6}]}
}}}
```

The child value `{"child":5}` is validated as a value of `ValuesAlias`,
including the nearest nullable view and the checks retained by its actual
collection and scalar types. It is not inserted into `Rows` or used by
`ValuesAlias`'s factory. Target, alias, scalar-child and outer annotations
remain independent. Multiple eligible levels are validated separately,
including when a collection reference is declared before its target.
Invalid values report their own child `/default` Pointer even when an outer
default is empty or a request initializer is suppressed.

The target may instead end in an inline builtin scalar, for example:

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Values":{"type":"array","items":{"type":"integer"},"default":[1]},
  "ValuesAlias":{"$ref":"#/components/schemas/Values","default":[2]},
  "Rows":{"type":"array",
          "items":{"$ref":"#/components/schemas/ValuesAlias","default":[3]}}
}}}
```

The three annotations are validated independently; the child value `[3]`
does not become an element of `Rows`.

This role does not admit internal references between containers, DTO leaves or
recursive collection paths. Inline enum leaves use the separate
[enum child contract](collection-reference-child-defaults.md). The collection containing the
annotated reference only gains an outer default where a separate existing role
admits it. Those sources and outer defaults retain their separate contracts.

The original physical parent and source path, canonical scalar binding,
linked element type, validation checks and retained component factories remain
authoritative. Named intermediate collections and aliases retain their actual
declarations, backing, nullable bindings and cached heights. Invalid values
report their respective original escaped `/default` Pointer. Existing input
8 MiB, source depth 48, cumulative source work 16,384, default value 16,384
and 32 KiB, checked collection depth 16 and JVM slot 240 limits remain.

Previously rejected eligible child or outer annotations now generate.
Adding only a valid child annotation changes no generated Kotlin API,
initializer, JSON wire behavior, name lock or output ownership format.
An outer component annotation adds its own opt-in factory; an ordinary
field annotation uses the existing field-default behavior.

[Core tests](../core/scalar_reference_child_defaults_test.go), including
[independent component and child defaults](../core/scalar_reference_child_independent_defaults_test.go),
[outer-default interactions](../core/scalar_reference_child_outer_defaults_test.go),
[collection-valued child tests](../core/collection_reference_collection_child_defaults_test.go),
[inline-scalar collection child tests](../core/collection_reference_inline_scalar_child_defaults_test.go),
[inline-special-scalar collection child tests](../core/collection_reference_inline_special_child_defaults_test.go),
[source binding tests](../internal/schema/scalar_reference_child_defaults_test.go),
[collection binding tests](../internal/schema/collection_reference_collection_child_defaults_test.go),
[inline-scalar collection factory authority tests](../internal/schema/collection_reference_inline_scalar_child_defaults_test.go),
[factory authority tests](../internal/schema/referenced_scalar_collection_factories_test.go),
and [CLI failure-preservation tests](../internal/cli/scalar_reference_child_defaults_process_test.go)
cover this bounded role. The
[platform fixture](../tests/fixtures/platform/enum-field-bounds.json) and
[common consumer](../samples/kotlin-platform-smoke/src/commonMain/kotlin/org/example/platform/EnumFieldBoundsContracts.kt)
exercise referenced integer, date and byte children, as well as
collection-valued children through named scalar-reference and inline-byte
collections. Their exact runtime and compilation results are recorded in the
[platform probe](platform-enum-field-bounds.md). The
[HTTP fixture](../tests/fixtures/http/string-enum-collection-bounds.json) and
[MockEngine consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/StringEnumCollectionBoundsContracts.kt)
include referenced scalar children in JSON request, response and SSE values;
`bash scripts/test.sh feature string-enum-collection-bounds` runs the profiles.
Report compilation and
runtime results for the exact candidate and target; preparation alone is not
runtime evidence. General default graphs and platform acceptance remain open.
