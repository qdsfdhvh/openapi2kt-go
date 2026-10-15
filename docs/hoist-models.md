# Shared operation roots (initial hoisting subset)

`GenerationOptions.HoistSharedShapes`, generate/validate
`--hoist-shared-shapes[=true|false]`, and configuration boolean
`hoist_shared_shapes` opt in to shared ordinary operation DTO/String-enum roots.
The default is false. `QualifyHoistedNames`, `--qualify-hoisted-names[=true|false]`
and `qualify_hoisted_names` choose owner-qualified shared names; their default is
false and true requires effective hoisting. Invalid dependencies return core
`E_OPTIONS`, direct CLI usage2, or configuration `E_OPTIONS` before selected
input IO. Configuration false replaces inherited true; overrides require one
selected spec. File and explicit stdin input share these generation options; no download command remains.

## Supported decisions

Only repeated eligible roots with the same recorded role and conservative typed
shape are moved. All owned nested declarations must be ordinary DTOs/enums.
Empty/raw and other declaration trees, components and helpers are not candidates.
The initial [deduplication contract](deduplicate-models.md) governs semantic
comparison: different checks, defaults, codecs, counts, presence/Patch,
nullability, property order or external nominal identity are not absorbed.
Hoisting works independently of deduplication and runs before it and before
[plain presentation](model-style.md). Original source/work/slot checks run on
every occurrence before either pass.

The lexical allocated canonical root moves to `<package>.models`; every former
root name remains an exported alias in `<package>.api.models`. Without
qualification, the recorded role plus suffix supplies the shared base name
(`InputDto`, `ResponseDto`, `ModeDto`). With qualification, retain the canonical
allocated owner-qualified name (`AlphaResponseDto`). Owner/role provenance is
recorded during allocation, not inferred from arbitrary identifier prefixes.
Ambiguous or missing provenance does not authorize a move.

Component/peer and actual same-package runtime classifier/file names reserve
case-insensitively. Allocate the base then base2/base3, visiting canonical groups
in lexical order. Runtime helpers in another package do not reserve names here.
Components are never silently replaced or merged. This is deterministic for
identical input/options, not a persisted name lock: adding a conflicting model
can change a numbered name. The additive [initial name lock](name-lock.md)
can retain names across supported source changes through opt-in CLI/config
transactional metadata; profile/drift/removal errors require explicit review.

Owned named references follow the shared canonical path. Provider classifiers,
external nominal types and codec/constant expressions remain opaque. Incoming
opaque code/type references mentioning an operation owner conservatively protect
that owner from aliasing or relocation, including references from other
components/operation surfaces. This can miss otherwise equivalent candidates;
it is preferable to removing a provider's previously valid nested classifier.
It is not classpath/provider semantic equivalence or general symbol rewriting.

## Opt-in migration

Even the formerly canonical API-model root becomes an alias. Constructors,
getters and canonical companion/serializer APIs remain; Data copy/component APIs
remain, while Plain intentionally has neither. Nested classifiers must use the
shared canonical owner: replace `AlphaResponseDto.Child` or
`BetaResponseDto.Child` under `api.models` with `models.ResponseDto.Child`
(or `models.AlphaResponseDto.Child` when qualified). Known sealed-enum data objects such as `FAST` are also nested classifiers: use the shared canonical owner or the old root's retained `fromSerialName("fast")`/`entries` companion API. Do not infer that a companion method preserving access means nested objects remain accessible through typealiases.

Class/reflection identity, serializer descriptor packages/names and
cross-former-name value equality follow the shared class. JSON wire data and
existing constructor/serializer checks remain the same for equivalent supported
trees. Class/descriptor keyed registries must migrate. Disable both options to
preserve historical package/class ownership; explicit hoist false with inherited
qualification true is invalid, so reset both settings.

This first subset does not collect arbitrary nested-only shapes, union/hierarchy/
SSE declarations, dictionary value aliases or inline error bodies. General
hoist/merge, suffix/merge exclusions, lock reset/owner-rename/root-move migrations
and complete optimization combinations remain outside it. No independent runtime library, compiler/RSS
speedup, Cartesian resource guarantee or additional platform support is implied.
See [configuration](configuration.md) and [compatibility](compatibility-matrix.md).
