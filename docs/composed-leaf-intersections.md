# Composed hierarchy leaf nested DTOs in allOf

An allOf operand may borrow a unique ordinary property/collection DTO nested
under a named hierarchy leaf whose original source is an ordinary allOf.
For example, `Leaf/allOf/0/properties/payload` retains the actual nested
constructor under `LeafDto`, and Consumer receives the payload's original
fields through existing allOf lowering. Direct internal Pointers, finite
internal chains and root bridges share this behavior.

This extends the [plain-source leaf contract](hierarchy-dto-intersections.md).
The target must remain an ordinary nested object; an allOf target itself,
count-bearing/root-default/const-only targets, hierarchy roots/common fields,
synthetic inline leaves and ambiguous/discarded/scratch sources retain their
separate boundaries. Source syntax alone does not authorize the target: the
final retained declarations must prove exactly one original nested constructor
under the actual named leaf. Removing the real leaf cannot be compensated by
a copied Consumer. Catalog/Pointer prefixes and cached identities remain
checked, with no manufactured component or predicted classifier.

The leaf's parent discriminator and inherited/common fields are not payload
fields. They do not appear in Consumer JSON or consume Consumer constructor
slots. Ordinary allOf later overrides and approximation warnings, nullable
views, required/Patch, original field/default/check metadata, canonical nested
aliases/codecs and recursive references keep their existing rules. Original
AST nodes remain unchanged. Existing hierarchy dispatch and Unknown behavior
remain under their existing contracts.

The shared internal-intersection capture budget remains16,384 work units/8MiB
per document. Repeated targets reuse capture, overlapping distinct targets are
charged separately and analyzer contexts share the same budget. Source48,
checked16, JVMslots240 and default/value/input limits remain unchanged.
Payload239+Consumer own1 is accepted;240+1 is rejected independently of the
parent tag.

`bash scripts/test.sh feature composed-leaf-intersections` generates fresh
Data/Plain×validation projects, each with nullable/non-null baseline and
combined optimized/locked profiles plus a239+1 constructor. The
[non-null](../tests/fixtures/http/composed-leaf-intersections.json) and
[nullable](../tests/fixtures/http/nullable-composed-leaf-intersections.json)
fixtures exercise original nested constructors/aliases/recursion, nested
enum/DTO defaults, four Json settings, required/Patch, HTTP and fragmented SSE
null/data/error/closure/reuse, and absence of the parent's kind tag in Consumer
JSON. Core/source tests cover collection-owned targets, discarded sources,
root annotations, slots, original AST/catalog/Pointer and actual owner proof;
native tests compare core bytes, readonly/no-op and original failure Pointers
with old-output preservation. Bounded fuzz varies chains and nullable views.
Broader hierarchy graphs and platforms require separate evidence.

**Migration:** formerly rejected eligible named composed-leaf nested ordinary
DTO allOf operands now generate with existing field/API/wire behavior. Existing
accepted declarations, hierarchy dispatch and manifest/name-lock formats keep
their contracts. General hierarchy/common/inline ownership remains separate.

[Composed intersection targets across lexical owners](composed-target-owners.md) extend D01/D02/F04/F07 to actual unique nested ordinary allOf constructors under allOf/union owners and named plain/allOf hierarchy leaves. Original source/alias snapshots, final unique owner/capture proof and parenttag isolation remain; shared16384work/8MiB/source48/checked16/JVM240 and default/value limits hold. Nullable/Patch/default/check/recursion/override/warning/API/wire/locks keep their contracts. Ambiguous/discarded/common/inline/product/count/root-default roles and broader graph/platform completion remain separate.
