# Internal chains to named temporal scalars

Finite typed schema-internal reference chains and component-root bridges may end at an existing named `date`, `date-time` or native `date` scalar, including its supported named aliases. The canonical mappings remain `kotlinx.datetime.LocalDate` and `kotlin.time.Instant`, with their parse APIs and provider serializers. Public root bindings export transparent scalar typealiases; no DTO, wrapper, serializer, catalog entry or source node is copied.

For example, `Bridge → Hops/properties/time → TimeAlias → Time` preserves the original temporal source and every internal/named edge. A chain ending at an existing root bridge keeps the same scalar backing and adds that bridge's complete source work.

## Source identity, nullable views and wire

The resolver retains the real declared catalog separately from the original canonical temporal component, schema node and Pointer. Successful construction records the actual `ScalarAlias` or intrinsic nullable `ValueAlias`; final shared declarations recheck the original profile and unique declaration before artifacts are returned. A coincident caller-owned classifier named Instant or LocalDate does not authorize a temporal binding.

An explicit consumer overrides the nearest internal/root/named alias true/false; absent intent inherits intrinsic temporal nullability. Explicit false restores the original non-null scalar. Reified nullable codecs handle Kotlin null; direct provider serializers remain non-null. Parse calls, ISO JSON strings and invalid-value failures remain the provider's existing behavior. Shared optional null defaults and inline request required-nullable/Patch presence keep their existing contracts.

The generated TypeRef remains Scalar. Query values use the existing Ktor scalar stringification. JSON HTTP bodies/responses and SSE data retain the temporal provider codec; canonical erasure still determines deterministic method names. These mappings consume one serialized JVM slot, with the existing 240-slot model/operation budgets. No companion declaration, wrapper rawValue or provider layout is invented.

Temporal aliases are transparent. Every erased internal hop, final named edge and named/root alias contributes to the 48-level source budget. Cached height survives nullable wrapping and false narrowing. A public typealias or native mapping does not create a nominal source-depth barrier. Cycles, malformed sources, reference siblings and unused invalid fields still reject before output; native generation preserves previous owned files on failure.

## Boundaries and migration

Only existing named temporal profiles are extended. Direct internal special-scalar declarations, builtin primitive/bigint/byte/caller-native/raw projections, temporal enum/const source profiles, value-class declarations and shape borrowing keep their separate policies. Native date support does not add arbitrary caller-provider temporal inference, new constraints or general cross-reference aggregation.

Previously rejected finite chains and root bridges to these named temporals now produce canonical scalar bindings. Existing accepted Kotlin and wire behavior remain unchanged. The original date-time negative chain input is preserved as an explicit scalar-binding positive regression; other excluded targets remain diagnosed. This is a bounded F04/F07/M02 extension, not complete reference graphs or additional Kotlin-platform acceptance.

The [synthetic fixture](../tests/fixtures/http/temporal-reference-chains.json), [core bindings](../core/temporal_reference_chains_test.go), [source/slot/other-target guards](../core/temporal_reference_chains_guards_test.go), [private identity/cache tests](../internal/schema/temporal_reference_chains_test.go) and [native output preservation](../internal/cli/temporal_reference_chains_process_test.go) define the contract. Run `bash scripts/test.sh feature temporal-reference-chains` for fresh Data/Plain × validation projects with Plain optimization options, four Json configurations, ISO/nullable/false codecs, constructor/copy/Patch, controlled query/body/response and fragmented still-open SSE closure/reuse/coldness. Independent 240-slot and 48-hop inputs compile and run in each project. The [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TemporalReferenceChainsContracts.kt) also runs in the full HTTP suite.
