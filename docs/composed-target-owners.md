# Composed DTO intersection targets across lexical owners

An allOf operand may reference a unique original nested ordinary allOf DTO
under an allOf/union lexical owner or a named hierarchy leaf. This extends the
[direct ordinary owner contract](allof-target-intersections.md) through the
existing original lexical property/collection path and final retained owner
proof. Direct internal Pointers, finite chains and root bridges share the same
source; no component or classifier is manufactured.

For example, `Owner/allOf/0/properties/payload` may identify an allOf payload
under an ordinary allOf owner or a named allOf hierarchy leaf.
`Owner/oneOf/0/properties/payload` retains its actual variant-owned constructor.
The final declaration tree must contain exactly one occurrence of that
original composed target under the actual catalog owner. Repeated product
occurrences, discarded overrides, missing owners, hierarchy common sources
and synthetic inline leaves remain rejected. Source syntax alone cannot
choose an owner; typed binding alone cannot grant shape authority.

The payload's original semantic nodes, named shape-reference edges, alias
heights, nullable intent and actual catalog/Pointer remain authoritative.
Parent hierarchy tags and inherited/common fields do not become payload
fields, Consumer JSON or Consumer constructor slots. Nullable views,
required/Patch, original defaults/checks, canonical nested aliases/codecs and
recursion retain their rules. Existing allOf later overrides and approximation
warnings remain; no exact intersection validation or new dispatch is added.

Union products, target root defaults/counts and broader hierarchy graphs
retain their separate boundaries. Original ASTs are not rewritten. The shared
intersection capture budget stays16,384 work units/8MiB per document across
ordinary and composed targets, with repeated reuse, overlapping target charges
and shared analyzer contexts. Source48, checked16, JVMslots240 and all
input/default/value limits remain.

`bash scripts/test.sh feature composed-target-owners` generates four owner
roles: allOf owner, union owner, named plain-source leaf and named allOf leaf.
Each runs Data/Plain×validation with nullable/non-null baseline and combined
optimized/locked profiles plus an independent239+1 constructor. The
[main non-null](../tests/fixtures/http/composed-target-owners.json) and
[nullable](../tests/fixtures/http/nullable-composed-target-owners.json) fixtures
use a named allOf hierarchy leaf; explicit role fixtures cover the other
owners. Fresh Kotlin consumers exercise actual aliases/constructors,
recursion, nested DTO/enum defaults, four Json settings, required/Patch,
HTTP and fragmented SSE null/data/error/closure/reuse. Source tests confirm
actual occurrences across all four roles, original AST/catalog/Pointer,
shared capture, missing/repeated owner rejection and parent tag isolation.
Native tests retain exact formerly rejected sources as positive cases with
core byte comparison, readonly checks and original failure preservation.
Broader graphs/platforms require separate evidence.

**Migration:** formerly rejected eligible nested allOf DTO operands across
these lexical owners now generate with existing field/API/wire rules. Existing
accepted declarations, hierarchy dispatch and manifest/name-lock formats keep
their contracts. Ambiguous/common/inline ownership remains separate.
