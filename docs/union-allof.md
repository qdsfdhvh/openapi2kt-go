# Object unions within allOf

[Explicit nullable product roots](nullable-product-unions.md) use the existing standalone Null-carrier and inline nullable-codec policies while keeping physical variant shapes non-null. [Nullable object operands](nullable-intersections.md) derive the product null view from all operand null sets.

[Inline variant-owned fields](union-inline-values.md) can hold supported inline object/allOf/union values and finite containers; the product retains each actual nested owner.

An object `allOf` may contain object `oneOf` / `anyOf` members, directly or through supported named component references. The generator combines their borrowed object shapes into a sealed object union. Nested allOf members and multiple object unions use the same resolver; component roots and inline model fields share the resulting type and serializer policy.

```json
{
  "allOf": [
    {"type": "object", "required": ["id"], "properties": {"id": {"type": "string"}}},
    {"oneOf": [
      {"type": "object", "properties": {"kind": {"type": "string", "const": "note"}, "body": {"type": "string"}}},
      {"type": "object", "properties": {"kind": {"type": "string", "const": "count"}, "count": {"type": "integer"}}}
    ]}
  ]
}
```

The generated sealed type has `Note` and `Count` variants, each retaining required `id`, plus an Unknown carrier when content dispatch applies. With two object unions in allOf, alternatives form a bounded Cartesian product in member/branch declaration order. AllOf members contribute in order; the schema's own properties contribute last. A later field replaces an earlier field without moving its first insertion position. Required fields union in source order. This preserves the existing allOf generation approximation; it does not implement strict intersection or oneOf match-count validation.

Original schema nodes, field Pointers, reference hops and depths remain the source of validation and diagnostics. Combinations do not create expanded JSON trees. Overridden fields and additionalProperties schemas are still validated. Constrained sources may be rejected when their checks cannot be retained; allOf composition does not grant an exemption from validation noDrop.

## Dispatch and Kotlin use

Composed variants with pinned fields use conjunctions of their field tags. Each field's finite enum vocabulary is an alternative within that conjunction. For example, `kind=note` together with `mode=slow` selects the corresponding product variant. Rules preserve branch order; identical rules are claimed by the first variant. Required-key fallback retains the existing object-union policy. Unselected variants produce `W_UNION_UNREACHABLE_VARIANT`; unrecognized object payloads use the generated Unknown carrier and preserve raw JSON.

Variant names follow the existing object-union pin naming and numeric collision suffix rules. Common fields keep the current object-union type/presence policy. Constants are body properties and encode independently of encodeDefaults. Shared components retain ordinary nullable defaults; inline request fields retain their existing required-nullable/Patch behavior. Component HTTP bodies/responses and SSE data reference the same generated serializer. This change does not automatically widen operation-specific inline body/response projection contracts.

[References to retained product unions](composed-union-terminals.md) separately bind existing root, nested and collection-peer sealed declarations through typed internal chains and root aliases, including own-nullable/false views. Their source/codec identity remains canonical; this does not grant internal nominal shape borrowing.

Ordinary allOf inputs without a union and ordinary unions retain their existing artifacts, diagnostics and traversal policy.

## Limits and exclusions

The existing 48-level source budget, 16,384 expansion-work budget, 240 union branches and 240 JVM serialized constructor slots apply. A Cartesian product exceeding 240 is rejected before allocating combinations. Transparent composition cycles remain errors. These independent limits are not a promise that every combination of maximum width and depth compiles on all Kotlin targets.

[Nullable object/intersection alternatives](nullable-intersections.md) now retain separate typed null sets over their original physical shapes. Primitive/dictionary alternatives, explicit discriminator hierarchy composition, internal nominal shape borrowing and general JSON Schema intersections remain separate contracts. [Common dictionary fields](union-common-dictionaries.md) have a separate compatible Map/JsonObject property contract. [Common nominal references](union-common-nominals.md) reuse actual DTO/union/hierarchy classifiers. General cross-reference constraint aggregation and inline common object ownership remain pending.

## Migration and evidence

Previously rejected object union-in-allOf inputs now produce a sealed type rather than failing. There was no accepted constructor contract for those inputs. Existing ordinary DTO/union wire behavior is unchanged; projection warnings identify the approximation.

The [synthetic fixture](../tests/fixtures/http/union-allof.json), [core behavior](../core/union_allof_test.go), [source/cycle/product guards](../core/union_allof_guards_test.go), [borrowed-source identity](../internal/schema/union_allof_test.go), [native lifecycle](../internal/cli/union_allof_process_test.go) and [fresh Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/UnionAllOfContracts.kt) define the contract. Run `bash scripts/test.sh feature union-allof` for fresh Data/Plain × validation profiles with four Json configurations, all product variants, Unknown, required fields, Patch, controlled HTTP and fragmented still-open SSE closure/reuse.

[Product union tuple sources](tuple-product-union-sources.md) extend D02/D03/D06/F08/M03/M07/Q05 through actual Cartesian product declarations, original contribution/dispatch/null policies and strict outer own annotations. Original product/source/variant/origin/type/address/cache proof and budgets remain, including discarded validation and internal wrappers. General recursive/hierarchy/intersection/default/platform and final acceptance remain open.
