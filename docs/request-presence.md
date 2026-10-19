# Request-only presence and Patch

[直接嵌套 native enum 集合字段默认值](nested-native-enum-collection-defaults.md)完整验证原树、词表、checks 与预算后，required（含 nullable）仍无 initializer，ordinary inline request 的 optional explicitly nullable 字段仍为 Patch.Absent。逐层 nullable 仅授权该节点的注解 null，不创建 entry-level Patch；原词表 null 成员仍 validation-only。missing/default、explicit null 和 replacement 值保留既有 wire。

[Referenced temporal collection field own defaults](referenced-temporal-collection-field-defaults.md) validate the full original annotation before retaining required arguments and Patch.Absent. [Referenced String enum collection field own defaults](referenced-string-enum-collection-defaults.md) share the same presence rules with open runtime vocabulary; [referenced native enum collection field own defaults](referenced-native-enum-collection-defaults.md) add retained numeric bounds under validation. Actual per-layer resolver nullability authorizes null; optional presence does not. Targets and intermediate defaults are never inherited.

[Referenced standard temporal scalar field own defaults](referenced-temporal-scalar-field-defaults.md) validate original own annotations before retaining required arguments and Patch.Absent. Effective resolver nullability authorizes explicit null; target/chain defaults are never inherited, and temporal leaf/container roles remain separate.

[Referenced scalar field own defaults](referenced-scalar-field-defaults.md) validate the original annotation against already admitted actual scalar/check/nullable views before retaining required arguments or Patch.Absent. Only optional ordinary fields acquire literal defaults; no target/chain default is inherited.

[Direct nested temporal defaults](nested-temporal-collection-defaults.md) validate the entire original field annotation before retaining required arguments or Patch.Absent. Inner container/leaf nullability remains independent and does not create entry-level Patch; missing/default, explicit null and replacement values use the existing codecs.

[Referenced builtin collection field defaults](referenced-collection-field-defaults.md) apply the same required/Patch rules after validating the field's own annotation against the original linked type/checks. Target defaults are not inherited; nearest nullable intent remains independent at every layer.

[Finite nested builtin collection defaults](nested-builtin-collection-defaults.md) validate the entire direct field annotation before retaining required arguments or Patch.Absent. Only optional ordinary fields acquire typed initializers; inner container/leaf nullability never creates entry-level Patch.

[Direct nested String enum collection defaults](nested-string-enum-collection-defaults.md) validate the full original tree and vocabulary before required/Patch initializer suppression or allOf override. Explicit nullable intent at each container/leaf authorizes only that node's annotation null; optional presence alone does not. Ordinary missing fields use fresh typed defaults, while request Patch.Absent, Null and Value retain their states and runtime String enums remain open.

[Scalar DTO defaults](model-defaults.md) let optional non-null schema fields use validated literal initializers. Required fields with a default retain required arguments; optional explicitly nullable plain request fields retain Patch.Absent. These are explicit exceptions/additions to the no-schema-default table below.

[Own-nullable raw aliases](model-semantics.md#root-nullable-raw-aliases) share inherited root intent/consumer overrides, shared default-null, required nullable request fields without defaults and optional nullable Patch. False-narrowed required JsonElement can carry JsonNull; nullable decoding collapses it to Kotlin null, and Patch.Value(JsonNull) decodes as Patch.Null. This is not carrier-identity preservation.

[Intrinsic nullable caller-native components](model-semantics.md#intrinsic-nullable-caller-native-components) independently apply the same direct/wrapper/alias inherited-false root intent, shared default-null, required nullable no-default and optional nullable Patch rules. Public roots are nullable provider aliases; classifier/codec ownership and raw String/ByteArray HTTP null projection remain with existing providers/Ktor.

[Own-nullable caller-native aliases](model-semantics.md#root-nullable-caller-native-aliases) inherit provider-owned root intent through direct/wrapper/alias fields: shared nullable fields default null, required nullable request fields have no default, optional nullable fields use Patch. Explicit false narrows the provided classifier. Provider codecs remain authoritative; raw String/ByteArray HTTP responses containing `null` are text/bytes, not Kotlin null.

Plain inline JSON request objects use a request-only model context. This includes inline objects nested through properties, list items and dictionary values. Referenced components keep shared-model behavior unless an ordinary HTTP body needs the [directional DTO profile](directional-models.md): readOnly selects an operation-owned Request view, and writeOnly is omitted from its separate response view. Canonical components retain their original constructors and serializers.

| Request property | Generated shape |
|---|---|
| required, non-null | `T`, no default |
| required, nullable | `T?`, no default |
| optional, non-null | `T? = null` |
| optional, explicitly nullable | `Patch<T> = Patch.Absent` |

A local reference's explicit `nullable` override takes precedence over its target's nullability. Nullable aliases alone must not accidentally turn an explicitly non-null property into Patch. Const properties retain the existing constant-field contract. For the new [terminal scalar schema Pointer subset](model-semantics.md#terminal-scalar-schema-pointers), absent overrides inherit the original leaf's effective nullable spelling, while explicit true **and false** override it. The [finite collection schema Pointer profile](model-semantics.md#finite-collection-schema-pointers) applies the same override to only the container root: annotation-only wrappers without an override inherit it, optional nullable request uses become `Patch<List<T>>` / `Patch<Map<String,T>>`, and element/value nullability stays independent without entry-level Patch. Explicit false remains an ordinary optional field rather than Patch; required nullable collections have no default. Supported target checks retain Patch.Value/constructor/serde enforcement; named-component behavior is unchanged. [Finite collection reference chains](model-semantics.md#internal-collection-reference-chains) retain the nearest explicit true/false override across hops before applying the consumer override, without changing inner nullability or adding entry-level Patch.

[Nullable ordinary DTO root aliases](model-semantics.md#root-nullable-object-aliases) retain their explicit nearest true/false intent before a consumer override. Absent overrides inherit it, optional true uses become Patch and explicit false remains ordinary optional `T? = null`; required nullable request properties have no default. Referenced shared DTOs keep shared field defaults. [Intrinsic nullable direct DTO components](model-semantics.md#intrinsic-nullable-dto-components) independently apply the same inherited/overridden typed root intent while preserving the non-null physical classifier and shared defaults. Broader nullable composition and existing direct dictionary behavior are unchanged.

[Own-nullable plain String-enum aliases](model-semantics.md#root-nullable-string-enum-aliases) inherit nearest explicit intent before the consumer override, with the same required/Patch/default rules. A null value is distinct from a non-null Unrecognized enum value; the companion serializer remains non-null.

[Intrinsic nullable plain String enums](model-semantics.md#intrinsic-nullable-string-enums) retain the same typed root intent, required/Patch/default rules and explicit false narrowing; a nullable schema does not admit null enum members or change the canonical non-null companion serializer.

[Own-nullable primitive value-class aliases](model-semantics.md#root-nullable-value-class-aliases) use the same nearest-intent/required/Patch/default table. Wrapped constructor/decoder checks run for Value, not Null/Absent; explicit false narrows the root without changing the canonical wrapper or primitive wire.

[Intrinsic nullable primitive value classes](model-semantics.md#intrinsic-nullable-value-class-components) preserve the same direct/wrapper/alias required/Patch/default rules and explicit false narrowing; the physical constructor and its content checks remain non-null.

[Own-nullable temporal aliases](model-semantics.md#root-nullable-temporal-aliases) retain the same required/Patch/default and nearest-intent rules. Null/Absent do not parse temporal contents; Value uses the original ISO codec. False narrowing does not make an optional field required.

[Intrinsic nullable temporal components](model-semantics.md#intrinsic-nullable-temporal-components) export nullable public native aliases while direct/wrapper/alias fields retain the existing required/Patch/default and false-narrowing table. Canonical `.parse` and non-null companion codecs remain distinct from the public reified nullable root codec.

[Own-nullable bigint aliases](model-semantics.md#root-nullable-bigint-aliases) preserve that table while keeping direct fields and `Patch.Value` as decimal strings; root/list/map values remain native Long JSON numbers. Serializer attachment is a field-use decision, not a global nullable alias policy.

[Intrinsic nullable bigint components](model-semantics.md#intrinsic-nullable-bigint-components) inherit the same table for direct/wrapper/alias consumers: public reified root codecs are nullable, but direct fields and Patch values retain their decimal-string serializer while root/list/map values are JSON numbers.

[Own-nullable byte/binary aliases](model-semantics.md#root-nullable-bytebinary-aliases) keep the same model/Patch presence rules with signed-array JSON. Simple HTTP roots instead retain Ktor raw bytes: null requests send ASCII `null`, and that response body returns a ByteArray containing those bytes, not Kotlin null. Do not infer root HTTP null semantics from the model table.

[Intrinsic nullable byte/binary components](model-semantics.md#intrinsic-nullable-bytebinary-components) also inherit that table in direct/wrapper/alias fields without globalizing a serializer or changing raw HTTP null-ASCII projection. Their exported root alias itself is nullable, while explicitly false-narrowed consumers use the canonical non-null array backing.

[Primitive dictionary defaults](primitive-map-defaults.md) on ordinary inline request fields retain this table: required maps have no initializer, optional explicitly nullable maps remain `Patch.Absent`, and entry nullability never creates entry-level Patch. Default annotations are validated even when they supply no constructor initializer.

## Three wire states

`Patch<T>` is a generated sealed class in the model package, with these members:

- `Patch.Absent`: omit the key, even with `encodeDefaults = true`.
- `Patch.Null`: send JSON `null`, even with `explicitNulls = false`.
- `Patch.Value(value)`: send the bare serialized value, without an envelope.
- `valueOrNull`: the value for `Value`, otherwise null.
- `Patch.ofNullable(value)`: null means `Null` (clear).
- `Patch.orAbsent(value)`: null means `Absent` (leave unchanged).

Generated Patch properties use `@EncodeDefault(NEVER)`. `PatchSerializer<T>` delegates its descriptor to the element serializer. Standalone encoding of `Absent` throws `SerializationException`: silently encoding it as null would change the requested operation. A present null decodes as `Null`; an omitted property uses its `Absent` default.

Required nullable properties have no Kotlin constructor default. Their wire behavior still obeys the caller's JSON configuration: `explicitNulls = true` emits a null key, while false omits it. With false, decoding a missing nullable key can also infer null; this is not strict required-field validation. Shared components retain their previous nullable defaults.

## Ordinary DTO JSON property counts

[Direct explicitly nullable component DTO root own counts](direct-nullable-component-counts.md) retain the same required-nullable/Patch and shared-default policies below. Counts constrain non-null JSON objects in the original nominal serializer; root null uses the existing nullable adapter and does not validate an object. Nullable allOf component counts remain separate.

`--validation` enables [wire-property counts](model-semantics.md#ordinary-dto-json-property-counts) on supported direct or [flat-inline](model-semantics.md#flat-inline-allof-dto-property-counts) ordinary non-inherited request DTOs with constructor fields, without changing this presence table. Raw incoming null/unknown keys each count once before defaults/unknown handling; missing const defaults do not backfill input counts. Outgoing counts use the caller Json's actual automatic serializer output: Patch Absent remains omitted, Null remains a null member and Value remains a bare value. Required nullable and ordinary optional null behavior still follows `explicitNulls` / `encodeDefaults`.

Constructors/copy are not object-count barriers; JSON string/element encode/decode are. Ordinary decoded const overrides are preserved. Even min=0 is JSON-only; non-JSON codecs reject. HTTP callbacks may precede serialization count failures, but no invalid request reaches the engine. Existing typed preflight checks keep their earlier order, and arbitrary body overrides remain caller-owned. Union variants, reference-composed/inherited counts, empty/const-only and raw projections stay unsupported. This is an opt-in migration from previously rejected ordinary count schemas, not a presence API change.

## Flat inline allOf primitive bounds

The opt-in [ordinary-model bound conjunction](model-semantics.md#flat-inline-allof-primitive-bounds) retains winning field presence and required-name union. Nested ordinary request models can use the flat inline profile; optional explicitly nullable fields remain Patch, and required nullable fields retain no default. Constructor/copy/automatic serde check non-null Value contents, not Absent/Null. This does not enable inline root allOf request bodies, defaults, union/common-field aggregation or wider composition. Flat-inline whole-object counts are independently supported on nested ordinary request DTOs; Absent remains omitted, and leaf-valid Null/Value may still exceed the outgoing object maximum.

## Union contexts

Object-union request bodies keep ordinary `T? = null` field defaults, including their nested objects. These roots use the inline union tag/key dispatcher and raw Unknown passthrough, not component-root default polymorphism.

For a nominal union retaining multiple object branches, root schema nullability does not add a `Null` subtype or make a required `input` nullable. `requestBody.required` alone controls that parameter's nullability. Raw JSON null can be supplied for encoding by explicitly constructing the generated `Unknown<Input>` peer; optional null inputs send JSON null. Content-based decoding remains object-only: decoding JSON null directly as the non-null union fails rather than producing Unknown. `W_HTTP_REQUEST_UNION_PROJECTION` identifies this caller-specific contract. A single object plus null still folds to an ordinary nullable plain input.

An object union nested inside a plain request object uses a different context: required nullable variant fields have no default; optional nullable variant fields remain plain `T? = null`, not Patch. Inline plain objects reached inside those variants (including list items and dictionary values) regain the request-object Patch rules. A single object plus a null branch folds into a nullable plain input with those same request-object rules.

## Request-only field unions

After the existing nullable/scalar/enum folds, a union in a request-only property, list element or dictionary value that is not a supported object union uses raw `JsonElement`. This includes validated primitive combinations, array branches and mixed object/scalar branches. No native payload wrappers or branch DTOs are emitted for this projection. `W_HTTP_FIELD_UNION_PROJECTION` identifies the original union location. Each branch still passes the shared schema subset's metadata, reference, depth, naming and resource checks.

The presence table applies to the resulting type: a required nullable field is `JsonElement?` without a default; an optional explicitly nullable plain-object field is `Patch<JsonElement>`. Nested request variants instead keep optional nullable fields as `JsonElement? = null`; their plain-object descendants regain Patch. Dictionary values retain their existing container mapping rather than Patch; raw JSON-value dictionaries use `JsonObject`, preserving explicit `JsonNull` entries. Existing component references, component/response primitive wrappers and ordinary root-union field policies are unchanged. Object-only unions retain their typed variants.

This is not runtime union-membership validation: a raw field can carry values outside its declared branches, including `JsonNull` in a non-null `JsonElement`. For a Patch field, `Patch.Value(JsonNull)` and `Patch.Null` both encode a null key, which decodes as `Patch.Null`. Common abstract union properties across object variants and branch shapes outside the shared subset remain unsupported.

## Body and runtime ownership

Inline inputs are named `<Operation>Input` plus the model suffix in `.api.models`. Fallback method renaming also re-plans input type identities. Direct dictionaries use aliases and package-level value declarations. Dictionary **entries** already represent absence; a nullable scalar dictionary value remains `T?`, not Patch. Simple list bodies need no local model. Inline objects inside root request arrays use raw JSON maps, not request-only DTOs; see [structural operation values](http-client.md#structural-values-below-simple-containers).

The pure generation plan selects the runtime package and emits one `Patch.kt` in the default models package or the explicit [shared runtime package](runtime-package.md) only when included declaration trees need it, including objects inside union variants. Excluded APIs and unselected media cannot leave a runtime behind. Components that conflict with `Patch` or `PatchSerializer` are rejected in the same package when that runtime is needed; a model suffix can separate the identities. Regeneration removes obsolete manifest-owned files only.

## Limits and verification

This is the plain-object presence contract, not complete request-schema support. Request-field unions are bounded by the raw projection above; additional composition projections, validation constraints, special scalar serializers are separate extensions. Shared runtime relocation is covered by the independent package contract, not by this original presence extension. Root request unions have a separate [empty-singleton/Unknown projection](http-client.md#root-request-unions-with-primitive-or-array-branches), not Patch or native primitive-payload wrappers.

`core/http_request_presence*_test.go`, native CLI ownership tests and the fresh CLI-generated Kotlin HTTP consumer cover presence, source/name guards and constructor limits. The Kotlin consumer checks absent/null/value and required nullable behavior under all four `encodeDefaults`/`explicitNulls` combinations, including nested/list/map fields, dictionary/list bodies, referenced aliases, shared response models and fallback-renamed inputs. `core/http_request_union_test.go` and the union HTTP consumer separately prove root/variant/nested-object defaults, missing nullable fields, nullable object folding and Unknown passthrough. `core/http_request_field_union*_test.go` and the request-field Kotlin consumer additionally cover raw kinds/unknowns, nullable and Patch carriers, list/map elements, variant descendants and unchanged component wrappers. The JVM compilation gate also includes 240 boxed `Patch<Long>` fields. These checks do not establish Android/KMP or minimum dependency versions.
