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

- String branches with a `format` (temporal/byte/uuid spellings);
- content object unions with no tag, conjunction or presence dispatch rule;
- per-branch `nullable`, same-kind unions that fold to a scalar;
- request-context operation unions (those use the separate raw `JsonElement` carrier) and root unions without a direct local `$ref` consumer.

An ordinary DTO field whose direct local `$ref` resolves to a root union component may also carry its own default. The component union is replanned on a silenced copy to obtain its declaration, the value is validated exactly as for an inline union (kinds, pins, object-union variant selection), and the helper binds the actual component type. Reference chains to such a component stay excluded. A [closed subtype root object union](closed-union-defaults.md) also accepts a non-null consuming-field default using Json.Default's String `type` discriminator and an actual emitted `SerialName`. It validates the selected original plain constructor variant, preserves target warnings/types and admits caller custom-discriminator/array-polymorphism use of the resulting typed value. Business `type` collisions, hierarchy/internal-owned/chain roles and closed collection/object-member defaults remain separate.

The same field-own default role also covers an inline union used as a container element (`array` items or typed `Map` values) or as a property of a defaulted object: the field helper decodes the whole collection/object through the final serializer, and each union member is validated against its kinds and pins with the element/member Pointer. The element union's own `default` annotation is not used.

A **tag-dispatched content object union** (`oneOf`/`anyOf` of plain object variants with a const/enum discriminator) is also admitted. The default must be a JSON object whose discriminator selects a known variant; the value is then validated against that variant's original source through the shared object walker, so unknown keys, missing required members and leaf checks keep the member Pointer. Both `enum`- and `const`-tag discriminators are supported, and a union with no discriminator selects the first variant whose required presence field the default carries (mirroring the emitted selector order). A **Cartesian composed union** (union-in-allOf) is also admitted. Its variants select by conjoined tag pins across several fixed fields (`TagCombinations`), so the default must be a JSON object whose pinned members carry primitive spellings matching one conjunction rule (OR within a field, AND between fields, first full match wins); presence-only rules may still select pin-less variants. The value is validated against the selected variant's merged original shape — all factors' fields and required members — through the shared walk, so unknown keys, missing required members and leaf checks keep their member Pointers. Inline composed fields and direct local `$ref` fields to a composed root component are both admitted; reference chains stay excluded. The `Unknown`/content fallback is never selected by a default; closed subtype selection follows the separate [class-discriminator contract](closed-union-defaults.md).

The planner borrows the original schema nodes and the consuming `/default` value; it never rewrites the source. `enum`/`const` branches are admitted, including a direct local `$ref` to an already accepted scalar/enum component (a reference chain stays excluded). The default must belong to the matched branch's vocabulary after resolution: a `const` branch requires exact equality, and an `enum` branch requires membership in its original String/native/Boolean vocabulary. The default must be a JSON primitive whose token matches an accepted branch. Because the runtime selector chooses its wrapper from the JSON token kind in the fixed order String → Boolean → Integer → Number, the original numeric token is preserved (`1` stays Integer, `1.0` stays Number) rather than canonicalized. A token that matches no accepted kind, a non-primitive value, or a null without explicit union nullability is rejected at the original `/default` Pointer. Canonical JSON participates in deduplication, merging, hoisting and name-lock identity; the helper binds the final relocated union type after optimization.

## Validation and resources

The existing input8MiB, source48, checked-collection16 and 240 serialized constructor-slot limits are unchanged; the annotation reuses the 16,384 value-node and 32 KiB compact/canonical JSON budgets. Direct primitive/object unions and referenced root union defaults preflight the original value before encoding or variant dispatch, retaining source order and member Pointers. Required/Patch initializer suppression and later `allOf` overrides cannot hide an oversized or invalid annotation. Discarded union contributions use a local scratch declaration plan; their generated declarations and warnings do not enter the winning model. The generator's existing `W_PRIMITIVE_UNION_APPROXIMATION` warning is unchanged: it still documents that the runtime dispatches by JSON kind and does not enforce exact `oneOf` match counts or branch constraints. This feature adds no runtime branch-membership enforcement.

## Compatibility and verification

These inline primitive union field defaults previously failed with `E_SCHEMA_UNSUPPORTED`. Existing union declarations, call sites, wire behavior and emitted bytes for fields without defaults are unchanged; the runtime wrappers still do not enforce membership, so pin enforcement is generation-time only. This closes the inline primitive union, composed object union and bounded closed subtype root-union field-default subitems of D03 (including `enum`/`const` pins, collection-element/object-property union members and composed conjunction dispatch) and extends M01/M03/M04/M07/M08 with the existing A03/A05/A08 consumers; hierarchy field defaults, request-context unions and root/component defaults remain open.

The [fixture](../tests/fixtures/http/union-field-defaults.json), [core typed/presence and graduation tests](../core/union_field_defaults_test.go), [guards and null-branch tests](../core/union_field_defaults_test.go), [declaration/selector authority tests](../internal/schema/union_field_defaults_test.go), [budget/discarded-contribution guards](../core/carrier_defaults_guards_test.go), [native guard preservation](../internal/cli/carrier_defaults_guards_process_test.go) and [native parity/preservation/optimization tests](../internal/cli/union_field_defaults_process_test.go) define the increment. `bash scripts/test.sh feature union-field-defaults` freshly generates Data/Plain × validation off/on projects and checks constructor/serde, four Json configurations, required presence, controlled HTTP and fragmented live SSE, plus a combined optimized/locked package whose helper still binds its relocated type. The fixture/consumer also run in `http`. `OPENAPI_KT_PREPARE_ONLY=1` provides native preparation evidence only.

**Migration:** formerly rejected inline primitive, composed object union and eligible closed subtype root-union field defaults now generate. Oversized annotations are rejected with `E_MODEL_LIMIT` before output is committed, including annotations on required, Patch and overridden fields. Valid discarded union defaults validate without emitting temporary declarations. Ordinary allOf fields with disjoint property contributions use the separate [object merge default contract](allof-object-defaults.md); wider ordinary allOf defaults retain their exclusions. General D03/D04, wider platforms and release gates remain open.

### Cartesian constructor enum-own defaults

[Composed selected-constructor member defaults](composed-member-defaults.md) retain original String/native enum own annotations while validating an outer union default. Original numeric pin tokens are retained only when canonical formatting would change the selected index. This corrects already accepted numeric-pin defaults too: known subtype, equality and `encodeDefaults=false` omission can change; regenerate/rebuild and review the documented migration. Existing runtime floating approximation, source/value budgets and broader D03/platform boundaries remain.
