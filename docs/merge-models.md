# Resolved ordinary model merging

`--merge-identical-models` (core `MergeIdenticalModels`, configuration `merge_identical_models`) is opt-in and defaults to false. It compares already validated ordinary DTO and plain String-enum owned root trees across components, shared hoisted models and operation models. Unlike operation-only [deduplication](deduplicate-models.md), eligible acyclic model references are compared by their resolved typed shape, not just their classifier spelling.

```sh
openapi-kt generate --input api.json --output build/generated \
  --package-name org.example.api --merge-identical-models \
  --exclude-merge-model org.example.api.models.KeepSeparate
```

`--exclude-merge-model` exclusions remain repeatable **exact fully qualified generated root names**, including the effective model suffix. They are not regular expressions or schema wire names. Additional [pattern/suffix exclusions](merge-exclusions.md) use separate flags and preserve this exact-name contract. An excluded root neither absorbs another root nor is absorbed. Nonempty exclusions require merging. In configuration, `exclude_merge_models = []` explicitly replaces an inherited list; an explicit CLI list replaces the effective configured list. Boolean false likewise overrides true. Unknown valid names simply match no root.

## Equivalence and canonical owners

Comparison preserves property/wire order, required/default-null state, nullability, constants, nested names, Patch context, validation, collection/object counts and serializer choices. Description annotations do not participate. Different referenced excluded, protected or unsupported classifiers remain distinct nominal types.

Canonical preference is component root, then shared hoisted root, then operation root; within each priority it uses fully qualified lexical order. Losing root files remain top-level Kotlin `typealias` declarations. Typed references to absorbed owned classifiers are rebound to the canonical owner, retaining original source/check/slot accounting. Private validation emission hints are combined across equivalent positions. Opaque provider classifiers, serializer expressions and constant expressions protect their referenced owners; those strings are not rewritten.

The initial subset uses bottom-up typed dependency fingerprints without expanding schema trees. Cyclic/unresolved root dependencies and resolved comparison graphs taller than 48 conservatively remain unmerged. That is an optimizer skip, not a new input rejection or replacement for the existing schema budget. Union/hierarchy/value-class/collection alias roots, nested-only shape extraction and arbitrary recursive structural equivalence are outside this subset.

## Kotlin migration

Merging intentionally changes classifier identity, runtime class names, serializer descriptors and cross-root equality. Constructors, data-class `copy`, companions and reified codecs use the canonical classifier; wire fields, encoding and typed checks retain their contracts. In plain style, `toString()` reports the canonical class name and no `copy`/components are added.

A typealias does **not** export nested classifiers/singletons. Replace `OldOwner.Nested` or `OldEnum.FAST` uses with the canonical owner; canonical `fromSerialName()` and `entries` remain available. Do not treat typealiases as independent extension/JVM overload identities. Resolved aliases can also collapse previously distinct nominal HTTP/SSE overloads. After merging, method allocation reuses path/method order and reserves existing natural names: for example, a second `submit` becomes `submit3` when `submit2` already exists. Operation paths, public model aliases and name-lock occurrences are retained; migrate affected method call sites. Rebuild affected compiled consumers; this is not a binary ABI guarantee.

Pass order is analysis → hoist/locked hoist → operation deduplication → merge → plain style → emission. Hoisted name-lock entries/fingerprints are computed before merging and keep their existing names via aliases; merging does not reset, delete or migrate lock state. [Name-lock](name-lock.md) storage/CAS/ownership rules remain separate.

## Evidence and limits

[Core contracts](../core/merge_test.go), [semantic/source guards](../core/merge_guards_test.go), [native configuration and preservation tests](../internal/cli/merge_process_test.go), [synthetic envelope](../tests/fixtures/http/merge-models.json), [codec/nested probes](../tests/fixtures/http/merge-probes.json) and [generated consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/MergeModelContracts.kt) define the subset. `bash scripts/test.sh feature merge` generates fresh data/plain basic and hoist/dedup/lock short/qualified profiles with four Json settings, constructor/serde bounds, Patch, controlled HTTP and fragmented live SSE checks.

The broader O04 contract, all optimizer combinations, downstream performance gains and Android/KMP/minimum-toolchain/platform coverage are not established by these independent profiles. Existing schema/collection/member/constructor/hierarchy budgets are unchanged.
