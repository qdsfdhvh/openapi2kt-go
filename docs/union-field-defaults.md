# Inline primitive union field defaults

A direct ordinary DTO field whose schema is an inline `oneOf`/`anyOf` of plain String, Integer, Number and Boolean branches may declare its own `default`. The field keeps its generated native primitive-union sealed type; the initializer decodes the canonical JSON through the final actual union serializer, so the call-site type and wire behavior are unchanged.

```json
{
  "v": {
    "oneOf": [
      {"type": "string"},
      {"type": "integer", "format": "int32"},
      {"type": "number", "format": "double"},
      {"type": "boolean"}
    ],
    "default": 1.5
  }
}
```

An optional field exposes e.g. `val v: V? = `openapi-kt model default org.example.models.HolderDto V`("1.5")`, which decodes to `HolderDto.NumberV(1.5)`. Missing JSON uses the default, a supplied value replaces it and explicit null remains null when the union is explicitly nullable. Each evaluation produces a fresh wrapper. Required fields validate the annotation and keep their constructor argument; optional explicitly nullable inline request fields keep `Patch.Absent`.

## Admission and source fidelity

Only the consuming ordinary DTO field's own `default` grants this role, and only when the shared plan produces the native primitive-union declaration. Excluded here:

- branches with `enum` or `const` pins, and String branches with a `format` (temporal/byte/uuid spellings);
- object branches, `$ref` branches and per-branch `nullable`;
- same-kind unions that fold to a scalar, and unions that produce an `ObjectUnion`;
- request-context operation unions, root unions and any collection element/value union.

The planner borrows the original schema nodes and the consuming `/default` value; it never rewrites the source. The default must be a JSON primitive whose token matches an accepted branch. Because the runtime selector chooses its wrapper from the JSON token kind in the fixed order String → Boolean → Integer → Number, the original numeric token is preserved (`1` stays Integer, `1.0` stays Number) rather than canonicalized. A token that matches no accepted kind, a non-primitive value, or a null without explicit union nullability is rejected at the original `/default` Pointer. Canonical JSON participates in deduplication, merging, hoisting and name-lock identity; the helper binds the final relocated union type after optimization.

## Validation and resources

The existing input8MiB, source48, checked-collection16 and 240 serialized constructor-slot limits are unchanged; the annotation reuses the 16,384 value-node and 32 KiB compact/canonical JSON budgets. The generator's existing `W_PRIMITIVE_UNION_APPROXIMATION` warning is unchanged: it still documents that the runtime dispatches by JSON kind and does not enforce exact `oneOf` match counts or branch constraints. This feature adds no branch-membership enforcement beyond selecting an accepted kind.

## Compatibility and verification

These inline primitive union field defaults previously failed with `E_SCHEMA_UNSUPPORTED`. Existing union declarations, call sites, wire behavior and emitted bytes for fields without defaults are unchanged. This closes only the inline primitive union field-default subitem of D03 and extends M01/M03/M04/M07/M08 with the existing A03/A05/A08 consumers; pinned/object/reference unions, union-of-collection elements, object/hierarchy/union branches and root/component defaults remain open.

The [fixture](../tests/fixtures/http/union-field-defaults.json), [core typed/presence and graduation tests](../core/union_field_defaults_test.go), [guards and null-branch tests](../core/union_field_defaults_test.go), [declaration/selector authority tests](../internal/schema/union_field_defaults_test.go) and [native parity/preservation/optimization tests](../internal/cli/union_field_defaults_process_test.go) define the increment. `bash scripts/test.sh feature union-field-defaults` freshly generates Data/Plain × validation off/on projects and checks constructor/serde, four Json configurations, required presence, controlled HTTP and fragmented live SSE, plus a combined optimized/locked package whose helper still binds its relocated type. The fixture/consumer also run in `http`. `OPENAPI_KT_PREPARE_ONLY=1` provides native preparation evidence only.

**Migration:** formerly rejected inline primitive union field defaults now generate. General D03/D04, wider platforms and release gates remain open.
