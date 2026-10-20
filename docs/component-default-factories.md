# Component default factories

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

Supported leaves may be nullable independently of the outer collection: a non-null `List<String?>` can carry `[null, "a"]`, and a nullable `Map<String, Int?>?` can carry either `{"a": null}` or `null`. A null annotation still validates the whole original schema. Only the outer schema authorizes a null collection; only the leaf schema authorizes null members. The existing scalar, temporal, enum and signed-byte leaf grammar and budgets apply. Finite nested List/typed Map trees also admit root factories under the [nested component contract](nested-component-defaults.md). Reference/composition elements and child defaults retain their exclusions. Eligible `$ref`-only wrapper factories also admit these collection targets under the [wrapper contract](reference-wrapper-component-defaults.md).

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

Explicitly nullable roots may own null or non-null values. Eligible finite named-component alias chains use nearest nullable intent and may independently own defaults, with diagnostics at each annotation's original `/default`. Null still validates the full source. Fields and unannotated aliases do not inherit defaults; required parameters retain their constructor presence. Unknown/mismatched formats, UUID, caller-native/value-class/union roles and other default contexts gain no admission. Source identity and existing depth/value/JSON budgets remain.

This additive D03/M01/M03/M04/M07/M12 profile accepts formerly rejected annotations. Existing valid Kotlin call sites, wire behavior and persisted locks require no migration. [Core acceptance/value guards](../core/numeric_component_defaults_test.go), [source ownership](../internal/schema/numeric_component_defaults_test.go), [native parity/failure preservation](../internal/cli/numeric_component_defaults_process_test.go), [fixture](../tests/fixtures/http/numeric-component-defaults.json) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NumericComponentDefaultsContracts.kt) cover mapped extrema, Float rounding/negative zero, nullable chains, constraints and non-inheritance. The validation-on Data/Plain projects in `feature reference-wrapper-component-defaults` and the full `http` registry run the consumer under all four Json default/null configurations.

## Admission

Admission reuses the existing default planner for property defaults; this contract adds no new shape grammar. A component is eligible when its own node carries `default` and the value validates through the same walk that the equivalent **property** default would use:

- plain inline ordinary object (explicit or inferred `type: "object"` with declared properties);
- disjoint ordinary `allOf` constructor roots;
- builtin scalar roots (`string`, `integer`, `number`, `boolean`, including the numeric formats above), standard temporal roots (`format: date` / `date-time`), plain unformatted String-enum roots, and numeric/Boolean native enum roots (`int32`/`int64`/`float`/`double` numeric vocabularies and Boolean vocabularies);
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
- An **unformatted numeric** enum root keeps its existing rejection; a numeric native vocabulary needs the explicit native `format`. The native enum's scalar wire type, exact pre-rounding vocabulary membership and the existing `W_ENUM_NATIVE_APPROXIMATION` warning are retained. A temporal root validates its lexeme through the existing temporal role, so a bad date or instant value is rejected at the annotation.
- Collection roots keep every exclusion the field role enforces: an `allOf`-wrapped container, a `$ref` element, a composition element and an out-of-vocabulary element all stay rejected. Nested inline List/typed Map elements follow the [finite tree contract](nested-component-defaults.md). Nullable roots follow the [nullable component contract](nullable-component-defaults.md). Counts and `uniqueItems` retain their existing opt-in behaviour.
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

A nullable root that also declares whole-object counts keeps its separate nullable-component-counts role. Null and non-null annotations follow the precise [nullable component root contract](nullable-component-defaults.md); other nullable compositions and excluded roles receive no additional permission.

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
