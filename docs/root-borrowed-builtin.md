# Root aliases to borrowed builtin leaves

A component-root reference may point directly or through finite internal hops to an existing ordinary String, Boolean, Int, Long, Float or Double schema leaf inside a component. Pure named aliases and later typed chains may reference that bridge. The exported root remains a native scalar typealias, with the original leaf's checks and path-local nullable intent.

For example, `Bridge → Source/properties/hop → Other/properties/leaf` retains the terminal node in Other, its original Pointer and lexical source catalog. Source is the entry owner, not the terminal owner. No canonical component, nominal declaration, DTO, wrapper or serializer is fabricated for the internal leaf.

## Source and exported-root authority

The resolver admits real schema edges: properties, array items, typed additionalProperties and composition member positions, with RFC6901 escaping. It then validates the entire terminal scalar grammar, including supported format, nullable form and opt-in bounds. Unused bridges are validated as well. Newly supported roots expose illegal leaf keywords at the original leaf Pointer; excluded source profiles retain reference-use diagnostics.

A new source carrier records the original owner/schema/node/Pointer and parsed Scalar/check object. Final validation rechecks the original source path, profile and check provenance. Separately, the bridge's original retained ValueAlias declaration authorizes its export. Missing, same-name replacement or multiple retained exports cannot substitute for that declaration; copied source nodes or coincident Kotlin scalar spelling cannot authorize the borrowed leaf.

New bridge consumers share the checks parsed for their original leaf. Historical direct borrowed consumers retain their existing per-use check objects and emission hints. The new cache does not globally canonicalize validation objects or rewrite old helper/inline-check output.

## Nullable intent, budgets and consumption

From the consumer toward the terminal, the nearest explicit true or false wins; absence inherits the suffix and finally the leaf's own intrinsic intent. Owner object/container nullability does not flow into its leaf. False uses the real non-null Scalar backing. Root, named and internal hops retain complete source work through caching and narrowing; aliases introduce no nominal depth or validation barrier.

Existing source depth 48, work 16,384, checked collection depth 16 and constructor/method slot cap 240 remain. Required non-null Long/Double consume two slots, boxed nullable views consume one. Width, source-depth and checked-edge profiles are independent probes, not a maximum Cartesian compile guarantee.

Shared typed consumers preserve constructor/copy/decode checks, List/Map leaves, Patch.Value and request presence. HTTP query stringification, raw String bodies and native numeric/Boolean JSON codecs are unchanged. Preflight/postflight and SSE data checks use the original leaf; null/absent views keep existing behavior. Standalone alias assignment or scalar serialization remains unchecked.

## Boundaries and migration

Only ordinary builtin internal leaves are added. Entire borrowed collection targets, enum/const/valueclass, temporal/native/bigint/bytes/raw/true/recursive JSON sources, shape borrowing and borrowed collection trees with named terminals retain separate profiles. Existing named builtin and nominal contracts remain supported. Unsupported intermediate aliases and malformed neighboring source nodes are still validated; a new bridge cannot bypass them.

Previously rejected root references to these leaves now generate scalar aliases. Historical root-leaf inputs are preserved and explicitly graduate with canonical backing assertions; shape expansion and excluded neighboring profiles keep their negative tests. This bounded F04/F07/M02 increment does not complete general graphs, aggregation or other Kotlin platforms.

The [fixture](../tests/fixtures/http/root-borrowed-builtin.json), [core bindings](../core/root_borrowed_builtin_test.go), [guards](../core/root_borrowed_builtin_guards_test.go), [private authority tests](../internal/schema/root_borrowed_builtin_test.go) and [native preservation](../internal/cli/root_borrowed_builtin_process_test.go) define the contract. Run `bash scripts/test.sh feature root-borrowed-builtin` for fresh Data/Plain × validation projects, four Json configurations, native codecs/constructor/copy/Patch, HTTP/query and fragmented still-open SSE closure/reuse/coldness, plus independent source/slot/checked-edge inputs. The [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/RootBorrowedBuiltinContracts.kt) also runs in the full HTTP suite.
