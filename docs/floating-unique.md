# Finite floating list uniqueness

With `--validation`, arrays whose original elements map to Float or Double accept `uniqueItems: true`. Direct ordinary number sources (`format: float`, `double`, or the existing unformatted Double mapping), admitted scalar references/aliases and native floating enum leaves retain their existing types, codecs, nullable intent and source authority. Existing single scalar-plus-null folds, nested List/typed Map paths, root collection aliases and typed HTTP/SSE consumers reuse the check. No new reference, wrapper or default role is admitted.

```json
{
  "type": "array",
  "items": {"type": "number", "format": "float", "nullable": true},
  "uniqueItems": true
}
```

## Mapped equality and finite values

Equality follows the actual mapped IEEE value. Float inputs `1` and `1.000000001` round to the same Float and collide; Double inputs `1` and `1.0000000000000001` likewise collide. Adjacent representable Float/Double values remain distinct. Positive and negative zero compare equally. At most one nullable null is allowed, distinct from either zero; null containers skip their own checks. Empty/singleton lists pass uniqueness unless count constraints exclude them.

Every non-null value participating in this check must be finite. NaN and positive/negative infinity fail, including singleton lists and when the caller Json explicitly permits special floating values. Ordinary unconstrained floating fields and standalone scalar codecs retain their separate behavior. No epsilon, decimal tolerance or nonfinite equality policy is introduced.

This deliberately uses mapped-value equality. JSON Schema may distinguish original numeric values that round to the same Kotlin float, so this is not exact original JSON-number uniqueness or general JSON Schema validation. The existing [direct JSON-value profile](json-unique.md) keeps its own exact equality contract.

A dedicated private helper widens already mapped Float values exactly to Double, verifies finiteness, sorts a copy and compares adjacent concrete Double values. Concrete floating equality handles signed zero; the existing generic Comparable helper for other leaves keeps its output and behavior. No caller list mutation, public wrapper, custom serializer or additional serialized state is introduced. Helpers are non-inline and emitted from typed consumer metadata, including transparent aliases and direct SSE collection payloads.

## Consumers and defaults

DTO constructors, Data-class copy, automatic decoding and Patch.Value enforce the retained checks. Typed HTTP arguments check before callback/engine dispatch, decoded results check before return, and known SSE payloads check at their event/DTO construction. Invalid or cancelled fragmented live SSE closes its input; cold collection and the caller client remain reusable. Alias assignment, independent reified collection serialization, later caller mutation and arbitrary requestBuilder body replacements keep their existing boundaries.

Native floating enums retain mapped membership, finite checks and already admitted numeric bounds alongside uniqueness. Their defaults remain strict members of the original source vocabulary before rounding; two distinct admitted vocabulary annotations can still collide after mapping. Floating enum `multipleOf` and broader aggregation retain their exclusions. Plain numeric leaf bounds/factors keep their own existing contracts.

Already admitted direct/nested/referenced collection defaults and component-own factories check the same mapped equality. Comparison keys round to Float or Double and normalize signed zero; existing emitted default formatting stays unchanged. Duplicate errors identify the second original `/default/index`, including nested paths. Literal grammar, representability, required/Patch suppression, full empty/null schema validation and original default budgets remain unchanged.

## Sources, identity and limits

Both an eligible actual scalar type and its original source are required. Native enum sources additionally retain actual membership metadata and their existing original vocabulary/export validation. Const/native overrides/value classes, typed container/DTO elements, lossy projections and general allOf/union intersections remain outside this profile. A caller-owned classifier spelled Float/Double gains no equality authority. `uniqueItems: false` remains effect-free with validation enabled; both Boolean values require the opt-in.

Input 8 MiB, source depth 48, checked collection edges 16, analysis work 16,384, raw enum/serialized slots 240 and existing default value/byte limits remain. The existing Unique flag plus actual scalar/check identity determine optimization/name-lock identity and helper selection; persisted formats do not change. These inputs previously rejected, while historical accepted artifacts and generic scalar/enum/temporal/JSON uniqueness retain their contracts. New consumers should regenerate and compile. This is a D05/M03/M07/M08/M12/V05 increment with existing A03/A05/A08 consumers; wider graph, structured equality and platform support remain open.

## Verification

The [fixture](../tests/fixtures/http/floating-unique.json), [core source/default/budget regressions](../core/floating_unique_test.go), [native old-output preservation](../internal/cli/floating_unique_process_test.go) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FloatingUniqueContracts.kt) define this profile. Historical direct/root/internal/ref/default floating uniqueness negatives remain explicit original-input opt-in positives; unrelated projections remain negative.

`bash scripts/test.sh feature floating-unique` builds fresh Data/Plain × validation projects. Plain also enables dedup/hoist/merge/name-lock. The unchecked scalar control removes checks/vocabularies and substitutes existing direct scalar references for checked-only aliases; the original constrained fixture still rejects without validation. Four Json default/null settings exercise rounding ties/adjacent values/extrema, signed zero, constructor and permissive-Json nonfinite rejection, defaults/factories, native membership/bounds, Patch, actual request JSON/preflight, decoded responses and fragmented live SSE for both DTO and direct collection payloads, failure/cancellation/closure/cold collection/client reuse. Checked profiles independently compile and construct a 240-field Float-list probe. The complete `http` route includes the checked consumer. Independent source48/checked16/width probes are not combined-maxima, Android/KMP or minimum-toolchain evidence; preparation-only output does not prove JVM execution.
