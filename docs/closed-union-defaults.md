# Closed subtype object union field defaults

An ordinary DTO direct local `$ref` field may own a non-null object default when its target is an already accepted named standalone object union using closed kotlinx sealed polymorphism (`SerialNames`). The annotation uses the `Json.Default` object representation: a String `type` member selects an actual emitted subtype serial name.

```json
{
  "Choice": {
    "oneOf": [
      {"properties": {"value": {"type": "string"}, "age": {"type": "integer", "default": 7}}, "required": ["value"]},
      {"properties": {"value": {"type": "integer"}}, "required": ["value"]}
    ]
  },
  "Holder": {
    "properties": {
      "choice": {"$ref": "#/components/schemas/Choice", "default": {"type": "Variant1", "value": "initial"}}
    }
  }
}
```

Both branches require `value`, so no exclusive required key or pinned field provides content dispatch. The ordinary optional field keeps its actual named union type and decodes a fresh subtype through the existing file-level model-default helper. Its String variant above uses its own scalar initializer for omitted `age`. A supplied value replaces the default; this is not a branch-shape matching algorithm.

## Annotation and type authority

Use the emitted class-level `@SerialName`, rather than assuming a name from a schema title or a component reference. Inline root variants often use `Variant1` and `Variant2`; referenced variants retain the generated component name, including a configured suffix. Root-name collisions can change these names. Default validation replans under the actual root name and requires the original named component, final retained nominal binding and actual consuming field type.

The `type` member is backend metadata, not an added schema or Kotlin property. The validator borrows the selected variant's original plain-object source and all original supplied values. Unknown business keys, missing required members and invalid leaf kinds/vocabularies fail with original `/default/...` Pointers. Business members validate in their original order before canonical key sorting. A missing discriminator fails at `/default`; an unknown or non-String discriminator fails at `/default/type`. There is no Unknown fallback on this backend.

The selected variant must be an already accepted plain constructor DTO. Its supported scalar, enum, temporal, nested-object and collection members follow the existing [object default walk](nested-object-defaults.md). Member-owned defaults retain the union variant planner's existing admission: for example scalar own defaults are supported, while nested object own defaults and broader primitive-bound roles are not enabled here. Native enum default members remain strict in both validation modes; supplied runtime values retain the existing validation opt-in. Adding the field annotation preserves target declaration bytes and source-owned warnings.

Every subtype's directly serialized properties/constants must avoid the wire name `type`, since it would collide with the helper's default class discriminator. Such a default is rejected at the consuming `/default` even if another subtype was selected. This check does not prohibit ordinary nested business objects from declaring their own `type` property.

## Caller codecs and presence

The annotation representation always uses `Json.Default`'s `type` object, independently of a consumer's codec. Once initialized, the typed value uses the caller's configured discriminator key or array polymorphism on encoding and decoding. No custom serializer, new variant or altered subtype inventory is introduced.

Required annotations validate and retain required constructor arguments. Optional explicitly nullable inline request fields retain `Patch.Absent`, `Patch.Null` and `Patch.Value`; the default never converts omission into a request update. Overridden `allOf` contributions validate before being discarded. These checks and budgets apply even when no initializer is emitted.

Existing root-null-carrier behavior is unchanged: an explicitly nullable standalone union admits a literal null annotation; optional Kotlin presence or a consumer nullable wrapper alone does not invent a root Null subtype. Nullable closed roots also accept known non-null subtype defaults through their existing wrapper serializer. This role does not change caller `explicitNulls` behavior.

## Limits and excluded roles

The original annotation is preflighted before subtype selection with the shared 16,384 JSON value-node and 32 KiB compact JSON budgets. Canonical output independently remains at most 32 KiB, including the discriminator. Limit failures return `E_MODEL_LIMIT` at the original offending value, or the annotation root when canonical output grows beyond its cap. Input8MiB, source48 and constructor-slot240 limits remain unchanged. Generation failure returns no partial artifacts and preserves existing CLI output.

Defaults on components or intermediate aliases are not inherited. Union alias/reference-chain defaults, internal-owned union targets, closed allOf variant/product defaults, hierarchy fields, selected empty/const-only projections, closed unions as collection/defaulted-object members, root/component defaults and discriminator collisions retain separate roles. [Content/tag/presence/composed union defaults](union-field-defaults.md) keep their existing contracts. This is a bounded D03 increment, not general schema-match validation or default inheritance.

## Verification and compatibility

This extends M01/M03/M04/M07/M08/M09 through the existing typed A03/A05/A08 consumers. Previously rejected eligible field annotations now generate; fields without defaults retain their public types, codecs and emitted declarations. Defaults using referenced subtype names must use the configured suffix and actual collision-resolved serial names.

The [core regressions](../core/closed_union_defaults_test.go), [source/type authority regressions](../internal/schema/closed_union_defaults_test.go) and [CLI parity/check/failure-preservation tests](../internal/cli/closed_union_defaults_process_test.go) define the boundary. The [union fixture](../tests/fixtures/http/union-field-defaults.json), [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/UnionFieldDefaultsContracts.kt) and `bash scripts/test.sh feature union-field-defaults` cover Data/Plain × validation off/on, four default/null configurations with default/custom discriminator and array-polymorphism modes, fresh nested values, missing/replacement/null, nullable roots, Patch wire states, HTTP and fragmented SSE, plus optimized/locked helper binding. `OPENAPI_KT_PREPARE_ONLY=1` is only native preparation. General D03/D04/D05 and platform/release acceptance remain separate.
