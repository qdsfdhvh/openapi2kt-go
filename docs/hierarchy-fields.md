# Hierarchy fixed and inherited discriminator fields

This profile extends [typed discriminator hierarchies](model-semantics.md#typed-discriminator-hierarchies) with a single-child fixed explicit level, generated mapping fields, and varying ancestor values. It preserves the existing top-level sealed classes, open enum tag types, source-order mapping selection and serializer policies.

## Single-child fixed levels

An explicit level with exactly one child may have one fixed discriminator value. The parent still exposes a typed open enum and a raw-preserving Unknown. A source const becomes an always-encoded enum body property; an actual enum, including a nullable or one-member enum, remains a required non-null constructor parameter without a default. Const-only custom leaves retain their consuming fixed-wire singleton codec. Multiple children with an identical fixed source pin remain rejected by this profile.

Mapping selection remains independent from a source const: the first mapping targeting a child wins, otherwise the original component key is lowercased. Suffixes never enter fallback wire values. Secondary mapping aliases are not extra routes, and source const values do not become extra explicit routes.

## Missing tag fields

When a leaf omits an explicit ancestor discriminator property, generation adds a typed enum body constant from the selected edge of that ancestor. For example, a `kind` mapping of `first` to a leaf containing only required `id` produces a constructor accepting `id` and an always-encoded `kind` body whose `serialName` is `first`. Several missing ancestor properties may be added to the same leaf, retaining each original ancestor enum type.

Generated properties belong to the Kotlin representation. They do not add schema properties or validate a missing source declaration. A source `required` entry naming an absent property remains invalid. Original fields, discarded allOf fields and additionalProperties sources are validated before a model is accepted. Normalized name collisions remain errors. Each generated tag uses one serialized constructor slot and counts toward the existing 240-slot limit.

A recognized dispatch route may decode a payload whose generated body tag is absent and then re-encode its fixed tag. Dispatch still requires the tag at each custom level being decoded: a missing or unknown route tag selects that level's Unknown. Original required constructor fields still fail when missing or invalid.

## Varying ancestors

A nested custom level may contain descendants with several values of an ancestor tag. Its abstract override and leaf properties keep the ancestor's existing enum type. An actual enum remains a required constructor argument; actual const bodies preserve their individual values. A subtree with one shared constant ancestor value retains the existing fixed projection.

Wire results depend on the selected serializer. An inner serializer can encode and decode an `alternate` ancestor value while an outer explicit mapping that only recognizes `group` returns its own Unknown for that payload. Encoding never invents an outer dispatch alias. Every custom layer's Unknown still preserves complete raw JSON and exposes typed empty `Unrecognized("")` stand-ins; recognized invalid known payloads fail rather than becoming Unknown.

## Boundaries and compatibility

This is an additive profile for previously rejected input. Existing accepted hierarchy declarations and wire rules remain unchanged. Several varying implicit fields and reused discriminator fields have a separate [compound dispatch contract](hierarchy-dispatch.md). Inline members named by their discriminator constant are covered by [their own contract](hierarchy-inline-members.md). Default-polymorphic fixed singleton contexts, multiple parents, primitive/nullable graph members and general internal nominal ownership still require separate contracts. Object-only union/allOf composition uses its [own contract](union-allof.md).

The source remains the original schema graph. Generated tag bodies are IR values derived from declared edges; generation does not fabricate expanded JSON schemas. Existing source depth 48, graph work 16,384, branch/enum count 240 and serialized slot 240 limits apply independently.

The [synthetic HTTP input](../tests/fixtures/http/hierarchy-fields.json), [core regressions](../core/hierarchy_fields_test.go), [native preservation checks](../internal/cli/hierarchy_fields_process_test.go) and [fresh Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/HierarchyFieldsContracts.kt) cover the field rules, four Json configurations, model styles, validation options, HTTP/Patch and live fragmented SSE closure/reuse. Independent slot and depth compilation probes do not establish every maximum-width/depth combination or Android/KMP acceptance.
