# Nullable object operands in allOf

Ordinary object `allOf` and object-union products now accept nullable object operands. Without an explicit nullable setting on the containing schema, `allOf` accepts null only when every operand does. An object union used as an operand contributes null when at least one of its accepted alternatives does. An explicit root `nullable: true`, `nullable: false` or supported object/null type array keeps the existing root-override policy.

```json
{"allOf":[
  {"nullable":true,"properties":{"left":{"type":"string"}},"required":["left"]},
  {"nullable":true,"properties":{"right":{"type":"string"}},"required":["right"]}
]}
```

This object has nullable typed uses. Making either operand non-null narrows the result. The physical ordinary DTO remains the same constructor-bearing class and non-null companion serializer. Reified `serializer<T?>()` and supported nullable aliases handle null; constructors and `copy` still create non-null instances.

## References and union products

Existing named ordinary DTO and object-union/product aliases may contribute their original object shapes. The nearest explicit consuming-reference or alias nullable setting controls the operand; false narrows the original physical source. Member annotations never rewrite the target declaration, duplicate catalog entries or change where its nested types and serializers live. A nominal property that refers back to its DTO remains a recursion barrier; cyclic shape expansion still fails.

Object union alternatives retain source order and the existing bounded Cartesian product. Their fields, later overrides, required names and const bodies follow the existing allOf approximation. The null bit is separate from physical alternatives. Standalone named products retain a Null singleton when the resulting product allows null; ordinary aliases retain that carrier, nullable aliases decode to Kotlin null, and false aliases restore the physical carrier. Inline products and their supported internal aliases expose nullable Kotlin views over non-null physical variants. Unknown payload fallback is unchanged.

A supported explicit null union branch participates in the union null set without becoming an object constructor. This does not make primitive/dictionary/discriminator intersections valid or establish exact oneOf match counting. Unsupported original fields, discarded overrides, counts and other excluded source roles remain checked at their original Pointers. Nullable counted nominal operands keep their separate shape-borrowing restriction.

## Presence, defaults and resources

Effective object nullability reaches the existing request presence planner. Optional nullable request fields use Patch, required nullable fields have no default, shared fields keep ordinary nullable defaults, and false-narrowed fields retain their existing requiredness. Annotation-only wrappers over named scalars retain their separate historical presence policy.

Default annotations use the effective resulting nullability. A null default that previously depended on only the first nullable member now fails if another operand is non-null. Make the intended source nullability explicit as described below. Required/Patch suppression does not excuse invalid annotations; default roles, target non-inheritance and original value/source budgets remain unchanged.

Source depth 48, composition work 16,384, checked collection edges 16 and JVM slots 240 remain. A derived standalone Null singleton counts toward the existing total branch budget, leaving at most 239 physical object alternatives. Independent depth/width/branch probes do not prove combined maximums or cross-platform support.

## Migration

This intentionally changes previously accepted inline ordinary allOf schemas that inherited nullable intent from the first member. With no root override, a nullable member followed by a non-null member now produces a non-null required field and rejects root null. Supported internal/root aliases and referenced defaults follow the same result. Regenerate and recompile consumers; replace invalid null annotations/data, mark every operand nullable when that reflects the schema, or use explicit root `nullable: true` to retain the established widened view. Explicit root overrides and unrelated non-null inputs preserve their behavior.

Example: `allOf: [nullable object A, non-null object B]` changes a required inline field from `Value? = null` to `Value`. A field on an optional shared parent may still use `Value? = null` because optional presence is independent. `allOf: [nullable A, nullable B]` now also works as a component root with canonical DTO and nullable aliases. No output/name-lock format changes are required; normal regeneration updates owned files and `--check` reports drift until regeneration. The published alpha remains unchanged. Classify the API/wire change under the next pre-1.0 MINOR release.

The [core matrix and migration guards](../core/nullable_intersections_test.go), [original-source/physical-model checks](../internal/schema/intersection_nullability_test.go), [native output protection](../internal/cli/nullable_intersections_process_test.go), extended [fixture](../tests/fixtures/http/nullable-product-unions.json) and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullableProductUnionsContracts.kt) define this D01/D02/F04/F07/M09/M11 increment. `bash scripts/test.sh feature nullable-product-unions` exercises Data/Plain × validation and four Json settings, canonical constructors/aliases, derived null carriers, explicit false, ordinary recursion, raw JSON equality, Patch, controlled HTTP and fragmented live SSE closure/cancellation/cold-flow/client reuse, plus independent 240-field, source-depth and branch probes. General composition/ownership graphs, common-field constraints and Android/KMP/minimum toolchains remain separate tasks.

[Named DTO bridge intersections](bridge-intersections.md) extend D01/D02/F04/F07 to already-bound root bridges through finite internal chains ending at original direct named non-null ordinary DTOs. The allOf planner reuses the real catalog/node/field Pointers and full cached height, preserving original constructors, nested copies, defaults/checks, required/Patch, recursion and approximation warnings. Nullable/composed/internal lexical/union/hierarchy/count/raw roles remain separate; source/work/checked/JVM240 caps, identity and failure preservation hold. Existing API/wire/state formats are stable; broader graph/platform completion stays open.

[Nullable named DTO bridge intersections](nullable-bridge-intersections.md) extend D01/D02/F04/F07 to intrinsic target and nearest bridge/member null intent in the existing actual named ordinary allOf role. Early discovery follows original internal ref edges without admission, preserves every named/internal node's work/depth before and after binding, and final source/catalog/constructor checks remain authoritative. Nullable codecs/AND intersection/required/Patch, original fields/AST/owners, caps and failure preservation retain their contracts. Other endpoint/shape/platform roles remain separate; accepted non-null output/state formats are stable.

[Actual internal DTO intersections](internal-dto-intersections.md) extend D01/D02/F04/F07 to direct/internal-chain/root-bridge allOf operands with unique actual ordinary lexical constructors. Resolved carriers keep original nodes/Pointers and real named catalogs without creating internal component identities; existing final owner proof rejects copies/scratch/missing owners. Shared document capture caps16,384work/8MiB charge overlapping targets while repeated targets reuse captures; prior source/composition/checked/slots/default budgets hold. Defaults/checks/nullable views/Patch/nominal recursion/warnings and accepted API/wire/state remain. Composed/hierarchy/count/root-default and wider graph/platform roles stay separate.
