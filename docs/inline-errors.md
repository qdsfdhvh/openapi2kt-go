# Inline JSON error DTOs

Selected JSON error responses retain the inline shared-model types already accepted by the model planner. Generated declarations live in `<package>.api.models`. Direct ordinary objects generate DTOs; ordinary allOf, object/primitive unions and plain String enums retain their existing model representations. Typed List/Map roots retain actual item/value declarations; scalar and raw JSON roots use their existing Kotlin carrier without a synthetic wrapper. Names are `<Operation>Error<Status><Suffix>`: for example `FetchError400Dto`, `FetchError4XXDto`, `FetchError5XXDto` and `FetchErrorDefaultDto`. Existing component `$ref` errors retain their component type.

The DTO uses shared-model field semantics, including supported defaults, enum vocabularies, nullable roots, nested objects and opt-in validation. Required arguments remain required. Borrowed HTTP response objects keep the original schema Pointer; each operation binds its own generated name. Fallback operation renaming replans error model identities alongside request/response/parameter types.

Only the preferred JSON media creates the error DTO for that status. All other declared media schemas are still validated before generation. API exclusion removes its local error declarations and removes `ErrorBody.kt` when no included operation needs it. Malformed unused media and excluded operations still fail explicitly with no output commit.

KDoc retains numeric/range/default ordering and now links the generated DTO type. The existing `ResponseException.errorBody<T>()` is unchanged: callers choose the type and configure HTTP validation, for example `expectSuccess = true`. The generator does not dispatch status types, throw merely because a status is declared, install plugins or replace the success return type. `default` remains an error declaration, never a success candidate. Ordinary decoding failures return null; cancellation is rethrown with its identity. Ktor saved responses may be decoded again.

```kotlin
try {
    api.fetch()
} catch (error: ResponseException) {
    val problem = error.errorBody<FetchError400Dto>()
    // Use problem?.message according to the application's error policy.
}
```

The new DTOs participate in ordinary operation deduplication, hoisting and identical-model merging. Error status roles have separate naming-lock identities; qualified/combined options preserve the actual relocated type in KDoc. The [options fixture](../tests/fixtures/http/inline-error-options.json) covers shared error roots. This does not extend optimization to arbitrary union/hierarchy/error projections.

## Migration and limits

Previously accepted inline error objects now add DTO files and typed KDoc links. The existing helper and success call signatures/wire behavior remain unchanged. If a newly generated name collides with another local declaration, generation fails explicitly and preserves existing output. No automatic filename override occurs. The historical [errors fixture](../tests/fixtures/http/errors.json) is preserved; its 409 object now adds `FetchError409Dto` and the expected file count changes from five to six.

The error role adds no new schema grammar. Accepted allOf/union/enum/container/const/Boolean/null/temporal roots reuse shared model mappings, nullable policy, approximation warnings and codecs. Structured const remains a kind projection, true remains JsonElement including JsonNull, and primitive unions retain their existing JSON-kind dispatch. Unsupported roots still reject. Non-JSON errors keep their previous validated projection. Broader HTTP request/response composition and automatic status dispatch remain separate. Source depth 48, constructor slots 240 and existing reference/work limits remain unchanged.

The [synthetic envelope](../tests/fixtures/http/inline-errors.json), original error regressions, Go source/media/optimization/limits and native preservation tests, and `bash scripts/test.sh feature inline-errors` cover the feature. Fresh Data/Plain × validation consumers run four Json configurations with typed numeric/range/default/borrowed errors, nullable bodies, defaults and required fields, malformed/missing payloads, repeated decoding, cancellation identity and caller-owned validation. These JVM checks do not establish Android/KMP or broader HTTP error dispatch support.

## Shared-model error roots

`bash scripts/test.sh feature inline-error-roots` freshly generates Data/Plain × validation off/on, an optimized/name-locked package and an independent 240-field allOf error constructor. The [fixture](../tests/fixtures/http/inline-error-roots.json) and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/InlineErrorRootsContracts.kt) exercise allOf required/default fields, object-union known/Unknown payloads, open enum unknowns, typed List/Map DTO leaves, primitive unions, structured const/raw true, nullable objects, temporal values and null-only bodies. Four Json settings, repeated `errorBody<T>()`, malformed/required/checked failures, cancellation identity and subsequent successful client calls are checked. `http` also invokes the consumer. Scalar aliases and raw JSON carriers acquire no new standalone validation barrier.

Every original preferred/unselected media schema still validates, including excluded operations. Actual item/value classifiers are qualified in KDoc; no root wrapper is manufactured for a List/Map. Sources retain their original Pointers and shared depth/work/constructor limits. Ordinary optimization eligibility is unchanged; this does not claim arbitrary union/hierarchy structural optimization.

**Migration:** previously accepted inline errors may now add shared-model declarations and typed KDoc. Success signatures, wire behavior, caller-owned HTTP validation and persisted lock formats are unchanged. Newly exposed names may reveal a collision, which fails explicitly and preserves existing output. Existing ordinary error DTO names remain stable. General D07 and platform/final-candidate gates remain separate.
