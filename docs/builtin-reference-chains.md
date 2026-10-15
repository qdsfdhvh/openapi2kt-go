# Internal chains to named builtin scalars

Finite typed schema-internal reference chains and component-root bridges may end at an existing ordinary named String, Boolean, Int, Long, Float or Double component, including its already supported named aliases. Consumers and exported bridge aliases keep the canonical builtin scalar backing and existing codec. No wrapper, DTO, serializer or catalog component is invented.

For example, `Bridge → Hops/properties/value → ValueAlias → Value` retains the original scalar component, schema node, Pointer and opt-in checks. Catalog aliases, the original canonical terminal and the internal source path are separate facts. The actual retained `ScalarAlias` or intrinsic nullable `ValueAlias` authorizes the canonical source; a predicted name or coincident caller-native classifier does not.

## Nullable intent and checks

From the consumer toward the terminal, the nearest explicit nullable true or false wins. Absent intent inherits the remaining path, then intrinsic terminal nullability. A root bridge's false precedes a true on a later internal hop; an internal true before that bridge precedes its false. Cached suffix intent is never changed by an upstream consumer. Container and item/value nullability remain independent.

The TypeRef stays Scalar. Explicit false restores the canonical non-null backing and all source work. A nullable Long or Double occupies one boxed JVM slot; a required non-null Long or Double occupies two. The existing 240 model/method slot cap remains, so 120 non-null two-slot fields and 240 boxed fields are independent boundaries.

With validation enabled, the original named component's check object and source Pointer follow typed consumers: DTO construction/copy/automatic decode, List/Map leaves, Patch.Value, HTTP preflight/postflight and SSE event data. Null and absent views keep their existing behavior. Transparent typealias assignment and standalone scalar serialization do not enforce these checks. Use an existing explicit value class when a constructor validation barrier is needed.

Existing unmarked or false-only aliases to a non-null ordinary primitive still require validation opt-in. A new nullable consumer cannot bypass an invalid intermediate. Intrinsic nullable and supported own-nullable aliases keep their separate admission rules.

## Source budgets and wire

Each internal reference, final named edge and root/named alias contributes its original source work, including cached bridges and false narrowing. Transparent scalars add no nominal recursion or checked-collection barrier. Source depth 48, expansion work 16,384 and checked collection depth 16 remain unchanged; original failing use/leaf Pointers remain diagnostic locations.

String HTTP roots keep Ktor's raw text path. Other builtin scalar roots keep their configured JSON codecs; query parameters retain native stringification and omission. Actual Kotlin backing types determine method overloads: distinct scalar types may share a method name, while the same erased signature receives the existing deterministic numeric suffix. Patch presence, enum/union Unknown behavior, streaming cancellation and caller client ownership are unchanged.

## Boundaries and migration

Only ordinary named builtin sources are extended. Root aliases directly to borrowed internal builtin leaves, borrowed collection trees with new named terminals, bigint, byte/binary, caller-native classifiers, raw/true/recursive JSON carriers and shape borrowing retain separate policies. Existing temporal and nominal reference contracts remain supported.

Previously rejected finite chains now generate native scalar bindings. Existing accepted output and wire remain compatible. Historical String/Long chain inputs stay in their original tests and explicitly graduate only to canonical builtin assertions; unsupported neighboring profiles remain negative regressions. This bounded F04/F07/M02 extension does not complete general graphs, constraint aggregation or additional Kotlin platforms.

The [fixture](../tests/fixtures/http/builtin-reference-chains.json), [core bindings](../core/builtin_reference_chains_test.go), [depth/slot/source guards](../core/builtin_reference_chains_guards_test.go), [first-use tests](../core/builtin_reference_chains_late_use_test.go), [private identity tests](../internal/schema/builtin_reference_chains_test.go) and [native preservation test](../internal/cli/builtin_reference_chains_process_test.go) define the contract. Run `bash scripts/test.sh feature builtin-reference-chains` for fresh Data/Plain × validation projects, four Json configurations, constructor/copy/Patch, native/raw HTTP and query, fragmented still-open SSE closure/reuse/coldness, and independent 120 two-slot / 240 boxed / 48 source / 16 checked-edge inputs. The [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/BuiltinReferenceChainsContracts.kt) also runs in the full HTTP suite.
