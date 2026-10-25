# Nullable named DTO bridge intersections

The [named ordinary DTO bridge allOf profile](bridge-intersections.md) also
accepts intrinsically nullable direct named DTO terminals and bridge/member
nullable overrides. It retains the same real terminal catalog and source node;
this does not authorize internal lexical, composed, union, hierarchy, counted,
raw or const-only endpoints as shapes.

A consumer reference's explicit nullable true/false precedes the bridge's
nearest explicit intent, which precedes the terminal's intrinsic intent.
AllOf combines the resulting operand null sets with AND unless the root has
its existing explicit override. A bridge operand and another nullable object
therefore make a nullable intersection; a false narrowing on either operand
removes null from the inferred intersection. Members and aliases never rewrite
the physical source DTO or its non-null constructor/classifier/companion codec.
Nullable typed references use the existing nullable adapter; required/optional
and request Patch retain their existing presence rules.

The catalog needs derived null intent before aliases are bound. Discovery now
follows original internal schema/ref edges in addition to the existing original
named-edge walk. The internal lookup reads the original node and Pointer and
grants no type, model, source-annotation or ownership permission. Every original
ref node recurses and consumes the same source/depth/work charge before and
after alias binding; cached alias heights do not skip nodes in this traversal.
The later intersection resolver checks the actual named ordinary catalog and
final nominal validation confirms its real retained constructor. Cycles, invalid
or discarded sources, wrong roles and missing targets remain fail-closed.

Original field Pointers, shared composition work16,384, source-depth48,
checked-depth16, JVMslots240, literal/default budgets, deterministic naming and
persisted manifest/name-lock formats retain their contracts. No expanded schema
or copied catalog is used to infer nullability or ownership.

`bash scripts/test.sh feature nullable-bridge-intersections` generates
Data/Plain×validation with baseline, combined optimized/locked and independent
240-slot output. The [fixture](../tests/fixtures/http/nullable-bridge-intersections.json)
and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullableBridgeIntersectionsContracts.kt)
test four Json settings, physical constructors/bridge nullable codecs, scalar
and enum defaults, nested copies, nominal recursion, required/Patch, checks,
HTTP null request/response and fragmented SSE null/data/error/closure/client
reuse. Core tests cover the 18 target/alias/member precedence combinations and
keep the former nullable refusals as explicit positives. Source tests verify
original AST/catalog/field identity and the exact seven-node discovery work
boundary before and after aliases bind, without manufacturing source authority.
Native tests compare core/CLI bytes, readonly/no-op behavior and old-output
preservation for invalid sources. Bounded fuzz varies original finite chains,
target/member null intent and validation while checking determinism. Broader
source roles, graph combinations and platforms require independent evidence.

**Migration:** formerly rejected eligible nullable bridge intersections now
produce existing nullable DTO/Patch/codec APIs. Previously accepted non-null
bridge inputs retain their generated output and wire behavior. No source schema,
canonical classifier or persisted state format is changed. This closes a bounded
D01/D02 null-intent role, not the general reference/composition roadmap.

[Actual internal DTO intersections](internal-dto-intersections.md) extend D01/D02/F04/F07 to direct/internal-chain/root-bridge allOf operands with unique actual ordinary lexical constructors. Resolved carriers keep original nodes/Pointers and real named catalogs without creating internal component identities; existing final owner proof rejects copies/scratch/missing owners. Shared document capture caps16,384work/8MiB charge overlapping targets while repeated targets reuse captures; prior source/composition/checked/slots/default budgets hold. Defaults/checks/nullable views/Patch/nominal recursion/warnings and accepted API/wire/state remain. Composed/hierarchy/count/root-default and wider graph/platform roles stay separate.

[Unique composed-owned nested DTO intersections](composed-dto-intersections.md) extend D01/D02/F04/F07 to actual ordinary property/collection DTOs under allOf/union lexical owners, using original source carriers and existing deferred unique-owner proof. Variant roots/common fields/hierarchy/ambiguous/discarded/scratch/count/root-default roles stay separate. Canonical recursion, nullable/Patch/default/check/override/warning contracts and shared16384work/8MiB capture/source/checked/JVM240 limits remain; cache identity rejects foreign Pointers/catalogs. Existing API/wire/locks remain stable; wider graph/platform completion stays open.
