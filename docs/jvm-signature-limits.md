# Certain JVM constructor signature overflow

Generation rejects a retained DTO constructor when a guaranteed minimum size
of its generic signature already exceeds the JVM classfile UTF-8 constant limit
of 65,535 bytes. The error is `E_MODEL_LIMIT` at the original component, nested
object or HTTP schema Pointer, before any artifacts are returned or written.
Native generate/check/validate preserve existing output on that failure.

The proof counts fixed inline List/typed Map signature syntax, fixed String keys
and known builtin primitive/reference types. Nullable primitives use their
boxed signature inside generic types. A separate [known inline List covariance
proof](jvm-list-variance-limits.md) counts its demonstrated public-constructor
wildcards. Other variance, unknown classifiers, transparent alias spelling and
other uncertain bytes contribute nothing to the minimum; the proof never guesses their layout. Only retained model constructors
are checked. Discarded validation-only declarations do not own emitted methods.
Diagnostic location tracking grants no new source or annotation permission.

For example, 240 parameters of List nested 16 levels around Int need at least
73,683 bytes even after omitting wildcard variance. That is already impossible;
the corresponding ordinary Kotlin/JVM constructor signature is larger still.
The [compiled resource profiles](wide-deep-validation.md), including 200×16 and
240×12, remain admitted. Source/work/value/default-byte/checked-depth/JVM-slot
limits and public Kotlin types are unchanged. This guard adds no erasure, cast,
wrapper type or serializer state.

This is deliberately a one-sided feasibility check, not a JVM signature encoder
or complete backend guarantee. A minimum at or below the limit does not prove
that wildcard, alias, nominal/caller-owned names, copy signatures or other class
metadata will fit. Those wider representability cases remain separate. Existing
independent caps do not imply a complete Cartesian product, and JVM checks do
not establish full cross-platform acceptance.

Go regressions cover the known impossible profile, the preserved compiled
profiles, boxed/unknown/Map accounting and original nested/HTTP source Pointers.
Native regressions cover actual generate/check/validate diagnostics and failure
preservation. `feature wide-deep-validation` remains the actual compilation and
runtime route for its documented feasible workloads and optimization matrix.

**Migration:** inputs proven unable to produce a representable JVM constructor
now fail early with a structured resource diagnostic instead of emitting
unbuildable Kotlin. Existing working call sites, wire behavior and persisted
manifest/name-lock formats retain their contracts. No consumer rewrite is
required for admitted inputs.

[Retained Data copy signature limits](jvm-copy-signature-limits.md) add a post-transform lower-bound check using known parameter syntax/List variance plus the ASCII package return prefix. Final Plain classes stay admitted without a Data copy API; impossible Data signatures fail at original Pointers with no artifacts/old-output changes. Private Plan diagnostics remain outside declaration/lock identity and source permissions. Existing types/wire/caps/state formats hold; exact names/aliases/other metadata/platforms and wider D08 stay open.
