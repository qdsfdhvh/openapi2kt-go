# Component default factories

[Enum-reference collection component factories](referenced-enum-collection-factories.md) retain independent root/alias annotations and existing enum-component annotations through accepted finite List/typed Map paths. Actual serializers, strict original defaults, nearest nullable, required/Patch and original source/export/backing/check/height proof remain; no inheritance or child role is added. General graphs, complete platforms and final acceptance remain open.

[Scalar-reference collection factories](scalar-reference-collection-child-defaults.md) separately admit finite paths ending in eligible physical scalar references, including paths through existing named collections and aliases. Eligible physical scalar and named-collection reference children can also validate independent annotations under that contract; no annotation is inherited.


[Bounded lexical enums in object defaults](bounded-enum-object-defaults.md) extend D03/D05/M01/M03/M04/M07/M08/M12 to existing ordinary factory/field/DTO-collection walks. Exact direct enum sources retain String checks and strict original vocabularies; original plain DTO component/alias factories and consuming annotations remain independent. Required/Patch, actual serializers/optimization, original Pointer/owner and budgets remain. Generic graphs, other root/child/composition roles, platforms and final acceptance stay separate.

[Internal String enum alias factories](reference-wrapper-component-defaults.md#internal-string-enum-targets) extend D01/D03/F04/F07/M03/M04/M07/M08 to existing actual lexical property/collection/composition-owned paths and finite pure aliases. Each original factory, field and consumer annotation remains independent; nearest nullable, strict vocabulary, required/Patch, source/declaration/type/height proof and shared budgets remain. Generic graphs, platforms and final acceptance stay open.

[Model String enum metadata](metadata-string-enums.md) extends nullable roots/aliases, actual internal lexical references, existing independent scalar/collection/object default roles and serialName uniqueness. Original ownership, source/null-vocabulary boundaries, presence and budgets remain. Equivalent enum merges now retain each component-owned factory; generic graphs, platforms and final acceptance remain open.


A **component schema's own `default`** describes the value a server or client applies when the whole value is absent. Unlike a property default there is no call site: the generated class keeps its normal constructor, and the generator adds an explicit factory that decodes the annotation into the class.

```json
{
  "Simple": {
    "type": "object",
    "required": ["z"],
    "properties": {"z": {"type": "string"}},
    "default": {"z": "1"}
  }
}
```

```kotlin
@kotlinx.serialization.Serializable
public data class SimpleDto(
    @kotlinx.serialization.SerialName("z")
    public val z: kotlin.String,
) {
    public companion object {
        public fun default(): SimpleDto =
            `openapi-kt component default org example models SimpleDto`("{\"z\":\"1\"}")
    }
}

private fun `openapi-kt component default org example models SimpleDto`(source: kotlin.String): org.example.models.SimpleDto =
    kotlinx.serialization.json.Json.decodeFromString(kotlinx.serialization.serializer<org.example.models.SimpleDto>(), source)
```

A scalar or String-enum component owns its value outright, so its factory returns the alias type. A Kotlin `typealias` cannot own members, so that form is a top-level function:

```kotlin
public typealias NameDto = kotlin.String

// The OpenAPI component default for this schema.
public fun `NameDto default`(): NameDto = `openapi-kt component default ... NameDto`("\"x\"")
```

Enum components already emit a companion, so their factory uses the same top-level form as scalars for one uniform call shape.

A single-layer List/typed Map root also emits the top-level form, because its declaration is a `typealias`:

```kotlin
public typealias NamesDto = kotlin.collections.List<kotlin.String>

// The OpenAPI component default for this schema.
public fun `NamesDto default`(): NamesDto = `openapi-kt component default ... NamesDto`("[\"a\"]")
```

Supported leaves may be nullable independently of the outer collection: a non-null `List<String?>` can carry `[null, "a"]`, and a nullable `Map<String, Int?>?` can carry either `{"a": null}` or `null`. A null annotation still validates the whole original schema. Only the outer schema authorizes a null collection; only the leaf schema authorizes null members. The existing scalar, temporal, enum and signed-byte leaf grammar and budgets apply. Finite nested List/typed Map trees also admit root factories under the [nested component contract](nested-component-defaults.md). Reference elements and child annotations use their separate contracts linked above; composition elements retain their exclusions. Eligible `$ref`-only wrapper factories also admit these collection targets under the [wrapper contract](reference-wrapper-component-defaults.md).

A disjoint ordinary `allOf` constructor emits the companion form, mirroring the consuming-field allOf walk:

```kotlin
public data class AllOfDto(
    @kotlinx.serialization.SerialName("a") public val a: kotlin.String,
    @kotlinx.serialization.SerialName("b") public val b: kotlin.Int? = null,
) {
    public companion object {
        public fun default(): AllOfDto =
            `openapi-kt component default ... AllOfDto`("{\"a\":\"x\"}")
    }
}
```

## Why a factory, not a constructor initializer

A constructor initializer would silently change wire behavior. `required: ["z"]` with a component default means "the caller may omit the value and the producer supplies it", not "the Kotlin property is optional". Giving `z` an initializer would:

- remove it from the constructor's required argument list, breaking existing call sites;
- make `encodeDefaults = false` omit a value that the schema marks required;
- conflict with the established rule that a `required` property has no initializer.

The factory keeps the constructor, the public property list, the `required`/Patch classification and every emitted byte of the class body unchanged. Adopting the default is an explicit caller decision.

## Constrained plain-scalar roots

With validation enabled, an existing unformatted plain scalar component may combine its own `default` with supported primitive constraints. String roots retain minLength/maxLength/portable pattern; integer roots retain supported bounds and integer multipleOf; number roots retain supported inclusive/exclusive bounds. No new format or constraint grammar is introduced.

```json
{"Label": {"type": "string", "minLength": 2, "maxLength": 6, "pattern": "^[a-z]+$", "default": "given"},
 "Count": {"type": "integer", "minimum": 0, "maximum": 10, "multipleOf": 2, "default": 4}}
```

Factories retain the top-level alias form. The annotation is checked through the shared scalar value checker before emission; invalid values report `E_SCHEMA_INVALID` at the original component `/default`. Original JSON tokens are preserved, including decimal spelling. Portable pattern defaults retain the existing printable-ASCII matching boundary. Nullable roots may own null factories, but null does not skip malformed constraint/source validation. Non-nullable null remains invalid.

Each admitted component or alias on a finite named-reference chain may own a separate factory under the [wrapper contract](reference-wrapper-component-defaults.md). Nearest alias nullable intent applies, and every annotation satisfies its retained constraints. No field or unannotated alias inherits a default. Generated typealias types, required/optional constructor parameters, warnings, descriptors and wire behavior stay unchanged. Typed DTO consumers retain constructor/copy/decode checks; standalone scalar alias codecs remain unchecked.

Validation-off constrained roots, new formats, floating multipleOf, temporal/bytes/native/valueclass/enum/composition roles and collection/child defaults receive no new permission. Original source/depth/value/JSON budgets remain. Default admission is tied to an original component source node, so a copied or arbitrary bounded use does not gain permission.

This is an additive D03/M01/M07/M12 profile for formerly rejected constrained root annotations. Valid prior inputs require no migration. Bounded scalar alias factories without explicit nullable intent now enforce their retained constraints too: invalid annotations previously accepted by the legacy kind-only path fail at the alias's original `/default`. Malformed nullable scalar root constraints now report the actual constraint Pointer (for example `/minLength`), rather than being masked by the previous unsupported-default diagnostic. Valid output, Kotlin APIs and persisted lock formats are unchanged.

[Core acceptance/value/source/opt-in guards](../core/constrained_component_defaults_test.go), [source identity/permission](../internal/schema/constrained_component_defaults_test.go), [native parity/read-only/failure preservation](../internal/cli/constrained_component_defaults_process_test.go), [fixture](../tests/fixtures/http/constrained-component-defaults.json) and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ConstrainedComponentDefaultsContracts.kt) cover this profile. The validation-on Data/Plain projects in `feature reference-wrapper-component-defaults` and the full `http` registry execute its JSON and typed-consumer assertions.

## Numeric builtin formats

Numeric components may own defaults with `integer` formats `int32`/`int64` or `number` formats `float`/`double`. Factories return the existing `Int`/`Long`/`Float`/`Double` aliases through the same top-level `X default()` API. Unconstrained direct roots work with validation off or on; primitive bounds require validation. Named reference aliases keep their existing validation opt-in.

```json
{"Count": {"type": "integer", "format": "int64", "default": 9223372036854775807},
 "Ratio": {"type": "number", "format": "float", "default": 0.300000004}}
```

Every non-null annotation must fit its mapped Kotlin type, even without explicit bounds. Integers require integral JSON tokens in Int/Long range; decimal/exponent tokens and integer `-0` remain rejected. Float values must stay finite after mapping and use ordinary IEEE rounding; Double retains the input loader's finite-number boundary. The factory preserves the original JSON token and decodes through the actual alias serializer. A Float factory for the example returns `0.300000004.toFloat()`; this does not promise decimal-exact arithmetic. Existing retained bounds and integer multipleOf use the shared value checker, including mapped floating-bound comparisons. Floating multipleOf remains excluded.

Explicitly nullable roots may own null or non-null values. Eligible finite named-component alias chains use nearest nullable intent and may independently own defaults, with diagnostics at each annotation's original `/default`. Null still validates the full source. Fields and unannotated aliases do not inherit defaults; required parameters retain their constructor presence. Unknown/mismatched formats, caller-native/value-class/union roles and other default contexts gain no admission. UUID follows its separate profile below. Source identity and existing depth/value/JSON budgets remain.

This additive D03/M01/M03/M04/M07/M12 profile accepts formerly rejected annotations. Existing valid Kotlin call sites, wire behavior and persisted locks require no migration. [Core acceptance/value guards](../core/numeric_component_defaults_test.go), [source ownership](../internal/schema/numeric_component_defaults_test.go), [native parity/failure preservation](../internal/cli/numeric_component_defaults_process_test.go), [fixture](../tests/fixtures/http/numeric-component-defaults.json) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NumericComponentDefaultsContracts.kt) cover mapped extrema, Float rounding/negative zero, nullable chains, constraints and non-inheritance. The validation-on Data/Plain projects in `feature reference-wrapper-component-defaults` and the full `http` registry run the consumer under all four Json default/null configurations.

## UUID roots

A `type: string`, `format: uuid` component may own a UUID default factory, including nullable roots:

```json
{"Identifier": {"type": "string", "format": "uuid", "default": "550e8400-e29b-41d4-a716-446655440000"}}
```

The factory returns the `kotlin.uuid.Uuid` alias (or Uuid? for nullable roots), using the actual JSON serializer. Non-null annotations must parse as UUIDs. Compact and uppercase hexadecimal forms are accepted; returned values encode canonical lowercase, hyphenated text. Unformatted String factories continue to support arbitrary and escaped Unicode strings.

Unconstrained direct roots work with validation off or on and emit identical factory source. String length/pattern checks require validation and observe canonical UUID text. Eligible finite named-component alias factories also require validation. Each target/intermediate/outer default validates at its original `/default`; nearest nullable intent, independent ownership, non-inheritance and required presence remain. Null still validates the complete source. Formatted enum factories, native/value-class/union default ownership and other formats gain no permission. Original source-node, Pointer and default-budget limits remain.

The Uuid mapping intentionally changes earlier String APIs and raw root HTTP behavior; see [UUID migration](nullable-uuid.md#migration). D03/M01/M03/M04/M07/M12 evidence includes [core](../core/uuid_component_defaults_test.go), [source authority](../internal/schema/uuid_component_defaults_test.go), [native parity/failure preservation](../internal/cli/uuid_component_defaults_process_test.go) and [Kotlin contracts](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/UUIDComponentDefaultsContracts.kt). Validation-on Data/Plain projects in `feature reference-wrapper-component-defaults` and the full `http` registry execute factories and consumer checks under four Json configurations.

## Admission

Admission reuses the existing default planner for property defaults; this contract adds no new shape grammar. A component is eligible when its own node carries `default` and the value validates through the same walk that the equivalent **property** default would use:

- plain inline ordinary object (explicit or inferred `type: "object"` with declared properties);
- disjoint ordinary `allOf` constructor roots;
- builtin scalar roots (`string`, `integer`, `number`, `boolean`, including the numeric formats and UUID profile above), standard temporal roots (`format: date` / `date-time`), plain unformatted String-enum roots, and numeric/Boolean native enum roots (unformatted Int/Double, explicit `int32`/`int64`/`float`/`double` numeric vocabularies and Boolean vocabularies);
- a finite inline List/typed Map tree whose terminal is a nullable or non-nullable leaf the nested collection default walk already supports (builtin scalar, temporal, String enum, native enum, or [signed bytes](bytes-default-trees.md));
- nullable direct scalar/temporal/bytes/enum/finite inline collection roots with a null annotation under the [nullable component contract](nullable-component-defaults.md);
- disjoint ordinary `allOf` constructor;
- named/internal `$ref` collection trees already admitted by the referenced collection default roles;
- scalar, temporal, String enum, numeric/Boolean native enum, `bigint`, and primitive union leaves;
- raw `JsonElement`/`JsonObject` carriers in the roles that admit them.

`required` members of the component are required in the annotation as well. Every declared property is validated even when the annotation omits it. Unknown keys, wrong leaf kinds, out-of-vocabulary enum members, violated counts/bounds and non-nullable `null` all fail at the original annotation Pointer.

## Exclusions

- Reference wrappers use the separate [wrapper contract](reference-wrapper-component-defaults.md); the direct nullable-root role does not widen their null permission.
- Other scalar format roots, `x-native-type`, `x-kotlin-value-class`, `const`, `nullable`, `oneOf`/`anyOf` and `$ref` wrappers keep their own roles and their existing rejection at the original `.../default` Pointer.
- An **unformatted numeric** enum root uses the existing Int/Double mapping for its own factory; see [unformatted native factories](unformatted-native-factories.md). The native enum's scalar wire type, exact pre-rounding vocabulary membership and the existing `W_ENUM_NATIVE_APPROXIMATION` warning are retained. A temporal root validates its lexeme through the existing temporal role, so a bad date or instant value is rejected at the annotation.
- Direct inline collection roots keep the corresponding field-role exclusions, including `allOf`-wrapped containers, composition elements and out-of-vocabulary elements. Eligible reference elements have the separate bounded contracts linked above. Nested inline List/typed Map elements follow the [finite tree contract](nested-component-defaults.md). Nullable roots follow the [nullable component contract](nullable-component-defaults.md). Counts and `uniqueItems` retain their existing opt-in behaviour.
- Empty/raw object projections now have the separate [raw JSON carrier component contract](raw-json-component-defaults.md); const-only projections, hierarchy sources and object-union products keep their separate contracts.
- A component default is **not inherited**: a property referencing the component does not acquire it, and a consumer's own annotation is unchanged.
- Request/response inline schemas and operation parameters are unaffected.

## Nullability and presence

`default: null` requires an explicitly nullable component root and its admitted factory shape, matching the rule that optional presence alone never authorizes annotation null. Ordinary object and explicitly nullable disjoint ordinary `allOf` factories return null directly:

```kotlin
public companion object {
    public fun default(): MaybeRecDto? = null
}
```

Eligible scalar, temporal, bytes and finite inline collection aliases use their actual nullable serializer to decode the null literal. Plain String enums retain the physical classifier and receive a top-level `X default(): X? = null`; their non-null serializers and non-null factories remain unchanged. Native enum roots keep their existing validation-on source role and factory warning.

An ordinary constructor DTO root may combine its own default factory with admitted whole-object counts when validation is enabled; the counted factory contract below preserves its separate value and codec checks. Null and non-null annotations follow the precise [nullable component root contract](nullable-component-defaults.md); other nullable compositions and excluded roles receive no additional permission.

A nullable component still emits its normal nullable public type; the factory signature follows the component's own generated type, not the annotation.

## Diagnostics and budgets

Failures keep the original Pointer: a schema violation reports at `/components/schemas/<Name>/default/...`, matching the property-default member Pointer shape. The factory body is produced from the same canonical JSON, so it shares the 16,384 value-node and 32 KiB original/canonical budgets. Schema source depth 48, composition work 16,384 and the 240-slot limits are unchanged.

A component default never changes a type, a declaration, a serializer, a warning or another component's emitted bytes; only the factory member and its helper are added.

## Naming and collisions

The factory is named `default` in both forms: a companion member for object components and a top-level function for scalar, String-enum and other alias components (whose `typealias` cannot own members). The helper keeps the established dotted-path form (`openapi-kt component default <package path> <ClassName>`) so two components with the same short name in different packages cannot collide, and it is emitted once per file even when hoisting or deduplication shares one declaration across consumers.

A property may itself be named `default`; Kotlin resolves the property access and the companion function independently, so both may coexist.

## Compatibility

This is an additive role for previously rejected input. Components without a `default` retain every emitted byte. Adding the annotation never changes the constructor, `copy`, the serializer or the wire behavior of the component itself; it only adds an opt-in factory.

## Verification

- [Core contract tests](../core/component_default_factories_test.go) cover Data/Plain × validation off/on, adoption identity, required/Patch preservation and original diagnostics.
- [Source authority tests](../internal/schema/component_default_factories_test.go) cover the eligible shapes, exclusion list and budget behaviour on original nodes.
- [Native parity/preservation](../internal/cli/component_default_factories_process_test.go) checks native output, readonly `--check` and unchanged output on failure.
- The nested-object feature fixture and Kotlin consumer call generated factories for an object, a scalar and a String-enum component, proving each form compiles and decodes to the same value as an explicitly constructed instance.

## Counted DTO component factories

An ordinary direct or disjoint `allOf` constructor component may declare its own `default` with admitted whole-object property counts, including explicitly nullable roots. Validation must be enabled. Direct bounds and original referenced/nested allOf contributions are validated and folded by the same model count planner. A non-null annotation must contain a permitted number of JSON members as well as satisfying the existing complete object-default grammar, required members, leaf checks and budgets. Its `default()` factory decodes that canonical JSON with the component's actual serializer. A schema-authorized null annotation returns Kotlin null directly and skips value counts, while still validating the entire source shape and every bound.

Permission is scoped to this one original component's own default walk. It does not transfer through copied views, wrapper factories, consuming-field defaults, child annotations or a coincident marked target. Raw carriers/dictionaries, empty or const-only models, discarded contributions and other unaccepted object-default roles keep their existing rules. The annotation is opt-in through `default()` and is never inherited by fields or aliases. A factory value can later fail re-encoding if caller Json settings change its physical member count; constructors/copy are not wire-count barriers.

[Core regressions](../core/counted_component_defaults_test.go) preserve source-role and invalid-count rejection; the existing [counted nullable DTO consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/DirectNullableComponentCountsContracts.kt) exercises null and composed object factories with fresh generated Kotlin. Existing accepted API/wire output and output metadata formats remain unchanged.

## Tuple reference consumers

[Existing tuple references](tuple-target-defaults.md) can consume an independently eligible component or named-alias factory source without inheriting its annotation. The component/alias keeps its own factory, while missing required List-element fields and required HTTP inputs remain required. Every factory still validates its original literal and budgets before raw/tail projection. The shared borrowed String-enum validator likewise preserves component-own factory permission for ordinary collection refs. This introduces no new factory shape or internal/positional annotation role.

[Ordinary property-own String enum annotations](object-enum-own-defaults.md) are preserved within eligible object/allOf factories and independent aliases. Structured alias validation borrows original source nodes/Pointers and applies nearest nullable intent only to the root plan; values retain their own annotation Pointer. Scalar/raw policies and persisted formats remain unchanged.

[Native enum property-own annotations](object-native-enum-own-defaults.md) are retained in ordinary object/allOf factories and independent aliases. The original typed field supplies missing-value decoding defaults; root factory values never transfer to another owner.

### Repeated String contributions in outer defaults

[Outer allOf defaults with retained repeated plain String checks](allof-string-defaults.md) reuse complete original ordinary primitive groups under validation. Source-own defaults validate before override, the winning annotation satisfies retained checks, supplied outer members satisfy the same conjunction, and omitted members use the actual constructor default. Original source/Pointer/final codec, required/Patch, work/value/width limits and independent default roles remain. Numeric/ref/enum/unchecked duplicate aggregation and broad D03/D05/platform completion stay separate.

### Ordinary DTO collection root factories

[DTO List/typed Map component factories](object-collection-factories.md) reuse existing ordinary object/allOf walks on finite inline container paths, including direct named plain DTO elements. Actual root alias/serializer, original source Pointers, field defaults, required/Patch, nullable layers, independent wrapper annotations and no inheritance remain. Probe permissions are isolated and factories cannot silently drop eligible annotations. Existing source/value/width caps and wider D03/platform/source-role boundaries remain.

[Outer allOf defaults with retained numeric checks](allof-numeric-defaults.md) extend D03/D05 under validation for complete original plain Int/Long/Float/Double groups. Selected original endpoints/factors stay in a bounded literal-only plan; exact integer and existing mapped floating semantics, own/winning/outer annotations, actual constructor defaults, required/Patch, original Pointers and source/value/slot caps remain. Formerly rejected eligible annotations now generate; accepted API/wire/state formats retain their contracts. Wider default/composition/platform completion remains separate.

[Scalar tuple component factories](tuple-root-defaults.md) extend D03/D06 for original direct scalar prefix/tail List sources and independent component aliases. Original nodes/Pointers, selected codec/nullability, strict annotations/temporal checks, no inheritance, required/Patch and source/value/slot budgets remain. Positional/length approximation and excluded nominal/raw/member-own roles retain their contracts. Formerly rejected eligible root annotations gain factories; accepted API/wire/state formats remain stable, and wider/default/platform work stays open.

[Outer allOf UUID defaults](allof-uuid-defaults.md) extend D03/D05 for complete original ordinary string/uuid contribution groups under validation. Retained length/pattern checks inspect canonical Uuid text, while original own/winning/outer annotations, actual constructor/codec, required/Patch, source Pointers and shared budgets remain. Invalid values fail at original annotations; no schema copying, new codec or persisted identity is introduced. Wider references/enum/formats/defaults/aggregation/platforms stay separate.

The [JsonNull default extension](null-value-defaults.md) adds original direct plain/null-only enum members to admitted ordinary object walks and direct JsonNull leaves to finite inline List/typed Map annotations, including admitted collection component/wrapper factories. Its source, nullability, role and budget restrictions apply independently.

[JsonNull scalar component factories](null-component-factories.md) separately admit nonnullable plain/null-only enum component own default:null and eligible independent finite component aliases. Actual JsonNull singleton returns, marked original source/vocabulary authority, strict annotation Pointers and required/Patch/no inheritance remain; nullable/const/format/general graph roles stay separate.

[Own defaults on named false collections](false-collection-defaults.md) extend D03/E03 for finite original named List/typed Map trees. Actual root-owned factories validate empty/nested/nullable-leaf literals through the actual nominal codec, retain original literal nodes even under metadata-named JSON keys, and bind actual factory/canonical initializer/type/source/cached-height identity. Source48/work16384/value16384/32KiB/checked16/JVM240 budgets and no inheritance remain. Consuming field/child/wrapper/object-member/composed/internal and broader default/platform/final acceptance remain separate.

[Nullable false alias own null factories](false-alias-defaults.md) extend D03/E03 with original pure named aliases whose nearest nullable intent is true. Each own annotation validates independently and binds the actual nullable-root codec, canonical null factory, original owner/reference/nullable/value nodes and actual declarations. Existing collection factories may consume independently annotated aliases without inheritance. Source48/work16384/value/bytes/checked16/JVM240 budgets, field presence, API/wire and persisted state remain. Consuming field/child/object-member/default and direct false HTTP/composed/internal/general/platform roles remain separate.

[False-reference members in object root defaults](false-object-default-members.md) extend D03/E03 for actual ordinary named Shared root-owned annotations. Original logical nullable members admit null, optional non-null false members admit only absence, and literal required keys remain mandatory. Unique actual getter/source/type/required/codec and model factory/canonical initializer/literal-node authority, source48/work16384/value16384/32KiB/checked16/JVM240 and existing wire-count/JSON policies remain. Nested/child/wrapper/composed/internal/directional/inline and general/platform/final acceptance remain separate; direct consuming-field defaults have the separate profile below.

Direct ordinary named shared DTO-reference fields may own object defaults using actual retained false-reference target getters, with no component default on consumer or target; required annotations validate before initializer suppression, and optional annotations decode fresh target state. Pure target aliases keep this field behavior. Source/value/depth/work limits and strict false codecs remain; nested/container/composed/directional/inline/internal roles and general defaults stay separate. See [DTO field defaults with false members](false-dto-field-defaults.md). Validation-on [target property counts](false-dto-field-counts.md) also validate original supplied keys against the actual retained target serializer; caller JSON omission and runtime wire counts remain. [Independent target root and consuming field annotations](false-dto-independent-defaults.md) also validate separately through the actual target factory and field initializer; no annotation is inherited.

[Own defaults on inline false collection fields](false-inline-collection-defaults.md) additionally validate actual ordinary named Shared outer field annotations, retaining canonical literal/initializer identity, required suppression, fresh decoding and original budgets. Empty containers and logically nullable false leaves are admitted; no owner-root/child/wrapper/composed/internal/directional inheritance is added.

[Bounded lexical enum List/Map roots](bounded-enum-collection-roots.md) retain independent component and pure-alias factories alongside consuming field/object annotations. Their actual typed consumers check serialName; standalone alias codecs retain their existing open policy.
