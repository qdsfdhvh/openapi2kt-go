# Special scalar subset

[Own-nullable empty-model raw aliases](model-semantics.md#root-nullable-raw-aliases) are a separate JsonElement projection, not a caller-native marker or special scalar. They retain all JSON kinds; nullable codecs collapse JSON null to Kotlin null, whereas a narrowed non-null JsonElement can carry JsonNull. No custom codec or DTO API is introduced.

The shared model/HTTP resolver supports these direct scalar mappings:

| Schema | Kotlin type | Model JSON |
|---|---|---|
| `string`, `format: date-time` | `kotlin.time.Instant` | ISO timestamp string |
| `string`, `format: date` | `kotlinx.datetime.LocalDate` | ISO date string |
| `string`, `format: byte` or `binary` | `ByteArray` | signed numeric array, **not Base64** |
| scalar `x-native-type: date` | `kotlin.time.Instant` | ISO timestamp string |
| scalar `x-native-type: bigint` | `Long` | context-dependent, below |
| `type: null` or `x-void` in a property/item | `JsonNull` | JSON null singleton |

Add `kotlinx-datetime` when compiling LocalDate models. The executable JVM sample uses version 0.7.1 with Kotlin 2.4.10 and serialization 1.11.0. These checks do not establish Android/KMP or minimum-version compatibility.

## bigint use sites and presence

A direct model field marked bigint, including a reference to a bigint component, uses a generated `LongAsStringSerializer`: decimal strings on encoding and `decodeString().toLong()` on decoding. The caller's decoder remains in control; this is not a strict JSON-string-kind validator. Values are limited to signed Long, not arbitrary precision.

For an optional nullable request field, the serializer annotates the **Long type argument of Patch**, not Patch itself. Absent remains omitted, Value encodes a decimal string, and Null remains JSON null regardless of `encodeDefaults`/`explicitNulls`.

A scalar component alias, simple HTTP root, or bigint directly inside a list/map uses ordinary Long serialization (JSON numbers). Objects inside collections retain their direct-field serializer. Only included emitted declarations needing a field override own the generated helper; scalar aliases or un-emitted temporary validation plans do not. Its model filename is reserved when needed.

[Finite internal chains to named temporals](temporal-reference-chains.md) separately preserve the original supported date/date-time/native-date scalar or named alias through typed chains and root bridges. Instant/LocalDate backing, provider codecs, ISO wire, parse APIs and nearest nullable/false intent remain unchanged; every erased transparent alias hop stays in the source-depth budget. Other special/native profiles and shape borrowing are not added.

[Finite internal chains to named bigint scalars](bigint-reference-chains.md) preserve the original bigint marker/actual source declaration and native Long backing through typed internal/named paths and root bridges. Field/Patch string codecs stay separate from numeric root/List/Map/SSE codecs; nearest path-local nullable/false and complete cached hops remain. Other native/byte/JSON or borrowed special leaf profiles are separate.

[Finite internal chains to named byte/binary scalars](bytes-reference-chains.md) independently preserve original format/source identity and actual transparent declaration through typed internal/named paths/root bridges. ByteArray JSON signed arrays and raw Ktor roots (including null ASCII) remain separate; caller-native ByteArray and borrowed special leaves stay distinct.

[Finite internal chains to named caller-native scalars](caller-native-reference-chains.md) preserve original native-string/source and actual generator alias identity through typed internal/named paths/root bridges. The real provider owns class/API/codecs; caller Long remains numeric, String/ByteArray roots raw, JSON/Instant nullable/provider codecs unchanged. Borrowed/structured native and raw/true/cover roles remain separate.

## HTTP delegation

Ktor bypasses ContentNegotiation for a root ByteArray request/response: the bytes travel raw even with `application/json`. ByteArray **inside a model** uses the builtin numeric-array JSON serializer. A simple root String still travels as raw text. Do not infer body encoding solely from a schema format or generated type name.

Instant query values delegate to Ktor stringification. Null/void success response bodies retain the independent Unit projection; JsonNull model fields are not Unit.

## Enum and const caller contexts

An object property, list item or component string enum with a supported special format remains an **open string vocabulary**. Its known/unknown `serialName` values are not parsed as dates or bytes. A direct simple HTTP enum instead uses the format's scalar mapper: LocalDate/Instant serialize ISO strings and ByteArray still delegates raw bytes to Ktor. Enum membership is not enforced in either context.

A string or untyped enum may also carry a valid `x-native-type`. Component, field, item, dictionary-value and parameter declarations still use the open string vocabulary, including unknown nonnumeric/non-temporal values. A simple HTTP body uses the native mapper instead (`date` → Instant, `bigint` → Long, or the caller-owned classifier). A bigint marker on a vocabulary does **not** attach LongAsStringSerializer, including inside Patch; the vocabulary's own string serializer remains in control. Native metadata is validated even where its type mapping is unused.

Date/Instant const body defaults use typed `.parse(...)` initializers, not ill-typed String assignments. Accepted defaults are valid four-digit-year ISO dates or RFC3339 timestamps. They always encode; missing keys use the default, while decoding another valid temporal value retains that value. This is a corrective compilation contract, not const-membership validation. A native-bigint const body encodes an ordinary Long number: const generation exits before the direct-field string-serializer rule. ByteArray/custom-class consts remain rejected because their String literals do not provide a defensible typed default.

## Caller-owned types and boundaries

A scalar or type-omitted `x-native-type` may name a safe dot-separated ASCII Kotlin classifier. The caller supplies its real declaration and serializer; the generator does not create a replacement DTO, install dependencies, or infer its wire representation. Generic syntax, code fragments and keywords are rejected.

This subset does not yet support structured custom-native schemas, numeric/boolean native-type enum combinations, ByteArray/custom const defaults, nullable scalar component roots or general normalization. Pure named root bindings to accepted non-null special/native targets now have the separate [root special alias contract](model-semantics.md#root-special-aliases); they preserve the original field-versus-root wire rules and caller-owned classifier/serializer. Own-nullable root bindings to accepted non-null date/date-time/date-native targets have the independent [temporal alias contract](model-semantics.md#root-nullable-temporal-aliases): canonical Instant/LocalDate parse and ISO codecs remain, nearest nullable/false intent affects only typed roots. Intrinsic nullable temporal components have the independent [transparent nullable-root contract](model-semantics.md#intrinsic-nullable-temporal-components), with public nullable alias/reified codec and false native backing; [Own-nullable bigint root aliases](model-semantics.md#root-nullable-bigint-aliases) independently preserve DTO/Patch decimal-string versus root/list/map JSON-number codecs with nearest root intent and nullable reified Long adapters. Intrinsic nullable bigint components separately have the [transparent nullable Long-root contract](model-semantics.md#intrinsic-nullable-bigint-components), inheriting/narrowing intent without changing that field-versus-root codec. [Own-nullable byte/binary aliases](model-semantics.md#root-nullable-bytebinary-aliases) retain signed-array model/Patch/SSE codecs versus raw Ktor roots. A null root request sends ASCII `null`; a raw response containing `null` returns those four bytes, **not Kotlin null**. This preserves existing inline nullable-byte behavior, not a nullable-JSON HTTP round-trip contract. [Intrinsic nullable byte/binary components](model-semantics.md#intrinsic-nullable-bytebinary-components) separately export transparent ByteArray? roots/reified nullable codecs and inherited/false intent, retaining those JSON-versus-raw HTTP rules. [Own-nullable caller-native root aliases](model-semantics.md#root-nullable-caller-native-aliases) independently retain provider-owned classifiers/serializers and nearest nullable/false intent for accepted non-null scalar terminals. A caller Long marker does not attach the bigint field helper; caller String/ByteArray roots keep raw Ktor projection, so a `null` response is raw text/bytes, not Kotlin null. Constructor/companion APIs depend on the provided class. [Intrinsic nullable caller-native components](model-semantics.md#intrinsic-nullable-caller-native-components) separately export transparent nullable provider roots and inherit/narrow typed intent without changing provider API/wire or inferring physical layouts. Native enum/const and structured custom-native profiles are not added. Unknown keywords remain errors. Existing mixed primitive unions retain their established kind carriers: a special-format string branch does not retarget the accepted String payload API.

**Migration:** these direct inputs were previously rejected. Seven original negative corpus cases now have explicit positive core/native-CLI artifact expectations, while their historical manifest, JSON inputs and 50 goldens remain immutable. No accepted scalar API or wire mapping is silently rewritten. Native string enums were previously rejected; suppressing a Long serializer on their vocabulary type is a corrective compilation contract, not a claim that the legacy emitter compiled that combination.

[Go fixture contracts](../core/http_special_scalar_test.go), [enum/const contexts](../core/special_scalar_values_test.go), [native enum contexts](../core/native_enum_context_test.go), [guards](../core/special_scalar_guards_test.go), and [fresh CLI-generated Kotlin/Ktor assertions](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/SpecialScalarContracts.kt) cover JSON configuration combinations, extrema, malformed wire values, nullable/Patch/nested fields, numeric containers and raw ByteArray bodies. These establish the bounded subset, not complete special-scalar or HTTP parity.
