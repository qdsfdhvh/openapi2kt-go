# Tuple references to components with default factories

An already accepted tuple reference may consume a component that owns an
independently supported [default factory](component-default-factories.md). The
component's annotation remains its own factory; it does not initialize a tuple
position, a list element or a required request argument.

This combines the existing tuple profiles for builtin/temporal scalar, plain
String enum, ordinary DTO and actual allOf constructor targets with their
existing component factory roles. Eligible intermediate named aliases may each
retain their own factory through the existing wrapper contract. The original
component node must already have factory permission and its actual catalog
identity/pointer. An annotation on a positional `$ref`, an internal property or
an unsupported wrapper cannot borrow that permission from its target.

Every original factory is validated independently, including before a prefix is
discarded by a mixed raw projection or explicit scalar tail. Strict original
enum membership, required object properties, leaf kinds, nullable intent,
constraints and default node/32KiB byte budgets remain. Invalid annotations fail
at their original component or alias `/default` Pointer. Runtime open enum
values remain open; strict annotation membership does not close their codec.
The factory value is not copied into the tuple type or used as a missing-key
fallback. For example, an ordinary DTO factory may provide its required `value`,
while decoding `[{}]` through `List<that DTO>` still fails for missing `value`.

The original constructor, serializer, canonical factory/helper, nullable view and
lexical owner remain authoritative. Same-source/nullability uniform Lists,
mixed raw projections, explicit-tail selection, existing approximation warnings,
nominal recursion, optimization and name-lock behavior retain their contracts.
Original alias/default source nodes are checked even for discarded prefixes.
Source48, per-array reference work16384, checked16, constructor slots240 and the
new allOf tuple cumulative snapshot limits remain unchanged.

The shared borrowed String-enum validator also honors the original component's
own factory permission for existing ordinary collection references. It still
checks the actual original vocabulary and aliases; it neither grants field-own
annotations nor makes factory defaults into element defaults. The exact former
borrowed-enum default regression remains an accepted source/bridge case.

No new factory shape is introduced. Numeric/native/valueclass/union/hierarchy
tuple targets, additional default-bearing source roles, internal wrapper
factories, tuple-owned default annotations and nominal uniqueItems remain under
their existing independent contracts. In particular, adding a factory to a
composition not already accepted by the factory planner stays unsupported.

The [synthetic input](../tests/fixtures/http/tuple-target-defaults.json),
[core contracts and graduated original cases](../core/tuple_target_defaults_test.go),
[original-source guards](../internal/schema/tuple_target_defaults_test.go),
[bounded fuzz](../core/tuple_target_defaults_fuzz_test.go),
[native parity/policy/old-output preservation](../internal/cli/tuple_target_defaults_process_test.go)
and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleTargetDefaultsReferencesContracts.kt)
define this D03/D06 increment. Run `bash scripts/test.sh feature tuple-target-defaults`
for fresh Data/Plain × validation, baseline/allopts/nonemptylocks, width240,
four Json settings, separate target/alias factory values, fresh helper values,
absence of inheritance, open enum/temporal codecs, typed HTTP/Patch and fragmented
SSE error/close/cancellation/reuse. The complete `http` route invokes the same
consumer. PREPARE_ONLY remains preparation only; other platforms require their
own evidence.

**Migration:** formerly rejected references to independently eligible component
factories now generate. Existing accepted call sites, wire behavior, warning
policies and persisted output formats retain their contracts. This does not
close general D03/D06 or the broader default/composition roadmap.
