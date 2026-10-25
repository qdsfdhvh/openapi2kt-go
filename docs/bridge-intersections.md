# Named ordinary DTO bridges in allOf

An already-bound component root bridge through a finite internal `$ref` chain
can serve as an allOf operand when its terminal is an original direct named,
non-null ordinary constructor DTO. The existing bridge binding supplies the
real terminal catalog entry, source node and complete cached reference height.
The intersection planner borrows that original named source, not the internal
hop's syntax or a fabricated component for a lexical nested target.

```json
{"Value":{"properties":{"id":{"type":"string"}}},
 "Hops":{"properties":{"hop":{"$ref":"#/components/schemas/Value"}}},
 "Bridge":{"$ref":"#/components/schemas/Hops/properties/hop"},
 "Consumer":{"allOf":[{"$ref":"#/components/schemas/Bridge"},
                         {"properties":{"extra":{"type":"boolean"}}}]}}
```

The Consumer follows the existing ordinary allOf lowering: original property
order, required fields, later overrides, field defaults/checks and
`W_ALL_OF_APPROXIMATION` remain. The target stays a separate canonical DTO; its
existing nested declarations, recursion and typed references retain their
owners. Borrowed nested copies stay under the consuming DTO as before. The
bridge's constructor alias/serializer continues to resolve the original Value.
This is not a general intersection validator or exact oneOf match counter.

Original endpoint fields and Pointers feed the shared model planner. Source and
catalog identity are checked; a copied catalog carrier does not authorize a
shape. Final nominal ownership validation still requires the original actual
constructor, including unused declarations. The ordinary nominal resolver used
by other roles keeps its boundary. Missing/invalid source, discarded fields and
excluded operations do not bypass validation. Native failure preserves old
output and returns no artifacts.

This profile excludes intrinsically nullable targets and bridges with true
nullable intent, composed/union/hierarchy/raw/const-only/count-bearing targets,
internal lexical DTO endpoints, dictionary shapes and new oneOf member/SSE
wrapper ownership. Explicit true member widening remains a separate nullable
classification role; absent/false bridge members retain non-null intersection
intent. Existing typed nullable bridge use keeps its earlier contract. Source
48, composition work16,384, checked-depth16, JVM slots240 and all other resource
budgets remain. Every cached erased hop is charged again at the shape consumer.
A 45-internal-hop bridge plus its incoming and named-terminal edges reaches the
existing field source-depth boundary; the next hop rejects.

`bash scripts/test.sh feature bridge-intersections` generates Data/Plain with
validation off/on, baseline and combined dedup/hoist/merge/locked output. The
[fixture](../tests/fixtures/http/bridge-intersections.json) and
[consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/BridgeIntersectionsContracts.kt)
exercise constructor/alias codecs, original scalar/enum defaults, nested DTO
copies, canonical nominal recursion, four Json configurations, required/Patch,
checks on borrowed nested fields, controlled HTTP and fragmented SSE
failure/closure/client reuse. An independent constructor probe combines239
borrowed fields and one own field. Go/source/native tests retain the exact former
bridge-allOf refusal as a positive, enforce source/catalog/AST identity, cached
height and constructor width, keep other roles rejected, compare core/native
bytes and readonly/no-op behavior, and preserve old output on failure. Bounded
fuzz checks finite reference heights and determinism. Wider graph roles and
platforms require independent evidence.

**Migration:** formerly rejected eligible named-terminal bridge intersections
now generate using existing DTO field/API/wire rules. Existing accepted inputs,
canonical aliases, manifests and name-lock formats retain their contracts.
Nullable/internal/composed targets and broader D01/D02 remain open.

[Nullable named DTO bridge intersections](nullable-bridge-intersections.md) extend D01/D02/F04/F07 to intrinsic target and nearest bridge/member null intent in the existing actual named ordinary allOf role. Early discovery follows original internal ref edges without admission, preserves every named/internal node's work/depth before and after binding, and final source/catalog/constructor checks remain authoritative. Nullable codecs/AND intersection/required/Patch, original fields/AST/owners, caps and failure preservation retain their contracts. Other endpoint/shape/platform roles remain separate; accepted non-null output/state formats are stable.

[Actual internal DTO intersections](internal-dto-intersections.md) extend D01/D02/F04/F07 to direct/internal-chain/root-bridge allOf operands with unique actual ordinary lexical constructors. Resolved carriers keep original nodes/Pointers and real named catalogs without creating internal component identities; existing final owner proof rejects copies/scratch/missing owners. Shared document capture caps16,384work/8MiB charge overlapping targets while repeated targets reuse captures; prior source/composition/checked/slots/default budgets hold. Defaults/checks/nullable views/Patch/nominal recursion/warnings and accepted API/wire/state remain. Composed/hierarchy/count/root-default and wider graph/platform roles stay separate.

[Unique composed-owned nested DTO intersections](composed-dto-intersections.md) extend D01/D02/F04/F07 to actual ordinary property/collection DTOs under allOf/union lexical owners, using original source carriers and existing deferred unique-owner proof. Variant roots/common fields/hierarchy/ambiguous/discarded/scratch/count/root-default roles stay separate. Canonical recursion, nullable/Patch/default/check/override/warning contracts and shared16384work/8MiB capture/source/checked/JVM240 limits remain; cache identity rejects foreign Pointers/catalogs. Existing API/wire/locks remain stable; wider graph/platform completion stays open.
