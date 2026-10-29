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
object-default walks. It does not admit a new reference graph or a default on
an ordinary scalar, DTO or collection reference. Direct inline builtin scalar
children have a separate [limited contract](inline-builtin-collection-child-defaults.md).
Tuple positions and tails,
discarded extra properties and other child annotation roles remain separate.
Each invalid value is reported at the original escaped child `/default`
Pointer. The original physical parent, enum vocabulary, reference target when
present, actual linked type/declaration/check objects and source/value budgets
remain authoritative.

**Compatibility:** eligible inputs that were formerly rejected now generate.
For otherwise identical input, the child annotation changes no generated Kotlin
API, initializer, wire behavior, name-lock or output ownership format. Existing
source depth48, cumulative source work16,384, default value16,384 and 32 KiB,
checked collection16, enum240 and JVM240 limits continue to apply.

[Core tests](../core/collection_reference_child_defaults_test.go),
[inline core tests](../core/inline_enum_collection_child_defaults_test.go),
[reference source and type authority tests](../internal/schema/collection_reference_child_defaults_test.go),
[inline source and type authority tests](../internal/schema/inline_enum_collection_child_defaults_test.go)
and [CLI failure-preservation tests](../internal/cli/collection_reference_child_defaults_process_test.go)
cover the bounded role. `bash scripts/test.sh feature string-enum-collection-bounds`
generates fresh Data/Plain, validation-on/off Kotlin consumers with four JSON
configurations and existing HTTP/SSE assertions. Test entry points and
prepare-only generation are not compilation or runtime evidence; check the
specific candidate and target. The [platform probe](platform-enum-field-bounds.md)
also exercises inline and referenced child annotations. General default graphs,
other platforms and final acceptance remain separate.
