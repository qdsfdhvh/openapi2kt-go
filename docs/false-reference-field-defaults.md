# Own null defaults on false-reference fields

An ordinary named shared DTO may declare a field's own `default: null` when the field directly references an actual [false component or pure alias](false-component-roots.md) and its logical nearest nullable intent is true. This extends the [direct false-reference field profile](false-reference-fields.md).

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Denied":false,
  "Holder":{"properties":{"value":{
    "$ref":"#/components/schemas/Denied","nullable":true,"default":null
  }}}
}}}
```

Logical nullability comes from the original field override and alias chain, before optional-presence wrapping. A nullable alias can supply that intent; an explicit true override can restore it after a non-null intermediate. A non-null or false-narrowed source gains no default role from its optional absence carrier. Non-null annotations on an eligible nullable false field fail at the original field `/default` Pointer.

Required fields follow the existing own-default rule: the Kotlin nullable parameter has no initializer and callers pass its argument explicitly. Optional fields initialize to Kotlin null. Both use the actual false-root nullable codec, accepting null and rejecting every non-null JSON value. No new serializer or Patch policy is added.

The existing Json nullable policy still applies. With `explicitNulls=true`, a required nullable field without a Kotlin initializer rejects a missing key and writes its null key. With `explicitNulls=false`, the nullable decoder may supply implicit null for a missing required key and the encoder may omit the null key. Optional nullable defaults follow `encodeDefaults` and `explicitNulls`. Ordinary required-nullable fields without an own default keep their established shared default-null behavior; target annotations do not change those fields.

A strict optional non-null false field retains its separate always-omitted absence carrier and supplied-value/null refusal. A nullable target factory does not relax a false-narrowed field. The root's factory, the field's annotation and the constructor's presence policy remain independent.

Admission belongs to the original retained ordinary named shared constructor. The actual owner/property/type tree, root codec, required list, own null node and immutable initializer/default-null/omission signature authorize the field. Removing an initializer or changing a required parameter to optional cannot reuse equal generated text. Existing source member/item/Get consistency and source48/work16384, literal/32 KiB, checked16 and JVM240 limits remain.

AllOf/union/hierarchy/directional owners, nested inline owners, container/child defaults, internal targets and object-default members keep their separate boundaries. This increment does not add those roles or direct false HTTP roots. Existing ordinary DTO JSON HTTP consumers use the same field codec and presence contract.

Previously rejected eligible field annotations now generate. Existing accepted APIs, wire behavior, naming locks and persisted formats need no migration. Data/Plain and current dedup/hoist/merge settings preserve required/optional and codec identity. Broader default/Boolean graphs, platforms and final plan acceptance stay open.

[Fixture](../tests/fixtures/http/false-reference-field-defaults.json), [core/value/role/budget checks](../core/false_reference_field_defaults_test.go), [actual property/source guards](../internal/schema/false_reference_field_defaults_test.go), [native lifecycle](../internal/cli/false_reference_field_defaults_process_test.go), and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FalseReferenceFieldDefaultsContracts.kt) cover the profile. `bash scripts/test.sh feature false-reference-field-defaults` generates Data/Plain × validation off/on projects with baseline and optimized locked output, four Json settings and controlled JSON HTTP. Separate 240-field required and optional DTOs compile/run actual constructors, codecs, descriptor presence and omission behavior. These are JVM checks, not Android/KMP or combined maximum-resource evidence.

[False-reference members in object root defaults](false-object-default-members.md) extend D03/E03 for actual ordinary named Shared root-owned annotations. Original logical nullable members admit null, optional non-null false members admit only absence, and literal required keys remain mandatory. Unique actual getter/source/type/required/codec and model factory/canonical initializer/literal-node authority, source48/work16384/value16384/32KiB/checked16/JVM240 and existing wire-count/JSON policies remain. Consuming-field/nested/child/wrapper/composed/internal/directional/inline and general/platform/final acceptance remain separate.
