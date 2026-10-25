# Resolved model merging

`--merge-identical-models` (core `MergeIdenticalModels`, configuration `merge_identical_models`) is opt-in and defaults to false. It compares already validated ordinary DTO and plain String-enum owned root trees across components, shared hoisted models and operation models. Pure [primitive JSON-kind union roots](#primitive-json-kind-union-roots) have a separate bounded eligibility rule. Unlike operation-only [deduplication](deduplicate-models.md), eligible acyclic references and [owned self references](#owned-self-references) are compared by their resolved typed shape, not just their classifier spelling.

```sh
openapi-kt generate --input api.json --output build/generated \
  --package-name org.example.api --merge-identical-models \
  --exclude-merge-model org.example.api.models.KeepSeparate
```

`--exclude-merge-model` exclusions remain repeatable **exact fully qualified generated root names**, including the effective model suffix. They are not regular expressions or schema wire names. Additional [pattern/suffix exclusions](merge-exclusions.md) use separate flags and preserve this exact-name contract. An excluded root neither absorbs another root nor is absorbed. Nonempty exclusions require merging. In configuration, `exclude_merge_models = []` explicitly replaces an inherited list; an explicit CLI list replaces the effective configured list. Boolean false likewise overrides true. Unknown valid names simply match no root.

## Equivalence and canonical owners

Comparison preserves property/wire order, required/default-null state, nullability, constants, nested names, Patch context, validation, collection/object counts and serializer choices. Description annotations do not participate. Different referenced excluded, protected or unsupported classifiers remain distinct nominal types.

Canonical preference is component root, then shared hoisted root, then operation root; within each priority it uses fully qualified lexical order. Losing root files remain top-level Kotlin `typealias` declarations. Typed references to absorbed owned classifiers are rebound to the canonical owner, retaining original source/check/slot accounting. Private validation emission hints are combined across equivalent positions. Opaque provider classifiers, serializer expressions and constant expressions protect their referenced owners; those strings are not rewritten.

The initial subset uses bottom-up typed dependency fingerprints without expanding schema trees. Direct references to a root's own owned classifier normalize inside its finite declaration view. Mutual/external root cycles, unresolved dependencies and resolved comparison graphs taller than 48 conservatively remain unmerged. That is an optimizer skip, not a new input rejection or replacement for the existing schema budget. Primitive unions with a structured object peer, object unions, hierarchies, value-class/collection alias roots, nested-only extraction and arbitrary recursive structural equivalence remain outside this subset.

## Owned self references

Equivalent ordinary DTO roots may now merge when their typed fields refer to their own declared root, directly or through existing List/Map/nullable paths and nested owned declarations. For example, `A.next: A?` and `Z.next: Z?` can share canonical `A`, with `Z` emitted as a typealias. The original self-only pair in the fixture remains an executable acceptance case. This compares finite validated IR; it does not unfold recursive schemas or calculate general graph equivalence.

Every existing identity dimension still participates: field order, required/default/Patch/nullability, typed codecs, effective bounds, owned nested names and component factories. A recursive root referring to another owner does not gain self identity. Mutual root cycles remain skipped. Excluded or opaque-protected owners neither absorb nor are absorbed, and external dependency height limits remain unchanged. Option-off generation is unchanged.

This expands the explicit merge option's classifier/descriptor/equality migration to the newly eligible roots. Use the canonical owner for nested classifier access. Self-recursive shapes with different defaults/checks remain separate. Analysis diagnostics and resource checks still run before optimization; no source budget or name-lock format changes.

[Core self-reference guards](../core/merge_self_recursive_test.go), native lifecycle checks and the existing fresh `merge` profiles cover the extension. The consumer exercises canonical type assignment, finite recursive serde, nullable/List/Map/nested fields, defaults, constructor/copy checks, typed HTTP and live fragmented SSE rejection/closure and caller-client reuse across Data/Plain and short/qualified lock profiles. These checks make no performance or general recursive-equivalence claim.

## Kotlin migration

Merging intentionally changes classifier identity, runtime class names, serializer descriptors and cross-root equality. Constructors, data-class `copy`, companions and reified codecs use the canonical classifier; wire fields, encoding and typed checks retain their contracts. In plain style, `toString()` reports the canonical class name and no `copy`/components are added.

A typealias does **not** export nested classifiers/singletons. Replace `OldOwner.Nested` or `OldEnum.FAST` uses with the canonical owner; canonical `fromSerialName()` and `entries` remain available. Do not treat typealiases as independent extension/JVM overload identities. Resolved aliases can also collapse previously distinct nominal HTTP/SSE overloads. After merging, method allocation reuses path/method order and reserves existing natural names: for example, a second `submit` becomes `submit3` when `submit2` already exists. Operation paths, public model aliases and name-lock occurrences are retained; migrate affected method call sites. Rebuild affected compiled consumers; this is not a binary ABI guarantee.

Pass order is analysis → hoist/locked hoist → operation deduplication → merge → plain style → emission. Hoisted name-lock entries/fingerprints are computed before merging and keep their existing names via aliases; merging does not reset, delete or migrate lock state. [Name-lock](name-lock.md) storage/CAS/ownership rules remain separate.

## Evidence and limits

[Core contracts](../core/merge_test.go), [semantic/source guards](../core/merge_guards_test.go), [native configuration and preservation tests](../internal/cli/merge_process_test.go), [synthetic envelope](../tests/fixtures/http/merge-models.json), [codec/nested probes](../tests/fixtures/http/merge-probes.json) and [generated consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/MergeModelContracts.kt) define the subset. `bash scripts/test.sh feature merge` generates fresh data/plain basic and hoist/dedup/lock short/qualified profiles with four Json settings, constructor/serde bounds, Patch, controlled HTTP and fragmented live SSE checks.

The broader O04 contract, all optimizer combinations, downstream performance gains and Android/KMP/minimum-toolchain/platform coverage are not established by these independent profiles. Existing schema/collection/member/constructor/hierarchy budgets are unchanged.

## Primitive JSON-kind union roots

The explicit merge option also compares already validated `PrimitiveUnion` roots with no structured object peer. Eligibility does not expand operation deduplication or hoisting. Ordered emitted primitive kinds and the explicit null-carrier bit must match; different order/nullability, object peers and nested union trees remain distinct. Existing component-first/lexical canonical preference, resolved dependencies, exclusions and conservative opaque protection apply. In particular, incoming provider/type/serializer expressions naming a root, its generated sibling wrapper or its codec protect the physical union from replacement.

A losing root's original file exports aliases for **the root and every public sibling wrapper**: String/Int/Number/Boolean plus the old root name, and its Null singleton when present. For example, `ZDto = ADto` is accompanied by `StringZDto = StringADto`; old wrapper constructors, getters, Data copy/component APIs, serializer companions and Null singleton references remain usable. Plain style retains its existing no-copy policy. The aliases introduce no new codecs or replacement wrapper classes. Actual typed fields, containers, requests/responses and SSE data bind the canonical root; the original public names remain usable in the same Kotlin module.

Enabling merge intentionally changes runtime class identity, descriptor names and equality across old names to the canonical root/wrappers. Method signatures whose formerly distinct nominal arguments become equal are reallocated by the existing natural-name reservation policy: a `submit` collision can become `submit3` while an existing `submit2` remains unchanged. Migrate those affected method calls and rebuild compiled consumers. This is not a binary ABI guarantee. Disabling merge restores distinct source classes; original hoisted-name metadata remains valid because locked planning runs before merge.

Every source is analyzed before comparison; original warnings and diagnostics remain. JSON-kind dispatch, numeric Long/Double precedence, strict null carriers and existing lossy primitive-union constraints retain their contracts. There is no new Unknown fallback or strict oneOf match-count validation. Unexpected JSON kinds still reject. No source/default/check/depth/slot budget or persisted metadata encoding changes.

[Core eligibility, distinctions, exclusions and opaque guards](../core/merge_primitive_unions_test.go), [alias-group contracts](../internal/schema/merge_primitive_unions_test.go), [bounded fuzz](../core/merge_primitive_unions_fuzz_test.go), [native toggle/lock/core parity and output preservation](../internal/cli/merge_primitive_unions_process_test.go), [synthetic fixture](../tests/fixtures/http/merge-primitive-unions.json) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/MergePrimitiveUnionContracts.kt) define the profile. `bash scripts/test.sh feature merge-primitive-unions` compiles/runs Data/Plain × validation off/on, unmerged controls, merge-only and all-option locked packages, four Json settings, original wrapper/null alias calls, canonical descriptors/equality, controlled HTTP overload routing and live SSE error/cancellation/closure/reuse. Preparation-only mode does not execute Kotlin; the full `http` route includes the same consumer. Broader union/hierarchy/recursive/nested-only optimization and platform/resource completion remain separate; no performance gain is implied by functional checks.
