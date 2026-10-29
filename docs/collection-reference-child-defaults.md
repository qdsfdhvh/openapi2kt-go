# Defaults on enum collection elements

An already supported finite List or typed Map may have an independent `default`
on its physical `items` or `additionalProperties` schema when that schema is an
inline lexical String or native numeric/Boolean enum, or a local `$ref` to
an already admitted enum of those kinds. Existing finite aliases and admitted
internal references keep their own source and nullability rules. For example:

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Word":{"enum":["one","two"],"default":"one"},
  "Rows":{"type":"array",
          "items":{"$ref":"#/components/schemas/Word","default":"two"},
          "default":["one"]},
  "Codes":{"type":"array",
           "items":{"enum":["one","two"],"default":"two"},
           "default":["one"]},
  "Holder":{"properties":{
    "rows":{"type":"array",
            "items":{"$ref":"#/components/schemas/Word","default":"two"}}
  }}
}}}
```

Each annotation is validated independently. The element annotations above do
not populate the containers, add elements, change `Holder.rows`, or inherit the
component's annotation. They are validated even when an outer default is empty or
null and before required/Patch suppression. A non-null String default must
belong to the original enum vocabulary; native numeric/Boolean defaults must
match the original vocabulary before floating-point mapping. A null default
requires the element's effective nullable view. Supported bounds are checked
when validation is enabled. Runtime String codecs remain open to unknown words
and native aliases retain their existing scalar codecs.

The same source role can occur in already admitted nested collection and
object-default walks. A physical List or typed Map child may also reference an
already admitted named finite collection or its aliases whose direct inline
leaf is one of these enum kinds. It may also reference a finite List/Map path
through eligible named collections and aliases that ends in a physical
reference to an eligible named enum or its named aliases. The child's
collection-valued annotation, enum and collection factories, and alias
factories are checked independently. For example:

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Values":{"type":"array","items":{"enum":["one","two"]},"default":["one"]},
  "Alias":{"$ref":"#/components/schemas/Values","default":["two"]},
  "Rows":{"type":"array",
          "items":{"$ref":"#/components/schemas/Alias","default":["one"]}},
  "Word":{"enum":["one","two"],"default":"one"},
  "ReferencedValues":{"type":"array",
                      "items":{"$ref":"#/components/schemas/Word","default":"two"}},
  "ReferencedRows":{"type":"array",
                    "items":{"$ref":"#/components/schemas/ReferencedValues","default":["one"]}},
  "NestedRows":{"type":"array",
                "items":{"$ref":"#/components/schemas/ReferencedRows","default":[["two"]]}}
}}}
```

The references between collections and the enum leaf in this collection-valued
role must resolve through named component roots and aliases. Internal
collection references, internal enum leaves, DTO leaves and other collection
outer-default roles remain separate. Eligible ordinary
scalar references and direct inline builtin scalar children have separate
[reference](scalar-reference-collection-child-defaults.md) and
[inline](inline-builtin-collection-child-defaults.md) contracts.
Tuple positions and tails,
discarded extra properties and other child annotation roles remain separate.
Each invalid value is reported at the original escaped child `/default`
Pointer. The original physical parent, enum vocabulary, reference targets,
actual linked collection declarations and cached heights, type/check objects
and source/value budgets remain authoritative.

**Compatibility:** eligible inputs that were formerly rejected now generate.
For otherwise identical input, the child annotation changes no generated Kotlin
API, initializer, wire behavior, name-lock or output ownership format. Existing
source depth48, cumulative source work16,384, default value16,384 and 32 KiB,
checked collection16, enum240 and JVM240 limits continue to apply.

[Core tests](../core/collection_reference_child_defaults_test.go),
[inline core tests](../core/inline_enum_collection_child_defaults_test.go),
[inline enum collection reference tests](../core/collection_reference_inline_enum_child_defaults_test.go),
[referenced enum collection tests](../core/collection_reference_enum_reference_child_defaults_test.go),
[reference source and type authority tests](../internal/schema/collection_reference_child_defaults_test.go),
[inline enum collection reference authority tests](../internal/schema/collection_reference_inline_enum_child_defaults_test.go),
[referenced enum collection authority tests](../internal/schema/collection_reference_enum_reference_child_defaults_test.go),
[inline source and type authority tests](../internal/schema/inline_enum_collection_child_defaults_test.go)
and CLI failure-preservation tests for [enum children](../internal/cli/collection_reference_child_defaults_process_test.go)
and [collection-reference children](../internal/cli/scalar_reference_child_defaults_process_test.go)
cover the bounded role. `bash scripts/test.sh feature string-enum-collection-bounds`
generates fresh Data/Plain, validation-on/off Kotlin consumers with four JSON
configurations and existing HTTP/SSE assertions. Test entry points and
prepare-only generation are not compilation or runtime evidence; check the
specific candidate and target. The [platform probe](platform-enum-field-bounds.md)
also exercises inline and referenced child annotations, including
collection-valued children through inline and referenced enum leaves and named
collection aliases. General default graphs, other platforms and final
acceptance remain separate.
