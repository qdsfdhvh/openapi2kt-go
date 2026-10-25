# Unique composed-owned nested DTOs in allOf

The [actual internal ordinary DTO intersection](internal-dto-intersections.md)
profile also accepts a unique ordinary nested property/collection DTO retained
under an ordinary allOf or object/JSON-kind union lexical owner. Direct Pointers,
finite internal chains and existing root bridges keep the original target
schema and Pointer. The target itself must remain an ordinary constructor DTO;
this does not admit a composed/union/hierarchy target as an ordinary shape.

For example, `Owner/allOf/0/properties/payload` can borrow fields from the actual
`OwnerDto.Payload`. A payload under an object-union variant uses its real
retained variant scope. References do not predict that scope or choose an
arbitrary occurrence from a composition product. The existing deferred source
binding and final unique-original-constructor proof remain authoritative.

Eligible targets must occur below original property/items/schema-valued
additionalProperties edges. A variant root is excluded: its physical constructor
may include owner common fields that are not part of that referenced nested
schema. Hierarchy ownership, ambiguous product copies, discarded overrides,
scratch additionalProperties, raw/const-only/count-bearing targets and target
root-default annotations retain their separate boundaries. Typed references and
oneOf members do not gain a new shape permission from this allOf extension.

Original catalog identity and lexical Pointer prefixes are checked, and a cached
schema cannot be reused under a different Pointer or foreign catalog. The final
source snapshot/retained occurrence check rejects missing or copied owners;
a borrowed consumer cannot become the canonical owner. Every property retains
its original source/default/check metadata. Consumers keep existing allOf
later overrides, required/Patch, nested copies, canonical nominal recursion,
nullable views and approximation warnings. No expanded schema, synthetic
internal component or alternate name allocator is introduced.

The existing shared internal-intersection capture budget remains16,384 semantic
node/edge units and8MiB per document, including separate charges for overlapping
targets and reuse of identical targets across analyzer contexts. Source-depth48,
ordinary composition work, checked-depth16, JVMslots240, default/value/input
limits and all prior tuple/non-tuple capture policies are unchanged. Kotlin API,
wire behavior and manifest/name-lock identity for existing inputs remain stable.

`bash scripts/test.sh feature composed-dto-intersections` generates Data/Plain
with validation off/on. Each project consumes nullable/non-null baseline and
combined dedup/hoist/merge/locked models and an independent239+1 constructor.
The [non-null](../tests/fixtures/http/composed-dto-intersections.json) and
[nullable](../tests/fixtures/http/nullable-composed-dto-intersections.json)
fixtures exercise real allOf and union lexical owners, canonical aliases and
recursive property types, nested DTO/enum defaults, four Json settings,
required/Patch, HTTP null/data and fragmented SSE null/data/failure/closure/reuse.
Core tests keep two former allOf refusal groups as original-input positives and
retain variant-root/hierarchy/ambiguity/discarded/scratch exclusions. Source
tests verify actual deferred owner occurrences, original AST/catalog/Pointer,
cache identity and shared capture bytes. Native tests compare core output,
readonly/no-op behavior and original failure Pointers/old-output preservation;
bounded fuzz varies allOf/union owners, original chains, nullable intent,
validation and direct/bridge use. Wider graphs and platforms require separate
evidence.

**Migration:** previously rejected eligible composed-owned nested DTO allOf
operands now generate using existing ordinary field/API/wire rules. Existing
named/direct internal operands and persisted formats retain their contracts.
General hierarchy/common/ambiguous ownership and broader D01/D02 stay open.

[Named hierarchy leaf nested DTO intersections](hierarchy-dto-intersections.md) extend D01/D02/F04/F07 only to actual ordinary nested DTOs directly owned by named plain-source hierarchy leaves. Original catalog/Pointer and unique retained owner/capture proof remain; parent tags/common fields do not become nested payload fields or Consumer slots. Nullable/Patch/default/check/recursion/warnings, shared16384work/8MiB capture and source/checked/JVM240 caps hold. Root/common/inline/composed-leaf/count/root-default/ambiguous roles stay separate; existing dispatch/API/wire/state remain stable, and broader graph/platform completion stays open.

[Named composed hierarchy-leaf nested DTO intersections](composed-leaf-intersections.md) extend D01/D02/F04/F07 to actual ordinary property/collection DTOs under named ordinary allOf leaf sources. Original catalog/Pointer and unique retained owner/capture proof remain; parent tags/common fields do not become payload fields or Consumer slots. Nullable/Patch/default/check/recursion/warnings and shared16384work/8MiB/source48/checked16/JVM240 limits hold. Root/common/inline/ambiguous/discarded/count/root-default roles stay separate; existing dispatch/API/wire/locks remain stable. Broader graph/platform completion remains open.

[Nested allOf DTO intersection targets](allof-target-intersections.md) extend D01/D02/F04/F07 to actual unique nested ordinary allOf constructors under direct ordinary object owners. Original composed source/alias snapshots and final retained owner proof remain; shared16384work/8MiB capture, source48/checked16/JVM240 and default/value caps hold. Nullable/Patch/default/check/recursion/override/warning/API/wire/locks keep their contracts. Composed/union/hierarchy lexical owners, products/count/root-default targets and broader graph/platform completion remain separate.

[Composed intersection targets across lexical owners](composed-target-owners.md) extend D01/D02/F04/F07 to actual unique nested ordinary allOf constructors under allOf/union owners and named plain/allOf hierarchy leaves. Original source/alias snapshots, final unique owner/capture proof and parenttag isolation remain; shared16384work/8MiB/source48/checked16/JVM240 and default/value limits hold. Nullable/Patch/default/check/recursion/override/warning/API/wire/locks keep their contracts. Ambiguous/discarded/common/inline/product/count/root-default roles and broader graph/platform completion remain separate.
