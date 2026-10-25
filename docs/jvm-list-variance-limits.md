# Known inline List variance and JVM signature limits

The [certain constructor overflow guard](jvm-signature-limits.md) additionally
counts the known covariance bytes of an entire direct inline List chain ending
in builtin String/Boolean/Int/Long/Float/Double. The public Kotlin constructor
signature emits one wildcard byte for each nested List classifier. Root
nullability is allowed; nullable element nodes, aliases, maps, mixed containers
and unknown terminal classifiers contribute no variance bytes to this separate
proof. The fixed-syntax lower bound still applies independently to all its
previously recognized types. Neither proof guesses or expands uncertain names.

For example, each parameter of List nested 16 levels around Int contributes
307 fixed syntax bytes and 15 nested List covariance bytes. A 203-parameter
constructor has a 65,369-byte generic signature and fits the JVM UTF-8 constant
limit. A 204-parameter constructor needs 65,691 bytes and is rejected with
`E_MODEL_LIMIT` at the original retained model Pointer before any artifacts.
Native generate/check/validate preserve old output. Source/checked/JVM-slot and
other independent caps remain unchanged; no public types are erased or wrapped.

Fresh Data/Plain 203×16 controls compile and run JSON/invalid-last-field checks.
The Data control's copy signature is 65,405 bytes for its package/class spelling;
Plain does not introduce the Data copy API. Exact alias, nominal/long-name,
nullable-element/Map variance, copy-return-name and other metadata feasibility
remain separate: passing this lower bound is not a complete backend guarantee.
No compiler performance improvement or full width×depth Cartesian support is
implied.

`bash scripts/test.sh feature jvm-signature-limits` freshly generates Data/Plain
baseline and combined dedup/hoist/merge/locked 203×16 models under the existing
heap limits. It tests four Json configurations, Data copy, a failing constraint
in field 202, deterministic checks and rejection of 204×16 while preserving the
previous output. Native regressions also exercise original component/HTTP
schema Pointers, core/native consistency and readonly generate/check/validate
behavior. The existing [wide/deep resource route](wide-deep-validation.md)
continues to cover 200×16 and 240×12 across all eight optimization combinations.
These JVM profiles do not establish other platforms or general signature
representability.

**Migration:** additional inputs already unable to compile with the supported
Kotlin/JVM signature mapping now fail early with a resource diagnostic. Proven
feasible inputs, Kotlin call sites, wire behavior and persisted formats retain
their contracts.

[Retained Data copy signature limits](jvm-copy-signature-limits.md) add a post-transform lower-bound check using known parameter syntax/List variance plus the ASCII package return prefix. Final Plain classes stay admitted without a Data copy API; impossible Data signatures fail at original Pointers with no artifacts/old-output changes. Private Plan diagnostics remain outside declaration/lock identity and source permissions. Existing types/wire/caps/state formats hold; exact names/aliases/other metadata/platforms and wider D08 stay open.
