# Finite reference chains to named bigint scalars

Typed internal references may follow finite `$ref`/nullable-only paths to an original named bigint component or eligible named alias. Component-root bridges and later named/internal chains retain the original `kotlin.Long` backing, catalog Schema/Pointer and bigint source marker. These transparent aliases do not create a DTO, value class, serializer or nominal recursion barrier.

## Source authority and nullable intent

The shared resolver records the canonical bigint source separately from the declared root entry and internal hops. Original source bytes/node/Pointer/profile and the unique retained ScalarAlias or intrinsic nullable ValueAlias authorize the target. Missing, replacement, duplicate retained declarations or changed source reject even if Kotlin names match. First HTTP and union-field uses pass the same source decision and final authority checks.

Along the actual path from consumer to terminal, the nearest explicit nullable true or false wins; absent inherits the cached suffix and finally intrinsic source intent. Root override before an internal hop differs from an incoming internal override before a root. Consumers cannot change the shared cache. False preserves actual primitive Long layout and all erased hop work.

Source depth 48, work 16,384 and slot cap 240 remain. Required non-null Long takes two JVM slots (120 fields/parameters), nullable Long one (240). SSE method profiles reserve their existing extra slot. Width and depth are verified separately; combined maxima and other platforms require their own evidence.

## Field-only string codec and numeric roots

A direct DTO field, including a reference through the new chains, retains the generated `LongAsStringSerializer`: encoding decimal strings and decoding with `decodeString().toLong()`. The original bigint marker decides this field use. Patch annotates its Long value argument, preserving Absent, Null and Value. Real retained fields own the helper; alias-only sources do not emit it.

Root aliases, List/Map elements, ordinary HTTP scalar roots and SSE data retain ordinary numeric Long serialization. Caller-native Long remains its own source profile and does not acquire the magic field helper. Overflow/kind/null handling remains the provider's current contract; this adds neither arbitrary precision nor strict string-kind validation.

Constructors, MIN_VALUE/MAX_VALUE, scalar companion serializer versus reified nullable adapters, copy where available, query stringification, method erasure suffixes and HTTP numeric bodies preserve the accepted API/wire. A still-open fragmented SSE stream closes on bad data/cancellation and remains reusable as a cold flow; reconnect policy remains caller-owned.

## Boundaries and verification

Only original named bigint source profiles gain chains. The original full bigint grammar still rejects bounds, enum/const/wrapper or unsupported structured forms at their original Pointers. Borrowed internal bigint leaves, byte/binary, caller-native, raw/true/recursive carriers and shape borrowing retain separate contracts. Borrowed collection trees do not gain named terminals from this change.

Previously rejected named-bigint internal/root chains now emit native Long aliases. Historical negative inputs are kept and explicitly graduate with native backing and field-helper assertions. Existing direct inputs, fixture bytes and generated output/report profiles remain unchanged.

The [fixture](../tests/fixtures/http/bigint-reference-chains.json), [core codec tests](../core/bigint_reference_chains_test.go), [budgets/source guards](../core/bigint_reference_chains_guards_test.go), [late uses](../core/bigint_reference_chains_late_use_test.go), [private source identity](../internal/schema/bigint_reference_chains_test.go) and [native preservation](../internal/cli/bigint_reference_chains_process_test.go) define this bounded F04/F07/M02 increment. `bash scripts/test.sh feature bigint-reference-chains` generates fresh Data/Plain × validation projects with four Json configurations, field versus numeric root/List/Map/Patch wire, query/HTTP and fragmented SSE closure/reuse/coldness, plus independent 48-hop and 120/240 slot inputs. The [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/BigintReferenceChainsContracts.kt) runs in the full HTTP suite. General graphs, special source roles and platform acceptance remain separate.
