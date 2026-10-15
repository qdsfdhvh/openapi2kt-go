# Operation model deduplication (initial subset)

Generation remains offline and deterministic. The default is **off**. Opt in with
`GenerationOptions.DeduplicateModels`, `--deduplicate-models` (or
`--deduplicate-models=true|false`) on generate/validate, or boolean
`deduplicate_models` in configuration defaults/specs. A separate CLI boolean
argument is not accepted; use `=false` to reset an inherited setting. CLI overrides
require a single selected spec as described in [configuration](configuration.md).
File and explicit stdin input share this option; no download command remains.

## Folded declarations

The initial pass compares operation-local ordinary DTO and String-enum roots.
An eligible DTO's owned nested declarations must also be ordinary DTOs/enums;
raw empty projections and other declaration trees remain unchanged. Components,
provider types and runtime helpers are not candidates. Canonical selection uses
the lexical order of already allocated class names. Former root names remain
exported typealiases to that canonical classifier in the same API-model package.
Original semantic checks, source-depth and constructor budgets run on **every**
occurrence before folding; errors and warnings keep their original provenance.

Typed equivalence retains property order, names/wire names, nullability,
defaults, Patch/presence, constants/initializer text, vocabulary, checks, counts,
codec overrides and owned nested declarations. Only declared owned classifier
names normalize; arbitrary provider names, external nominal refs and serializer
expressions remain opaque. Source hop accounting is not wire identity, but it is
checked before comparison. Constructor emission-safety hints combine into the
retained declaration. No JSON AST expansion, network access or Kotlin-text
comparison is used.

Incoming opaque provider/type/code expressions mentioning an operation owner conservatively protect its classifier from aliasing, including dependencies outside that candidate tree. This can miss otherwise equivalent candidates instead of breaking a provider's old nested classifier access. [Initial shared-root hoisting](hoist-models.md) uses the same protection and runs before deduplication.

This deliberately conservative subset is **not** general resolved-reference
model merging, shared-shape hoisting, name locking or arbitrary operation
union/hierarchy/SSE deduplication. It does not promise performance improvements,
Cartesian resource bounds, arbitrary caller layouts or additional platforms.

## Opt-in migration

Root alias constructor/getters and canonical companion methods are available.
For default data classes, copy/component APIs remain available through the alias;
[plain classes](model-style.md) retain their intentional absence. Serializer
companions, enum entries/fromSerialName and existing validators use the canonical
declaration.

A typealias does **not** own nested classifiers: migrate
`BetaResponseDto.Child` to the canonical `AlphaResponseDto.Child`, and access enum
known objects/Unrecognized through the canonical enum classifier. Keep ordinary
root calls using the exported alias if desired. Nested type access failures and
canonical-qualified positive calls are checked with a real Kotlin compiler.

Class identity, reflection, serializer descriptor names and cross-former-name
value equality now follow the canonical class. Previously distinct Alpha/Beta
instances may now compare equal; class-keyed registries and descriptor-dependent
consumers must account for this. JSON field/value wire behavior remains the same
for equivalent supported declarations. Different defaults/checks/codecs/type
identities are not silently merged. Explicitly disable the option to preserve
historical separate-class APIs/identity.

See the [compatibility matrix](compatibility-matrix.md) for the
broader optimization work and [output batches](output-batches.md) for ordinary
failure rollback. This pass adds no filesystem transaction or naming-lock format.
