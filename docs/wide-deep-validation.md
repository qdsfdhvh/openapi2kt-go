# Repeated deep collection validation

Large constructor bodies containing repeated inline List/typed Map validation
traversals use private typed check methods. The emitter measures only traversals
that would otherwise remain inline; after 12 KiB of emitted traversal source in
one model, it moves those bodies behind ordinary private methods. This is an
emission strategy, not a new schema or value limit.

Methods share only an identical rendered Kotlin parameter type and identical
retained checks. Different bounds, nullable roots or nominal alias spellings do
not share a method. The original full collection/Patch type remains at the call
boundary, property names reserve helper identifiers, and no instance state,
casts, serializer properties or regex caches are introduced. Constructors/copy,
decode and HTTP/SSE keep the same checks, order, messages and failure behavior.
Small bodies and existing scalar/shared uniqueness helper routes retain their
emission policy. Declaration identity, descriptors and persisted locks do not
include this private method planning.

`bash scripts/test.sh feature wide-deep-validation` checks two explicit workloads:
200 optional List<Int> fields with 16 checked collection edges, and 240 fields
with 12 edges. Each has retained leaf bounds, a distinct stronger contribution,
an identifier that collides with the preferred helper name and an independent
component default factory. Repeated HTTP request/response shapes activate
hoist/dedup/merge, and a referenced SSE payload keeps its actual model codec.
The deterministic [fixture](../tests/fixtures/http/wide-deep-validation.json)
uses values i+1, widths 200/240 and depths 16/12, without external input.

Data/Plain each compile all eight combinations of dedup/hoist/merge, including
locked hoisting. Four Json settings check fresh factories, actual missing/null
behavior without factory inheritance, distinct bound rejection and serde;
Data additionally checks copy. Controlled HTTP and live fragmented UTF-8 SSE
check validation before dispatch, stream failure/closure and client reuse.
The consumer partitions its complete original source inventory into compiler
invocations with a 128 KiB packing target, using the same inventory/dependency
planner as complete HTTP checks. It uses in-process Kotlin compilation and the
existing heap settings: no case, source, width or depth is reduced to fit.
This profile records compilation/runtime acceptance, not a general speedup.

The independent source48/work16384, checked16, value16384/default32KiB and
JVMslots240 limits remain. They do not imply that their complete Cartesian
product is representable. JVM classfiles additionally limit UTF-8 constants,
including generic method signatures, to 65,535 bytes. For example, a constructor
with 240 parameters of List nested 16 levels around Int needs a signature above
that limit. Moving validation bodies does not shorten a public signature;
a [certain-overflow generation guard](jvm-signature-limits.md) now rejects
constructors whose minimum signature already exceeds that hard limit. Broader
exact signature feasibility remains separate.
The tested workloads above do not establish arbitrary leaf/name/Map signatures,
maximum width×depth, other compiler/dependency versions or other platforms.

**Migration:** large repeated traversal bodies may gain/reuse private typed
methods. Kotlin public constructor/property/copy call sites, JSON/wire values,
validation results and manifest/name-lock formats retain their contracts.
Regenerate/rebuild to use the private implementation; no consumer rewrite is
required by this traversal change.

[Known inline List variance limits](jvm-list-variance-limits.md) tighten the one-sided JVM constructor proof only for complete direct List chains ending builtin6, including optional roots. Known nested List covariance rejects impossible204×16 while preserving freshly compiled203×16 Data/Plain and earlier200×16/240×12. Original Pointers, no artifacts/failure preservation, public types/wire and all existing caps remain. Alias/Map/nullable-element/mixed/copy/name exactness and wider platforms/general D08 remain open.

[Retained Data copy signature limits](jvm-copy-signature-limits.md) add a post-transform lower-bound check using known parameter syntax/List variance plus the ASCII package return prefix. Final Plain classes stay admitted without a Data copy API; impossible Data signatures fail at original Pointers with no artifacts/old-output changes. Private Plan diagnostics remain outside declaration/lock identity and source permissions. Existing types/wire/caps/state formats hold; exact names/aliases/other metadata/platforms and wider D08 stay open.
