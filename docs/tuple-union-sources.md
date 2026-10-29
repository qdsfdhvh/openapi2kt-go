# Union tuple sources

OpenAPI 3.1 positional arrays accept existing direct `oneOf`/`anyOf` sources and existing named/internal union references, including pure aliases. Each original prefix and schema tail completes ordinary source validation before a typed or raw projection is chosen. This extends D02/D03/D06, F08/M03/M07/Q05; it does not add general composition or resolver permission.

A uniform prefix uses its original union source. Effective nullable intent and original terminal identity must agree; equal-looking independent inline unions remain distinct and yield `List<JsonElement>`. An explicit schema tail owns the actual element type after all prefixes validate. Boolean tails retain the [tuple approximation](primitive-tuples.md): positions, exact length and closed-tail maximum length are not enforced.

The ordinary consumer planner owns every selected type and serializer. Direct shared primitive unions keep JSON-kind payload wrappers and integer-versus-number token selection. Object unions retain tag/key-presence dispatch and Unknown fallback. References use the actual retained root/internal declaration, including root subtype names and Null carriers; inline and root dispatch are not interchangeable. Existing same-kind scalar folds and known-vocabulary String folds retain their mapped codecs and warnings. Request-only field unions keep existing raw JSON projection; direct operation tuple arrays may retain actual operation-owned peers.

## Defaults

Existing ordinary outer field, component-root and independent collection-alias annotations may use the selected union codec. Target, intermediate and element annotations are not inherited. Required and Patch annotations validate before initializer suppression; required arguments and `Patch.Absent` remain.

Primitive defaults use original branch kinds and enum/const pins. Tagged/presence object defaults must select a known variant and satisfy its original required fields and member literals; Unknown fallback does not authorize an annotation. Closed root object defaults use existing actual subtype/discriminator rules. Normalized String vocabulary defaults remain strict although runtime strings remain open. Nullable views and root Null carriers retain their separate codec semantics. Raw operation field projections retain their existing arbitrary-JSON default policy.

## Source and limits

Discarded original union declarations complete deferred validation in a fresh isolated analyzer. Scratch declarations, initializers and warnings do not become emitted authority. Selected original source/path/catalog nodes, actual declaration/selector, backing types and exact element addresses are checked against the final retained plan. Literal JSON under const/default/enum remains data rather than schema-reference authority.

Original source depth48, cumulative16384-step/8MiB source budgets, branch/member240, default16384-node/32KiB, checked-depth16 and JVM240 limits remain. Erased union branch/reference depth participates in collection alias height caching, including raw and explicit-tail projections. Generation remains deterministic and offline; failures preserve existing output and return no partial artifacts.

## Verification entry points

`bash scripts/test.sh feature tuple-union-sources` generates fresh Data/Plain × validation projects with baseline and combined optimizations, nonempty naming locks and an independent240-field DTO. Runtime assertions cover four Json settings, defaults, integer/number selection, nullable/Unknown/open enums, Patch, shared/direct HTTP and fragmented SSE closure/reuse. The fixture is also part of the HTTP registry. These entry points do not prove other platforms or the final release candidate.

General recursive/composed union tuple graphs, union branches outside their existing source contracts, broader default roles, platform matrices and final candidate acceptance remain separate.

[Product union tuple sources](tuple-product-union-sources.md) extend D02/D03/D06/F08/M03/M07/Q05 through actual Cartesian product declarations, original contribution/dispatch/null policies and strict outer own annotations. Original product/source/variant/origin/type/address/cache proof and budgets remain, including discarded validation and internal wrappers. General recursive/hierarchy/intersection/default/platform and final acceptance remain open.

[Hierarchy tuple references](tuple-hierarchy-references.md) extend D01/D02/D03/D06/F08/M03/M07/Q05 through actual named root/intermediate/leaf codecs and strict outer own defaults. Original dispatch/inherited tags/Unknown/null adapters, actual lineage/leaf/source/type/address/cache proof and budgets remain. General inline/multiple-parent/recursive/default/platform and final acceptance stay open.
