# Ordinary allOf object field defaults

An ordinary DTO field may declare its own `default` on an inline object `allOf` whose contributions have distinct wire property names. The shared object planner still owns its nested Kotlin DTO, required arguments and serializers. For example:

```json
{
  "allOf": [
    {"properties": {"label": {"type": "string"}}, "required": ["label"]},
    {"properties": {"count": {"type": "integer"}}, "required": ["count"]}
  ],
  "default": {"count": 2, "label": "initial"}
}
```

An optional field uses a fresh typed object decoded through the final nested DTO serializer. A missing key uses the default; a supplied object replaces it. Required fields validate the complete annotation and retain their constructor argument. Optional nullable inline request fields retain `Patch.Absent`, `Patch.Null` and `Patch.Value`; a schema default never makes a missing key an update.

## Source and value contract

The field must produce an ordinary constructor Model with at least one constructor property. Its `allOf` members may contain ordinary objects, nested object compositions and already accepted object-member references. Root own properties contribute after the members. Every wire property has exactly one contribution; repeated names, even with identical schemas, remain outside this default role.

The default walk borrows the shared merged shape. Each field retains its original schema node, Pointer and source depth, including fields expanded from named members. All supplied keys must name declared fields; all merged required names must be present. Leaf membership, nullable values, supported bounds and collection counts follow the existing [object default contract](nested-object-defaults.md). A plain object member may carry its own validated default; omitting that member from the outer default uses the nested field's initializer. Schema errors point to their source; supplied-value errors point to the original `/default/...` member. Validation visits default keys in source order before sorting the encoded keys.

Effective shape nullability authorizes `default: null`; optional Kotlin presence alone does not. Nested objects, supported collections and scalar/enum/temporal leaves retain their own nullable intent and validation. Required/Patch/discarded annotations undergo the same full validation before initializer suppression or a later parent `allOf` override.

The decode helper binds the final actual DTO type after deduplication, merging and hoisting. Adding the annotation preserves the public type and declaration ownership. The existing `W_ALL_OF_APPROXIMATION` remains owned by type planning, with no duplicate warnings from the default walk.

## References to named allOf DTOs

An ordinary direct local `$ref` field may also own a default when its original terminal is an already accepted named ordinary allOf constructor DTO with the same disjoint contributions. For example, place the `allOf` above in component `Value` without a root default, then consume it as:

```json
{"$ref": "#/components/schemas/Value", "default": {"count": 2, "label": "initial"}}
```

Transparent named aliases and already accepted finite internal reference hops may reach that named component. The source walk retains every intermediate annotation and original Pointer. Final retained declaration ownership authorizes the actual component and serializer; a provisional or coincident type name does not. The field retains the resolver's existing public type, and adding a consumer default leaves target declaration bytes and warnings unchanged.

Reference nullability follows the shared reference planner before optional presence: explicit consumer intent, nearest alias/internal hop intent, then terminal intent. Explicit false narrows even an intrinsically nullable allOf component. A valid null annotation emits literal `null`; non-null values use the final actual DTO serializer and create fresh objects, including omitted members with their own defaults. Required, Patch and discarded defaults undergo complete validation before suppression.

The same referenced role also covers eligible [internal allOf DTOs](#references-to-internal-allof-dtos). Annotation-only allOf reference wrappers and hierarchy/common/inherited projections retain separate roles. [Referenced collection fields](#referenced-collection-field-defaults) separately admit DTO element references. Defaults on the target component or an intermediate reference are rejected rather than inherited. Repeated properties, fixed extras and whole-object counts remain excluded.

## References to internal allOf DTOs

A consuming ordinary direct local `$ref` field may own a default when an already admitted [internal allOf terminal](allof-nominal-terminals.md) retains a unique constructor Model under its original lexical component. For example, a reference to `#/components/schemas/Owner/properties/node` can use `default: {"label":"initial","count":2}` when `node` contains the disjoint allOf above. A root alias or finite internal/named reference chain may reach that same node. The public type stays the physical constructor, such as `OwnerDto.Node`, rather than an alias-owned or copied class.

Eligible existing scopes include properties/nested DTOs, List item and typed Map value Models, root collection peers and unique composition-owned constructor Models. A reference to an allOf object-union variant uses that concrete Model's serializer and ordinary business payload; it does not add a closed-root class discriminator. The source must still pass the disjoint ordinary object default walk. Typed nominal recursion or wider typed admission alone does not grant a default role.

Source authorization combines the final retained nominal binding with the original node, Pointer, current component identity and schema lookup's lexical `SourceCatalog`. A generated name or a scratch/borrowed occurrence cannot grant ownership. Hierarchy sources and hierarchy lexical owners remain excluded. No resolver or declaration admission is added by the annotation.

Original field nodes, required names, leaf checks and source depths are reused. Nearest consumer/alias/internal nullable intent precedes terminal effective shape intent, including member-derived nullable on internal Models; explicit false narrows it and a later true may re-enable null. The helper binds the final actual type after optimization and produces fresh values. Target/chain annotations remain rejected, even when the target field's own initializer is otherwise valid. Required/Patch/discarded annotations and original/canonical budgets follow the same full validation before suppression.

The consuming field defaults a single DTO value. [Referenced collection fields](#referenced-collection-field-defaults) separately support references to these Models as elements. [Direct inline allOf properties inside object defaults](#inline-allof-properties-inside-object-defaults) and [references to named/internal allOf members](#referenced-allof-properties-inside-object-defaults) have their own bounded roles.

## Direct inline collection field defaults

An ordinary direct inline finite List/typed Map field may own an outer default when its original terminal element is an ordinary disjoint allOf constructor DTO. For example:

```json
{
  "type": "array",
  "items": {
    "allOf": [
      {"properties": {"label": {"type": "string"}}, "required": ["label"]},
      {"properties": {"count": {"type": "integer", "default": 2}}}
    ]
  },
  "default": [{"label": "initial"}]
}
```

List, typed Map and finite mixed paths such as List of Map values keep their original nested constructor and public collection type. The shared type plan must create exactly one constructor Model with at least one property on that path. Empty/raw, const-only and union projections do not obtain the role. The value walk borrows the original leaf and its merged fields; no component, source schema copy or borrowed classifier is added.

Permission is scoped to that consuming field's original allOf element node in a shallow validation copy. A structurally equal copy or another object member cannot acquire it. It follows only inline container edges and stops at reference edges. [Direct inline collections with referenced elements](#direct-inline-collections-with-referenced-elements) use the final nominal validation role below; [Bare allOf elements inside referenced collections](#bare-allof-elements-in-referenced-collections) use final retained declaration authority. Direct allOf object properties use the [member role](#inline-allof-properties-inside-object-defaults); inline collections under an object-default property use the [collection member role](#inline-collection-properties-inside-object-defaults). Primitive allOf compositions keep their existing default/diagnostic path.

The merged required names, leaf checks, original Pointers/depths, source planning and original/canonical value budgets apply even to empty/null annotations. Container and element nullable intent remain independent; effective element intent may come from an allOf member, and an explicit root false narrows it. Optional presence alone does not authorize annotation null. Required/Patch/discarded annotations validate before suppression, and existing no-drop guards still reject discarded runtime constraints. Element/container-child own defaults remain rejected; ordinary fields inside the element retain their own admitted defaults.

The helper decodes the whole collection through its final actual serializer after optimization. Each evaluation creates fresh containers, element DTOs and supplied nested objects; missing fields inside an element use its existing field initializers. Supplied collections replace the outer default and preserve existing HTTP/SSE/Patch behavior.

[Core admission/nullable/diagnostic/budget regressions](../core/inline_allof_collection_defaults_test.go), [original leaf identity/scoping/type/depth tests](../internal/schema/inline_allof_collection_defaults_test.go) and [native parity/check/failure preservation](../internal/cli/inline_allof_collection_defaults_process_test.go) define the extension. The [nested collection fixture](../tests/fixtures/http/nested-object-collection-defaults.json), [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NestedObjectCollectionDefaultsContracts.kt) and `bash scripts/test.sh feature nested-object-collection-defaults` exercise Data/Plain × validation off/on, four Json configurations, fresh nested values, nullable/required/Patch, HTTP/live SSE and optimized/locked helpers. The checked profile adds admitted collection counts to its generated input; preparation alone is not Kotlin compilation/runtime evidence.

## Direct inline collections with referenced elements

An ordinary direct inline finite List/typed Map field may also own an outer default when its element crosses an actual `$ref` edge, optionally through already admitted finite named/internal hops, to an eligible ordinary allOf constructor Model. For example, use `{"type":"array","items":{"$ref":"#/components/schemas/Value"},"default":[{"label":"initial","count":2}]}` with the disjoint `Value` above. Mixed List/Map paths retain the same contract. Named and internal terminals use their actual physical lexical owner and serializer.

The shared collection/reference planner first establishes the public type, nullable layers and existing checks. The consuming annotation is then queued with its original field node and pre-presence TypeRef. After final nominal linking, the existing referenced collection walker checks the allOf terminal's retained constructor binding, original node/Pointer/owner/SourceCatalog and exact final element name; generated spelling alone never authorizes a source. Adding the annotation leaves the target's declaration bytes and warnings unchanged. The helper binds the final collection serializer after optimization and creates fresh containers and nested values.

Only the consuming inline container accepts its own `default`. Element refs, intermediate hops/aliases, target components and inner containers cannot own or supply inherited defaults. Object-member paths, hierarchy sources/owners, repeated-property aggregation, wrappers and fixed extras/counts retain their separate boundaries. [Bare allOf leaves inside referenced collection trees](#bare-allof-elements-in-referenced-collections) use their own retained declaration role. A reference ending in another collection does not extend this routing role; its terminal here must be the admitted allOf Model.

Container and element nullability remain independent, using the original pre-presence type and nearest reference intent. Optional Kotlin presence does not authorize null. Every declared source and supplied value is checked before required/Patch/discarded initializer suppression, including empty/null collections, native enum membership, original/canonical value budgets and the independent source-walk limit. Existing no-drop checks continue to reject discarded runtime constraints. Element fields may retain their own already admitted initializers; missing request keys remain Patch.Absent.

[Core type/nullable/suppression/diagnostic/budget regressions](../core/inline_referenced_allof_defaults_test.go), [final ownership/original source/depth and role checks](../internal/schema/inline_referenced_allof_defaults_test.go) and [native parity/check/failure preservation](../internal/cli/inline_referenced_allof_defaults_process_test.go) define this extension. The [referenced collection fixture](../tests/fixtures/http/referenced-object-collection-defaults.json), [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedObjectCollectionDefaultsContracts.kt) and `bash scripts/test.sh feature referenced-object-collection-defaults` cover Data/Plain × validation off/on, four Json configurations, aliases/internal bridges, fresh nested values, required/Patch, HTTP/live SSE and optimized/locked helpers. The checked profile adds admitted collection counts; preparation alone is not Kotlin compilation/runtime evidence.

## Referenced collection field defaults

An ordinary DTO field whose direct local `$ref` resolves to an already admitted finite named/internal List/typed Map tree may own an outer default when its element references an eligible named or internal ordinary allOf constructor Model. Accepted collection aliases and element reference chains retain their existing physical types and source limits. For example, `Rows` may declare `items: {"$ref":"#/components/schemas/Value"}` for the disjoint allOf above, and a consuming field may use `{"$ref":"#/components/schemas/Rows","default":[{"label":"initial","count":2}]}`. Mixed finite paths such as List of typed Map values are supported by the same walk.

When crossing an actual element reference edge, the final named element type, retained nominal binding, original source node/Pointer/owner and hierarchy exclusions are checked by the same referenced DTO authority above. [Bare inline elements](#bare-allof-elements-in-referenced-collections) use the retained collection constructor role below. [Direct inline collection fields](#direct-inline-collection-field-defaults) have their own source role; [their referenced elements](#direct-inline-collections-with-referenced-elements) reuse final nominal authority. AllOf-valued object properties use the [direct inline member role](#inline-allof-properties-inside-object-defaults) or the [direct named reference role](#referenced-allof-properties-inside-object-defaults).

Every element validates the original disjoint shape, required names and leaves. Container and element nullable intent remain independent; an explicit false element reference narrows a nullable terminal, and optional field presence alone does not authorize annotation null. Source planning and original/canonical value budgets apply even to empty/null collections. Required/Patch/discarded annotations fully validate before initializer suppression; existing no-drop checks still reject discarded runtime constraints. Source annotations are not inherited. Named collection count constraints retain their existing validation opt-in.

The helper decodes the complete collection using its final actual collection/alias serializer. Each default evaluation creates fresh containers, DTOs and supplied nested objects; omitted DTO fields retain their own declared initializers. Optimization and locks retain the effective type/default/check identity.

[Core scope/nullable/suppression/diagnostic/budget regressions](../core/referenced_allof_collection_defaults_test.go), [original source/final element authority checks](../internal/schema/referenced_allof_collection_defaults_test.go) and [native parity/check/failure preservation](../internal/cli/referenced_allof_collection_defaults_process_test.go) define this extension. The [referenced collection fixture](../tests/fixtures/http/referenced-object-collection-defaults.json), [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedObjectCollectionDefaultsContracts.kt) and `bash scripts/test.sh feature referenced-object-collection-defaults` cover Data/Plain × validation off/on, four Json configurations, concrete internal serializers, fresh nested values, required/Patch, HTTP/live SSE and optimized/locked helpers. The checked profile adds admitted named collection counts to its generated input; preparation alone is not compilation/runtime evidence.

## Bare allOf elements in referenced collections

An ordinary direct local `$ref` field may own an outer default when its existing finite collection type has an inline disjoint ordinary allOf constructor element. This covers named List/typed Map components, finite mixed paths and already admitted collection aliases. For example:

```json
{
  "Rows": {
    "type": "array",
    "items": {"allOf": [
      {"properties": {"label": {"type": "string"}}, "required": ["label"]},
      {"properties": {"count": {"type": "integer", "default": 2}}}
    ]}
  },
  "Holder": {"properties": {
    "rows": {"$ref": "#/components/schemas/Rows", "default": [{"label": "initial"}]}
  }}
}
```

The shared collection plan must already retain a unique nonempty constructor Model for the original element under its lexical component. Final nominal linking, original node/Pointer/SourceCatalog, actual classifier and hierarchy exclusions authorize the source. Root collection peers may retain a package-relative type name; validation matches that same emitted classifier to its complete final name. A matching spelling or another typed use alone cannot grant ownership. The annotation changes neither public collection/element types nor target declaration bytes/warnings, and adds no reference or type admission.

Every original merged field, required name, nullable layer, source depth and leaf check is retained. Member-derived element nullability and explicit false remain independent of container/alias intent and optional presence. Complete source and value validation precedes required/Patch/discarded suppression, including empty/null annotations; source16,384, value16,384 and original/canonical32KiB budgets apply. No-drop runtime-constraint guards remain. Only the consuming outer field owns a default; aliases, target components, elements and inner containers cannot supply inherited defaults. Reference chains ending in other terminal roles, hierarchy, wrappers, repeated-property aggregation and extras/counts keep their existing exclusions.

The helper uses the final actual collection serializer after optimization and produces fresh containers, DTOs and supplied nested objects. Missing element fields keep their own admitted initializers; supplied collections replace the outer default. [Core type/nullable/suppression/diagnostic/budget regressions](../core/referenced_inline_allof_defaults_test.go), [original constructor/name/owner/depth checks](../internal/schema/referenced_inline_allof_defaults_test.go) and [native parity/check/failure preservation](../internal/cli/referenced_inline_allof_defaults_process_test.go) define this role. The referenced collection fixture/consumer and `bash scripts/test.sh feature referenced-object-collection-defaults` cover List/Map/mixed, aliases, fresh nested defaults, nullable/required/Patch, four Json configurations, HTTP/live SSE and optimized/locked helper binding across Data/Plain × validation off/on. Checked profiles add admitted named collection counts; preparation alone is not Kotlin compilation/runtime evidence.

## Inline allOf properties inside object defaults

An already accepted object-default walk may contain a direct inline ordinary allOf property with disjoint wire contributions. For example:

```json
{
  "properties": {
    "member": {
      "allOf": [
        {"properties": {"label": {"type": "string"}}, "required": ["label"]},
        {"properties": {"count": {"type": "integer", "default": 2}}}
      ]
    }
  },
  "default": {"member": {"label": "initial"}}
}
```

The enclosing field's shared type plan owns the nested constructor and serializer. The value walk borrows the member's original merged fields, required names, nullable intent, Pointers and depths. At least one non-const constructor property is required. Empty/raw, const-only, hierarchy and object-union compositions, reference wrappers, repeated contributions, fixed extras and whole-object counts remain excluded. A property whose schema is an actual `$ref` to an allOf target uses the separate [direct named reference rule](#referenced-allof-properties-inside-object-defaults). AllOf leaves under a collection-valued property use the separate [inline collection member rule](#inline-collection-properties-inside-object-defaults).

The rule applies recursively wherever the existing object walk validates ordinary properties, including referenced plain DTO targets, object elements in accepted collections and ordinary merged shapes. Every declared member is checked even when omitted from the annotation. A member may retain its independently validated own field default; omission uses that initializer, while a supplied member replaces it and still uses its own nested field initializers. Required names must be supplied. Explicit or member-derived nullable intent permits null; explicit false narrows it, and optional presence does not authorize annotation null.

Default values retain original-order diagnostics and canonical key sorting. Required/Patch/discarded annotations validate before suppression. Source depth and the shared 16,384-step source budget continue across member boundaries, alongside original/canonical value budgets. No new declaration, reference admission, warning or helper codec is introduced; decoding uses the final enclosing serializer after optimization and produces fresh nested values.

[Core contexts/presence/boundaries](../core/allof_object_default_members_test.go), [source identity/depth/budget checks](../internal/schema/allof_object_default_members_test.go) and [native parity/check/failure preservation](../internal/cli/allof_object_default_members_process_test.go) define the increment. The nested-object fixture and consumer cover missing/replacement/null values, required member decoding, nested own defaults, independent instances and HTTP/live SSE under four Json configurations. `bash scripts/test.sh feature nested-object-defaults` runs Data/Plain × validation off/on and optimized/locked helper binding.

## Referenced allOf properties inside object defaults

An existing object-default walk may contain a local `$ref` property to an eligible named or internal ordinary allOf constructor, directly or through a finite chain of named aliases and already admitted internal reference hops. For example, with the disjoint component `Value` above:

```json
{
  "properties": {"member": {"$ref": "#/components/schemas/Value"}},
  "default": {"member": {"label": "initial", "count": 2}}
}
```

Every named alias edge must match its original component node and Pointer. Internal hops retain their original schema node, Pointer and current lexical `SourceCatalog`; they do not acquire nominal identity. Internal terminals use the original lexical binding described below. The enclosing shared type plan owns the nominal reference and its final declaration; validation creates no classifier. The same nonempty constructor, disjoint fields and hierarchy/union/wrapper/extras/count exclusions as the inline member rule apply. Defaults on intermediate aliases/hops or the terminal, and chains ending in other source roles remain excluded. This member rule grants no new collection-element role; [references inside inline collection properties](#referenced-elements-in-collection-properties) have their own original-element permission.

For example, components `Alias: {"$ref":"#/components/schemas/Value","nullable":true}` and `Outer: {"$ref":"#/components/schemas/Alias","nullable":false}` may be consumed by `member: {"$ref":"#/components/schemas/Outer"}`. The member uses the actual `Value` DTO and rejects annotation null. Setting `nullable:true` on that member permits null again. Alias declaration bytes and canonical target ownership remain unchanged; aliases do not contribute inherited defaults.

The walker borrows the target's merged original fields, required names, leaf checks and source locations. Nullable precedence is explicit consuming property intent, then the nearest explicit named-alias/internal-hop intent, then the target's effective intent; false is distinct from absence. Optional Kotlin presence alone does not allow annotation null. Every declared member is checked even when omitted. Target fields retain their own admitted initializers, and the referring property may retain an independently validated own default. A supplied object replaces that member default; it still uses omitted target-field initializers.

The rule is shared by inline objects, referenced DTO targets, accepted object-containing collections and ordinary merged shapes. Required/Patch/discarded annotations validate before suppression. Every reference traversal consumes source depth, and each named alias or internal hop consumes a step of the enclosing 16,384-step source budget. Cached canonical bindings do not skip original hops, source annotations or nullable intent. These limits apply even to omitted or recursive members. Original-order value diagnostics, canonical keys and original/canonical 32 KiB value budgets are unchanged. Decoding uses the final enclosing serializer after optimization and creates fresh referenced objects and nested defaults. Adding the annotation leaves the referenced component's generated bytes and warnings unchanged.

[Core contexts/nullable/diagnostic tests](../core/referenced_allof_object_default_members_test.go), [named alias tests](../core/aliased_allof_object_default_members_test.go), [original source/depth/budget tests](../internal/schema/referenced_allof_object_default_members_test.go), [alias hop budgets](../internal/schema/aliased_allof_object_default_members_test.go) and [native parity/check/failure preservation](../internal/cli/referenced_allof_object_default_members_process_test.go) define the increment. The nested-object fixture and consumer cover direct and aliased members, missing/replacement/null, required members, nested own defaults, fresh instances, Patch states, four Json configurations and HTTP/live SSE. `bash scripts/test.sh feature nested-object-defaults` runs Data/Plain × validation off/on and optimized/locked helper binding.

## Internal allOf properties inside object defaults

A member may directly reference an already admitted internal ordinary allOf constructor, for example `{"$ref":"#/components/schemas/Owner/properties/node"}`, or reach it through a finite mixed named/internal reference chain. Its public type remains the original lexical constructor, such as `OwnerDto.Node`. Existing eligible scopes include nested properties, List/typed Map elements, root collection peers and unique concrete composition-owned Models. A concrete union variant uses its own ordinary serializer and business fields, without adding a root-union discriminator envelope.

The default walk requires the shared type planner's existing binding for the exact source node, Pointer and current lexical component. Schema lookup must confirm the same `SourceCatalog`. It neither registers a new binding nor grants new reference admission. Source validation can run before a forward target is emitted; successful generation still passes the shared final linker, which requires a unique retained nonempty constructor at that original scope. Missing, copied or ambiguous declarations cannot authorize output. Hierarchy sources and hierarchy lexical owners remain excluded.

Merged required names, original member Pointers, nullable intent (including member-derived target nullability), leaf checks and all source/value budgets are preserved. Explicit consumer intent precedes nearest named-alias/internal-hop intent and the target. Required/Patch/discarded annotations validate before suppression. An internal target's or intermediate hop's own default is not inherited, even if valid as an independent field annotation. The enclosing serializer binds the actual final type after optimization and creates fresh referenced and nested default values.

[Internal member contexts/nullable/guards](../core/internal_allof_object_default_members_test.go), [source authority and final declaration checks](../internal/schema/internal_allof_object_default_members_test.go) and the shared native parity/preservation test define this extension. The nested-object fixture/consumer cover internal properties and aliases, root collection peers, concrete variants, fresh defaults, missing/replacement/null/required members, Patch, HTTP/live SSE and optimized helper binding under Data/Plain × validation and four Json configurations.

## Internal reference chains inside object defaults

An object-default member may traverse already admitted finite internal `$ref` hops interleaved with named aliases, ending in an eligible named or internal allOf constructor. For example, `member → Bridge → Hops.properties.first → Hops.properties.second → Alias → Owner.properties.node` keeps the actual `OwnerDto.Node` constructor and serializer. This extends only the annotation walk; the shared reference planner must already accept the typed path.

Each internal hop is checked against its original node, Pointer and current lexical `SourceCatalog`. The walker follows original edges instead of the resolver's cached terminal so intermediate defaults and invalid keys cannot be hidden. Hierarchy-owned hops remain excluded, as do chains ending in plain objects, scalars or other roles outside this allOf-member contract. The terminal keeps its original ownership and final retained-declaration checks.

Nullable precedence follows the complete consumer-to-terminal path, with the nearest explicit true or false winning. Every hop consumes depth and shared source-work budget even after type-resolution caches are populated; cycles remain bounded. All declared source members validate even when omitted or null in the annotation. Required/Patch/discarded handling, original-order value diagnostics and original/canonical value limits remain unchanged.

[Core mixed-chain regressions](../core/chained_allof_object_default_members_test.go), [original-hop identity and exact budget checks](../internal/schema/chained_allof_object_default_members_test.go) and the shared native parity/preservation test define this increment. The nested-object feature adds named and internal terminals through mixed chains, nullable overrides, fresh defaults, missing/replacement/null/required fields, Patch and optimized/locked helper binding to its four-profile HTTP/live SSE runtime coverage.

## Inline collection properties inside object defaults

An object-default property may contain a finite direct inline List/typed Map tree whose bare leaf is an ordinary disjoint allOf constructor. For example:

```json
{
  "properties": {
    "rows": {
      "type": "array",
      "items": {
        "allOf": [
          {"properties": {"label": {"type": "string"}}, "required": ["label"]},
          {"properties": {"count": {"type": "integer", "default": 2}}}
        ]
      }
    }
  },
  "default": {"rows": [{"label": "initial"}]}
}
```

This also covers typed Map values and mixed paths such as `List<Map<String, DTO>>`, recursively within existing plain, referenced or merged object-default walks and object elements. The shared type planner owns the original nonempty constructor and public collection type. Each property grants permission only to its own original inline allOf leaf on a shallow validation copy. Structurally equal nodes, sibling paths and reference edges cannot borrow that permission. The same ordinary constructor gate rejects empty/raw, const-only, hierarchy/union, wrapper, repeated-field and extras/count projections.

Each container retains independent nullable intent and supported count checks; each element retains original merged fields, required names, bounds and effective nullable intent, including member-derived true and explicit root false. Optional Kotlin presence alone does not authorize annotation null. The collection property may retain its own independently validated field default. Inner-container and element defaults remain rejected. Supplied collections replace the member default; omitted fields inside each supplied element use their own existing initializers.

Every source is validated even when the member is omitted, null or empty. The complete object shares source depth/work and original/canonical value budgets across sibling collections and nested objects. Diagnostics retain original schema locations and value order, including escaped Map keys and array indices. Required/Patch/discarded annotations validate before suppression. Decoding through the final enclosing serializer produces fresh containers, DTO elements and nested defaults, including after optimization and name locking.

Bare-leaf permission stops at reference edges. [Referenced elements in inline collection properties](#referenced-elements-in-collection-properties) use their own original-reference permission; [named collection properties](#named-collection-properties-inside-object-defaults) have a separate bounded role. Fields without defaults retain their public type and wire behavior. This closes only the direct inline collection-member subitem of D03.

[Core collection-member regressions](../core/allof_collection_object_default_members_test.go), [source identity/scope/budget checks](../internal/schema/allof_collection_object_default_members_test.go) and [native parity/check/failure preservation](../internal/cli/allof_object_default_members_process_test.go) define admission. The nested-object fixture and runtime consumer cover List/Map/mixed members, fresh values, independent nullable layers, missing/replacement/required elements, Patch and HTTP/live SSE, with optimized/locked helpers in all four feature profiles.

## Referenced elements in collection properties

An object-default property may also contain a finite direct inline List/typed Map tree whose element is an actual local `$ref` to an eligible named or internal ordinary allOf constructor. In the `rows` example above, `items` may be `{"$ref":"#/components/schemas/Value"}` with the disjoint component `Value`. Mixed paths such as `List<Map<String, ValueDto>>` and admitted finite named/internal element-reference chains retain the same contract.

Permission is scoped to the property's original element-reference node, not a copied schema or the resolved terminal alone. The source walk follows every original hop and reuses the [allOf member source rules](#referenced-allof-properties-inside-object-defaults). Internal terminals require their exact node, Pointer, current lexical owner and lookup `SourceCatalog`; the shared type plan and final linker retain the actual unique constructor. Source validation may precede final linking and does not register new declarations. Existing internal property/collection-owned Models, root collection peers and concrete union variants use their physical constructor and serializer. A concrete variant keeps its business payload without adding a root-union envelope.

Element nullability follows the explicit element reference, then nearest alias/internal-hop intent, then the terminal. Explicit false narrows true; container nullability remains independent. Every container and reference edge consumes source depth, and all siblings/terminal fields share the source-work counter even when the collection is omitted, null or empty. The original-order value diagnostics, required names, leaf checks, container counts and original/canonical value budgets apply unchanged. Required/Patch/discarded annotations validate before suppression.

The collection property may retain its separately validated own field default. Element, intermediate-hop, target and inner-container defaults remain rejected. The same constructor/hierarchy/wrapper/repeated-field/extras/count restrictions apply as for referenced object members. References ending in collections do not obtain this element role; properties referencing named collections use the [named collection member contract](#named-collection-properties-inside-object-defaults). Adding an enclosing annotation preserves target/alias declaration bytes and warnings; decoding binds the final enclosing serializer after optimization and creates fresh containers, elements and nested defaults.

[Core reference/context/nullable/guard tests](../core/referenced_allof_collection_object_default_members_test.go), [source/permission/budget checks](../internal/schema/referenced_allof_collection_object_default_members_test.go), [final retained-declaration checks](../internal/schema/internal_allof_object_default_members_test.go) and [native parity/check/failure preservation](../internal/cli/referenced_allof_object_default_members_process_test.go) define this extension. The shared nested-object feature tests named/internal mixed chains, root peers and concrete variants, missing/replacement/null/required elements, Patch, independent instances, HTTP/live SSE and optimized/locked helper binding under Data/Plain × validation and four Json configurations.

## Named collection properties inside object defaults

An object-default property may directly reference a named List/typed Map component, or a finite named alias chain to it, when its inline container tree ends in an actual reference to an eligible **named** ordinary allOf constructor. For example:

```json
{
  "Value": {
    "allOf": [
      {"properties": {"label": {"type": "string"}}, "required": ["label"]},
      {"properties": {"count": {"type": "integer", "default": 2}}}
    ]
  },
  "Rows": {"type": "array", "items": {"$ref": "#/components/schemas/Value"}},
  "Holder": {
    "properties": {
      "cfg": {
        "properties": {"rows": {"$ref": "#/components/schemas/Rows"}},
        "default": {"rows": [{"label": "initial"}]}
      }
    }
  }
}
```

Typed Map and mixed inline paths such as `List<Map<String, ValueDto>>` are also eligible. Element references may traverse already admitted named/internal hops, but must end at a named allOf component. The collection path before the containers uses only original named component nodes and finite named aliases. Each hop retains its node and Pointer, and the terminal container scopes permission to its original element reference. The shared type plan owns the collection alias and element classifier; adding the enclosing default leaves those declarations and warnings unchanged, including when the collection declaration follows its consumer.

Root nullability comes from the existing named component/alias view and consuming override. This preserves the historical direct nullable Map behavior: a direct named Map's `nullable:false` consumer does not remove its existing intrinsic nullability, while named collection aliases and arrays retain their existing narrowing behavior. Inner container and element nullability remain independent. Supported container counts and element checks apply before required/Patch/discarded suppression; supplied values replace defaults, and omitted element fields retain their own initializers. The collection property may own a separately validated default, but collection-target, intermediate-alias, element and inner-container defaults remain rejected.

Original source depth and the shared source-work budget continue across collection aliases, containers, element references and object members. Omitted/null/empty members still validate the complete source. Original-order value diagnostics, canonical keys and original/canonical value budgets remain. Decoding uses the final enclosing serializer after optimization and creates fresh containers, referenced DTOs and nested defaults.

This bounded member role excludes internal collection roots, bare allOf leaves, references between container layers and internal allOf element terminals. The corresponding consuming-field own-default roles and inline collection-member roles retain their separate contracts. General D03 remains open.

[Core named-collection/context/nullable/guard tests](../core/referenced_collection_allof_object_default_members_test.go), [original source/depth/work checks](../internal/schema/referenced_collection_object_members_test.go) and the shared [native parity/preservation test](../internal/cli/referenced_allof_object_default_members_process_test.go) define admission. The nested-object feature covers aliases, nullable Maps, mixed containers, fresh defaults, missing/replacement/null/required elements, Patch, HTTP/live SSE and optimized/locked helpers in all four profiles.

## Bounded admission

Inline fields and references to eligible named or internal constructor Models apply in ordinary shared/response/request DTOs. These remain separate:

- annotation-only reference wrappers: a single component ref plus metadata/nullable members must retain its canonical referenced type;
- target/root/component defaults and default inheritance;
- repeated-property/default/constraint aggregation, fixed extras (`additionalProperties`) and whole-object property counts;
- empty/raw and const-only projections, hierarchy/common/inherited fields and chains to other terminal roles inside object defaults;
- object-union products, which use the separate [composed union default contract](union-field-defaults.md).

Existing input8MiB, source48, composition-work16,384, checked-collection16 and constructor-slot240 limits remain. Object default walks also share the independent [16,384-step source planning budget](nested-object-defaults.md#object-source-planning-budget). The annotation retains the shared 16,384 JSON value-node and 32 KiB original/canonical JSON budgets. Exceeding a value budget returns `E_MODEL_LIMIT` at the original member; generation failure leaves existing output intact.

## Verification and compatibility

This extends M01/M03/M04/M07/M08 and the bounded D03 default profile; general D03/D05 remain open. Previously rejected eligible inline and referenced named/internal ordinary allOf object field defaults now generate. Fields without defaults retain existing source, type and wire behavior. Repeated-field aggregation, hierarchy and root/component default semantics are not added.

The [core contract tests](../core/allof_object_defaults_test.go), [source authority tests](../internal/schema/allof_object_defaults_test.go) and [native parity/preservation tests](../internal/cli/allof_object_defaults_process_test.go) define admission and diagnostics. The existing [nested object fixture](../tests/fixtures/http/nested-object-defaults.json) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NestedObjectDefaultsContracts.kt) cover ordinary allOf defaults, member refs, fresh nested instances, missing/replacement/null, Patch states and HTTP/SSE consumption under four Json configurations. Run `bash scripts/test.sh feature nested-object-defaults` for Data/Plain × validation off/on compilation/runtime and optimized/locked helper binding.

The referenced role is defined by [core admission/nullable/bridge regressions](../core/referenced_allof_object_defaults_test.go), [source and final nominal authority tests](../internal/schema/referenced_allof_object_defaults_test.go) and [CLI parity/readonly check/failure preservation](../internal/cli/referenced_allof_object_defaults_process_test.go). Its [referenced-object fixture](../tests/fixtures/http/referenced-object-defaults.json), [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedObjectDefaultsContracts.kt) and `bash scripts/test.sh feature referenced-object-defaults` exercise actual named/alias/internal-bridge defaults, intrinsic nullability, required/Patch, optimized helper binding and controlled HTTP/live SSE under the same four profiles.

The internal extension has [core scope/nullable/presence regressions](../core/internal_allof_defaults_test.go), [source/owner/depth tests](../internal/schema/internal_allof_defaults_test.go) and [native parity/check/failure preservation](../internal/cli/internal_allof_defaults_process_test.go). The same fixture/consumer exercises property and collection-owned Models, root peers, aliases/bridges, concrete composition variants, inherited nullability, fresh nested member defaults, Patch wire states and original validation through HTTP/live SSE. `OPENAPI_KT_PREPARE_ONLY=1` provides native preparation only. JVM verification does not establish Android/KMP or release acceptance.
