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

## Admission

Admission reuses the existing default planner for property defaults; this contract adds no new shape grammar. A component is eligible when its own node carries `default` and the value validates through the same walk that the equivalent **property** default would use:

- plain inline ordinary object (explicit or inferred `type: "object"` with declared properties);
- disjoint ordinary `allOf` constructor roots;
- plain scalar roots (`string`, `integer`, `number`, `boolean`), standard temporal roots (`format: date` / `date-time`), plain unformatted String-enum roots, and numeric/Boolean native enum roots (`int32`/`int64`/`float`/`double` numeric vocabularies and Boolean vocabularies);
- a single-layer List or typed Map root whose element/value is a leaf the nested collection default walk already supports (builtin scalar, temporal, String enum, native enum);
- a nullable ordinary object root whose annotation is the `null` literal;
- disjoint ordinary `allOf` constructor;
- named/internal `$ref` collection trees already admitted by the referenced collection default roles;
- scalar, temporal, String enum, numeric/Boolean native enum, `bigint`, and primitive union leaves;
- raw `JsonElement`/`JsonObject` carriers in the roles that admit them.

`required` members of the component are required in the annotation as well. Every declared property is validated even when the annotation omits it. Unknown keys, wrong leaf kinds, out-of-vocabulary enum members, violated counts/bounds and non-nullable `null` all fail at the original annotation Pointer.

## Exclusions

- A bare reference wrapper (a component whose body is only `$ref` plus `nullable`) has no value of its own and stays rejected.
- Scalar roots with a non-temporal `format` (bytes), `x-native-type`, `x-kotlin-value-class`, `const`, `nullable`, `oneOf`/`anyOf` and `$ref` wrappers keep their own roles and their existing rejection at the original `.../default` Pointer.
- An **unformatted numeric** enum root keeps its existing rejection; a numeric native vocabulary needs the explicit native `format`. The native enum's scalar wire type, exact pre-rounding vocabulary membership and the existing `W_ENUM_NATIVE_APPROXIMATION` warning are retained. A temporal root validates its lexeme through the existing temporal role, so a bad date or instant value is rejected at the annotation.
- Collection roots keep every exclusion the field role enforces: a nested container element, an `allOf`-wrapped container, a `$ref` element, a nullable root, a composition element and an out-of-vocabulary element all stay rejected. Counts and `uniqueItems` retain their existing opt-in behaviour.
- Empty/raw and const-only object projections, hierarchy sources and object-union products keep their separate contracts.
- A component default is **not inherited**: a property referencing the component does not acquire it, and a consumer's own annotation is unchanged.
- Request/response inline schemas and operation parameters are unaffected.

## Nullability and presence

`default: null` is admitted only when the component root is explicitly nullable, matching the existing rule that optional presence alone never authorizes annotation null. The factory then returns the nullable type directly and needs no decode helper, because the non-null component serializer cannot decode `null`:

```kotlin
public companion object {
    public fun default(): MaybeRecDto? = null
}
```

A nullable root that also declares whole-object counts keeps its separate nullable-component-counts role, and a non-null annotation on a nullable root keeps its existing rejection.

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
