# Temporal list uniqueness

With `--validation`, arrays whose already admitted elements are ordinary `string` schemas with `format: date` or `date-time` may use `uniqueItems: true`. Direct temporal elements and supported transparent scalar references retain their original LocalDate/Instant type, codecs, nullable intent and source authority. Containers may be roots, DTO properties, nested List/Map values, or existing typed request/response/SSE consumers. This D05/M03/M07/M12/V05 increment does not add new reference paths.

Direct inline typed dictionaries with LocalDate/Instant values can also be array elements under the separate [typed Map uniqueness contract](map-unique.md). It uses structural Map equality and mapped default keys; item/value references and nested-container values retain that contract's exclusions.

## Equality and barriers

LocalDate values compare by date; Instant values compare by the actual time point, including nanoseconds. For example, `2026-01-02T03:04:05Z` and `2026-01-02T05:04:05.000+02:00` decode to the same Instant and are rejected as duplicate entries. Different nanoseconds remain distinct. At most one nullable null member is allowed; null containers skip their checks. Empty/singleton lists pass uniqueness unless another count constraint excludes them.

This is uniqueness of mapped Kotlin values. JSON Schema compares the original JSON strings, so different textual timestamp offsets may be distinct there. The generator's temporal codec already normalizes those spellings; temporal uniqueness deliberately follows the mapped values and is not exact JSON-string uniqueness or general JSON Schema validation.

The existing shared scalar helper sorts a copy of non-null Comparable values and compares adjacent entries. It leaves caller iteration order unchanged. No temporal runtime helper, custom serializer, extra serialized state or global Json configuration is introduced. Constructors, Data-class copy and automatic DTO decoding enforce the check; typed collection HTTP arguments enforce it before the final callback/engine dispatch. Invalid typed responses and known SSE payloads fail decoding. SSE input closes on failure/cancellation, while a new cold collection and the client remain reusable.

Independent collection alias assignment and standalone reified serialization are not validation barriers. Mutating caller-owned collections after construction and arbitrary requestBuilder body replacements keep their existing boundaries. The public model/client API and ISO wire codecs follow the existing temporal contracts.

## Defaults and original sources

Existing admitted direct/nested/referenced collection defaults and component-own factories validate uniqueness of their original annotation values. Instant comparison keys normalize offsets and fractional precision before duplicate detection, including times before the Unix epoch. Emitted default JSON retains its original timestamp spelling; only the comparison key is normalized. Errors identify the second duplicate at its original `/default/index` path, including nested values. Required annotations and Patch-suppressed initializers still validate before suppression.

Literal grammar, scalar parsing, ISO precision, nearest nullable intent and default source/value budgets remain unchanged. Bad temporal values, child defaults, unsupported member constraints and discarded source annotations cannot acquire permission from an empty/null default. JSON-value uniqueness and scalar/native roles remain separate.

Original format/source identity is required in addition to a compatible Kotlin type. Native-date markers, caller-owned Instant/LocalDate spellings, enum/const/valueclass/bytes sources, structured containers as elements and general projections remain excluded from this scalar profile. The linked typed Map profile has its own source gate. A prefix-only tuple projection does not gain uniqueness eligibility; an existing explicit eligible scalar `items` source retains its own policy. Temporal bounds/order validation are not added. `uniqueItems: false` remains exactly effect-free with validation enabled; both flags still reject without validation.

Source depth 48, checked container edges 16, work 16,384, constructor/method slots 240 and existing default limits remain. Temporal constraints participate in the existing collection-check identity and optimization/name-lock policy. No persisted-format migration is needed. Previously rejected eligible temporal inputs now generate checks; existing accepted inputs retain their API/wire contracts.

## Verification

The [synthetic fixture](../tests/fixtures/http/temporal-unique.json), [core equality/source/default/resource contracts](../core/temporal_unique_test.go), [native old-output preservation](../internal/cli/temporal_unique_process_test.go) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TemporalUniqueContracts.kt) define this profile. Historical direct and nested/referenced temporal uniqueness negatives retain their exact input as explicit opt-in positives; other source exclusions remain.

`bash scripts/test.sh feature temporal-unique` generates Data/Plain × validation projects. Validation-off uses the unbounded control and verifies that the constrained original still rejects. Plain also exercises dedup/hoist/merge/name-lock. Four Json default/null settings exercise mapped equality, nullable roots/members, defaults and factories, Patch, actual request JSON, preflight, HTTP responses and fragmented live SSE failure/cancellation/closure/cold collection/reuse. The complete `http` route includes the checked consumer. This is Kotlin/JVM evidence; Android/KMP, minimum toolchains and general uniqueness remain separate.
