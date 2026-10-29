# Kotlin integration

- [Discarded structured const declarations](../../../docs/discarded-structured-const.md) are validated without acquiring Kotlin fields/defaults/helpers. Extra keys keep existing Json unknown-key behavior; winning field codecs and actual body/dictionary values remain separate.

- [Pinned structured const data](../../../docs/pinned-structured-const.md) uses non-null JSON body defaults in union/hierarchy data fields. Const-only structured variants are parameterless classes; same-kind decoded values persist. Scalar discriminators, Unknown policy and existing scalar-only singletons remain.

- [Referenced native enum collection field own defaults](../../../docs/referenced-native-enum-collection-defaults.md) accept the same `$ref`-field role for native numeric/Boolean enum leaves. Scalar wire typealiases, exact numeric default membership before rounding, the original approximation warning, retained validation bounds/counts and original budgets remain; String/temporal leaves and target/chain/root/child defaults stay separate.
- [Referenced String enum collection field own defaults](../../../docs/referenced-string-enum-collection-defaults.md) accept ordinary direct local `$ref` fields' own outer defaults through already admitted finite named/internal List/typed Map paths ending at plain String enum leaves (explicit or typeless). Default members stay strict against the original vocabulary while runtime keeps `Unrecognized`; actual lexical classifiers, per-layer nullable intents, required/Patch and original budgets remain, and native/bounded/formatted leaves plus target/chain/root/child defaults stay separate.
- [Referenced temporal collection field own defaults](../../../docs/referenced-temporal-collection-field-defaults.md) accept only ordinary direct local `$ref` fields' own outer annotations through already admitted finite named/internal List/typed Map paths with standard date/date-time leaves. Keep actual LocalDate/Instant aliases/codecs, independent nullable layers and original source/check/default budgets. Required arguments and Patch.Absent remain after full annotation validation; target/chain/root/child/native/nominal defaults remain separate. Missing fields use fresh typed trees; explicit null and supplied values retain their states. Original temporal lexemes participate in optimization and frozen-lock identity.

- With `--validation`, [direct explicitly nullable ordinary component DTO roots](../../../docs/direct-nullable-component-counts.md) may retain their own min/maxProperties. Incoming counts use the original whole JsonObject before defaults/unknown loss; outgoing counts use actual delegate output under caller settings. Canonical class/constructor/non-null companion stay unchanged; reified nullable/adapter codecs handle root null and skip counts. Constructor/copy are not count barriers. Nullable allOf/ref-wrapped/member-inferred roots and hierarchy/common/inherited/union/raw projections keep their exclusions; required/Patch and primitive checks retain their existing boundaries.

- [Referenced ordinary DTO object field own defaults](../../../docs/referenced-object-defaults.md) extend the same own-`default` role to ordinary direct local `$ref` fields whose target is an already accepted plain named ordinary DTO (transparent aliases collapse to the canonical type). [Disjoint ordinary allOf DTOs](../../../docs/allof-object-defaults.md) also accept the consuming field's own annotation through admitted named aliases/internal hops to named or unique internal constructor Models; use the actual final lexical owner and serializer. Keep the actual nominal type and emit-time helper binding, target original nodes, nearest nullable, required/Patch, per-leaf checks and whole-tree budgets. Target/chain/root defaults, repeated-field aggregation and hierarchy sources/owners remain separate.
- [Inline primitive union field own defaults](../../../docs/union-field-defaults.md) accept an ordinary DTO field's own `default` when its inline `oneOf`/`anyOf` is plain String/Integer/Number/Boolean branches and the shared plan emits the native primitive-union declaration. Keep the token (the runtime selector picks its wrapper by token kind in String→Boolean→Integer→Number order) and the emit-time helper binding; required/Patch remain. `enum`/`const` pins are enforced at generation time against the matched branch vocabulary while runtime wrappers still do not enforce membership. String formats, object/`$ref`/nullable branches, same-kind folds, `ObjectUnion`, request-context unions and root/component defaults stay separate, and an ordinary field `$ref` to a root union component is admitted and validated like an inline union.
- [Referenced ordinary DTO elements in collection field own defaults](../../../docs/referenced-object-collection-defaults.md) cover an inline `array`/typed `Map` field whose element is a direct local `$ref` to an already accepted plain named DTO, and a field `$ref` to a named collection whose elements resolve to such a DTO. Keep the actual collection/alias type, emit-time helper binding, element original nodes, nearest nullable, required/Patch, per-leaf checks/counts and whole-tree budgets. Other target/root annotation roles, alias/internal graphs, scalar/enum elements, `$ref`/composition target properties and union/hierarchy fields stay separate. Original plain DTO target/alias factories have the independent object-default ownership profile below.
- [Direct inline List/typed Map of ordinary DTO objects field own defaults](../../../docs/nested-object-collection-defaults.md) accept the same own-`default` role on a direct inline `array`/typed `Map` tree whose element tree contains an already accepted plain object leaf, including longer finite mixed paths like `List<Map<String, DTO>>`. Scalar-only collections keep the existing builtin/temporal/enum helpers. Keep the actual collection type and emit-time helper binding per wrapper, per-layer nullability, required/Patch, per-leaf checks, container counts and whole-tree budgets. Element-owned/container-child defaults, `$ref`/composition/dictionaries/const/enum-object/native/empty and referenced named-DTO elements stay separate.
- Object-containing default walks share a [16,384-step source planning budget](../../../docs/nested-object-defaults.md#object-source-planning-budget), independently of JSON value size. Repeated reference expansion, including empty/null and required/Patch/discarded annotations, returns `E_MODEL_LIMIT` at an original schema Pointer when over budget. Each annotation starts a fresh walk; typed reference admission is unchanged.
- [Direct inline ordinary DTO object field own defaults](../../../docs/nested-object-defaults.md) accept an ordinary field's own `default` when its inline plain object schema (explicit `object` or inferred from non-empty `properties`) is already accepted. The shared object planner still owns the field type and nested declarations; required arguments, `Patch.Absent`, per-layer nullability, canonical keys, original diagnostics and whole-tree budgets remain. `$ref`/named DTO, union/allOf, dictionary/fixed extras, const, empty objects and List/Map-of-object defaults stay separate.
- [Finite nested builtin collection defaults](../../../docs/nested-builtin-collection-defaults.md) accept outer defaults on direct ordinary DTO List/typed Map trees with builtin String/Boolean/Int/Long/Float/Double leaves. Each layer keeps its own nullable intent and opt-in counts/leaf checks; optional fields use typed initializers, required arguments and Patch.Absent remain. Original diagnostics, canonical dictionary keys and whole-tree resource budgets are enforced; [direct temporal trees](../../../docs/nested-temporal-collection-defaults.md) and [direct nested String enum trees](#nested-string-enum-collection-defaults) have separate leaf contracts, while named/ref/root defaults keep their own boundaries.

- [Nullable structured const values](../../../docs/nullable-structured-const.md) preserve JsonObject?/JsonArray? roots and nearest false views. Ordinary referenced body constants keep non-null defaults; no membership or root/value default is introduced.

- [Structured const values](../../../docs/structured-const-values.md) use non-null JsonObject/JsonArray root/alias and collection/HTTP/SSE kind projection. Same-kind alternatives retain their data; roots/values have no default, while ordinary DTO references keep body constants. Intrinsic nullable roots/aliases remain excluded.

- [Structured const fields](../../../docs/structured-const-fields.md) expose ordinary JsonObject/JsonArray body values; defaults and same-kind alternatives follow existing const approximation. Roots/ref literals/leaves and exact membership remain separate.

- [Null-only DTO defaults](../../../docs/null-only-defaults.md) accept actual null annotations on direct JsonNull fields; required arguments and Patch.Absent remain. General root/ref/collection/const defaults stay outside this permission.

- [Inferred null literals](../../../docs/inferred-null-literals.md) let plain null const/enum schemas omit type while retaining the same JsonNull API, Patch states and HTTP/SSE projection as explicitly typed inputs. This does not infer nullable intent, mixed vocabularies or general composition/defaults.

- [Explicit null literals](../../../docs/explicit-null-literals.md) use JsonNull for plain `type:null` const/enum values in both validation modes. JSON null, Kotlin null and absence differ; Patch.Value(JsonNull) and Patch.Null share a wire null. Direct inline HTTP null success retains Unit, and singleton dispatch retains its approximation.

- With `--validation`, [ordinary referenced/nested allOf DTO counts](../../../docs/referenced-allof-counts.md) constrain the same whole JSON object through the retained automatic delegate. Input counts precede defaults/unknown loss; output uses actual Json/Patch/const members. Constructor/copy do not check object counts. Nullable allOf component roots, inherited/common/raw and inline root HTTP allOf remain excluded; direct explicitly nullable root own counts follow the separate contract above.

- With `--validation`, [ordinary referenced/nested object allOf DTOs](../../../docs/referenced-allof-bounds.md) may retain same-wire direct primitive bounds through admitted object references/aliases/nesting. Original contributions must share kind/format/nullability; types/presence/Patch and owned effective lock identity remain. Property-ref/default/enum/collection aggregation, inherited/common and inline root HTTP allOf remain outside this subset.

- [Plain nullable String/typeless enum vocabularies](../../../docs/string-enum-null-members.md) may contain one null and at least one String. A [typeless plain String/null vocabulary](../../../docs/implicit-string-enum-null.md) supplies intrinsic nullable intent when type/nullable is absent; explicit non-null intent remains. [Narrowed component-own enum factories](../../../docs/narrowed-enum-wrapper-defaults.md) validate the original vocabulary and nearest alias intent independently; unannotated aliases and referencing fields inherit no value. Actual enum classifiers/Unrecognized remain String-based; nullable views handle JSON null, while a vocabulary String `"null"` remains a token. Both validation settings support this profile. Direct inline HTTP String projection retains the raw Ktor response reader; named enum codecs stay typed.

- With `--validation`, [explicit nullable numeric/Boolean enum vocabularies](../../../docs/native-enum-null-members.md) may contain one null and at least one non-null scalar member. Null uses the nullable codec; typed consumers retain original membership/bounds and direct DTO defaults. Null counts toward the 240 raw-member budget. Default-off, String/null-only and general composition remain separate.

- With `--validation`, [plain nullable numeric/Boolean enum roots and aliases](../../../docs/nullable-native-enums.md) retain scalar/typealias APIs, nearest nullable overrides and typed membership/bounds. Standalone alias codecs remain unchecked; null vocabulary members follow the separate validation-only contract above.

- With `--validation`, [plain named numeric/Boolean enum sources](../../../docs/named-native-enums.md) retain checks through internal reference chains and borrowed finite List/Map. Actual scalar exports authorize the source; independent aliases stay unchecked.

- With `--validation`, [direct internal numeric/Boolean enum Pointers](../../../docs/internal-native-enums.md) retain source vocabulary/bounds in typed consumers and transparent root bridges. Standalone aliases remain unchecked; original schema/source/slot limits remain.

- With `--validation`, [plain numeric enum plus bounds](../../../docs/native-enum-bounds.md) checks membership and inclusive/exclusive endpoints together; Int/Long `multipleOf` is supported. Empty non-null intersections and invalid direct DTO defaults are rejected. Floating enum `multipleOf`, String/const/general composition remain outside this profile.

- With `--validation`, direct numeric/Boolean enum DTO fields enforce mapped scalar membership in constructors/decoders; optional/null/default/Patch policies stay unchanged. Float/Double compare finite mapped values; rounding-equivalent source numbers share a value. Default-off retains the approximation warning. [Typed native enum consumers](../../../docs/native-enum-consumers.md) retain checks through plain scalar component aliases/named refs, List/Map and operation inputs/results/SSE data. Independent aliases, generic pins and open String enums retain their separate policy. Raw/discarded projections that lose direct field checks fail; vocabulary participates in merge/dedup/name-lock identity. See [native enum DTO validation](../../../docs/native-enum-validation.md).

- Selected direct ordinary inline JSON error objects generate operation/status DTOs in api.models; KDoc links the type for caller-selected errorBody decoding. See [inline error DTOs](../../../docs/inline-errors.md). Caller-owned validation and cancellation behavior remain.

- Validation-enabled direct JSON-value arrays support recursive `uniqueItems` using exact JSON semantics and explicit runtime bounds. See [JSON uniqueness](../../../docs/json-unique.md); typed nominal/projection equality remains separate.

- With `--validation`, ordinary DTO direct inline plain String enum List/typed Map leaves accept builtin String lengths and portable patterns, with or without an outer default. Check actual `serialName` at DTO constructor/copy/automatic decoding boundaries, including typed HTTP/SSE; valid Unrecognized values remain open. Defaults must be original members and satisfy constraints; required/Patch annotations validate before initializer suppression. Keep actual lexical enum owners after optimization. [Finite inline nested trees](checked-enum-collections.md#nested-inline-bounds) separately retain the same bounds through actual enum terminals, outer defaults and existing object walks. Roots/ref, union/hierarchy fields and numeric bounds retain their exclusions. Direct enum list uniqueness follows the [serialName contract](../../../docs/string-enum-unique.md): compare known and Unrecognized payloads, preserve open runtime Strings and strict default membership; already admitted named/internal enum leaf references and aliases use the same equality and original declaration authority. Use actual canonical owners for singleton access after explicit merging; wider graphs and projected vocabularies remain excluded. See [String enum collection bounds](../../../docs/string-enum-collection-bounds.md).

- Direct plain String/typeless and native scalar enum DTO fields accept vocabulary-checked defaults. Actual enum classifier/codec and required/Patch presence remain; unknown String values retain Unrecognized. See [enum DTO defaults](../../../docs/enum-defaults.md).

- Ordinary DTO direct local `$ref` fields accept their own representable default through already admitted finite named/internal paths ending at builtin String/Boolean/Int/Long/Float/Double. The actual scalar literal/check/nullable view supplies the initializer (`Long? = 2L`, for example). Required arguments and Patch.Absent remain after annotation validation. Target/chain/root/composition-wrapper defaults, enum/const/native/valueclass/nominal terminals and nullable folds retain their exclusions. [Referenced standard temporal scalar own defaults](../../../docs/referenced-temporal-scalar-field-defaults.md) separately admit only already accepted date/date-time paths with actual LocalDate/Instant, original lexeme identity and the same required/Patch/source/budget rules; collection temporal own defaults use the separate profile above, while native providers retain their exclusions. Defaults never authorize an otherwise rejected alias/root; constraints and legacy non-null scalar aliases still require their existing validation opt-in. Bare aliases/codecs remain unchecked. Effective defaults participate in optimization/name-lock identity; original reference, default-byte/value-node and source/check/slot limits remain.

- Direct scalar DTO fields accept representable schema defaults. Optional fields initialize/decode from the default; required fields remain required arguments and optional nullable request fields keep Patch.Absent. Unsupported default roles fail explicitly. See [scalar DTO defaults](../../../docs/model-defaults.md).

- Object unions may have nullable references to actual ordinary DTOs/aliases. Nearest member/alias false narrows the view; physical source constructors/codecs and original owner authority remain. See [nullable DTO reference members](../../../docs/nullable-object-reference-members.md).

- Ordinary inline object members of oneOf/anyOf may carry nullable intent. Original physical variants remain non-null; named roots use Null carriers and inline views use nullable codecs. See [nullable object members](../../../docs/nullable-object-members.md).

- Explicit nullable roots of supported allOf object products keep named Null carriers and inline nullable views. Aliases preserve nearest true/false intent and actual variants; intersection-member nullability remains separate. See [nullable product unions](../../../docs/nullable-product-unions.md).

- Inline object/union fields at the same wire name in multiple object-union variants keep separate nested classifiers and codecs. Select the actual variant before accessing the field; List/Map values and supported allOf values follow the same owner rule. See [inline union values](../../../docs/union-inline-values.md).

For model integration, use the sources/serialization section; load Ktor guidance only for an HTTP task. Inspect generated files for actual names and signatures. Components are in `<package>.models`; HTTP adds `<package>.api`, applicable `<package>.api.models` types and an aggregate in `<package>`. Helpers depend on the input.

## Raw JSON carrier field defaults

An ordinary DTO field may carry its own `default` when its shared plan produces a raw carrier: an open dictionary (`additionalProperties:true`) yields `JsonObject` and requires a JSON-object default; an unconstrained schema, an empty object or a request-context union projection yields `JsonElement` and accepts any JSON value. Initializers use file-level `Json.parseToJsonElement` helpers (plus an `as JsonObject` cast) so public qualifiers cannot be shadowed. Required/Patch, explicit null and the existing approximation warnings remain; typed dictionaries, arrays, enum/const/format/native/value-class and root/component defaults stay separate. Feature `raw-json-field-defaults`, fixture `tests/fixtures/http/raw-json-field-defaults.json`, consumer `RawJSONFieldDefaultsContracts.kt`; `OPENAPI_KT_PREPARE_ONLY=1` is not Kotlin evidence. General D03/D04 and release gates remain open.

## Inline primitive union field defaults

An ordinary DTO inline `oneOf`/`anyOf` field with plain String/Integer/Number/Boolean branches may carry its own `default` when the shared plan emits the native primitive-union sealed type. The initializer decodes through the final actual union serializer, so call-site types and wire behavior are unchanged. Preserve the original JSON token: the runtime selector maps token kind to wrapper in String→Boolean→Integer→Number order, so `1` stays `IntV` and `1.0` stays `NumberV`; do not canonicalize numbers. A null default requires explicit union nullability. `enum`/`const` pins are matched and enforced at generation time; the same role also covers an inline union used as an `array` item, typed `Map` value or defaulted-object property, validated per member. Direct local `$ref` scalar/enum branches (including chains) are resolved before membership, and a tag-dispatched content object union (`enum` or scalar `const` discriminator) or by presence is selected and validated per variant; scalar `const` object properties are exact-equality checked; String-`format`, per-branch-nullable branches, same-kind folds, request-context unions and root/component defaults remain separate; an ordinary field `$ref` to a root union component is admitted and validated like an inline union. A Cartesian composed union (union-in-allOf) default selects by conjoined tag pins mirroring the emitted first-match conjunction order and validates against the selected variant's merged original shape; ordinary allOf fields with disjoint property contributions use the object merge default role below. Canonical JSON participates in optimization and lock identity, and the helper binds the final relocated type. Feature `union-field-defaults`, fixture `tests/fixtures/http/union-field-defaults.json`, consumer `UnionFieldDefaultsContracts.kt`; `OPENAPI_KT_PREPARE_ONLY=1` is not Kotlin evidence. General D03/D04 and release gates remain open.

## Closed subtype root union field defaults

For a direct local `$ref` field's own default targeting an existing named closed root object union, use the [closed subtype contract](../../../docs/closed-union-defaults.md). The annotation is a Json.Default object with String `type` equal to an actual emitted `@SerialName`; generated suffixes and root-name collisions can affect that value. The selected original plain constructor variant supplies business fields/required/leaf checks. Keep final component/nominal/type authority, original values/Pointer order, budgets and required/Patch/discarded validation.

The typed default honors caller custom discriminator keys or array polymorphism after construction. Directly serialized business `type` collisions, aliases/internal-owned targets, selected empty/const-only/allOf variants, collection/object-member closed unions and target/chain/root/component default inheritance remain separate. Null retains the existing root Null-variant contract. The `union-field-defaults` feature checks Data/Plain × validation, four Json default/null configurations with three polymorphism modes, Patch wire states, HTTP/live SSE and optimized helper binding.

## Referenced ordinary DTO elements in collection field defaults

Two admitted forms: an inline `array`/typed `Map` field whose element is a direct local `$ref` to an already accepted plain named DTO, and a field `$ref` to a named `array`/typed `Map` component whose elements resolve to such a DTO. Keep the actual collection/alias type and emit-time helper binding/dedup, element original nodes, nearest explicit element/consumer nullability, required/Patch, per-leaf checks and container counts. A referenced target must not own a root `default`; target-internal `$ref` properties, alias/internal chains, scalar/enum elements, unions/hierarchies and root/component defaults stay separate. Canonical JSON sorts dictionary keys while `/default/<index>/<escaped-key>` diagnostics keep original order. For ordinary direct local `$ref` fields to admitted finite named/internal List/Map trees, [allOf DTO element defaults](../../../docs/allof-object-defaults.md#referenced-collection-field-defaults) also work when the element crosses an actual reference edge to an eligible retained named/internal constructor Model. Keep original lexical source/type authority, collection/element codecs, nullable layers, source/value budgets and required/Patch/discarded validation. Bare allOf elements in referenced collection trees also follow the [retained constructor role](../../../docs/allof-object-defaults.md#bare-allof-elements-in-referenced-collections): require the original unique nonempty Model/node/Pointer/SourceCatalog/owner and same final classifier, allowing its existing package-relative root-peer spelling. Preserve existing alias/element types, merged fields, nullable layers, budgets and suppression/no-drop checks. Referenced allOf object-default members and hierarchy remain separate; direct inline members follow the [object-default member contract](../../../docs/allof-object-defaults.md#inline-allof-properties-inside-object-defaults). [Direct inline collections with allOf element refs](../../../docs/allof-object-defaults.md#direct-inline-collections-with-referenced-elements) also accept their own outer default: keep the original inline type plan, queue pre-presence TypeRef until final nominal linking, and reuse retained original-source/owner/Pointer/name authority for admitted alias/internal paths. Target/chain/element/child defaults are not inherited. [Direct inline fields with bare allOf constructor elements](../../../docs/allof-object-defaults.md#direct-inline-collection-field-defaults) use the separate original-leaf role below. The feature's checked profile injects already admitted named collection counts; unchecked source grammar is unchanged.

Original/canonical defaults retain 32 KiB caps, 16,384 values, source depth48, checked edges16 and slots240. Feature `referenced-object-collection-defaults`, fixture `tests/fixtures/http/referenced-object-collection-defaults.json`, consumer `ReferencedObjectCollectionDefaultsContracts.kt`; `OPENAPI_KT_PREPARE_ONLY=1` is not Kotlin evidence. General D03/D04 and release gates remain open.

## Direct inline List/typed Map of ordinary DTO object field defaults

For this profile, check that the annotation belongs to an ordinary DTO direct inline `array` or typed `Map<String, T>` field and that its element tree contains an already accepted plain inline object leaf, including longer finite mixed paths such as `List<Map<String, DTO>>`. Scalar-only collections stay with the existing builtin/temporal/enum collection profiles and keep their emitted helpers. The field type comes from the shared collection/object planner, and the helper decodes the canonical JSON through the final actual collection serializer, binding the final relocated element type and deduplicating per wrapper/nullable shape.

Elements are validated like the direct object leaf: every declared property, unknown-key rejection, required presence, per-layer nullability, builtin/temporal/String-enum/native leaves, nested builtin collections, nested object recursion and property-owned defaults validated by the element model. Container counts and non-object leaves keep their existing rules. An element schema cannot own the default annotation, so container-child defaults remain rejected. `$ref`/composition/dictionaries/fixed extras/`const`/object-level `enum`/native/empty and referenced named-DTO elements stay outside this profile.

For [bare ordinary allOf elements](../../../docs/allof-object-defaults.md#direct-inline-collection-field-defaults), the shared type plan must create a nonempty constructor Model with disjoint original property contributions. Only the consuming inline collection's original leaf gets temporary validation permission for bare elements; object-member paths retain their separate role. Referenced allOf elements use the final nominal queue role above. Preserve merged required/nullable/fields/Pointers/depths, source work/value budgets and no-drop/suppression behavior. The element's own default stays rejected. Helper binding remains on the actual final collection type. The same feature covers fresh nested values, nullable/required/Patch and optimized HTTP/live SSE; checked input profiles add existing count constraints.

Canonical JSON sorts dictionary keys at every level while diagnostics keep original member order and `/default/<index>/<escaped-key>` Pointers. With `--validation`, element DTO `init` checks and retained counts/bounds apply. Original/canonical defaults each retain 32 KiB caps, whole trees 16,384 values, source depth48, checked edges16 and slots240. The feature name is `nested-object-collection-defaults`, with fixture `tests/fixtures/http/nested-object-collection-defaults.json` and consumer `samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NestedObjectCollectionDefaultsContracts.kt`. `OPENAPI_KT_PREPARE_ONLY=1` is native preparation only, not Kotlin evidence. General D03/D04 and release gates remain open.

## Direct inline ordinary DTO object field defaults

For this profile, check that the annotation belongs to an ordinary DTO field whose schema is a plain inline object, with `type: "object"` or an inferred object from a non-empty `properties` map, and that the no-default inline object is already accepted. The field type and every nested declaration still come from the shared ordinary object planner; the annotation never changes public types or adds declarations.

The default value must be a JSON object (or null on an explicitly nullable object). Every declared property is validated, unknown keys fail, and every `required` wire must be present. Property values reuse the finite leaf/container rules: builtin and temporal scalars, String enum membership against the original vocabulary with open runtime `Unrecognized`, native numeric/Boolean exact membership, nested builtin collections, and nested plain objects recursively. A property that is a direct local `$ref` to a plain object borrows that target recursively; a `$ref` to a scalar/enum keeps existing leaf validation against the resolved source; reference chains stay separate. A property may carry its own `default`, validated by the nested model plan; the outer default may omit it. Canonical JSON sorts object keys, while diagnostics keep original member order and Pointers.

Optional fields decode a fresh tree through the final DTO serializer; missing keys use the default, supplied values replace it and explicit null stays null when explicitly nullable. Required fields keep required arguments without initializers. Optional explicitly nullable inline request fields keep `Patch.Absent`/`Patch.Null`/`Patch.Value`. Discarded allOf annotations validate before override. With `--validation` the nested DTO `init` checks run when the decoded default is constructed.

Ordinary direct inline `allOf` object fields also accept their own defaults when the shared planner produces a constructor DTO and every wire property has one source contribution. Member objects, nested compositions and accepted object-member refs retain original field nodes/Pointers/depths, required union and nullable intent. Unknown/default member and leaf checks use the same walk and budgets. Annotation-only ref wrappers, repeated-property aggregation, fixed extras/counts, empty/const-only projections, hierarchy/common/inherited and root/component defaults remain separate. The same object walk accepts direct inline allOf-valued properties with disjoint contributions and at least one constructor field, preserving original required/nullable/source/depth and shared source/value budgets even when omitted. Their own field defaults remain independently validated; missing/replacement/null and fresh values use the final enclosing serializer. Actual `$ref` properties to allOf targets and allOf leaves under collection-valued properties retain separate roles. The `nested-object-defaults` feature covers this role, including optimized/locked final helper binding.

This direct inline profile keeps `$ref`/named DTO, union and List/Map-of-object defaults in their linked contracts. Dictionaries or fixed extras, empty objects, `const`, object-level `enum`, native/value-class/format markers and container-child defaults retain their separate boundaries. Original/canonical defaults each retain 32 KiB caps, whole trees 16,384 values, source depth48, checked edges16 and slots240. The feature name is `nested-object-defaults`, with fixture `tests/fixtures/http/nested-object-defaults.json` and consumer `samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NestedObjectDefaultsContracts.kt`. Its route is `bash scripts/test.sh feature nested-object-defaults`; `OPENAPI_KT_PREPARE_ONLY=1` is native preparation only, not Kotlin evidence. General D03/D04 and release gates remain open.

## Referenced ordinary DTO object field defaults

For this profile, check that the annotation belongs to an ordinary DTO direct local `$ref` field and that the referenced target is an already accepted plain ordinary named DTO component (explicit or inferred `object`, non-empty `properties`, no composition/dictionary/const/native/value-class/format), possibly reached through transparent aliases that collapse to the canonical nominal type. The emitted helper decodes the canonical JSON through the final actual DTO serializer and binds its final relocated type after optimization; fields that resolve to the same DTO share one helper per file.

For the [ordinary allOf extension](../../../docs/allof-object-defaults.md), require disjoint original contributions and an actual retained constructor Model under its original lexical component. Already admitted finite named aliases/internal hops may end at named or internal Models; internal property/List/Map/root-peer/composition-owned concrete Models use the original node/Pointer/SourceCatalog and final nominal binding. Inspect canonical constructor/serializer ownership after optimization. Root intent follows consumer, nearest alias/internal hop, then terminal, before optional presence, including internal member-derived nullability; explicit false narrows it. Target/chain defaults, annotation-only wrappers, repeated properties, extras/counts and hierarchy sources/owners remain separate. Defaults on referenced finite collections with actual allOf DTO element refs use the [separate collection role](../../../docs/allof-object-defaults.md#referenced-collection-field-defaults); bare allOf elements in referenced collection trees follow their [retained constructor role](../../../docs/allof-object-defaults.md#bare-allof-elements-in-referenced-collections), while direct inline allOf object-default members follow the [member contract](../../../docs/allof-object-defaults.md#inline-allof-properties-inside-object-defaults) and referenced member targets remain separate. Direct inline fields with bare allOf constructors use the [original-leaf collection role](../../../docs/allof-object-defaults.md#direct-inline-collection-field-defaults). The `referenced-object-defaults` feature checks actual lexical serializers, fresh member defaults, all four Json configurations, Patch/HTTP/live SSE and optimized helper binding.

Only the consuming field's own `default` is accepted. A target-root default and a default on any intermediate reference or target-internal container child are rejected. Nullability follows nearest explicit consumer/alias intent; a null default on a nearest `nullable:false` view is rejected. Property values reuse the finite leaf/container planner: builtin/temporal scalars, String enum membership against the original vocabulary with open runtime `Unrecognized`, native exact membership, nested builtin collections, nested plain-object recursion and per-layer nullability. Required fields keep required arguments; optional explicitly nullable inline request fields keep `Patch.Absent`/`Null`/`Value`; discarded allOf annotations validate before override.

Canonical JSON sorts object keys while diagnostics keep original member order and `/default/<escaped-key>` Pointers. With `--validation`, per-leaf bounds/counts and the nested DTO `init` checks apply. Original/canonical defaults each retain 32 KiB caps, whole trees 16,384 values, source depth48, checked edges16 and slots240. The feature name is `referenced-object-defaults`, with fixture `tests/fixtures/http/referenced-object-defaults.json` and consumer `samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedObjectDefaultsContracts.kt`. `OPENAPI_KT_PREPARE_ONLY=1` is native preparation only, not Kotlin evidence. General D03/D04 and release gates remain open.

## Nested String enum collection defaults

For this profile, check that the default belongs to the outer ordinary DTO field and the schema is an already supported finite direct inline List/typed `Map<String, T>` tree ending in a plain unformatted explicit String or typeless String enum. List/List, List/Map, Map/List, Map/Map and longer finite mixed paths share that rule. Keep the actual lexical enum classifier, entries, `serialName`, `fromSerialName`, serializer and open `Unrecognized` values; inspect final generated owners after dedup/merge/hoist instead of treating the leaf as a String or accessing alias-owned nested types. In the generator checkout, see the [nested String enum contract](../../../docs/nested-string-enum-collection-defaults.md).

Every non-null annotation leaf must be a JSON String in the original vocabulary with validation on or off. Unknown runtime Strings remain accepted by the enum codec. Each container and leaf keeps its own explicit nullable intent; optional presence alone does not allow a null default. An explicitly nullable enum vocabulary permits at most one null and requires at least one String; null counts toward the raw 240-member cap, and String `"null"` remains distinct. Empty/null defaults still validate every schema/vocabulary node before traversal or suppression.

Optional ordinary missing fields use fresh typed trees, including typed empty containers; explicit null stays null and supplied collections replace the default. Required fields retain required arguments after complete annotation validation. Optional explicitly nullable ordinary inline request fields retain Patch.Absent, Null and Value; defaults never turn missing keys into updates, and inner nullable nodes never create entry-level Patch. Discarded allOf annotations validate before override. With `--validation`, original per-layer List/Map counts apply to defaults and typed constructor/copy/serde/HTTP/SSE values. Keep `explicitNulls=false` missing-required-null inference separate from required-key validation.

The separate [finite inline bounds contract](checked-enum-collections.md#nested-inline-bounds) accepts nested String lengths/patterns and supported lexical metadata under validation. Typed container uniqueness follows [structural equality](checked-enum-collections.md#structural-equality). Numeric bounds, other codec families and ref/named/root/wrapper/child/target/chain annotation roles keep their own eligibility. It adds no inheritance, recursive tree or general intersection admission. Original direct plain String enum list leaves admit [serialName uniqueness](../../../docs/string-enum-unique.md), including these nested default paths. Single-layer String enum bounds and existing builtin6/temporal checks retain their separate contracts; `uniqueItems:false` is only the existing opt-in no-op on otherwise supported arrays.

Preserve errors at original source `/default/<index>/<escaped-key>/...` Pointers. Each Map validates in source order and sorts keys only after success; `$ref`, `type`, `/` and `~` keys remain opaque data. Actual default/type/check/nullability identity participates in optimization and frozen locks, with every Map level ignoring key order. Required/Patch annotations without an initializer add no runtime default identity. Original/canonical defaults each retain 32 KiB caps, whole trees 16,384 values, source depth48, checked edges16, enum/slots240, work16,384 and input8MiB. A default never exempts malformed schema or lost checks.

The generator checkout's feature name is `nested-string-enum-collection-defaults`, with fixture `tests/fixtures/http/nested-string-enum-collection-defaults.json` and consumer `samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NestedStringEnumCollectionDefaultsContracts.kt`. Its dedicated route is `bash scripts/test.sh feature nested-string-enum-collection-defaults`. `OPENAPI_KT_PREPARE_ONLY=1` performs native generation/check and project preparation only; preparation is not Kotlin compilation/runtime evidence. Report actual checks; preparation and existing JVM results for other features do not establish this feature's Kotlin, Android/KMP, minimum-dependency or release acceptance. General D03 remains open.

## Referenced String enum collection defaults

For this profile, check that the default belongs to an ordinary DTO direct local `$ref` field and that the referenced target is an already accepted finite List/typed `Map<String, T>` path whose innermost leaves are plain unformatted explicit or typeless String enums. The walk may traverse named aliases, internal property edges and nested collection layers exactly like the builtin and temporal referenced extensions. Keep the actual lexical enum declaration (named components keep their own sealed class; inline leaves keep their `...Inner` class), strict original default membership at generation time, and open `Unrecognized` runtime for supplied values.

Container counts apply to defaults and typed consumers under `--validation`; nested enum String bounds/pattern and native numeric/Boolean terminals stay rejected. Errors point at the consuming field's original `/default/<index>/<escaped-key>` tree; Map keys sort after validation. The generator checkout's feature name is `referenced-string-enum-collection-defaults` with fixture `tests/fixtures/http/referenced-string-enum-collection-defaults.json` and consumer `samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedStringEnumCollectionDefaultsContracts.kt`. `OPENAPI_KT_PREPARE_ONLY=1` is not Kotlin compilation/runtime evidence; general D03 and release gates remain open.

## Referenced String enum null-member defaults

Existing referenced scalar/finite collection own-default roles compose with already admitted explicitly nullable plain String/typeless enum vocabularies containing one null. Keep null independent of the String `"null"`; non-null defaults must match original String members, while null defaults and members require each actual typed layer to be nullable. A vocabulary null never widens nearest `nullable:false`. Required/Patch annotations validate before suppression; supplied unknown Strings remain open. Original raw240 includes null. See the [null-member default contract](../../../docs/referenced-string-enum-null-member-defaults.md) and feature `referenced-string-enum-null-member-defaults`; native preparation is separate from JVM evidence.

## Referenced String enum scalar defaults

For plain unformatted String/typeless enum scalar references, check that the consuming ordinary field owns the annotation and the no-default reference path is already accepted. Defaults must belong to the original non-null String vocabulary; supplied unknown Strings remain the actual classifier's `Unrecognized`. Null annotations require effective schema nullability. The typed helper binds the final lexical enum serializer after optimization and is deduplicated across nested DTOs in the same file. Required/Patch annotations validate before initializer suppression. Formats, bounds/patterns and target/chain/root/child defaults keep their separate boundaries. Null vocabulary members use the separate role linked above. See the [scalar contract](../../../docs/referenced-string-enum-scalar-defaults.md). The feature route is `referenced-string-enum-scalar-defaults`; report fresh Kotlin execution separately from native preparation.

## Referenced native enum scalar defaults

For numeric/Boolean scalar references, check that the consuming ordinary field owns the annotation and the no-default reference is already accepted. Defaults must belong to the original vocabulary before Float/Double mapping; output uses scalar literals. Internal references and intrinsic nullable native schemas retain their existing validation opt-in. With validation, typed DTO consumers enforce mapped membership and retained bounds; standalone scalar alias codecs remain scalar decoding. Required/Patch annotations validate before initializer suppression. See the [scalar contract](../../../docs/referenced-native-enum-scalar-defaults.md). The feature route is `referenced-native-enum-scalar-defaults`; report fresh Kotlin execution separately from native preparation.

## Referenced native enum collection defaults

For this profile, check that the default belongs to an ordinary DTO direct local `$ref` field and that the referenced target is an already accepted finite List/typed `Map<String, T>` path whose innermost leaves are native `integer`/`number`/`boolean` enum vocabularies. The leaves keep their scalar wire type (`List<Int>`, `Map<String, Double>`); default members are strict against the original vocabulary at generation time with exact numeric comparison before Float/Double rounding. Supplied values use native scalar codecs. Without `--validation`, typed consumers accept nonmembers; with it, DTO constructors and decoding enforce mapped enum membership, retained container counts and numeric bounds. Standalone scalar typealias codecs keep scalar decoding in both modes; floating `multipleOf` and `uniqueItems:true` stay rejected.

The generator checkout's feature name is `referenced-native-enum-collection-defaults` with fixture `tests/fixtures/http/referenced-native-enum-collection-defaults.json` and consumer `samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedNativeEnumCollectionDefaultsContracts.kt`. `OPENAPI_KT_PREPARE_ONLY=1` is not Kotlin compilation/runtime evidence; general D03 and release gates remain open.

## Nested native enum collection defaults

仅 ordinary DTO direct inline finite List/typed `Map<String, T>` 字段自己的 outer default 属于此 profile，叶为既有 Int/Long/Float/Double/Boolean enum；List/List、List/Map、Map/List、Map/Map 和更长有限路径遵循同一规则。保 native scalar numeric/Boolean wire，不引入 lexical enum wrapper/Unrecognized。每个非 null 注解成员在 validation off/on 都须有正确原 JSON kind、可表示且严格属于原 enum；Float enum `[1]` 不接受 `1.000000001` default，即使映射值相同。supplied 值仍遵循 validation-off 近似与 W_ENUM_NATIVE_APPROXIMATION，或 validation-on 原 mapped membership/bounds/counts。

每层 nullable 独立，optional Kotlin view 不授权 annotation null；原 enum null-member 仍仅 validation-on、最多一个且至少一个非 null scalar，null 计 raw240。empty/null default 先完整验证原树/词表/checks/预算。required（含 nullable）无 initializer，optional explicitly nullable ordinary inline request 保 Patch.Absent/Null/Value，allOf discarded 注解先于覆盖验证，checks 不能被丢弃。missing 普通字段使用 fresh typed default tree，explicit null 和 replacement 保原行为；explicitNulls=false 不提供严格 required-key validation。

保原 member/escaped Pointer，每层 Map 按 source order 验证后分别排序，opaque `$ref`/type/斜线/波浪号 keys 不解释成 schema。Helper 使用优化后的实际 alias/TypeRef，检查 canonical default/type/nullability/check tree 和 frozen lock identity；Map key 重排保持身份，required/Patch 无 initializer 不增加 runtime default identity。原 input8MiB、original/canonical32KiB、tree/work16,384、source48/checked16、enum/slots240 不变。不授权 ref/named/root/wrapper/child/target/chain/default-inheritance、union/hierarchy、其他 terminal、String bounds/pattern 或一般 D03/D04/D05；Int/Long/Boolean enum 列表另见 [native uniqueness](../../../docs/native-enum-unique.md)，保原 membership/bounds/来源和预算，Float/Double 列表另见[有限浮点唯一性](../../../docs/floating-unique.md)：按 mapped 值判重（含 signed zero/rounding ties），拒绝 nonfinite，保原 membership/default/来源与预算。

在 generator checkout 中，`bash scripts/test.sh feature nested-native-enum-collection-defaults` 提供 Data/Plain × validation 四 profile 消费入口；`bash scripts/test.sh models` 的 exporter 将普通模型断言接入既有 `EnumContextContracts.main`（`runEnumContextContracts`），`bash scripts/test.sh http` 通过既有 `runHttpContracts` 对应的 `HttpContracts` suspend registry 直接调用完整 HTTP 消费者 body。native/source 回归为 `go test ./internal/cli -run '^TestNestedNativeEnumCollectionDefaults'`。`OPENAPI_KT_PREPARE_ONLY=1` 只做 native generation/check 和项目准备；分别报告实际编译/runtime 与未验证项，不从准备或其他 feature 的 JVM 结果推导本 profile/platform/minimum-dependency/统一候选/release PASS。

## Gradle sources and serialization

Add the **entire output root** to the intended Kotlin source set, including models, APIs and helpers. CLI paths are relative to its working directory; Gradle paths are relative to the consumer project. For example, after generating into `build/generated/openapi` from the consumer root, use this JVM setup:

```kotlin
plugins {
    kotlin("jvm") version "2.4.10"
    kotlin("plugin.serialization") version "2.4.10"
}

repositories { mavenCentral() }

dependencies {
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.11.0")
}

kotlin.sourceSets.main {
    kotlin.srcDir(layout.buildDirectory.dir("generated/openapi"))
}
```

These versions come from verified JVM smoke runs, not a minimum-version promise. Reuse the consumer project's version catalog, plugin management and dependency/network policy; match the serialization compiler plugin to Kotlin. Configure generation explicitly before compilation rather than assuming an existing generator Gradle task. Compile the new sources using the project's relevant task; for serialization behavior changes, test the affected missing/null/unknown cases.

## Ktor clients

If generated sources are not yet wired into the build, also use [Sources and serialization](#gradle-sources-and-serialization).

HTTP consumers need client core, a platform-appropriate engine, ContentNegotiation and kotlinx-json serialization. The generator's JVM HTTP smoke uses Ktor 3.5.2 with test-only MockEngine; select the consumer's engine and compatible dependencies.

The caller provides, configures and closes `HttpClient`. Generated code installs no plugins, sets no authentication and does not own the client's lifetime. Configure `ContentNegotiation { json(...) }`; use `expectSuccess = true` when non-success statuses should throw. Declared errors produce KDoc and, when needed, an `api.errorBody<T>()` helper, not automatic error-type selection or a `Result` wrapper. The helper preserves cancellation.

Check these boundaries before relying on wire behavior:

- The first operation tag selects the group; no tag means `DefaultApi`. Aggregate names default to `info.title`-derived names and can be overridden by `--api-client-name`. Read the emitted signatures before calling them.
- Default basePath uses the first server URL and appends `/`; an existing trailing slash can yield `//`. Pass an explicit basePath for a different prefix. Relative prefixes still need caller configuration providing a host.
- Local schema-internal Pointers support builtin scalar leaves and finite builtin List/typed Map targets/chains below real `components.schemas` edges. A separate direct property-owned DTO profile binds existing nested classifiers in direct ordinary component object trees (for example `models.OwnerDto.Node`), including nominal recursion and nullable overrides. It creates no new/copy nominal types. Finite internal ref chains to those DTOs retain canonical classifiers, nearest true/false nullable intent and every source hop, even below transparent collection aliases. Nominal barriers permit DTO recursion; transparent-only cycles still fail. Borrowed builtin List/Map targets keep nominal leaves excluded. Direct List/typed Map-owned DTO targets also bind existing nested `OwnerDto.RowsInner`/`OwnerDto.EntriesValue` and root top-level `RowsDtoInner`/`MappingDtoValue` peers; containers are not namespaces and their nullability does not propagate to leaves. Annotation-only component-root aliases may also target those internal DTOs directly or through finite internal chains, and named root-alias chains may end at that binding. Exported aliases and typed uses retain the physical constructor/serializer; nested classifier access uses the canonical owner path, not alias.Child. Finite typed internal chains may also end at direct named ordinary DTOs or existing named aliases to eligible direct/nested DTOs. Consumers use the original physical constructor/serializer, not copied types. Nullable priority is explicit consumer, nearest internal hop, effective named-alias intent, then terminal intrinsic intent; every internal and named-alias edge remains in cached source depth, including false narrowing. Component-root aliases may also cross finite internal chains into these named DTOs/DTO aliases, retaining the same physical constructor/serializer. Their own root intent precedes internal/named intent; an explicit consumer still wins. Forward dependencies, transparent cycles and complete cached source depth use the existing catalog DFS. Nearest nullable intent and all named/internal source hops remain intact. Hierarchy/composition owners have the separate actual-provenance profiles below. Fixed-object extra-key scratch declarations, ambiguous ownership, nonordinary named terminals and shape expansion remain excluded. Explicit nullable true/false overrides the leaf; otherwise its effective nullability is inherited. With `--validation`, supported original leaf checks retain typed DTO/Patch/collection/HTTP/SSE enforcement; raw projections cannot erase them. Named-component behavior, count codecs and resource limits remain unchanged. This is not full reference graph or JSON Schema support.
- Hierarchy-owned direct property/array-item/typed-map-value ordinary constructor DTOs can be internal Pointer targets. Original node/catalog/lexical path bindings wait for actual retained hierarchy Leaf/Nested Model provenance; canonical physical classifiers authorize root aliases/chains/nullable and nominal recursion without guessed names or copied schema. Scratch/discarded/rawconst/ambiguous targets remain excluded; consult `docs/hierarchy-nested-owners.md`.
- Composition-owned constructor DTOs below original allOf/oneOf/anyOf paths use actual retained lexical Model occurrences. Root aliases/typed chains preserve canonical scopes, nullable intent and nominal recursion; primitive-union object wrappers remain peers. Borrowed copies cannot replace the source component's owner, and multiple occurrences under that lexical owner are rejected rather than selecting a variant. First HTTP uses bind only registered retained occurrences. Complex common/shape borrowing, discarded/scratch/rawconst/ambiguous targets remain excluded; consult `docs/composed-nested-owners.md`.
- Borrowed internal List/typed Map segments/root exports may end at original actual object/JSON-kind union classifiers and accepted aliases. Canonical variant dispatch/Unknown/Null carriers and nullable views remain; folded Models/raw/product/hierarchy roles stay separate. See `docs/borrowed-collection-union-leaves.md`.
- Borrowed internal List/typed Map segments/root exports may end at original retained allOf product-union classifiers and accepted aliases. Canonical variants, common fields, dispatch/Unknown and nullable views remain; constructor Models/direct unions/hierarchy source roles stay separate. Access nested variants through their canonical owner. See `docs/borrowed-collection-product-union-leaves.md`.
- Borrowed internal List/typed Map segments/root exports may end at original actual named hierarchy parents/intermediates/leaves and nullable aliases. Canonical inheritance/tag vocabularies/KnownUnknown codecs remain, with independent container/item nullable intent. See `docs/borrowed-collection-hierarchy-leaves.md`.
- Borrowed internal List/typed Map/root exports may end at original named ordinary builtin scalars and supported aliases. Native backing/nullability/checks remain, with transparent full source work and default-off validation; bare codecs do not validate. See `docs/borrowed-collection-builtin-leaves.md`.
- Borrowed internal List/typed Map/root exports may end at original named temporal scalars/aliases. Instant/LocalDate provider ISO codecs and nullability remain; malformed ISO decoding fails, container counts stay opt-in at typed consumers. See `docs/borrowed-collection-temporal-leaves.md`.
- Borrowed internal List/typed Map/root exports may end at original named bigint sources/aliases. Containers and HTTP/SSE roots encode Long numbers; direct object fields/Patch scalar keep decimal strings. Nullability and source work remain. See `docs/borrowed-collection-bigint-leaves.md`.
- Borrowed internal List/typed Map/root exports may end at original named byte/binary scalars/aliases. Nested ByteArray JSON remains signed numeric arrays, standalone Ktor ByteArray roots remain raw bytes/null ASCII. Compare array contents. See `docs/borrowed-collection-bytes-leaves.md`.
- Borrowed internal List/typed Map/root exports may end at original named caller-native scalars/aliases. Supply the real classifier and serializer; caller Long remains numeric, ByteArray nested JSON uses signed arrays, JSON/Instant keep provider codecs. Nullable/source work and typed count checks remain; existing JVM evidence does not verify arbitrary provider layout or Android/KMP.
- Borrowed internal List/dictionary/root exports may end at original named empty raw objects or literal true sources/aliases. Use reified JsonElement codecs; nullable JSON collapses JsonNull, non-null JSON retains it, raw dictionaries keep JsonObject. Existing source warnings and typed count boundaries remain; recursive covers/platforms require separate evidence.
- Borrowed internal List/dictionary/root exports may end at original named complete recursive JSON covers/aliases. Full source graph/SCC suffix and incoming hops remain bounded; JsonElement/JsonObject/provider numeric/null/Patch codecs and original warnings remain. This does not admit partial covers or general shape borrowing.
- Borrowed internal List/typed Map segments/root exports may end at original allOf nodes with unique actual constructor-bearing Models/accepted aliases. Original lexical constructor/serializer, shape-ref alias/target identity and merged nullable intent remain; product unions and general intersections stay separate. See `docs/borrowed-collection-allof-model-leaves.md`.
- Borrowed internal List/typed Map segments/root exports may end at existing named primitive value classes and accepted aliases. Original actual PrimitiveValueClass/InlineValue source, scalar/check identity preserve canonical constructor/rawValue/codec behavior; root/item nullable and container slots remain independent. See `docs/borrowed-collection-valueclass-leaves.md`.
- Borrowed internal List/typed Map segments/root exports may use original retained String enum leaves and their accepted aliases. Canonical entries/fromSerialName/Unrecognized codecs, unknown strings and container/item nullability remain; unique actual vocabulary/source member identity and alias caches authorize them. See `docs/borrowed-collection-enum-leaves.md`.
- Borrowed internal List/typed Map segments and root exports may end at existing actual ordinary DTOs/accepted aliases. Unique retained constructor/serializer ownership, original source/lexical scope and alias node/declaration/cache identity authorize these nominal leaves. DTO recursion is a barrier; prior ref/container work and root/item nullable remain. Other nominal source profiles stay separate. See `docs/borrowed-collection-dto-leaves.md`.
- Explicit component roots may export admitted finite internal List/typed Map builtin-leaf graphs. Original lexical/source graph/ordered path node identity, shared checks/cache/nullable and unique actual root ValueAlias authorize output without a fabricated component/classifier. Nested item/collection/ref work and root-item nullable stay distinct; Patch/HTTP/SSE checks retain their lifecycle, borrowed named/nominal leaves stay separate. See `docs/root-borrowed-collections.md`.
- Finite internal/named paths and root bridges may target original named List/Map/JsonObject collection aliases. Actual ValueAlias/linked backing/check objects and canonical item peers remain source authority; forward plans complete before final snapshots, full incoming/suffix budgets and root/item nullable stay distinct. Patch/HTTP/SSE checks remain, borrowed internal collection roots separate. See `docs/named-collection-reference-chains.md`.
- Finite internal/named paths and root bridges may target original named complete recursive covers. Original reachable graph/source bytes and unique actual JSON alias authorize the source; incoming hops add to SCC suffix and nullablefalse remains path-local. JSON/Patch/HTTP/SSE numeric backing and original warning ownership stay separate from raw/true/native roles. See `docs/recursive-cover-reference-chains.md`.
- Finite internal/named paths and root bridges may target original named raw empty ordinary Models or literal true aliases. Actual unique source identity authorizes JSON backing; nearest nullable/false/full source work and original warning ownership remain. JsonNull survives non-null codecs and collapses through nullable/Patch; recursive cover/borrowed raw leaves retain separate permission. See `docs/raw-true-reference-chains.md`.
- Finite internal/named paths and root bridges may target original named caller-native components/eligible aliases. Original native string/source and generator alias authority are preserved; caller provides real API/serializer. Caller Long is numeric, ByteArray/String Ktor roots raw/null ASCII, JSON/Instant provider codecs unchanged. Borrowed/structured native and raw/true/cover roles/layout assumptions remain separate; see `docs/caller-native-reference-chains.md`.
- Finite internal/named paths and root bridges may target original named byte/binary components/aliases with native ByteArray backing and real format authority. Model/ListMap/Patch/SSE keep signed-array JSON; Ktor root bodies stay raw bytes, including null ASCII responses as bytes. Caller-native ByteArray/borrowed special leaves remain separate; see `docs/bytes-reference-chains.md`.
- Finite internal/named paths and root bridges may target original named bigint components/eligible aliases. They keep native Long/source authority and path-local nullablefalse/fullhop budgets; direct fields/Patch remain decimal strings, root/ListMap/HTTP/SSE remain numeric. Caller Long and borrowed special leaves/other native/JSON roles stay separate; see `docs/bigint-reference-chains.md`.
- Component-root refs may borrow existing ordinary internal builtin scalar leaves directly or through finite internal hops; named root aliases and later typed chains retain original leaf node/Pointer/source owner, checks and native Scalar backing. No fake canonical component/declaration is added. Path-local nullable false/full cached source work and historical direct borrowed checks remain; special/native/JSON and whole borrowed collection profiles stay separate. Consult `docs/root-borrowed-builtin.md`.
- Finite internal/named chains and root bridges may retain ordinary named String/Boolean/Int/Long/Float/Double components and accepted aliases. Native scalar backing, original opt-in checks, actual canonical declaration identity and path-local nullable/false remain; every erased hop counts. Standalone typealias assignment/serde is unchecked, and legacy unmarked non-null aliases remain validation opt-in. Borrowed internal root leaves and special/native/JSON carriers keep separate policies; consult `docs/builtin-reference-chains.md`.
- Finite internal/named chains and root bridges may retain existing named date/date-time/native-date scalars and aliases. Use the canonical Instant/LocalDate parse APIs and ISO provider codecs; nearest nullable/false views and every transparent source hop remain bounded. No DTO/wrapper/companion or nominal depth barrier is introduced. Other builtin/native/raw profiles, temporal enum/const and shape borrowing remain separate; consult `docs/temporal-reference-chains.md`.
- Typed internal/named chains and root aliases may bind an already retained allOf product ObjectUnion at root/nested/ListMap/collection-peer scope. Canonical variant constructors/Unknown/codec remain; own-nullable aliases use nullable reified codecs and explicit false restores the physical non-null view. Product variants borrow the original source without becoming extra owners; ambiguous nested scopes, scratch/discarded sources and internal shape borrowing remain excluded. Source-hop caches and work/product/slot limits stay bounded; consult `docs/composed-union-terminals.md`.
- Finite internal chains and root bridges can end at existing named primitive value classes or their supported aliases. The original canonical constructor/rawValue/non-null serializer/init checks and InlineValue underlying primitive remain, including two JVM slots for non-null Long/Double and one boxed nullable slot. Nearest true/false and all cached source hops survive typed/root aliases; HTTP query unwraps rawValue. Inline/native/temporal/raw projections and shape borrowing remain separate; consult `docs/valueclass-reference-chains.md`.
- Typed internal Pointers, root bridges and finite chains can bind retained plain String-enum vocabularies in actual nested/variant/ListMap/collection-peer scopes, or named enums and their aliases. Use the canonical entries/fromSerialName/Unrecognized owner and serializer; nullable reified codecs accept Kotlin null, false restores the non-null vocabulary codec and every source hop stays bounded. Scratch/discarded/ambiguous/nativefold and borrowed-owner scopes remain excluded; consult `docs/enum-nominal-owners.md`.
- Finite internal/named chains and root bridges can bind actual retained nested, root and collection-peer union classifiers. Use canonical nested variants or native sibling wrappers; aliases do not own copies. Inline nullable union views decode JSON null as Kotlin null, while standalone roots can retain non-null Null variants. Nullable aliases apply a nullable codec and false restores canonical backing; nearest consumer intent and all source hops remain. Scratch/discarded/ambiguous/raw/fold targets and nullable shape borrowing remain excluded; consult `docs/union-nominal-owners.md`.
- Original allOf constructor nodes can also be nominal terminals, nested or named through finite typed chains/root bridges. Only unique retained actual Models authorize the original catalog/node/Pointer; product unions, rawconst/discarded/ambiguous/scratch results remain rejected. Nested allOf effective nullable intent includes original member-derived shape intent, with nearest true/false and all cached source hops preserved. Physical constructors/direct serializers stay non-null. Internal DTO shape borrowing retains its separate policy. Explicit nullable ordinary allOf component roots with actual constructor Models now follow the intrinsic nullable DTO codec policy: canonical companion stays non-null, reified nullable refs/aliases handle null, nearest false restores physical backing, and request Patch/HTTP/SSE keep original source checks. Empty/const/product/count-only/unmarked member-null roots and nullable shape borrowing remain excluded; consult `docs/allof-nominal-terminals.md` and `docs/nullable-allof-dtos.md`.
- Finite internal reference chains and component-root bridges may also end at actual named hierarchy parents, intermediates or leaves and their supported aliases. Canonical constructors/serializers, nearest true/false intent and all internal/named cached source hops remain; nominal recursion is allowed, transparent cycles reject. Final Hierarchy declarations authorize named terminals. Hierarchy-owned nested targets, allOf/union/ambiguous borrowed owners and shape expansion remain separate; consult `docs/hierarchy-reference-chains.md`.
- Typed discriminator hierarchies support single-child fixed explicit levels, missing direct/ancestor tag fields as generated typed mapping body constants, and varying ancestor enum values. Each generated tag keeps the original ancestor enum type, encodes its selected edge value and occupies one serialized slot; it does not satisfy absent source required declarations. Actual enum fields stay required non-null constructor arguments. Per-layer Unknown preserves raw JSON, and root versus inner explicit routes can differ for alternate ancestor values. Multiple parents/inline/nullable hierarchy members and general internal nominal ownership remain separate; consult `docs/hierarchy-fields.md` in the source checkout.
- An inferred hierarchy level may promote several common finite string const/enum fields. OR within each field and AND between fields select const-only branches first, then enum branches in source order; identical rules first-claim, invalid selected payloads fail. Every promoted field retains its typed enum and Unknown stand-in. Nested explicit levels may reuse the same ancestor tag enum, with one override and their own route mappings; generated missing bodies use the nearest matching edge. Root versus inner roundtrip can differ. Multiple parents/inline/nullable hierarchy and general nominal ownership remain separate; consult `docs/hierarchy-dispatch.md`.
- Named root aliases may add nullable intent to actual non-null hierarchy parents or leaves. Their nearest true/false intent and explicit consumer override retain the canonical classifier; nullable reified codecs decode JSON null as Kotlin null, while the direct canonical serializer remains non-null. Unknown raw payloads and mapping routes stay unchanged. Intrinsic nullable hierarchy roots/intermediates/leaves have their own physical-class/typed-intent contract: their refs/aliases inherit nullability and false can narrow; recognized object mappings still use non-null child serializers. The separate nullable-member contract supports pure null members, explicit nullable refs and aliases as members, and nullable aliases in physical child mappings. Member intent propagates through original alias dependencies to nullable parent typed refs; nearest false can narrow. Physical inheritance/codec and Unknown remain non-null; exact oneOf null overlap is not validated. Inline/multipleparent/borrowed ownership remain open; consult `docs/nullable-hierarchy-aliases.md` and `docs/hierarchy-null-members.md`.
- Compatible direct or named dictionaries shared by object-union variants expose Map<String,T> or JsonObject base properties. Actual named value classifiers/codecs and each branch's checks remain; incompatible value types keep variant-only fields. Common container nullability joins existing presence, separately from value nullability. Unknown preserves rawElement: non-null dictionary access throws the existing unavailable-property error, nullable access returns null. Inline nominal values requiring scratch declarations and general common object/union fields remain separate; consult `docs/union-common-dictionaries.md` in the source checkout.
- Compatible references shared by object-union variants may expose the actual generated DTO, native/object union or hierarchy classifier as a common base property. Original alias/source/unique declaration ownership, codecs and nearest nullable/false/presence policy stay canonical; distinct classifiers retain variant-only access. Outer Unknown keeps rawElement and non-null common access unavailable, nullable fallback remains null; inner Unknown/Null carriers use their original union codec. Complete JSON covers keep transparent backing. Inline objects requiring a shared scratch owner remain excluded; consult `docs/union-common-nominals.md`.
- Object-only allOf graphs may contain supported object oneOf/anyOf members. Generated sealed variants combine source fields in member order (own fields last), with required-field union and bounded branch products; product dispatch can conjoin multiple pinned fields. Unknown preserves raw JSON. Nullable/primitive/dictionary/discriminator-hierarchy intersections and operation-specific inline projections remain separate; consult `docs/union-allof.md` in the source checkout.
- Complete OpenAPI 3.1 JSON-kind cover roots may reference themselves or other complete covers through array items/object values, including finite annotation-only named-ref chains. They export JsonElement aliases; reified/backing codecs handle JsonNull and the caller's numeric behavior. Every reachable root must be a complete six-kind cover. Root aliases inherit nearest nullable intent or explicitly narrow with false; non-null JsonElement still contains JsonNull. Nullable codecs decode JSON null to Kotlin null. This does not make partial unions, ref-only cycles, constrained covers, nullable recursive edge siblings or general nullable graphs supported; consult `docs/recursive-json.md` in the source checkout.
- Simple component `x-kotlin-value-class: true` generates a nominal `@JvmInline` wrapper with `rawValue`. Construct it explicitly; parameters unwrap its raw value, JSON bodies/responses retain its serializer. Root String wrapper bodies are JSON-quoted rather than bare-String raw text, so toggling this flag requires a call-site/wire migration. Only the simple primitive subset is supported; supported String length/numeric bounds enforce init and decoding checks, but direct DTO scalar fields can enforce the same bounds with `--validation`; direct String slots also support the declared portable pattern grammar with substring matching. With `--validation`, supported primitive component bounds and scalar-ref chains propagate to typed DTO/Patch/list-map consumers, HTTP preflight/postflight and SSE event data. Transparent alias assignment/independent serialization remains unchecked; use intrinsic value classes when that guarantee is needed. General schema validation and full JVM/ECMAScript regex remain unsupported. DTO checks run in init (construction/copy/automatic decoding), skip nullable null/Patch Absent/Null, and preserve valid primitive field types/wire. Raw projections that would erase constrained source fields are rejected. Scalar-ref and directly bounded primitive collection elements retain checks across typed nested list/map/nullable/Patch and HTTP/SSE consumers. Without container bounds, empty collections pass and map keys are unchecked. Opt-in typed List min/maxItems and typed Map/open JsonObject min/maxProperties check current size independently of elements, with nonnegative Int-range integer bounds; null/Patch presence and typed HTTP/SSE consumers follow the same rules. Opt-in uniqueItems:true supports homogeneous native String/Int/Long/Boolean lists (nullable leaves/transparent scalar refs), exact parsed equality without Unicode normalization and at most one null; sorted-copy checks do not mutate input. Boolean false is a validated effect-free no-op on otherwise supported arrays, including non-scalar elements; omitted opt-in, malformed/wrong-kind keywords and unsupported true equality projections are rejected. Float/Double, nested/nominal/enum/const/native/date/byte equality, fixed DTO wire-property counts and direct bounded HTTP scalar slots remain unsupported; independent collection serializers and mutable post-construction changes are unchecked. Supported numeric bounds include3.0Boolean exclusives (true needs basebound/false noop),3.1numeric exclusives and positive representable nonzero multipleOf; floating checks are finite/strict mapped-IEEE remainder, not decimal multiples or epsilon tolerance. Caller requestBuilder overrides are unchecked. Treat bound failures from server payloads as decoding failures; generated code does not silently repair the value.
- The final requestBuilder callback can override requests. Clear `url.parameters` explicitly when replacing all query parameters; Ktor `url(...)` does not necessarily remove generated query values.
- Ktor uses raw text for root String bodies rather than automatically adding JSON quotes. In the tested JSON-media configuration, an optional body's default null sends JSON `null`; selected raw media use the separate absence contract below.
- Shared components retain ordinary model defaults rather than acquiring Patch semantics when used in requests. Request-only presence/Patch, root request unions, response unions and parameter projections have distinct policies. Review corresponding `W_HTTP_*` warnings and emitted serializers rather than treating them as complete JSON Schema validation.
- Structured parameters may use `toString()` instead of recursive JSON/query encoding. Verify actual URLs, headers and bodies with a controlled test using the target engine.

For generated SSE methods, also install Ktor's `SSE` plugin on a compatible caller-owned engine. These methods return cold Flow: each collection opens a session; creating the Flow does not send a request. The generated client/API can take a trailing Json argument (default ignores unknown payload fields), separately from ContentNegotiation. Known event data is JSON-decoded; unknown events are skipped. Keep the default no-reconnection policy unless the consumer explicitly owns a different one. Supply `lastEventId` for a cursor; remove/set rather than append the header when overriding it in the callback.

Compile the fresh client and test the relevant URL/header/body/response behavior with a controlled engine or loopback server. SSE changes additionally need fragmented-stream and cancellation tests; a buffered ordinary MockEngine response is not streaming evidence. Model-only tasks do not need HTTP tests.

For context-specific semantics, inspect emitted serializers and consult the matching source-checkout contracts: `docs/http-client.md`, `docs/sse.md`, `docs/request-presence.md` and `docs/model-semantics.md`. Request unavailable evidence only when it is needed to resolve the task.

## Generator repository smoke checks

When verification in the generator checkout is needed, `bash scripts/test.sh models` and the HTTP-specific `bash scripts/test.sh http` build fresh CLI output and compile/run offline. They require Go, Python 3, Gradle/JDK and cached dependencies. These scripts are not consumer-project tasks or prerequisites for ordinary CLI use.

The repository smoke is verified with Gradle 9.3.1 / JDK 21 (JVM 17 bytecode). It does not establish Android/KMP, every engine or minimum dependency versions. Report missing tools or offline caches as unverified prerequisites; use the consumer's actual build and dependency policy for consumer integration.

## Positional arrays with enum references

OpenAPI3.1 nonempty `prefixItems` remains a List approximation. Finite local references to already admitted plain String enum declarations (named/internal/alias) retain typed enum Lists only when original terminal source and effective nullability match. Different original enums, enum versus scalar or differing nullable views become `List<JsonElement>`; explicit admitted scalar/enum `items` wins after every prefix validates. No positional enums are synthesized. Use ordinary list construction and reified serializers, with the physical enum's known/fromSerialName/Unrecognized API. Nullable enum elements distinguish JSON null from String `"null"`.

Read `W_TUPLE_APPROXIMATION`: Lists do not preserve positions/tuple length, typed Lists can narrow unrestricted tails, raw Lists do not check positional membership and String enums remain open. Direct inline enums follow [their own profile](#inline-enum-tuples); bounds/default-bearing tuple members, valueclass enum and general graphs retain separate admission. String format/native metadata follows the inline enum tuple profile below, including actual named lexical enums and pure aliases. Boolean tails follow the [Boolean tail guidance](#boolean-tuple-tails). Keep original source/depth/work/ownership diagnostics; do not replace a rejected member with a plain String to suppress semantics. Compile freshly generated sources in the actual target; JVM feature evidence does not prove KMP or Android.

## Merged primitive union roots

With explicit `--merge-identical-models`, already validated primitive JSON-kind roots with no object peer may share one canonical root only when ordered kinds and null-carrier policy match. Old root and all public sibling-wrapper names remain aliases, so old wrapper constructors/getters/Data copy/serializer calls and Null singleton names still work. Plain retains no-copy behavior. Actual class, descriptor and cross-old-name equality becomes canonical; rebuild compiled callers. Collapsed nominal HTTP overloads can get a new suffix while existing natural method names stay reserved: verify affected calls against fresh sources.

The option does not enlarge dedup/hoist/locked-shape eligibility, add Unknown fallback or enforce exact oneOf count/constraints. JSON-kind/Long-versus-Double/null semantics and original warnings remain. Exclusions and opaque provider references to root/wrapper/codec protect the physical source. Object peers, object unions, hierarchy/nested/general-recursive optimization stay separate. Preserve output/name locks on failures; disabling merge restores distinct generated classes through ordinary generation.

## Positional arrays with DTO references

OpenAPI3.1 `prefixItems` or explicit `items` may reference an already admitted plain ordinary DTO constructor through finite local named/internal aliases. Same original source/effective nullable intent yields a typed List; distinct sources, DTO/scalar or nullability mismatches yield `List<JsonElement>`. Explicit tails validate every original prefix and alias first. Use the existing physical constructor/serializer and canonical owner for nested classifiers, including existing field defaults, checks/counts and directional views; no positional DTO is generated. Existing nominal recursion stays on the original codec.

Preserve `W_TUPLE_APPROXIMATION`: no positional or tuple-length enforcement, raw projection does not check object fields, and typed Lists can narrow unrestricted tails. Existing typed-consumer counts remain separate from standalone alias codecs. Independently supported original component-own factories use the target-default profile below. Own tuple-default roles, nominal uniqueness, composed/union/hierarchy/dictionary/raw targets and direct inline positional DTO declarations remain excluded from this ordinary-DTO profile. Source48/work16384/checked16/slots240 and existing annotation budgets remain. Original source/unique constructor/alias diagnostics are safety checks; keep output/locks on refusal and compile fresh sources in the target consumer.

## Form URL encoded requests

When JSON media is absent, selected `application/x-www-form-urlencoded` accepts only an inline ordinary object body (required or optional) with required or optional non-null direct String/Boolean/Int/Long/Float/Double source fields. Optional constructor null omits that key; empty/false/zero remain present. Existing builtin constraints retain validation opt-in; root/field grammar is closed. Original wire names and Ktor UTF-8 form encoding are shared by HTTP and SSE, followed by the final requestBuilder callback. References/defaults/schema-nullable/structured fields, Patch form presence and explicit encoding remain outside this profile. See `docs/form-request-media.md`; `bash scripts/test.sh feature form-request-media` provides four JVM profiles and the full HTTP registry invokes the consumer. Do not claim other targets from JVM evidence. Existing JSON precedence and invalid discarded-schema checks remain.

## allOf constructor tuple references

Finite named/internal refs in prefixItems/items can target actual original retained allOf constructor Models. Same source/effective nullability yields typed List; distinct/mixed sources yield JsonElement and explicit tails validate every prefix first. Existing constructor/default/check/owner/nominal recursion and both tuple/allOf warnings remain. New tuple snapshots share cumulative16384 node/edge/shape-ref work and8MiB original JSON bytes across distinct targets; repeated target reuse, old source48/per-target16384/checked16/slots240/default budgets remain. Constructorless aliases/raw/const-only/product/hierarchy targets and new root counts/default/directional/nominal uniqueness permission stay separate. See `docs/primitive-tuples.md#referenced-allof-constructor-members-and-tails` and `bash scripts/test.sh feature tuple-allof-references`; report JVM evidence separately from PREPARE_ONLY/platform claims.

## Tuple target default factories

Existing scalar/temporal/plain enum/ordinary/allOf tuple targets and eligible named aliases may keep independently admitted original component-own default factories. Only the marked original catalog root/pointer grants annotation permission. Factories remain separate; a target default never initializes a position/List element or required input, and invalid target/alias literals still reject before discarded prefix/tail projection. Original strict enum membership, source/nullable/required/Patch and default/graph budgets remain. Shared borrowed enum source validation also accepts eligible component-own factories for ordinary collection refs. No new positional/internal/unsupported factory grammar is admitted. See `docs/tuple-target-defaults.md` and `bash scripts/test.sh feature tuple-target-defaults`; distinguish JVM/fresh helper/wire evidence from PREPARE_ONLY and other targets.

## Raw scalar request media

After JSON/form preference, selected non-null text/plain String and octet-stream ByteArray schemas for required/optional bodies delegate actual raw text/charset/bytes to Ktor with original validated media spelling/parameters. Empty values and literal null text are payloads. Optional parameter null means absence; schema-nullable raw payloads remain rejected. Original shared scalar projections/warnings/checks retain their boundaries and all discarded media schemas still validate. HTTP/SSE share emission; callback runs after generated body, typed validation before callback/dispatch. See `docs/raw-request-media.md` and `bash scripts/test.sh feature raw-request-media`; report fresh JVM UTF8/ISO8859/rawbytes evidence separately from other targets.

## Scalar tuple field own defaults

Ordinary DTO fields with accepted prefixItems may own a direct scalar List default selected by uniform prefixes or explicit accepted scalar items tail. Reuse original scalar/UUID/temporal literal checks and typed helpers; validate every original prefix before raw/tail/default handling, and default before required/Patch suppression. Approximation does not enforce positions/tuple length. New own-field role keeps16384-value-node/32KiB JSON budgets and source48/work16384/checked16/slots240. Mixed raw/ref-selected/root/position/nominal defaults and prefix-only enforcing uniqueness remain separate; explicit scalar tail uses its existing unique source policy. See `docs/tuple-field-defaults.md` and `bash scripts/test.sh feature tuple-field-defaults`; distinguish JVM evidence from other targets.

Cartesian object-union consuming-field defaults retain actual constructor String/native enum own annotations. Supply every required conjunction pin; own defaults are not applied before selection. Numeric pin tokens are preserved in the initializer when necessary to keep the intended subtype. Regeneration can correct old Unknown defaults, changing subtype/equality and `encodeDefaults=false` omission; native floating runtime dispatch remains approximate.

Optional raw text/octet request parameters default to null for absence, including SSE; non-null empty text/bytes retain raw Content-Type. Schema nullable remains unsupported. Use `feature raw-request-media` for the combined required/optional wire contract.

For selected optional form bodies, omit the DTO or pass null for no content/Content-Type. A present DTO must supply all existing required scalar fields; empty Strings still emit `key=`. HTTP/SSE use the same omission guard and final callback. Field/Patch/schema-nullable grammar does not expand.

With validation enabled, existing ordinary allOf object default roles admit complete repeated plain String constraint groups. Earlier own annotations stay local; winning and supplied outer values satisfy the retained conjunction. Missing/required/Patch/source/budget/codec contracts remain. Use `feature allof-string-defaults` for Data/Plain/fourJson/optimized/width240/HTTP/SSE checks; numeric/ref/enum/unchecked duplicate roles remain separate.

Eligible finite inline List/typed Map component roots with ordinary DTO, direct named plain DTO or bare ordinary allOf leaves can own opt-in top-level `<Alias> default` factories. Use the actual emitted alias/serializer; referencing fields do not inherit root defaults. Existing nullable, required/Patch, own-property source and budget rules remain. `feature object-collection-factories` verifies four fresh profiles, optimized/locked output, wide/deep and HTTP/SSE consumers; element alias/internal/union/hierarchy and child-default roles stay separate.

Numeric enum component-own factories also accept no format, retaining Int (`integer`) or Double (`number`). Defaults strictly match original vocabulary/kind before rounding; nullable native roots/null members retain validation-on admission, and root factories do not initialize referencing fields. Use `feature unformatted-native-factories` for four profiles, optimized/locked output, IEEE/negative-zero/extrema/240-members+slots and HTTP/Patch/SSE consumers.

With validation, ordinary allOf outer defaults can consume complete repeated plain Int/Long/Float/Double groups already admitted by primitive aggregation. Integer checks retain exact original bounds/factors; Float/Double keep mapped comparisons and floating multipleOf-default exclusions. Original own annotations validate before override, the winning annotation satisfies the conjunction, and omitted members keep actual constructor defaults. Required/Patch/no-inheritance and resource caps remain; `feature allof-numeric-defaults` verifies Data/Plain, optimized/locked output, four Json settings and HTTP/Patch/SSE.

OpenAPI 3.1 direct scalar prefixItems component roots may own independent List default factories, including pure component aliases. Uniform prefixes or admitted scalar items tails retain their existing List type/codec and approximation; original prefix sources validate before empty/null defaults. Root factories are not inherited by fields; nominal/raw/member-own tuple factory roles remain separate. Use `feature tuple-root-defaults` for Data/Plain×validation, optimized/locked output, four Json settings, temporal/IEEE/Long/240field and HTTP/Patch/SSE consumers.

Selected flat form DTOs also accept optional non-null builtin fields: Kotlin nullable defaults mean omitted form keys, retaining empty Strings/false/zero values and original property order/wire names. A supplied all-null DTO sends an empty form with media, distinct from an absent optional body. Explicit schema-null/default/reference/Patch states stay excluded; use `feature optional-form-fields` for four profiles, optimized output, 240 fields and HTTP/live SSE consumers.

Flat form fields also accept direct Float/Double, including unformatted number→Double, using mapped runtime scalar toString text and Ktor UTF-8 encoding. Negative zero/exponent/rounding and unconstrained NaN/Infinity text remain; admitted numeric constraints retain opt-in finite/bounds checks. Optional null omits a key; schema-null/default/caller-native/Patch roles remain excluded. `feature floating-form-fields` checks four profiles, optimized output, Double JVM240slots and HTTP/live SSE.

Direct plain String/native scalar enum form fields retain actual local String declarations and mapped builtin types. Encode String serialName (including Unrecognized), native scalar text with opt-in membership/checks, optional null omission and independent body absence. Selected raw vocabularies cap240 under both validation modes; formatted String/ref/alias/default/schema-null/Patch roles remain separate. Optimized DTO aliases use the actual retained owner for nested enum access. `feature enum-form-fields` verifies four profiles, actualowner/240vocab, Long/IEEE and HTTP/live SSE.

Large repeated collection traversal checks may share private typed methods after a 12KiB inline-source planning threshold. Public types/constructor/copy/serde/checks/wire and persisted formats remain; no state/casts or schema cap changes. `feature wide-deep-validation` checks List<Int>200×16 and240×12 in Data/Plain×8optimization combinations, factory/copy/Json/HTTP/SSE, with complete 128KiB compilation partitions and existing heap. JVM UTF8 generic-signature limits remain separate: do not claim full width×depth Cartesian or cross-platform acceptance from those profiles.

Retained DTO constructors whose guaranteed minimum generic-signature bytes exceed JVM65535 UTF8 capacity fail with E_MODEL_LIMIT and original component/nested/HTTP schema Pointer before output. Known inline containers/builtin boxing only are counted; unknown aliases/names/wildcards are omitted, so passing is not a complete backend guarantee. Feasible wide-deep profiles/public types/wire/caps remain. Native generate/check/validate preserve old output. See `docs/jvm-signature-limits.md`; broad exact-signature and cross-platform work remains separate.

Selected flat form DTOs also accept direct UUID/date/date-time fields using existing Uuid/LocalDate/Instant types and canonical toString text through Ktor UTF-8 encoding. Optional field null omits its key; optional body null omits content. UUID checks remain validation opt-in, temporal bounds/ref/default/schema-null/native/formatted-enum roles stay excluded, and existing budgets hold. See [standard form fields](../../../docs/standard-form-fields.md); run `bash scripts/test.sh feature standard-form-fields` for fresh JVM profiles.

Validation-enabled ordinary allOf outer defaults also support complete repeated direct UUID fields already eligible for retained primitive checks. Length/pattern checks use canonical Uuid text, original own/winning/outer annotations validate with their source Pointers, and required/Patch/no-inheritance/shared budgets remain. See [allOf UUID defaults](../../../docs/allof-uuid-defaults.md); `feature allof-uuid-defaults` is the fresh JVM route. Other source/format/enum/reference roles remain separate.

The JVM constructor lower-bound guard also counts demonstrated variance bytes for entire direct builtin List chains. It rejects204×List16 while retaining compiled203×16 Data/Plain; uncertain alias/Map/mixed/nullable-element/copy/name metadata remains outside this narrow proof. Run `feature jvm-signature-limits`; see [known List variance limits](../../../docs/jvm-list-variance-limits.md). Public types, source Pointers, existing caps and old-output preservation remain.

A post-transform signature guard counts the known parameter lower bound plus the ASCII package prefix of retained Data copy return types. Final Plain models are exempt from the Data copy proof. Failures retain original Pointers and old output; diagnostic Plan metadata grants no source permission or shape/lock identity. See [Data copy signature limits](../../../docs/jvm-copy-signature-limits.md), and run `feature jvm-copy-signatures`. Exact aliases/names/other metadata and wider platform feasibility remain separate.

Already-bound root bridges through internal chains ending at original direct named non-null ordinary DTOs can contribute an allOf shape. Fields keep original nodes/Pointers, real catalog/owners, defaults/checks, required/Patch and full cached source height. Nullable/composed/internal lexical/union/hierarchy/count/raw targets remain excluded from this role; existing caps and failure preservation hold. See [bridge intersections](../../../docs/bridge-intersections.md), and run `feature bridge-intersections`. Other nominal and typed bridge contexts retain separate contracts.

The named ordinary DTO root-bridge allOf role also accepts intrinsic target nullable intent and nearest alias/member true/false overrides. Discovery traverses original ref nodes without granting ownership/annotation authority; existing source/work/depth/slot caps and final catalog validation hold. Use canonical non-null companion codecs or nullable adapters according to the typed view. See [nullable bridge intersections](../../../docs/nullable-bridge-intersections.md), and run `feature nullable-bridge-intersections`; other endpoint and shape roles remain separate.

AllOf can borrow fields from actual ordinary internal DTOs under direct ordinary owners through direct/finite-chain/root-bridge refs. Original nodes/Pointers/canonical owners and final unique-constructor proof remain; no internal component identity is synthesized. A shared16384work/8MiB capture budget charges distinct overlapping targets; existing source/checked/slot/default caps, nullable/Patch and failure preservation hold. Composed/hierarchy/count/root-default/scratch roles remain excluded. See [internal DTO intersections](../../../docs/internal-dto-intersections.md), and run `feature internal-dto-intersections` for fresh nullable/non-null Data/Plain profiles.

AllOf may also borrow a unique actual ordinary nested property/collection DTO under an allOf/union lexical owner. Original source/canonical owner and deferred unique-occurrence proof remain; variant roots/common/hierarchy/count/root-default/ambiguous/scratch roles stay excluded. Existing shared capture and source/checked/slot caps, defaults/Patch/wire and failure preservation hold. See [composed-owned DTO intersections](../../../docs/composed-dto-intersections.md), and run `feature composed-dto-intersections`; construct canonical nested types according to actual generated aliases.

AllOf may also borrow an actual ordinary nested DTO directly owned by a named plain-source hierarchy leaf, via internal chains/root bridges. Original unique-owner/capture proof, nullable/Patch/default/recursion and all prior caps remain; parent kind/common fields are not part of the nested schema or Consumer slots. Root/common/inline/composed-leaf/count/root-default/ambiguous roles remain separate. See [hierarchy-leaf nested DTO intersections](../../../docs/hierarchy-dto-intersections.md), and run `feature hierarchy-dto-intersections`. Existing dispatch and persisted formats are unchanged.

[Named composed hierarchy-leaf nested DTO intersections](../../../docs/composed-leaf-intersections.md) extend D01/D02/F04/F07 to actual ordinary property/collection DTOs under named ordinary allOf leaf sources. Original catalog/Pointer and unique retained owner/capture proof remain; parent tags/common fields do not become payload fields or Consumer slots. Nullable/Patch/default/check/recursion/warnings and shared16384work/8MiB/source48/checked16/JVM240 limits hold. Root/common/inline/ambiguous/discarded/count/root-default roles stay separate; existing dispatch/API/wire/locks remain stable. Broader graph/platform completion remains open.

[Nested allOf DTO intersection targets](../../../docs/allof-target-intersections.md) extend D01/D02/F04/F07 to actual unique nested ordinary allOf constructors under direct ordinary object owners. Original composed source/alias snapshots and final retained owner proof remain; shared16384work/8MiB capture, source48/checked16/JVM240 and default/value caps hold. Nullable/Patch/default/check/recursion/override/warning/API/wire/locks keep their contracts. Composed/union/hierarchy lexical owners, products/count/root-default targets and broader graph/platform completion remain separate.

[Composed intersection targets across lexical owners](../../../docs/composed-target-owners.md) extend D01/D02/F04/F07 to actual unique nested ordinary allOf constructors under allOf/union owners and named plain/allOf hierarchy leaves. Original source/alias snapshots, final unique owner/capture proof and parenttag isolation remain; shared16384work/8MiB/source48/checked16/JVM240 and default/value limits hold. Nullable/Patch/default/check/recursion/override/warning/API/wire/locks keep their contracts. Ambiguous/discarded/common/inline/product/count/root-default roles and broader graph/platform completion remain separate.

[Ordinary allOf directional HTTP DTOs](../../../docs/allof-directional-models.md) extend D07/A02/A03 to operation-owned request/public-response views from already validated winning original fields and merged required entries. Private analyzer source metadata stays outside Model identity; copies gain no original-source authority or new graph admission. Later overrides, hidden/discarded validation, required/Patch/default/check/recursion and original Pointers remain; shared DTO/API/wire/locks stay stable. Hierarchy/union/product/raw/count and changed-nominal default boundaries, source48/checked16/JVM240/clone/default/value budgets and broader platform/graph completion remain separate.

[Mapped scalar-field DTO uniqueness](../../../docs/dto-unique.md) extends D05/M03/M07/M12/V05 to actual ordinary named/original-inline DTO constructors with direct builtin scalar getters. Typed API/codec/nullable/default/required/Patch behavior remains; constructor initializer-aware default keys match runtime Float/UUID/Instant equality and retain original value Pointers. Actual source/member/constructor proof and independent shared16384work/8MiB source plus16384 default-key work budgets remain; enforcing getter/type fingerprints use a bounded cached reference while provenance/default keys stay private. General nested/enum/alias/composed/hierarchy/projected/count/native roles and broader platform/graph completion remain separate.

Direct String/native enum fields in ordinary DTO `uniqueItems` arrays use the [DTO enum-field profile](../../../docs/dto-enum-unique.md) under validation: actual serialName/native mapped getter equality, constructor defaults and retained source/enum identity. Nested/reference/composed/projected field roles retain separate boundaries.

Direct builtin scalar const bodies in ordinary DTO `uniqueItems` arrays use the [const-body profile](../../../docs/dto-const-unique.md) under validation. Keys read actual decoded body getters; scalar const decoding still accepts other mapped values. Constructor/copy/Patch APIs and original default grammar remain.

Direct JsonNull constructor fields and null const bodies in ordinary DTO `uniqueItems` arrays use the [JsonNull getter profile](../../../docs/dto-null-unique.md) under validation. Keys compare actual getter states; JsonNull/Kotlin null nullable-codec wire collapse and original field/default/Patch rules remain.

Direct inline List/typed Map builtin-scalar fields in ordinary DTO `uniqueItems` arrays use the [collection getter profile](../../../docs/dto-collection-unique.md) under validation. List order, Map keys and mapped Float/UUID/Instant equality remain; outer and getter collection edges share checked16 and default trees share key-work limits. References/aliases/enum/JsonNull/DTO/raw leaves retain separate boundaries.

Direct explicit/inferred null const members use the [ordinary object-default profile](../../../docs/null-const-object-defaults.md) in already admitted enclosing annotations. Supplied values must be actual JSON null; body getters stay non-null JsonNull, required/Patch/factories/no inheritance and budgets remain.

With `--validation`, ordinary DTO `uniqueItems` keys also admit already-supported builtin List/typed Map references through finite named/internal/bridge paths, including independently allowed field-own defaults. Map order and mapped Float/Double zero equality follow actual getters; nullable/required/Patch and no default inheritance remain. Target/alias defaults, reference leaves within the target tree and enum/temporal/nominal/raw/const/native/valueclass collection leaves are separate roles. Keep generator diagnostics and original resource limits.

The DTO collection reference uniqueness role also admits existing `uuid`/`date`/`date-time` mapped leaves under validation. UUID spelling and equal Instant offsets normalize through typed getter equality; strict own annotations, original source/format/alias/backing authority, nullable/required/Patch/no inheritance and old budgets remain. This grants no temporal bounds, target/alias factories or additional leaf/source roles.

Validated ordinary DTO collection reference keys also admit already-supported numeric/Boolean enum leaves. Original vocabulary/kind/precision and strict own defaults precede mapped equality; actual alias/backing/check objects retain authority. Standalone aliases and unchecked scalar approximation keep their contracts, and borrowed native enum references remain rejected without validation. String enum/other leaf/source roles and all old budgets remain separate.

With validation, ordinary DTO uniqueness supports plain String enum collection reference getters using runtime `serialName` equality and strict field-own defaults. Internal inline-enum collection references reuse their actual lexical enum declaration. See [String enum collection references](../../../docs/dto-string-collection-reference-unique.md) for boundaries and migration.

Validated ordinary DTO uniqueness also supports existing finite List/Map getters with scalar/plain enum reference leaves. Ordinary inline fields gain strict own defaults for these source trees; unchecked native enum inline-default admission stays closed. See [collection reference leaves](../../../docs/dto-collection-reference-leaves-unique.md) for actual source/getter authority and boundaries.

Validated ordinary allOf DTO uniqueness uses actual inherited getters and the original complete winning-field/required/default view. Existing aggregation, nullable and source/check identity rules remain. See [allOf DTO uniqueness](../../../docs/dto-allof-unique.md) for boundaries, tuple approximation and migration.

With validation, admitted ordinary allOf default walks accept complete same-kind/format/nullable primitive override groups when earlier occurrences have no bounds. Original own annotations remain strict; outer values and omissions use the winning checks/default, without new emitted aggregation or default inheritance. See [allOf default overrides](../../../docs/allof-default-overrides.md) for supported scalar and source roles.

For validation-on positional arrays referencing existing numeric/Boolean enum sources, see [native enum tuple references](../../../docs/tuple-native-enum-references.md). Same original source/nullability yields typed Lists; distinct or mixed sources remain JsonElement Lists. Codecs keep existing standalone-alias versus typed-consumer check boundaries, approximation warnings and independent default roles. Unchecked native enum references remain excluded; inline enums use [their own profile](#inline-enum-tuples), and general tuples remain separate.

For [nullable recursive JSON cover edges](../../../docs/nullable-recursive-cover-edges.md), graph item/value nulls remain JsonNull inside JsonElement; exported nullable aliases/typed consumers keep nearest intent and may decode Kotlin null. Original SCC/source budgets, source node/member authority and approximation warnings remain. This is not general nullable recursion.

[Literal false component roots](../../../docs/false-component-roots.md) are uninhabited sealed types with rejecting serializers. Reified JSON calls compile; every root JSON value rejects, while an eligible nullable pure alias may decode/encode Kotlin null. Data/Plain/optimization preserve nominal codecs. Typed consumer and general false roles remain separate.

[Ordinary named fields referencing false roots](../../../docs/false-reference-fields.md) distinguish strict optional absence from explicit null: default Kotlin null always omits and every supplied JSON value rejects. Required non-null arguments remain uninhabited; nullable source aliases retain existing shared default-null policy. Do not substitute raw JSON or remove a false field to bypass a rejected role.

[Named collections with false-reference leaves](../../../docs/false-collection-references.md) retain typed empty List/Map values and actual rejecting leaf codecs; nullable leaf aliases admit only null values. Original finite named collection source and actual alias/backing identity are required. Direct literal false, defaults, tuple/internal/composed and inline collection roles remain separate.

[Own defaults on named false collections](../../../docs/false-collection-defaults.md) emit root-owned top-level factories using actual collection codecs. Empty/nested containers and nullable false leaves validate strictly; consumers do not inherit annotations. Keep child/field/wrapper/object-member roles separate and preserve original source/value budgets and failure output.

[Own null factories on nullable false aliases](../../../docs/false-alias-defaults.md) retain the actual rejecting root and nullable wrapper. Only null is a valid own annotation; each factory/default is independent. Consumers keep existing presence/codecs and acquire no default from their target. Preserve original source/declaration/initializer authority and budgets.

[Own null defaults on direct false-reference fields](../../../docs/false-reference-field-defaults.md) require original logical nullable intent. Required Kotlin parameters keep no initializer; optional ones initialize null. Json missing/null behavior follows existing explicitNulls policy. Target factories are not inherited, and strict non-null optional false fields keep their separate absence codec.

[False-reference members in ordinary object root defaults](../../../docs/false-object-default-members.md) use the retained getter codec and logical source intent. Nullable members may be null; non-null optional false members must be absent, and literal required keys stay required. Companion factories are root-owned and do not relax wire counts or grant consuming-field/nested/wrapper/default roles.

Direct ordinary named shared DTO-reference fields may own object defaults using actual retained false-reference target getters, with no component default on consumer or target; required annotations validate before initializer suppression, and optional annotations decode fresh target state. Pure target aliases keep this field behavior. Source/value/depth/work limits and strict false codecs remain; nested/container/composed/directional/inline/internal roles and general defaults stay separate. See [DTO field defaults with false members](../../../docs/false-dto-field-defaults.md). Validation-on [target property counts](../../../docs/false-dto-field-counts.md) also validate original supplied keys against the actual retained target serializer; caller JSON omission and runtime wire counts remain. [Independent target root and consuming field annotations](../../../docs/false-dto-independent-defaults.md) also validate separately through the actual target factory and field initializer; no annotation is inherited.

[Inline collection fields with false-reference leaves](../../../docs/false-inline-collection-fields.md) separately admit ordinary named Shared DTO direct finite List/Map fields without field/owner defaults. Empty containers remain inhabited, logical nullable leaves admit null, non-null false leaves reject every supplied value, and optional containers retain normal presence rather than direct-false absence policy. Actual retained field/type/check/source/codec proof and existing depth/work/checked/JVM limits remain; other roles and general Boolean/platform acceptance stay separate.

[Own defaults on inline false collection fields](../../../docs/false-inline-collection-defaults.md) additionally validate actual ordinary named Shared outer field annotations, retaining canonical literal/initializer identity, required suppression, fresh decoding and original budgets. Empty containers and logically nullable false leaves are admitted; no owner-root/child/wrapper/composed/internal/directional inheritance is added.

[Literal false DTO fields](../../../docs/literal-false-fields.md) extend F10/D06/E05 for OpenAPI 3.1 ordinary named Shared owners. One actual nested zero-domain codec serves the owner's literal-false properties; required arguments remain uninhabited, optional nullable carriers admit only absence and reject explicit null. Actual owner/source/property/helper identity, existing optimizer eligibility and resource caps remain. Cumulative false-source accounting now survives model contexts: previously accepted over-budget inputs can fail `E_MODEL_LIMIT` without replacing existing output. Other roles and general/platform acceptance remain open.

[Known false-class JVM descriptors](../../../docs/false-nominal-jvm-limits.md) extend D08/E07 with a post-transform one-sided limit using actual retained false classifiers and direct nominal fields. Constructors and Data static copy descriptors use separate erased bounds; Plain and unknown aliases/value classes retain their own rules. Previously emitted sources with provably oversized constants now fail `E_MODEL_LIMIT` at the original Pointer with old output intact. The 240-slot cap and existing generic/resource probes remain; general feasibility/platform coverage stays open.

[Root defaults with literal false DTO fields](../../../docs/literal-false-root-defaults.md) extend D03/E03 for actual ordinary named Shared owners. Optional false members may be absent; every supplied value, including null, rejects, required keys remain mandatory, and nullable owners may own null factories after complete source validation. Actual owner/field/helper/factory/canonical/literal-node authority and all existing limits remain; borrowed/consuming/child/composed/internal/directional roles stay separate.

[Consuming field defaults with literal false targets](../../../docs/literal-false-dto-field-defaults.md) extend D03/E03 for actual ordinary named Shared consumer/target DTOs, including targets mixing literal and referenced false getters. Field and root annotations validate independently; strict absence, required suppression, logical nullable targets and validation-on target counts reuse existing contracts. Deferred validation rechecks each actual getter/helper/factory/source role; original budgets, API/wire/state and other role boundaries remain.

[Inline collections with literal false items](../../../docs/literal-false-collection-fields.md) extend F10/D06/E05 for actual ordinary named Shared DTO fields. Finite inline List/typed Map chains retain a non-null owner-nested false leaf at an original array items slot, empty-container semantics, independent nullable containers and ordinary optional-container presence. Actual source/field/helper/type/count proof and original limits remain. Boolean additionalProperties keyword semantics, default/tuple/unique/reference and other owner roles retain separate eligibility.

[Own defaults on literal false collection fields](../../../docs/literal-false-collection-defaults.md) extend D03/E03 for actual ordinary named Shared outer field annotations. Standard container/value checks, canonical typed initializers, required suppression and complete original literal-node capture retain actual owner/field/helper identity, including metadata-named JSON data keys. Empty/nested/nullable containers remain valid while every false leaf value rejects. Existing budgets and other default/owner/keyword roles remain independent.

[Known false-class JVM generic signatures](../../../docs/false-generic-jvm-limits.md) extend D08/E07 with post-transform constructor and instance Data copy bounds from actual retained false classifiers. Inline List/Map arguments and proven non-null false List variance contribute conservative lengths; static copy uses its separate erased descriptor. Previously emitted uncompilable inputs now fail `E_MODEL_LIMIT` at the original Pointer with old output intact. Plain independence, unknown aliases/value classes, all original caps and resource probes remain; general feasibility/platform coverage stays open.

[Owner defaults containing literal false collections](../../../docs/literal-false-collection-root-defaults.md) extend D03/E03 for actual ordinary named Shared component-own annotations. Root factories borrow the original retained field/helper/false-leaf codec, validate required keys and independent field annotations, and preserve canonical complete literal-node identity including metadata-named JSON data. Empty/nested/nullable containers retain normal presence; false leaves reject every value. No default inheritance, cap, API/wire or persisted-format change is introduced for accepted inputs; other source/default/platform roles remain independent.

[Field defaults targeting literal false collection DTOs](../../../docs/literal-false-collection-dto-field-defaults.md) extend D03/E03 within the existing actual deferred consumer/target scope. Original leaf/Pointer/model/helper/TypeRef and independent target factories/counts remain authoritative; complete canonical consuming literals preserve metadata-named JSON data. Required suppression, nearest nullable, fresh typed initializer state and no target-default inheritance retain their contracts. All source/work/value/default/checked/slot/member caps and accepted API/wire/state formats remain; other source/default/platform roles stay independent.

[Owner-root defaults borrowing false DTO members](../../../docs/false-dto-owner-root-defaults.md) extend D03/E03 for original ordinary named Shared root annotations and direct unannotated reference members. Actual owner/getter/ref/target/helper/type/count identity and complete canonical literal capture remain bound; independent target factories validate after the full factory loop regardless of component naming order. Required keys, nearest nullable, strict false absence, normal collection presence and no default inheritance retain their contracts. All source/work/value/default/checked/slot/member caps and accepted API/wire/state formats remain; broader graph/source/platform roles stay independent.

[Pure DTO aliases in owner-root defaults](../../../docs/false-dto-owner-alias-defaults.md) extend D03/E03 through original finite named component paths. Canonical target/getter/serializer identity, nearest alias nullable then field override, original binding/height/cache/source snapshots and late independent factory/count/literal proof remain. Repeated members share immutable alias captures within the existing cumulative budget; source depth counts every hop. Existing caps and accepted API/wire/state formats remain, with alias-own/default-graph/context/platform roles independent.

## Form field defaults

Flat form DTOs accept own defaults on existing direct scalar/enum properties. Omitted optional constructor arguments use their own defaults and send those getter values; explicit Kotlin null omits a key. Required annotations validate without supplying an argument. Caller Json settings do not control form omission. Validate annotations against original scalar kinds, enum vocabularies and retained checks before required suppression. References, schema-nullable fields, const and structured form fields keep their separate eligibility.

## Boolean tuple tails

OpenAPI3.1 prefixItems arrays accept Boolean items tails. True uses List<JsonElement>; false keeps the uniform original prefix type or mixed raw projection. W_TUPLE_APPROXIMATION explicitly reports lost positional/closed-tail length checks. All original prefix sources still validate, and existing scalar own-default roles remain independent. Positions and false-tail maximum length remain approximate. Review the warning before generation; deny it when that loss is unacceptable. Compile fresh sources and test the actual target, separating JVM evidence from Android/KMP evidence.

## Raw tuple defaults

Already admitted mixed or true-tail `List<JsonElement>` tuples may own field or
component/alias defaults. Validate each original prefix source before inspecting
the literal, including references discarded by the raw projection. Default arrays
retain JsonNull and arbitrary JSON data keys. Null containers need source nullable
intent. Optional fields initialize from their own annotation; required arguments
and Patch.Absent remain explicit, and consumers inherit no target annotation.
Independently annotated alias factories use nearest nullable intent and return
fresh decoded values. The List approximation retains its warning and does not
enforce positional membership or closed-tail length.

Keep the original whole-literal 16,384-node/32KiB/depth48 budgets. Existing opt-in
counts and literal true-tail JSON uniqueness also validate annotations: objects
ignore key order, exact decimal equivalents conflict, strings/Booleans/null stay
distinct, and numeric precision/exponent bounds apply even to singletons. Report
the original failing default Pointer and preserve output on failure. Compile fresh
sources on the actual target; JVM results do not establish other platforms.

## Inline enum tuples

OpenAPI3.1 tuple prefixes and schema tails accept direct String or native
Int/Long/Float/Double/Boolean enum sources. A single enum prefix retains the
actual collection element codec. Distinct original enum nodes stay distinct even
with equal vocabularies, so a mixed prefix becomes List<JsonElement>. Explicit
admitted tails win after all prefixes validate; true tails use raw JSON and false
tails retain the prefix decision. Read W_TUPLE_APPROXIMATION for lost positions,
length checks and narrowing of unrestricted tails.

Model String enums retain lexical classifiers and open Unrecognized strings;
simple inline HTTP String enums retain the existing scalar projection selected by
format/native metadata. Native
typed consumers check membership with validation, while unchecked consumers
retain approximation warnings. Native null vocabulary members require the
existing validation-on nullable profile. List aliases add no new check barrier.

Outer field, component root and independent alias own defaults validate strict
original vocabularies, including native numeric membership before rounding.
Containers and elements keep independent null intent. Required arguments have
no initializer, Patch stays Absent, factories decode fresh and consumers inherit
no target default. Raw projected defaults retain arbitrary JSON data. Member-own
annotations, bounds, valueclass String vocabularies and general graphs retain
separate roles.

Keep original physical paths/vocabulary/literal identities, enum240, cumulative
source16384/8MiB and existing source/default/checked/JVM limits. Invalid discarded
sources still fail. Preserve output on failure and compile fresh target sources.
With merge/hoist, use the actual canonical enum classifier for nested constants;
old aliases retain companion APIs such as fromSerialName. JVM smoke evidence
does not prove Android or other KMP targets.

### String metadata in tuples

Direct String vocabularies accept existing empty/uuid/date/date-time/byte/binary
formats and safe x-native-type metadata. Model enums keep their lexical String
codec, known/Unrecognized API and strict original String default membership;
metadata never parses words such as "ready" as dates/bytes/provider values.
Actual named metadata enum components and pure named aliases keep this lexical
codec in model and HTTP collections. Consumer nullable views remain independent.
Direct simple HTTP inline enums instead use the operation scalar codec (for
example LocalDate, signed ByteArray JSON arrays or numeric Long) with the existing
membership approximation warning. Keep these contexts when integrating clients.

Outer field/root/independent tuple-alias defaults are strict, fresh and never
inherited; required/Patch annotations validate before suppression. Original
vocabularies, retained declarations, alias heights and actual consumer types
remain authoritative even for discarded prefixes. Model enum roles below additionally admit nullable components/aliases, actual
internal lexical references and independent existing default roles. Null vocabulary
inference, bounds, valueclass and general graphs retain separate permissions. Reject malformed metadata at its original Pointer and preserve old
output. The tuple-inline-enums feature includes actual JVM codec/HTTP/Patch/SSE
contracts; additional target platforms need their own execution evidence.

## Selected tuple reference defaults

OpenAPI3.1 positional array fields, original component roots and independent
component aliases may own defaults when the selected element follows an already
admitted local reference to a builtin/UUID/temporal scalar, plain String enum,
validation-on native enum or ordinary DTO/allOf constructor. Named/internal and
alias paths retain their original gates; a default grants no new reference role.
Uniform prefixes use their original source, and an admitted schema tail wins only
after every prefix validates. Boolean tails/raw defaults retain their own rules.

Use the actual emitted List type/serializer. Optional arguments use their own
annotation, required arguments remain explicit, Patch stays Absent and factories
decode fresh. Target/alias defaults validate independently and are never inherited.
Scalar codecs/null intent, strict original enum vocabulary before float rounding,
DTO declared/required keys and existing ordinary counts/constraints govern values.
Nested properties keep the existing object-default source roles. Runtime String
enums remain open; standalone transparent aliases add no constructor check barrier.

Read W_TUPLE_APPROXIMATION: positions and exact lengths are lost, and typed Lists
can narrow unrestricted tails. Own annotations add no prefix-only uniqueness.
Retain original physical selected reference/ancestor/vocabulary/literal identities,
nearest nullable, source48/work16384/capture16384+8MiB/default16384+32KiB/checked16/
enum240/JVM240 limits and original error Pointers. Preserve output on failure.
Field helpers bind the final actual nominal leaf after optional hoist/merge;
canonical classifier migration remains separate. Position-owned/child defaults,
unadmitted sources and general recursive/default graphs remain independent.
Compile fresh sources on the actual target; JVM evidence does not prove KMP or
Android acceptance.

## Boolean positional members

OpenAPI3.1 prefixItems accepts literal true/false through the shared tuple
planner. A uniform false prefix with absent/false tail uses a real nested or
collection-peer uninhabited classifier: empty arrays work and every supplied
element, including JSON null, fails. A nullable outer container separately
allows null. True and mixed prefixes use JsonElement; explicit admitted schema
tails win after all original members validate. Read W_TUPLE_APPROXIMATION,
including its explicit warning when false positional restrictions are discarded.

Fields, roots and independent aliases keep own defaults: selected false Lists
allow empty arrays, nullable containers may own null, raw projections keep JSON
data and explicit tails use their actual codec. Required arguments stay explicit,
Patch stays Absent and annotations are not inherited. Existing typed counts and
uniqueItems source gates remain; aliases gain no new count validation barrier.

Actual model/operation peers and serializers retain original schema/member/path,
allocated name, unique declaration and element-type proof. Matching spelling or
copied helpers grant no role. Existing source48/refwork16384/capture16384+8MiB/
default16384+32KiB/checked16/JVM240 limits hold, including HTTP-only contexts.
Boolean additionalProperties and ordinary non-tuple false fields keep their own
keyword/presence roles. General Boolean references, structured/composed/recursive
positions and enclosing default graphs remain independent. Compile freshly
generated sources in the target; JVM smoke does not prove other platforms.

## Inline DTO tuples

OpenAPI 3.1 positional members and explicit schema tails accept direct ordinary
DTO objects with constructor properties and type/nullable/properties/required/
property-count grammar. Use the selected context's actual element type: peer
DTOs for model roots, nested DTOs for owner fields, and the existing raw Map
projection with `W_HTTP_VALUE_PROJECTION` for simple inline HTTP object values.
Distinct original inline DTO sources stay mixed raw Lists even if structurally
equal. Explicit tails win only after every original member validates.

Outer field/root/independent alias own defaults reuse eligible ordinary object
walks, strict required/unknown keys, nullable/check/count rules and fresh actual
final serializers. Required arguments keep no initializer; Patch stays Absent;
no target/alias/member/child default is inherited. Position-owned defaults and
composed/union/hierarchy/dictionary/const/empty-object sources retain separate
admission. Empty/null outer annotations cannot hide invalid discarded source
annotations, including deferred reference defaults.

Original physical nodes/paths/bytes remain authoritative. Source depth48,
cumulative capture16384 work/8MiB, value16384/32KiB, checked16 and JVM240 limits
remain. `W_TUPLE_APPROXIMATION` still means Lists do not enforce position or exact
length and may narrow unrestricted tails. Check fresh constructor/serde/Patch/
HTTP/SSE call sites after optimization, including nested enum access through the
actual canonical classifier. Existing API/wire/state formats retain their
contracts; new platform and complete candidate acceptance need separate evidence.

## Named Boolean tuple references

OpenAPI 3.1 positional members and explicit schema tails may reference existing
named literal true/false components through pure named aliases. Uniform false
references with the same original root and effective nullability use that real
uninhabited root classifier: empty Lists are valid, each non-null leaf rejects,
and nullable aliases admit null elements only. Nearest nullable overrides apply.
Distinct false roots, direct-versus-referenced false and mixed signatures retain
raw List projection. True references use JsonElement; its nullable wrapper can
collapse JsonNull wire values to Kotlin null. Named true aliases gain no new
constructor or serializer API.

Explicit tails win after complete original prefix validation. Outer field/root/
independent alias own defaults use actual final codecs, strict null/count/value
checks and existing uniqueness eligibility. Required annotations validate before
initializer suppression; Patch stays Absent, and target/alias/child defaults are
not inherited. Direct tuple-array HTTP and typed SSE reuse those codecs; direct
false operation roots and general internal/composed/recursive/default roles
remain separate.

Review `W_TUPLE_APPROXIMATION`, including lost false-prefix restrictions, and the
true root's original `W_BOOLEAN_SCHEMA_PROJECTION`. Lists do not enforce position
or exact length. Original alias/path/node/declaration and retained element/type
identity authorize the role. Erased prefix references still count toward cached
incoming source depth48; reference16384, cumulative capture16384/8MiB,
value16384/32KiB, checked16 and JVM240 remain. Compile fresh ctor/serde/default/
Patch/HTTP/SSE consumers, including optimized and locked output; additional
platforms and complete candidate acceptance need their own evidence.

## Inline allOf tuples

OpenAPI 3.1 positional members and explicit schema tails accept inline ordinary
allOf sources with a real nonempty retained DTO constructor. Header syntax is
type/nullable/allOf/properties/required/property counts plus descriptive metadata.
Use the actual selected peer/nested classifier and codec. Existing later-field
overrides, required union, supported bounds/count aggregation and member null
intersection remain; `W_ALL_OF_APPROXIMATION` is not a full intersection promise.
Distinct original sources remain mixed raw Lists, and explicit tails win only
after every original prefix validates. `W_TUPLE_APPROXIMATION` retains position/
length/tail and discarded-false restrictions.

Outer field/root/independent alias own defaults use the existing eligible allOf
walk, actual final codec and strict original literals/keys/required/null/counts.
Required initializers remain suppressed, Patch stays Absent, and target/alias/
child annotations are not inherited. Wider default/overlap roles remain separate.
Simple HTTP tuple arrays generate actual operation-owned DTO peers with ordinary
required/default/check and Json unknown-key behavior. Plain inline DTO Map
projection, direct allOf/union HTTP roots and non-tuple rules remain separate.

Discarded sources complete isolated ordinary/deferred validation; temporary
constructors and registrations are not emitted. Source node/path/Pointer facts,
unique actual constructor and retained element/type/check identity authorize the
role. Empty/const-only/dictionary/product/hierarchy sources do not gain that
constructor role. Repeated original captures reuse snapshots, while distinct
allOf sources borrowing one Base pay shared overlap work: tuple allOf16384/8MiB,
general source48/capture16384, value16384/32KiB, checked16 and JVM240 remain.
Erased allOf contribution depths remain in incoming/forward collection caches.
Compile fresh ctor/serde/default/Patch/HTTP/SSE and directional consumers after
optimization; additional platforms, option Cartesian products and final candidate
acceptance require their own evidence.

## Collection tuple sources

OpenAPI 3.1 positional members and explicit schema tails accept existing finite
List/typed Map sources, inline or through admitted named/internal references.
Use the actual context codec and collection backing. Uniform references reaching
one original source/null view can stay typed; distinct inline equal shapes stay
mixed raw Lists. Explicit tails win only after every original source validates.
Tuple position/length/tail approximation and original warning policies remain.

Outer field/root/independent alias annotations use the actual selected tree,
strict original literals/enum vocabularies/required keys and existing nullable/
counts/checks. Forward named backings link before field annotation validation.
Required/Patch annotations validate before initializer suppression; Patch stays
Absent and target/alias/child defaults are not inherited. Metadata-looking JSON
keys remain data. Helpers bind actual final DTO/enum types after optimization.

Discarded sources complete isolated ordinary/deferred validation without emitted
temporary declarations. Original nodes/paths/catalogs, final type/check and
retained element/backing addresses authorize the role. Preserved typed edges pay
their ordinary source height; only erased work is added after alias linking.
Source48/capture16384/8MiB, value16384/32KiB, checked16 and JVM240 remain independent
bounds. Generic resolver/recursive/union/const/Boolean leaf roles and context HTTP
projections retain their own gates. Compile fresh JSON/Patch/HTTP/SSE consumers;
source scans and PREPARE_ONLY do not prove runtime, platform or final acceptance.

## JsonNull tuple sources

OpenAPI3.1 positional members/schema tails accept plain type:null, singleton null enums and null consts, including inferred sources and existing named/pure-alias references. Equal effective nullable views share the actual JsonNull singleton codec; nonnullable List<JsonNull> decodes null as JsonNull, nullable leaves decode null as Kotlin null. Mixed domains use existing raw List projection and explicit tails win after all prefixes validate. Tuple position/length and unrestricted tails remain approximated.

Outer field/root/independent alias defaults validate original null domains: [null] is valid without nullable:true; wrong kinds reject. Required arguments and Patch.Absent remain after validation, and target/chain defaults are not inherited. Positional own annotations, unsupported format/native/vocabulary/default roles still reject before erasure. Actual source/ref/catalog/backing/element/type proof and source48/capture16384+8MiB/value16384+32KiB/checked16/JVM240 limits remain. Compile fresh Data/Plain x validation consumers, exercise four Json settings and actual HTTP/SSE, including invalid-value rejection and stream closure; PREPARE_ONLY is not runtime evidence. Direct null success roots retain their existing Unit projection. General const/union/recursive graphs and platform/final candidate acceptance remain independent.

## Scalar const tuple sources

OpenAPI3.1 positional members/schema tails accept existing scalar const sources and admitted named/pure-alias references. Builtin/numeric-format/UUID/date/Instant/native-bigint codecs retain their grammar; no enforcing wrapper is introduced. Same original terminal plus nullable view may be uniform; separate inline sources or const-versus-ordinary values remain raw. Explicit tails win only after complete original source validation. Retained scalar const warning and tuple lost-membership warnings remain; runtime accepts nonliteral values fitting its scalar codec.

Outer field/root/independent alias annotations must equal the original pin before Float rounding or native wire conversion, then fit the actual codec. Native bigint string pins validate original string membership and encode the helper payload for actual Long values. Required/Patch suppression follows validation; no target/child default is inherited. Source/literal/ref/catalog/backing/element/type snapshots remain authoritative, and retained overlapping alias-binding records count against shared16384work/8MiB; source48/default16384+32KiB/checked16/JVM240 remain. Compile fresh Data/Plain x validation baseline/optimized/locked consumers and exercise fourJson, nonliteral runtime values, defaults/Patch/HTTP/SSE failure and closure. PREPARE_ONLY is not runtime evidence. Structured const, union, recursive and general default roles remain independent.

## Structured const tuple sources

OpenAPI3.1 positional members/schema tails accept original object/array consts and existing named/pure aliases. Use actual JsonObject/JsonArray codecs, not ordinary List schemas for explicit array consts. Same original terminal/nullable may be uniform; distinct inline pins or mixed domains retain raw projection. Typed runtime accepts different data of the same JSON kind, with structured-const and tuple lost-membership warnings; position/length/tail approximation remains.

Outer field/root/independent alias defaults compare complete literal JSON: object order independent, array order/kinds retained and bounded exact decimal identity (256 numeric characters/±512 exponent), before decoding fresh values through the final codec. Null requires actual nullable intent; required/Patch suppression follows validation and no target/child annotation is inherited. Const/default/enum data keys such as $ref/allOf/properties/description/items remain data at every level and retain original node snapshots. Actual source/ref/catalog/backing/element/type and canonical-key proof, source48/capture16384+8MiB/default16384+32KiB/checked16/JVM240 remain. Canonical keys and literal edges pay shared budgets. Compile fresh Data/Plain x validation baseline/optimized/locked consumers and exercise metadata-shaped data, nonliteral same-kind runtime values, defaults/Patch/HTTP/live SSE failure and closure. PREPARE_ONLY is not runtime evidence; general union/recursive/default and platform/final acceptance remain independent.

For oneOf/anyOf tuple sources, references, normalization, context codecs and outer own defaults, read [union tuple sources](tuple-union-sources.md). Preserve actual root versus inline dispatch and strict annotations; runtime Unknown is not a default escape.

For allOf products in positional arrays, read [product union tuple sources](tuple-product-union-sources.md); preserve original conjunction/required/Unknown/null policies, actual final nominal ownership and independent outer annotations.

For hierarchy references inside positional arrays, read [hierarchy tuple references](tuple-hierarchy-references.md); root/parent/leaf and nullable adapters retain actual route-specific codecs and strict independent outer defaults.

## Metadata String enums in models

String enum empty/uuid/date/date-time/byte/binary formats and safe x-native-type
metadata preserve lexical String codecs in model contexts. Nullable component
roots, nearest explicit alias/reference intent and already eligible internal
lexical declarations share the existing String enum planners. Keep the actual
unique retained owner/classifier; JSON null is independent from the word "null".
Metadata adds no plain-enum null-member inference or general graph ownership.

Existing direct/referenced scalar, finite collection and object-default roles
use strict original String membership and final lexical serializers. Original
components and named aliases may each own a factory; consuming fields and outer
annotations remain independent, fresh and never inherit target defaults.
Required/Patch/discarded annotations validate before initializer suppression.
Reject child-owned annotations, unsupported bounds/valueclasses, malformed
metadata/vocabularies or unowned targets at their original Pointers.

Validation uses serialName for admitted metadata enum uniqueness, including
existing references, finite container trees and ordinary DTO getter keys.
Known and same-name Unrecognized values collide; differently cased UUID-looking
words remain distinct Strings. Simple direct HTTP inline enum projections keep
their actual scalar/provider codec and approximation warning. Model enum refs
keep their lexical codec in HTTP/SSE.

Merged equivalent component enums keep each original named `<ClassName> default`
function; null factories return nullable values while the physical classifier
stays non-null. Object component factories keep companion default(). Regenerate
owned output when adopting the corrected merged enum factories; persisted
formats remain. Original depth/work/default/member/slot limits and source/cached
alias/declaration identities still apply. Compile and run the target consumer;
feature tuple-inline-enums provides JVM profiles, not full platform acceptance.

### Internal String enum alias factories

A component alias may own `default` through an already admitted internal lexical
String enum property, nested property, List/typed Map leaf or composition-owned
path, including finite pure named/internal aliases. For example, an alias to
`#/components/schemas/Owner/properties/state` with `default:"paused"` receives
its own top-level `<AliasName> default` factory returning the actual Owner enum.
The nearest explicit nullable intent applies: a nullable alias may own null,
and `nullable:false` narrows the next alias's own factory.

Original field, intermediate factory and consuming scalar/collection/object
annotations are independent and strict against their original vocabularies.
Unannotated aliases and fields inherit no default. Required arguments and
Patch.Absent remain after annotation validation. Format/native metadata keeps
lexical String codecs; runtime unknown words still use Unrecognized. Reuse the
canonical physical enum owner for nested singleton access through typealiases.

Unique actual declaration/source/type/nullable/alias-height authority and shared
depth/work/value budgets remain. Cloned/discarded/ambiguous declarations, literal
`default`/`enum` data, cycles, unsupported metadata/bounds, numeric/Boolean enum
internal factories, child-owned defaults and general reference graphs retain
their exclusions. Run `bash scripts/test.sh feature tuple-inline-enums` for fresh
Data/Plain × validation, baseline/optimized locks, four Json settings and
HTTP/SSE contracts. This additive role changes no wire or lock format and does
not close broader defaults, platform or release acceptance.

### Direct String enum field bounds

With validation, ordinary direct scalar String enum fields accept minLength,
maxLength and portable pattern, with optional own defaults. Existing direct
List/typed Map enum leaf bounds also accept supported format/native metadata.
These remain lexical enum types: check actual serialName in DTO construction,
Data copy and automatic decoding, while standalone enum lookup/serialization
and runtime Unrecognized stay open. Metadata creates no provider/date/UUID codec.

Defaults must belong to the original vocabulary and satisfy checks. Validate
required/Patch annotations before suppressing initializers; preserve missing,
null and supplied values, and Patch.Absent/Null/Value. Constraints require
validation, including empty/null annotations; off must reject the original
bounded source. Keep original vocabulary/check Pointers, codepoint length,
portable regex/default ASCII policy and original limits.

Use canonical enum owners after optimization and verify frozen check/default
identity. Enum roots, referenced/borrowed checked leaves, nested collection-bound
trees, hierarchy/union/cross-contribution roles, const,
value classes and numeric bounds retain their separate boundaries. Run
`bash scripts/test.sh feature string-enum-collection-bounds` for fresh four
profiles, four Json settings, optimization/locks, independent240 collection and
scalar probes and HTTP/SSE behavior. No new wire/lock format or broader platform
acceptance is implied.

### Bounded enums in object defaults

With validation, existing ordinary object-default walks accept their original
direct String enum fields and direct List/typed Map enum leaves with lengths
and portable patterns. Metadata remains lexical String. Each supplied annotation
leaf must satisfy original membership and retained checks; runtime Unrecognized
stays open. Preserve original source/value Pointers, required/Patch validation
before suppression, nullable intent and all source/default/width budgets.

Original admitted plain DTO components and pure named aliases can keep their
own independent factories while field/collection annotations borrow the shape.
A valid outer value cannot hide a bad target or intermediate factory; unannotated
consumers inherit no initializer. Decode fresh state through the final actual
serializer after optimization. No copied/unmarked/internal source gains a new
annotation or resolver role.

Checked enum roots/references, deeper collection-bound trees, child-owned
annotations and unadmitted composition/hierarchy/union/general graphs remain
separate. Run `bash scripts/test.sh feature string-enum-collection-bounds` for
fresh Data/Plain × validation, four Json settings, root/alias/inline/ref/DTO-list
factories, optimization/locks, Patch and HTTP/SSE behavior. Other target platform
and final-candidate acceptance retain their own requirements.

For [bounded lexical enum collection roots](checked-enum-collections.md#collection-roots), use the actual root alias and original enum peer. Check independent root/alias/field/object annotations, nearest nullable and Patch. Typed DTO/HTTP/SSE consumers impose serialName checks; standalone alias assignments/codecs stay open. Preserve original backing/source/check proof and report actual runtime evidence.

[Checked lexical enum borrowing](checked-enum-collections.md#checked-references) uses original ordinary field/collection-root check objects and actual lexical classifiers through existing internal/named chains. Check source getters and strict independent factories/consuming annotations, nullable/required/Patch and typed HTTP/SSE; preserve standalone codec openness and existing resource/source-role boundaries.

[Borrowed enum collection defaults](checked-enum-collections.md#borrowed-collection-defaults) retain an original ordinary field annotation and independent factories on the actual collection alias/finite pure aliases. Test each own vocabulary/check/null rule and fresh state; unannotated consumers inherit no default. Factory binding/type/height and original graph/check identities remain authoritative; child defaults and general graphs stay separate.

[Checked String enum roots](checked-enum-collections.md#checked-enum-roots) retain lexical classifiers and shared source checks through existing named aliases and typed List/Map/default/HTTP/SSE consumers. Standalone codecs and assignments remain open; defaults require original vocabulary plus checks. Verify root/alias/null factories, nearest nullable, required/Patch and actual runtime boundaries; root declarations and factory records must remain authoritative even without consumers.
