# Direct true JSON value schemas

This initial **F08/Q05** profile supports OpenAPI3.1 literal `true` in ordinary DTO property slots and typed array items, including array items inside typed dictionaries and simple operation containers. The schema means an unconstrained JSON value; it maps to `kotlinx.serialization.json.JsonElement`, not a fabricated DTO or wrapper.

Original literal nodes remain in the document. Each projected source emits `W_BOOLEAN_SCHEMA_PROJECTION` at its original Pointer, once. The existing warning policy can reject it with `--deny-warnings` or permit it with `--allow-warning W_BOOLEAN_SCHEMA_PROJECTION`. Failure returns no partial files/state/warnings and leaves old output intact.

## JSON null and presence

A required true field has a **non-null Kotlin JsonElement** parameter. Literal JSON null is the **JsonNull value**, not a Kotlin null parameter. Its generated codec accepts all JSON kinds, including null, Boolean, number, String, array and object. Missing required fields still fail.

Optional fields use the existing unconstrained `{}` presence profile: nullable/default-null Kotlin fields, not implicit Patch. A schema `true` by itself does not invent explicit nullable intent. An optional JsonNull value encodes JSON null, but decoding it through a nullable JsonElement codec collapses it to Kotlin null. Missing and explicit null are therefore not distinct optional field states; `encodeDefaults`/`explicitNulls` keep their existing policy. Inline request DTOs follow the same rule. Explicitly nullable/Patch profiles elsewhere are unchanged.

Native list construction/reified JSON codecs remain available. True array items accept JsonNull as a value without adding a nominal validation barrier. Typed container counts retain the existing opt-in rules; standalone JSON/list codecs do not enforce those counts. Source depth48, checked edges16, constructor/method slots240 and other budgets are unchanged.

## Limits and migration

- Boolean `false` value slots remain rejected with their original diagnostic/Pointer. They are never deleted or silently generated as object/JsonElement.
- Named true component roots have a separate [named-root contract](boolean-roots.md). Direct true JSON body/success-response roots have a separate [HTTP contract](http-client.md#direct-true-json-roots); internal Boolean reference targets, Boolean union branches and discarded composition fields remain outside these profiles. This does not implement general Boolean-schema composition or strict JSON Schema validation.
- OpenAPI3.0 Boolean value slots remain rejected. Existing true/false `additionalProperties` behavior in both versions is unchanged.
- Previously rejected admitted true fields/items now generate with a source warning. Existing accepted non-Boolean value schemas retain their APIs/wire. The original `property-boolean` corpus input explicitly graduates through full-artifact/warning-Pointer assertions in core and the native CLI; its input, inventory and reviewed snapshots are retained.

## Verification surfaces

[Whole synthetic input](../tests/fixtures/http/boolean-values.json), [core contracts](../core/boolean_value_test.go), [source/budget guards](../core/boolean_value_guards_test.go), [explicit corpus graduation](../internal/testutil/boolean_value_graduation.go) and [native policy/no-op/failure preservation](../internal/cli/boolean_value_process_test.go) cover the initial profile.

[BooleanValueContracts.kt](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/BooleanValueContracts.kt) checks four Json configurations, all JSON kinds/required JsonNull/optional collapse, controlled request/response bodies and fragmented still-open SSE failure closure/reuse/coldness. `bash scripts/test.sh feature boolean-value` generates fresh Data/Plain × validation off/on projects; Data projects additionally check `copy`. The full HTTP smoke also invokes the consumer.

Independent Go depth/edge/slot boundaries and fresh Kotlin profiles do not establish combined maximum-resource, minimum-toolchain, Android/KMP, all-platform or full first-version support.
