# Direct inline ordinary DTO object field defaults

[Finite nested builtin collection defaults](nested-builtin-collection-defaults.md), [temporal leaves](nested-temporal-collection-defaults.md), [plain String enum terminals](nested-string-enum-collection-defaults.md) and [native enum terminals](nested-native-enum-collection-defaults.md) own the finite inline collection/leaf roles. The profile below adds a plain inline **ordinary DTO object** leaf to an ordinary field's own `default`. The field type and all nested declarations still come from the shared ordinary object planner, so public call sites are independent of the annotation.

```json
{
  "type": "object",
  "required": ["name"],
  "properties": {
    "name": {"type": "string"},
    "age": {"type": "integer", "default": 7},
    "status": {"type": "string", "enum": ["on", "off"]},
    "child": {"type": "object", "properties": {"flag": {"type": "boolean"}}, "default": {"flag": true}}
  },
  "default": {"name": "outer", "status": "on", "child": {"flag": false}}
}
```

An optional ordinary field gets a typed initializer that decodes the canonical JSON through the actual final DTO serializer, for example `val cfg: Cfg? = `openapi-kt model default org.example.models.HolderDto Cfg`("{...}")`. Missing JSON uses the default, a supplied value replaces it and explicit JSON null remains null when the field is explicitly nullable. Every default evaluation produces a fresh tree. A required field validates the same annotation but keeps its constructor argument without an initializer; optional explicitly nullable inline request fields keep `Patch.Absent`, and a missing request key never becomes an update. Required/nullable/Patch and `explicitNulls = false` behavior mirrors the collection profile and [request presence](request-presence.md).

## Admission and source fidelity

Only a direct ordinary DTO field's own `default` grants this role, and only when the field schema is a plain inline object: `type` may be explicit `"object"` or inferred from a non-empty `properties` map. Rejected sub-constructs keep their original diagnostics at the same or a more precise original Pointer. Excluded here:

- `$ref` to a named DTO (the separate [referenced object contract](referenced-object-defaults.md)), `oneOf`/`anyOf`/union and `allOf` objects (ordinary disjoint inline merges use the separate [allOf field default contract](allof-object-defaults.md));
- dictionaries or fixed extras (`additionalProperties`), empty-property objects and raw `JsonElement` projections;
- `const` on the object node itself (scalar `const` **properties** are validated for exact equality), `enum` on the object node, native (`x-native-type`), value-class, format and OpenAPI discriminator markers; structured (object/array) `const` properties;
- containers whose element is an inline object use the separate [object collection contract](nested-object-collection-defaults.md); container-child defaults remain rejected.

Every declared property is checked, including when the default object omits it, and every present key must name a declared property. All `required` wires must be present in the default object; `null` is accepted only for a nullable property. Property values reuse the existing finite leaf/container planner, so scalar/temporal leaves, String enum membership against the original vocabulary, native enum exact membership, nested builtin collections and per-layer nullability follow those contracts unchanged. Nested plain-object properties recurse through this same walk, so finite non-recursive object trees are supported. A property that is a direct local `$ref` to a plain object target borrows that target recursively; a `$ref` to a scalar or enum target keeps the existing leaf/container validation against the resolved source. Reference chains (a `$ref` whose target is itself a `$ref`) stay with their separate roles. A scalar `const` property (`type` explicit or inferred from the literal) requires exact equality against its literal. A property may declare its own `default`; the nested model's own field plan validates that annotation, and the outer default may omit the key.

The planner borrows the original schema and value nodes; it never rewrites the source. Canonical JSON sorts object keys with the same rule as typed dictionaries, but validation runs in original member order so diagnostics keep their source precedence. Success behavior therefore ignores key reordering and mapped Float/Double spelling, while changing an effective value, a nullable layer or a retained check changes the final shape seen by deduplication, merging, hoisting and name locks. The helper binds the final actual nominal TypeRef after relocation; a nested type is never spelled with a stale short name.

## Validation and resources

With `--validation`, the default value is checked against each property's retained bounds/counts/enum membership exactly as a supplied value would be. Native enum membership of the default annotation is strict in both modes, while runtime wire enforcement stays validation-gated. The nested DTO's own `init` checks run whenever the decoded default is constructed, so runtime validation also covers the default tree. Null bypasses only its own non-null checks.

The existing 8 MiB document, source-depth 48, checked-collection-depth 16 and 240 serialized constructor-slot limits are unchanged. The whole annotation reuses the nested default budget: 16,384 JSON value nodes and 32 KiB compact JSON bytes per original `/default`, with the canonical encoding independently limited to 32 KiB. Exceeded budgets return `E_MODEL_LIMIT` at an original Pointer with no partial artifacts. These bounds apply to required/Patch/discarded annotations as well as emitted initializers; independent width/depth/byte probes do not establish a maximum-width × maximum-depth guarantee.

## Object source planning budget

Object-default source planning has an independent 16,384-step budget per validation walk. Entering a plain object/container node, a merged object shape, a property dispatch or a container-element dispatch consumes one step; repeated references share the counter and each traversal consumes work. This bounds repeated expansion of a small nominal DAG even for `{}` or `null` annotations. Exceeding the budget returns `E_MODEL_LIMIT` at the original schema Pointer before an unbounded validation tree is allocated.

The same walk is used by referenced DTOs, object-containing collections, ordinary allOf defaults and selected object-union variants. Required/Patch/discarded annotations retain this bound before suppression. Independent annotations receive fresh counters; source nodes and typed-reference admission are unchanged. [Core fanout/suppression regressions](../core/object_default_source_budget_test.go), [exact boundary and walk-isolation checks](../internal/schema/object_default_source_budget_test.go) and [native failure-preservation checks](../internal/cli/object_default_source_budget_process_test.go) define the limit.

## Compatibility and verification

These inline object defaults previously failed with `E_SCHEMA_UNSUPPORTED`. Existing single-layer, collection, enum, temporal and referenced roles retain their contracts and emitted bytes. This closes only the direct inline ordinary DTO object field-default subitem of D03 and extends M01/M03/M04/M07/M08 with the existing M12/V05 and typed A03/A05/A08 consumers; referenced named DTO defaults, union/hierarchy fields, root/component defaults, container-child defaults and general D03/D04 remain open.

The [fixture](../tests/fixtures/http/nested-object-defaults.json), [core typed/presence and graduation tests](../core/nested_object_defaults_test.go), [guards/budgets](../core/nested_object_defaults_guards_test.go), [source/value authority tests](../internal/schema/nested_object_defaults_test.go) and [native parity/preservation/optimization tests](../internal/cli/nested_object_defaults_process_test.go) define the increment. `bash scripts/test.sh feature nested-object-defaults` freshly generates Data/Plain × validation off/on projects and checks constructor/serde/Patch/copy, required presence, four Json configurations, validation membership, controlled HTTP and fragmented live SSE closure/reuse, plus a combined optimized/locked package whose helper still binds its relocated type. The fixture/consumer also run in `http`. `OPENAPI_KT_PREPARE_ONLY=1` provides native preparation evidence only.

**Migration:** formerly rejected plain inline object field defaults now generate. Adding the annotation does not change the field's public type or declaration set. General D03/D04, wider platforms and release gates remain open.
