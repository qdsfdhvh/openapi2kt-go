# Named hierarchy leaf nested DTOs in allOf

An allOf operand may borrow a unique ordinary nested DTO directly owned by a
named hierarchy leaf whose original source is an ordinary direct object. Direct
internal Pointers, finite internal chains and root bridges use the existing
original lexical source and final unique-owner proof. For example,
`Leaf/properties/payload` retains the actual `LeafDto.Payload`, while the
Consumer receives its original ordinary fields through existing allOf lowering.
The payload itself remains an ordinary constructor DTO, not a hierarchy root.

The lexical source must remain direct and outside a nested composition. Root
hierarchy/leaf shapes, inherited/common fields, inline synthetic branches,
count-bearing/root-default/raw/const-only targets and
ambiguous/scratch/discarded ownership retain separate boundaries. This does not
flatten a hierarchy leaf constructor, select a subtype or change discriminator
dispatch. The leaf owner's parent tag and inherited fields are not properties
of the referenced nested payload and do not leak into Consumer JSON or its
constructor-slot accounting.

Source catalog/Pointer prefixes and cached identity remain checked. The final
retained declaration tree must confirm exactly one original nested constructor
under its actual named leaf; a borrowed Consumer cannot replace a missing leaf
owner. Original AST/schema nodes, field/default/check metadata, canonical
nominal recursion, nullable views, required/Patch, later overrides and
approximation warnings keep their contracts. No internal component or classifier
name is invented. Canonical leaf aliases/codecs and parent hierarchy behavior
remain under their existing contracts.

The shared internal-intersection capture budget stays16,384 work units/8MiB per
document, with repeated target reuse, overlapping target charges and shared
analyzer contexts. Source-depth48, ordinary composition work, checked-depth16,
JVMslots240 and default/value/input limits remain. Nested payload239+Consumer
own1 uses240 slots, independently of the leaf owner's serialized parent tag.
No erasure, public wrapper, persisted identity or budget relaxation is added.

`bash scripts/test.sh feature hierarchy-dto-intersections` generates fresh
Data/Plain×validation with nullable/non-null baseline and combined
optimized/locked inputs, plus an independent239+1 constructor. The
[non-null](../tests/fixtures/http/hierarchy-dto-intersections.json) and
[nullable](../tests/fixtures/http/nullable-hierarchy-dto-intersections.json)
fixtures and consumers exercise canonical nested aliases/constructors,
original-owner recursion, copied nested DTO/enum defaults, four Json settings,
required/Patch, HTTP null/data, fragmented SSE null/data/error/closure/reuse,
and absence of the parent's kind tag in Consumer JSON. Core tests preserve the
former direct named-leaf nested refusal as a positive, preserve preexisting
named-root shape behavior and retain common/inline exclusions
and combined-slot bounds. Source tests verify
actual hierarchy-leaf/nested origins, original AST/catalog/Pointer/capture,
shared contexts and missing-owner/foreign-parent rejection. Native checks
compare core bytes, readonly/no-op and original failure Pointers/old-output
preservation; bounded fuzz varies original chains, nullable intent, validation
and direct/bridge use. Broader hierarchy graphs and platforms require separate
evidence.

**Migration:** formerly rejected eligible named plain-leaf nested DTO allOf
operands now generate using existing ordinary field/API/wire rules. Existing
hierarchy dispatch, accepted named/internal/composed operands and manifest/name-
lock formats retain their contracts. General hierarchy/common/inline ownership
and broader D01/D02 remain open.

Named ordinary allOf leaf sources use the separate [composed-leaf extension](composed-leaf-intersections.md) and the same final original-owner proof.

[Composed intersection targets across lexical owners](composed-target-owners.md) extend D01/D02/F04/F07 to actual unique nested ordinary allOf constructors under allOf/union owners and named plain/allOf hierarchy leaves. Original source/alias snapshots, final unique owner/capture proof and parenttag isolation remain; shared16384work/8MiB/source48/checked16/JVM240 and default/value limits hold. Nullable/Patch/default/check/recursion/override/warning/API/wire/locks keep their contracts. Ambiguous/discarded/common/inline/product/count/root-default roles and broader graph/platform completion remain separate.
