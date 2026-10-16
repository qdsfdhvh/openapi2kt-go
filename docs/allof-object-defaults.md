# Ordinary allOf object field defaults

An ordinary DTO field may declare its own `default` on an inline object `allOf` whose contributions have distinct wire property names. The shared object planner still owns its nested Kotlin DTO, required arguments and serializers. For example:

```json
{
  "allOf": [
    {"properties": {"label": {"type": "string"}}, "required": ["label"]},
    {"properties": {"count": {"type": "integer"}}, "required": ["count"]}
  ],
  "default": {"count": 2, "label": "initial"}
}
```

An optional field uses a fresh typed object decoded through the final nested DTO serializer. A missing key uses the default; a supplied object replaces it. Required fields validate the complete annotation and retain their constructor argument. Optional nullable inline request fields retain `Patch.Absent`, `Patch.Null` and `Patch.Value`; a schema default never makes a missing key an update.

## Source and value contract

The field must produce an ordinary constructor Model with at least one constructor property. Its `allOf` members may contain ordinary objects, nested object compositions and already accepted object-member references. Root own properties contribute after the members. Every wire property has exactly one contribution; repeated names, even with identical schemas, remain outside this default role.

The default walk borrows the shared merged shape. Each field retains its original schema node, Pointer and source depth, including fields expanded from named members. All supplied keys must name declared fields; all merged required names must be present. Leaf membership, nullable values, supported bounds and collection counts follow the existing [object default contract](nested-object-defaults.md). A plain object member may carry its own validated default; omitting that member from the outer default uses the nested field's initializer. Schema errors point to their source; supplied-value errors point to the original `/default/...` member. Validation visits default keys in source order before sorting the encoded keys.

Effective shape nullability authorizes `default: null`; optional Kotlin presence alone does not. Nested objects, supported collections and scalar/enum/temporal leaves retain their own nullable intent and validation. Required/Patch/discarded annotations undergo the same full validation before initializer suppression or a later parent `allOf` override.

The decode helper binds the final actual DTO type after deduplication, merging and hoisting. Adding the annotation preserves the public type and declaration ownership. The existing `W_ALL_OF_APPROXIMATION` remains owned by type planning, with no duplicate warnings from the default walk.

## Bounded admission

This role applies to direct inline fields in ordinary shared/response/request DTOs. These remain separate:

- annotation-only reference wrappers: a single component ref plus metadata/nullable members must retain its canonical referenced type;
- fields whose root is `$ref`, target/root/component defaults and default inheritance;
- repeated-property/default/constraint aggregation, fixed extras (`additionalProperties`) and whole-object property counts;
- empty/raw and const-only projections, hierarchy/common/inherited fields, collection elements and nested defaulted-object members whose own schema is `allOf`;
- object-union products, which use the separate [composed union default contract](union-field-defaults.md).

Existing input8MiB, source48, composition-work16,384, checked-collection16 and constructor-slot240 limits remain. The annotation retains the shared 16,384 JSON value-node and 32 KiB original/canonical JSON budgets. Exceeding a value budget returns `E_MODEL_LIMIT` at the original member; generation failure leaves existing output intact.

## Verification and compatibility

This extends M01/M03/M04/M07/M08 and the bounded D03 default profile; general D03/D05 remain open. Previously rejected eligible inline ordinary allOf object defaults now generate. Fields without defaults retain existing source, type and wire behavior. Repeated-field aggregation, hierarchy and root/component default semantics are not added.

The [core contract tests](../core/allof_object_defaults_test.go), [source authority tests](../internal/schema/allof_object_defaults_test.go) and [native parity/preservation tests](../internal/cli/allof_object_defaults_process_test.go) define admission and diagnostics. The existing [nested object fixture](../tests/fixtures/http/nested-object-defaults.json) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NestedObjectDefaultsContracts.kt) cover ordinary allOf defaults, member refs, fresh nested instances, missing/replacement/null, Patch states and HTTP/SSE consumption under four Json configurations. Run `bash scripts/test.sh feature nested-object-defaults` for Data/Plain × validation off/on compilation/runtime and optimized/locked helper binding. `OPENAPI_KT_PREPARE_ONLY=1` provides native preparation only. JVM verification does not establish Android/KMP or release acceptance.
