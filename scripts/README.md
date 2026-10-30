# Development tooling

`bash scripts/test.sh feature nullable-component-defaults` covers existing non-null and newly admitted null factories in fresh Data/Plain × validation profiles and four Json settings. It checks nullable aliases/serializers, physical String-enum classifiers and open unknown values, signed bytes, collections, absence/null/value, non-inheritance and copy; independent 240-member String-enum and validation-on native-enum probes retain existing caps. The full `http` registry invokes both consumers, and CI's `profiles` job selects this route. `OPENAPI_KT_PREPARE_ONLY=1` performs native generation/check without Kotlin execution.

`bash scripts/test.sh feature referenced-bytes-field-defaults` generates fresh Data/Plain × validation profiles for own defaults on existing byte/binary scalar and finite referenced collection paths. Four Json settings cover signed-array wire, original aliases/ABI/descriptors, fresh buffers, required/Patch/copy, optimized/locked models, HTTP and fragmented live SSE failure closure/reuse. Independent source48/checked16/slots240 and exact node/32KiB scalar default probes compile bounded typed helpers. `OPENAPI_KT_PREPARE_ONLY=1` performs native generation/check only; `http` generates/checks the fixture and invokes the consumer.

`bash scripts/test.sh feature referenced-string-enum-null-member-defaults` generates fresh Data/Plain × validation consumers for existing referenced scalar/finite collection field-own defaults with explicitly nullable String enum null-member vocabularies. Four Json settings cover actual classifiers/nullable aliases, null versus `"null"`, strict annotations/open runtime, fresh trees, narrowed leaf rejection, required/Patch/copy, controlled HTTP and finite SSE, optimized/merged owner identity and independent 240-field/raw240-vocabulary probes. `OPENAPI_KT_PREPARE_ONLY=1` is native preparation only; `http` also generates/checks the fixture and calls its consumer.

`bash scripts/test.sh feature referenced-string-enum-scalar-defaults` runs fresh Data/Plain × validation profiles for plain String/typeless enum references' own scalar field defaults. It checks actual lexical types/aliases, file-wide helper reuse, escaping, four Json settings, required/Patch, strict annotations and open unknown Strings, optimization/locks, controlled HTTP and finite SSE decoding/reuse, plus independent 240-field and 240-member compilation probes. Native preparation with `OPENAPI_KT_PREPARE_ONLY=1` is not Kotlin execution evidence. The fixture and consumer also run in `http`.

`bash scripts/test.sh feature referenced-native-enum-scalar-defaults` runs fresh Data/Plain × validation profiles for native numeric/Boolean enum references' own scalar field defaults. It checks literal/alias ABI, minimum values, strict annotation vocabulary, four Json settings, required/Patch, checked internal aliases, optimization/locks, controlled HTTP and finite SSE decoding/reuse, plus independent 240-field and 240-member compilation probes. `OPENAPI_KT_PREPARE_ONLY=1` provides native generation/check evidence only. The fixture and consumer also run through `http`.

`bash scripts/test.sh feature referenced-object-defaults` is the consumer verification entry point for [referenced ordinary DTO object field defaults](../docs/referenced-object-defaults.md). It covers ordinary direct local `$ref` fields' own `default`, [existing named ordinary allOf DTOs](../docs/allof-object-defaults.md#references-to-named-allof-dtos) (disjoint fields, including named aliases/internal bridges), and [unique internal allOf constructor Models](../docs/allof-object-defaults.md#references-to-internal-allof-dtos) (properties/collection leaves/root peers/concrete variants, member-derived nullability), transparent alias folding, required/Patch, nearest nullability, four Json configurations, validation membership, typed HTTP, and fragmented live SSE. It also generates a combined optimized/locked package to verify helper binding to relocated types. `bash scripts/test.sh http` generates and checks the complete fixture and calls the full HTTP consumer body directly in the `HttpContracts` registry. Existing source48/checked16/slots240/tree16,384/default32KiB caps remain unchanged. `OPENAPI_KT_PREPARE_ONLY=1` performs only native generation/check and project preparation, not a Kotlin compilation/runtime PASS. The separate Go/native regression is `go test ./internal/cli -run '^Test(Referenced(Object|AllOf)Defaults|InternalAllOfDefaults)'`. Platform, unified candidate, and release acceptance are separate.

`bash scripts/test.sh feature raw-json-field-defaults` is the consumer verification entry point for [raw `JsonElement`/`JsonObject` field defaults](../docs/raw-json-field-defaults.md). It covers open dictionary/unconstrained/empty object/request-context union projections, file-level parse helpers, required/Patch/explicit null, four Json configurations, typed HTTP, and fragmented live SSE, and generates a combined optimized/locked package. `bash scripts/test.sh http` generates and checks the complete fixture and calls the full HTTP consumer body directly in the `HttpContracts` registry. Existing 8MiB/source48/checked16/slots240 budgets and approximation warnings remain unchanged. `OPENAPI_KT_PREPARE_ONLY=1` performs only native generation/check and project preparation, not a Kotlin compilation/runtime PASS. The separate Go/native regression is `go test ./internal/cli -run '^TestRawJSON'`. Platform, unified candidate, and release acceptance are separate.

`bash scripts/test.sh feature union-field-defaults` is the consumer verification entry point for [inline primitive union field defaults](../docs/union-field-defaults.md). It covers plain String/Integer/Number/Boolean branches' own `default`, generation-time enum/const pin membership (including direct local `$ref` scalar/enum branches), union members in container elements/object properties, tag/presence-dispatched object union variants and referenced root union component validation, original token preservation and runtime selector wrappers, nullable/required, [closed subtype root union field defaults](../docs/closed-union-defaults.md) (actual SerialName, default/custom discriminator, array polymorphism, nullable root, and three Patch states), four Json configurations, typed HTTP, and fragmented live SSE. It generates a combined optimized/locked package to verify helper binding to relocated union types. `bash scripts/test.sh http` generates and checks the complete fixture and calls the full HTTP consumer body directly in the `HttpContracts` registry. Existing source48/checked16/slots240/tree16,384/default32KiB caps and W_PRIMITIVE_UNION_APPROXIMATION remain unchanged. `OPENAPI_KT_PREPARE_ONLY=1` performs only native generation/check and project preparation, not a Kotlin compilation/runtime PASS. The separate Go/native regression is `go test ./internal/cli -run '^Test(UnionFieldDefaults|ClosedUnionDefaults)'`. Platform, unified candidate, and release acceptance are separate.

`bash scripts/test.sh feature referenced-object-collection-defaults` is the consumer verification entry point for [collection field defaults with referenced ordinary DTO elements](../docs/referenced-object-collection-defaults.md). It covers direct local `$ref` DTO elements in inline `array`/`Map` and DTO elements in named collection `$ref`, [allOf DTO element defaults in referenced finite collections](../docs/allof-object-defaults.md#referenced-collection-field-defaults) (named/internal constructors, aliases, and mixed List/Map trees), [referenced allOf DTO element defaults in inline collections](../docs/allof-object-defaults.md#direct-inline-collections-with-referenced-elements) (final nominal/source admission, pre-presence type, alias/internal bridge), and [bare allOf constructor elements in referenced collection trees](../docs/allof-object-defaults.md#bare-allof-elements-in-referenced-collections) (named List/Map/mixed/alias, original retained source/owner and package-relative peer types). It also covers per-layer nullability/required/Patch, fresh nested member defaults, four Json configurations, validation membership/counts, typed HTTP, and fragmented live SSE, and generates a combined optimized/locked package to verify helper binding to relocated types. `bash scripts/test.sh http` generates and checks the complete fixture and calls the full HTTP consumer body directly in the `HttpContracts` registry. Existing source48/checked16/slots240/tree16,384/default32KiB caps remain unchanged. `OPENAPI_KT_PREPARE_ONLY=1` performs only native generation/check and project preparation, not a Kotlin compilation/runtime PASS. The separate Go/native regression is `go test ./internal/cli -run '^Test(Referenced(ObjectCollectionDefaults|AllOfCollection|InlineAllOf)|InlineReferencedAllOf)'`. Platform, unified candidate, and release acceptance are separate.

`bash scripts/test.sh feature nested-object-collection-defaults` is the consumer verification entry point for [direct inline List/typed Map field defaults of ordinary DTO objects](../docs/nested-object-collection-defaults.md). It covers direct inline `array`/typed `Map` outer `default`, recursive plain object elements, and the original source-scoped walk for [bare ordinary allOf constructor leaves](../docs/allof-object-defaults.md#direct-inline-collection-field-defaults), List/Map/mixed trees, per-layer nullability, fresh nested defaults, required/Patch, four Json configurations, validation membership/counts, typed HTTP, and fragmented live SSE. It generates a combined optimized/locked package to verify helper binding to relocated element types; scalar-only collections retain the original builtin/temporal/enum helpers. `bash scripts/test.sh http` generates and checks the complete fixture and calls the full HTTP consumer body directly in the `HttpContracts` registry. Existing source48/checked16/slots240/tree16,384/default32KiB caps remain unchanged. `OPENAPI_KT_PREPARE_ONLY=1` performs only native generation/check and project preparation, not a Kotlin compilation/runtime PASS. The separate Go/native regression is `go test ./internal/cli -run '^Test(NestedObjectCollectionDefaults|InlineAllOfCollection)'`. Platform, unified candidate, and release acceptance are separate.

`bash scripts/test.sh feature nested-object-defaults` is the consumer verification entry point for [direct inline ordinary DTO object field defaults](../docs/nested-object-defaults.md). It covers ordinary direct inline plain object fields' own `default`, [ordinary allOf object field defaults with disjoint properties](../docs/allof-object-defaults.md), direct local `$ref`-valued properties, finite nested objects, builtin/temporal/enum leaves, nested builtin collections, per-layer nullability, required/Patch, four Json configurations, validation membership, typed HTTP, and fragmented live SSE. It generates a combined optimized/locked package to verify helper binding to relocated types. `bash scripts/test.sh http` generates and checks the complete fixture and calls the full HTTP consumer body directly in the `HttpContracts` registry. Existing source48/checked16/slots240/tree16,384/default32KiB caps remain unchanged. `OPENAPI_KT_PREPARE_ONLY=1` performs only native generation/check and project preparation, not a Kotlin compilation/runtime PASS. The separate Go/native regression is `go test ./internal/cli -run '^TestNestedObjectDefaults'`. Platform, unified candidate, and release acceptance are separate.

`bash scripts/test.sh feature nested-native-enum-collection-defaults` is the consumer verification entry point for [directly nested native enum own defaults](../docs/nested-native-enum-collection-defaults.md). It covers ordinary direct inline finite List/typed Map, the five existing native enum terminals, Data/Plain × validation, four Json presence/default/check configurations, actual optimizer aliases/locks, typed HTTP, and fragmented live SSE. `bash scripts/test.sh models` exports ordinary models through `TestExportModelSmoke` and extends the existing `runEnumContextContracts` via `EnumContextContracts.main` without adding a model runtime suite. `bash scripts/test.sh http` generates and checks the complete fixture; the existing `runHttpContracts` calls the full HTTP consumer body directly in the `HttpContracts` suspend registry. Existing source48/checked16/slots240/default32KiB caps remain unchanged. `OPENAPI_KT_PREPARE_ONLY=1` performs only native generation/check and project preparation, not a Kotlin compilation/runtime PASS. The separate Go/native regression is `go test ./internal/cli -run '^TestNestedNativeEnumCollectionDefaults'`. Platform, unified candidate, and release acceptance are separate.

`bash scripts/test.sh feature referenced-native-enum-collection-defaults` generates fresh Data/Plain × validation consumers for ordinary direct local `$ref` collection fields' own defaults with native numeric/Boolean enum leaves (int32/int64/float/double/Boolean, typed Map values, nullable root containers and scalar leaves, alias chains and internal bridges). Four Json settings cover scalar-wire aliases, strict generation-time default membership, retained validation bounds/counts, required/Patch, optimization/locks, HTTP and fragmented live SSE closure/reuse; independent source48/checked16/slots240/default32KiB probes retain existing caps. `OPENAPI_KT_PREPARE_ONLY=1` performs native generation/check only; the full `http` route generates/checks the fixture and calls the consumer once.

`bash scripts/test.sh feature referenced-string-enum-collection-defaults` generates fresh Data/Plain × validation consumers for ordinary direct local `$ref` collection fields' own defaults with plain String enum leaves (named/typeless/inline, per-layer nullable/null members, alias chains and internal bridges). Four Json settings cover actual lexical classifiers, open `Unrecognized` runtime, strict default vocabulary, required/Patch, optimization/locks, HTTP and fragmented live SSE closure/reuse; independent source48/checked16/slots240/default32KiB probes retain existing caps. `OPENAPI_KT_PREPARE_ONLY=1` performs native generation/check only; the full `http` route generates/checks the fixture and calls the consumer once. 

`bash scripts/test.sh feature referenced-temporal-collection-field-defaults` generates fresh Data/Plain × validation consumers for ordinary direct local `$ref` collection fields' own defaults with standard LocalDate/Instant leaves through already accepted named/internal paths. Four Json settings cover actual aliases/ABI/descriptors, temporal and builtin6 values/checks, required/Patch, independent trees, optimization/locks, HTTP and fragmented live SSE closure/cold reuse. Independent source48/checked16/slots240/default32KiB probes retain existing caps. `OPENAPI_KT_PREPARE_ONLY=1` performs native generation/check and preparation only; the full `http` route generates/checks the fixture and calls the consumer once.

`bash scripts/test.sh feature direct-nullable-component-counts` generates fresh Data/Plain × validation consumers for direct explicitly nullable ordinary component DTO own counts. Validation-off uses accepted no-count controls and verifies original counted rejection; validation-on checks four Json physical-object/null/default/unknown/Patch settings, actual delegates/aliases, optimized/locked identity, HTTP and fragmented SSE failure closure/reuse. Independent source48/checked16/shallow240 probes keep existing caps. `OPENAPI_KT_PREPARE_ONLY=1` prepares/checks native output without JVM evidence; the full `http` route generates the fixture and invokes its consumer once.

`bash scripts/test.sh feature referenced-scalar-field-defaults` generates fresh Data/Plain × validation off/on consumers for ordinary direct local `$ref` field own defaults ending in admitted builtin6 scalars. Four Json settings cover actual aliases/builtin ABI/descriptors, required/Patch, constructor/copy/serde, optimization/name-lock and HTTP/fragmented SSE failure closure/reuse. Independent source48/checked16/slots240/default32KiB probes retain existing limits. `OPENAPI_KT_PREPARE_ONLY=1` prepares/checks native output without JVM execution; the full `http` route also supports this preparation flag.

`bash scripts/test.sh feature nested-temporal-collection-defaults` generates fresh Data/Plain × validation consumers for direct finite LocalDate/Instant List/typed Map defaults, four Json states, actual aliases/classifiers/descriptors under opt/lock, typed HTTP and fragmented live SSE. Independent source48/checked16/slots240/default32KiB probes keep existing caps. `OPENAPI_KT_PREPARE_ONLY=1` runs native generation/check only.

`bash scripts/test.sh feature string-enum-collection-bounds` generates Data/Plain × validation checked/unbounded controls for ordinary DTO direct inline String enum List/typed Map bounds. Four Json settings retain open payloads, strict defaults, required/Patch, constructor/copy/serde, actual optimized lexical owners/descriptors and HTTP/fragmented SSE failure closure/reuse. `bash scripts/kotlin/features/string-enum-collection-bounds.sh --prepare-only` prepares/checks fresh native output without JVM execution. The full `http` route generates the bounded fixture and calls its consumer.

`bash scripts/test.sh feature referenced-collection-field-defaults` builds fresh Data/Plain × validation consumers for ordinary direct-reference builtin collection field defaults. It retains actual aliases/checks, four Json/default/Patch states, optimized/name-locked call sites and HTTP/still-open fragmented SSE closure/reuse, with independent source48/checked16/slots240/default32KiB probes. `OPENAPI_KT_PREPARE_ONLY=1` performs native generation/check only and provides no Kotlin compilation evidence.

`bash scripts/test.sh feature nested-builtin-collection-defaults` freshly generates Data/Plain × validation consumers for finite direct DTO builtin List/typed Map trees. It checks four Json configurations, each nullable layer, required/Patch, recursive counts/leaf checks, actual optimization/name-lock consumption, HTTP and still-open fragmented SSE failure closure/reuse, with independent width/depth/byte probes.

`bash scripts/test.sh feature http-servers` verifies server variable defaults, operation/path/document precedence, explicit caller overrides, relative URLs, path encoding and real SSE cancellation/closure with fresh Data/Plain × validation and four Json configurations.

`bash scripts/test.sh feature yaml-document` compares fresh JSON/YAML files/manifests/warnings, then compiles Data/Plain × validation with four Json settings for serde/Patch/GET/query/PATCH/response wire. See the [YAML profile](../docs/yaml-input.md).

`bash scripts/test.sh feature string-enum-collection-defaults` verifies direct List/typed Map String enum defaults with actual lexical classifiers, strict annotation membership and open runtime Unrecognized values. Fresh Data/Plain × validation profiles cover four Json configurations, required/Patch, counts, actual dedup/merge, combined hoist/name-lock, HTTP and fragmented SSE closure/reuse.

`bash scripts/test.sh feature http-namespace-collisions` verifies unchanged parameter/wire names and safe HttpMethod/ContentType/Flow/SSE/aggregate constructor aliases across ten package prefixes in fresh Data/Plain × validation and four Json configurations, including cold streaming closure/reuse.

`bash scripts/test.sh feature native-enum-collection-defaults` verifies numeric/Boolean enum List/Map defaults, required/Patch, original vocabulary matching, nullable members and checked bounds/counts in fresh Data/Plain × validation profiles, four JSON configurations, HTTP and fragmented SSE closure/reuse.

`bash scripts/test.sh feature primitive-map-defaults` exercises typed builtin/date/date-time dictionary defaults, key/value escaping, timezone normalization, required/Patch, opt-in counts/value checks and HTTP/live SSE closure/reuse with Data/Plain × validation and four JSON configurations.

`bash scripts/test.sh feature checked-array-defaults` exercises validation-enabled counts, scalar uniqueness and item bounds with Data/Plain, four JSON configurations, Patch and HTTP/live SSE.

`bash scripts/test.sh feature primitive-array-defaults` exercises typed defaults, nullable members, required/Patch rules and HTTP/live SSE with Data/Plain × validation and four JSON configurations.

`bash scripts/test.sh feature nullable-structured-const` covers canonical nullable roots/aliases, false narrowing, body defaults and typed HTTP/SSE null handling in fresh Data/Plain × validation and four JSON configurations.

`bash scripts/test.sh feature structured-const-values` verifies roots/aliases and value leaves, ordinary body references and typed HTTP/live SSE under fresh Data/Plain × validation and all four JSON settings.

`bash scripts/test.sh feature structured-const-fields` exercises fresh Data/Plain × validation, four JSON settings, typed body/default/kind/Patch, HTTP and fragmented live SSE cancellation/closure.

`bash scripts/test.sh feature null-only-defaults` compiles fresh Data/Plain × validation profiles for null default annotations, four Json settings, required/optional/Patch and real HTTP/live SSE. The full `http` route generates the same fixture and runs both the new default and existing null consumers.

`bash scripts/test.sh feature inferred-null-literals` compiles the existing null consumer against a separate typeless const/enum fixture in fresh Data/Plain × validation projects, with four Json settings and actual HTTP/live SSE.

`bash scripts/test.sh feature explicit-null-literals` builds fresh Data/Plain projects with validation off/on and runs four Json configurations for canonical JsonNull, const/singleton, presence/Patch and HTTP/live SSE. The `http` suite includes the same consumer.

`bash scripts/test.sh feature referenced-allof-counts` freshly compiles Data/Plain whole-object counted DTOs, then runs four Json configurations per style for source counts, retained delegate/default/const/Patch/null, non-JSON rejection and actual HTTP/live SSE. `http` includes the same fixture and consumer.

`bash scripts/test.sh feature referenced-allof-bounds` generates fresh Data/Plain ordinary DTOs with referenced/nested scalar conjunctions, then checks four Json configurations per style, nullable/Patch, numeric bounds and actual HTTP/live SSE. `http` includes the same fixture and consumer.

`bash scripts/test.sh feature string-enum-null-members` runs fresh Data/Plain × validation on/off for original nullable String vocabularies, unknown tokens, defaults/Patch and HTTP/SSE under four Json settings.

`bash scripts/test.sh feature native-enum-null-members` runs fresh Data/Plain profiles and four Json settings for original null vocabulary members, defaults, Patch and HTTP/SSE typed checks.

`bash scripts/test.sh feature nullable-native-enums` runs fresh Data/Plain with four Json settings for nullable alias codecs/Patch and actual HTTP null body/result/SSE checks.

`bash scripts/test.sh feature named-native-enums` runs fresh Data/Plain with four Json settings for named checked source/rootbridge/ListMap/Patch and HTTP/SSE behavior.

`bash scripts/test.sh feature internal-native-enums` runs fresh Data/Plain and four Json settings for original Pointer/root bridge/nullable narrowing/ListMap/Patch and HTTP/SSE checks.

`bash scripts/test.sh feature native-enum-bounds` runs fresh Data/Plain with four Json settings for combined numeric membership/bounds/default/Patch and HTTP/SSE checks.

`bash scripts/test.sh feature native-enum-consumers` runs fresh Data/Plain × validation on/off with four Json settings for alias/List/Map/Patch, operation preflight/result/SSE checks and default-off parity.

`bash scripts/test.sh feature native-enum-validation` runs fresh Data/Plain × validation on/off four-Json tests for mapped numeric/Boolean membership, defaults/Patch and HTTP/SSE.

`bash scripts/test.sh feature merge-exclusions` runs fresh data/plain combined optimizer profiles and four Json settings for protected classifier identities, aliases, validation and HTTP.

`bash scripts/test.sh feature inline-errors` runs Data/Plain × validation and four Json settings for typed status/borrowed error DTOs, cancellation/rereading and caller-owned HTTP validation.

`bash scripts/test.sh feature json-unique` runs validation-enabled Data/Plain with four Json settings for recursive JSON equality, runtime limits, Patch/aliases, HTTP and SSE.

`bash scripts/test.sh feature enum-defaults` runs Data/Plain × validation and four Json settings for lexical String enum/default vocabulary, native scalar defaults, required/Patch, allOf/variants/List/Map and HTTP/SSE.

`bash scripts/test.sh feature model-defaults` runs Data/Plain × validation with all four Json settings for literal defaults, required/Patch presence, temporal shadowing, unions/containers, HTTP and SSE.

`bash scripts/test.sh feature nullable-object-reference-members` runs Data/Plain × validation for actual ordinary DTO/reference members, four Json settings, original alias authority/variants/carriers/null-false codecs/Patch/HTTP/SSE and independent source/branch probes.

`bash scripts/test.sh feature nullable-object-members` runs Data/Plain × validation for original ordinary nullable object members, four Json settings, carriers/aliases/codecs/Patch/HTTP/SSE and independent source/branch probes.

`bash scripts/test.sh feature nullable-product-unions` runs Data/Plain × validation with explicit allOf product null intent, four Json settings, actual Null carrier/inline codecs, Patch/HTTP/SSE and independent source/branch probes.

`bash scripts/test.sh feature union-inline-values` runs Data/Plain × validation with original variant-owned inline object/union/ListMap constructors, four Json settings, Patch, HTTP/SSE and independent width/checked-depth probes.

These scripts verify or package the generator; they are not needed to run the installed CLI.

## Test entry point

From the repository root (the runner also works via its absolute path from other directories):

```bash
bash scripts/test.sh unit                       # format, vet, Go tests/race, Python tools
bash scripts/test.sh models                     # fresh generated Kotlin model contracts
bash scripts/test.sh http                       # fresh Ktor HTTP/SSE contracts
bash scripts/test.sh feature collection-owned-dto
bash scripts/test.sh feature root-dto-bridge
bash scripts/test.sh feature union-allof        # borrowed object union/allOf combinations
bash scripts/test.sh feature union-common-dictionaries # public Map/JsonObject properties and original branch checks
bash scripts/test.sh feature union-common-nominals # actual DTO/union/hierarchy common field codecs
bash scripts/test.sh feature nullable-allof-dto # canonical nullable allOf constructor/codecs and HTTP/SSE
bash scripts/test.sh feature hierarchy-dispatch # multiple inferred/reused hierarchy tags
bash scripts/test.sh feature hierarchy-fields   # fixed/missing/varying ancestor tags
bash scripts/test.sh feature hierarchy-intrinsic-nullable # root/intermediate/leaf intrinsic intent
bash scripts/test.sh feature hierarchy-reference-chains # internal chains to actual named hierarchies
bash scripts/test.sh feature hierarchy-nested-owners # retained hierarchy leaf property/list/map DTO identities
bash scripts/test.sh feature composed-nested-owners # actual lexical allOf/union DTO owners and ambiguity guards
bash scripts/test.sh feature allof-nominal-terminals # retained allOf constructor terminals and inherited nullable intent
bash scripts/test.sh feature union-nominal-owners # retained union scopes, nullable views and canonical Null carriers
bash scripts/test.sh feature enum-nominal-owners # retained nested vocabularies, unknown strings and nullable/false codecs
bash scripts/test.sh feature valueclass-reference-chains # canonical wrappers, cached hops, nullable/false and JVM/HTTP layout
bash scripts/test.sh feature composed-union-terminals # retained product classifiers, aliases, nullable/false and independent limits
bash scripts/test.sh feature borrowed-collection-union-leaves # original retained union dispatch/carriers, ListMap/Patch/HTTP-SSE
bash scripts/test.sh feature borrowed-collection-product-union-leaves # original retained union dispatch/carriers, ListMap/Patch/HTTP-SSE
bash scripts/test.sh feature borrowed-collection-hierarchy-leaves # original retained union dispatch/carriers, ListMap/Patch/HTTP-SSE
bash scripts/test.sh feature borrowed-collection-builtin-leaves # original named native backing/checks, nullable ListMap/Patch/HTTP-SSE
bash scripts/test.sh feature borrowed-collection-temporal-leaves # original Instant/LocalDate source/ISO, nullable ListMap/Patch/HTTP-SSE
bash scripts/test.sh feature borrowed-collection-bigint-leaves # Long container numbers vs scalar-field/Patch strings and HTTP/SSE
bash scripts/test.sh feature borrowed-collection-bytes-leaves # ByteArray signed JSON containers vs raw standalone HTTP
bash scripts/test.sh feature borrowed-collection-caller-native-leaves # real Money/native providers through ListMap/Patch/HTTP-SSE
bash scripts/test.sh feature borrowed-collection-raw-true-leaves # original raw/true JSON source and nullable List/JsonObject codecs
bash scripts/test.sh feature borrowed-collection-recursive-cover-leaves # original graph/SCC-backed JSON List/JsonObject containers
bash scripts/test.sh feature borrowed-collection-allof-model-leaves # original actual allOf Model/shape source authority, ListMap/Patch/HTTP-SSE
bash scripts/test.sh feature borrowed-collection-valueclass-leaves # original wrapper/InlineValue/check source authority, ListMap/Patch/HTTP-SSE
bash scripts/test.sh feature borrowed-collection-enum-leaves # original vocabulary/source authority, nullable ListMap/Patch/HTTP-SSE
bash scripts/test.sh feature borrowed-collection-dto-leaves # original actual DTO/alias-node authority, nominal barrier, ListMap/Patch/HTTP-SSE
bash scripts/test.sh feature root-borrowed-collection # original internal graph/root alias/check-cache authority, ListMap/Patch/HTTP-SSE
bash scripts/test.sh feature named-collection-reference-chains # actual alias/linked backing/peers, container-null/checks and source budgets
bash scripts/test.sh feature recursive-cover-reference-chains # original graph/actual alias authority, incoming hops + SCC suffix, JSON codecs
bash scripts/test.sh feature raw-true-reference-chains # actual raw/true source authority, all JSON kinds, nullable/Patch and source budgets
bash scripts/test.sh feature caller-native-reference-chains # original native profile/real provider APIs-codecs/native wire and source budgets
bash scripts/test.sh feature bytes-reference-chains # real byte/binary source authority, signed-array JSON and raw Ktor bytes
bash scripts/test.sh feature bigint-reference-chains # original bigint authority and field-string/numeric-root wire through chains
bash scripts/test.sh feature root-borrowed-builtin # original internal leaf authority, native root aliases, checked HTTP/SSE and resource inputs
bash scripts/test.sh feature builtin-reference-chains # native builtin backing/checks, path-local null views, HTTP/SSE and source/layout limits
bash scripts/test.sh feature temporal-reference-chains # temporal scalar backing, ISO/null codecs, mixed source hops and query layout
bash scripts/test.sh feature hierarchy-null-members # pure null/nullable refs/aliases in named hierarchy edges
bash scripts/test.sh feature hierarchy-nullable-alias # nullable aliases to canonical hierarchies
bash scripts/test.sh feature root-internal-dto
bash scripts/test.sh feature named-dto-chain
bash scripts/test.sh feature named-dto-alias-chain
bash scripts/test.sh features                   # all focused profile suites
bash scripts/test.sh all                        # all of the above; intentionally expensive
bash scripts/test.sh --list
```

Hosted CI first runs the three-host Go format/vet/test/race gate. After it passes, native generation/package checks and separate Ubuntu Kotlin/JVM jobs run. The Kotlin matrix executes fresh `models`, full `http`, and the selected Data/Plain × validation profiles (`directional-models` and `raw-response-media`). It pins JDK 21 and Gradle 9.3.1 and prepares declared Kotlin dependencies explicitly before the existing offline test commands. CI configuration is not evidence that a particular remote run passed; historical generation-only runs do not acquire Kotlin evidence retroactively.

`bash scripts/kotlin/prepare-dependencies.sh` is an explicit online development/CI preparation command. It copies the declared HTTP consumer build into an ignored scratch project and compiles/runs a small serialization/datetime/Ktor MockEngine probe, warming compiler and runtime dependencies. It does not read an OpenAPI document or perform a real HTTP request. `--offline` verifies preparation against an existing cache; missing dependencies fail. Generation and `scripts/test.sh` never invoke preparation automatically. Platform consumers use the separate `bash scripts/kotlin/prepare-platform-dependencies.sh TARGET [--offline]` for their declared target dependencies; `scripts/kotlin/platforms.sh` remains an offline fresh four-input probe. Android SDK installation stays separate. The local `unit` command still runs both Go and Python checks.

The complete HTTP runner generates/checks every existing fixture before compiling. `kotlin/http_compilations.py` then assigns every generated/provider/consumer Kotlin source exactly once to sequential Gradle compilations, retaining recursive type components and shared internal test helpers. The original `HttpContractsKt` registry runs with all compiled outputs on its classpath. `build/http-compilations/inventory.json` in the fresh project records the complete assignment, and preparation-only mode builds the same compilation plan. Packing uses a source-size target; an indivisible large width/depth probe stays intact even above that target. No schema cap, fixture, runtime assertion or compiler heap setting is relaxed. This avoids compiling the entire independent fixture corpus in a single compiler invocation; it is not a product performance or maximum-input memory guarantee.

Go commands are offline. Kotlin suites additionally need Python 3, Gradle/JDK and cached Kotlin/Ktor dependencies. `GO`, `GOFMT`, `PYTHON` and `GRADLE` select existing tools; model verification also accepts `OPENAPI_KT_BINARY`.

The runner and CI pass their actual Bash executable to Python entry-point tests via `OPENAPI_KT_TEST_BASH` (a native Windows path under Git Bash). This avoids resolving a different `bash` launcher when Python starts a subprocess. Direct Python test invocations may set this variable to select an existing compatible Bash; it does not change generator behavior.

## Layout

| Directory | Purpose |
|---|---|
| `kotlin/` | Model/HTTP runners, native exporter and shared offline bootstrap |
| `kotlin/features/` | Focused runtime profiles, including option and resource boundaries |
| `fixtures/` | Standalone fixture export |
| `perf/` | Opt-in specification inventory and correctness-checked benchmarks |
| `publication/` | Local candidate scans, without printing matched values |
| `release/` | Allowlisted binary archives; no uploads |
| `tests/` | Python tests for these tools |

```bash
python3 scripts/publication/check.py
python3 scripts/publication/check.py --policy .pi/public-policy.json
python3 scripts/perf/benchmark.py --help
python3 scripts/perf/spec_inventory.py --help
```

Old flat `scripts/kotlin-*-smoke.sh` entry points moved into `kotlin/` and `kotlin/features/`; use `test.sh` for routine checks. Coverage, fixture inputs, Kotlin dependencies and resource caps are unchanged. See [release instructions](../docs/releases.md) and [publication checks](../docs/publication.md).

`bash scripts/test.sh feature object-default-graphs` verifies [object-default member collection reference graphs](../docs/object-default-graphs.md): Data/Plain × validation, four Json configurations, default instance/factory isolation, Patch, HTTP/live SSE, optimized/name-lock, and independent width240/depth15 probes. `http` also registers this consumer directly.

`bash scripts/test.sh feature inline-error-roots` verifies [shared-model JSON error roots](../docs/inline-errors.md#shared-model-error-roots): Data/Plain × validation, four Json configurations, allOf/union/enum/typed List/Map/const/primitive/null/true/temporal, repeated errorBody reads, cancellation identity, optimized name locks, and a 240-field error DTO. The full `http` route also registers this consumer.

`bash scripts/test.sh feature tuple-references` verifies [builtin scalar tuple refs](../docs/primitive-tuples.md#referenced-builtin-scalar-members-and-tails): the original primitive fixture is retained, with independent Data/Plain × validation, four Json configurations, Patch, typed/mixed/tail, nullability, temporal failures, and HTTP/live SSE cancellation/reuse; the full `http` route registers this independent consumer.

`bash scripts/test.sh feature split-locked-adoption` verifies real distinct-owner file subsets whose union preserves the original locked Kotlin tree, metadata-only adoption/check/validate/v4 replay and ordinary configuration check. Fresh Data/Plain × validation off/on projects also compile singleton projections and run four Json configurations, descriptors, defaults, open enums and controlled HTTP. The `http` suite includes this consumer; PREPARE_ONLY does not execute Kotlin.

`bash scripts/test.sh feature tuple-enum-references` compiles/runs fresh Data/Plain × validation profiles with named/internal String enum tuple references, all optimization options plus locking, four Json settings, 240 fields, open/nullable vocabularies, required/Patch, controlled HTTP and live SSE/cancellation. `http` includes the same consumer. PREPARE_ONLY prepares files without Kotlin execution.

`bash scripts/test.sh feature merge-primitive-unions` compiles/runs Data/Plain × validation profiles with unmerged, merge-only and all-option locked packages. It verifies root/sibling wrapper/null aliases, canonical descriptors/equality, four Json settings, overload routes and live SSE error/cancellation/reuse; locks are compared before/after enabling merge. `http` includes the same consumer; PREPARE_ONLY is preparation only.

`bash scripts/test.sh feature tuple-dto-references` compiles/runs Data/Plain × validation with actual existing DTO tuple references, baseline/all-option packages and nonempty locks, 240 fields, four Json settings, fresh original defaults, checks/counts, nullable/internal/self-recursive types, Patch and controlled HTTP/live SSE/error/cancellation/reuse. Full `http` includes the same consumer; PREPARE_ONLY executes no Kotlin.

`bash scripts/test.sh feature form-request-media` compiles fresh Data/Plain × validation baseline and dedup/hoist/merge/lock outputs for [required scalar form requests](../docs/form-request-media.md), including a 240-field DTO, four Json settings, exact UTF-8 form bytes, JSON priority, namespace aliases, callback body replacement and fragmented SSE close/cancellation/reuse. The complete `http` registry executes the same consumer. Other targets and broader media remain separate.

`bash scripts/test.sh feature tuple-allof-references` exercises [allOf constructor tuple refs](../docs/primitive-tuples.md#referenced-allof-constructor-members-and-tails) in Data/Plain × validation, baseline/allopts/nonemptylocks, width240 and fourJson, retaining original defaults/open enums/checks/actual source owners/nominal recursion/Patch/HTTP/fragmented SSE close/cancel/reuse. Go contracts separately cover cumulative16384 source node/edge/ref and8MiB snapshot-byte accounting, repeated-target reuse and old non-tuple roles. Full `http` invokes the same consumer; PREPARE_ONLY and other targets remain separate.

`bash scripts/test.sh feature tuple-target-defaults` covers [independent target/alias default factories consumed by tuple refs](../docs/tuple-target-defaults.md), with Data/Plain × validation, baseline/allopts/nonemptylocks,240fields/fourJson, original factory values and fresh helpers, no inheritance, required decode failures, temporal/open-enum codecs, typed HTTP/Patch/fragmented SSE close/cancel/reuse. Full `http` invokes the same consumer. Source guards/native strict-default failures and bounded fuzz are separate; PREPARE_ONLY/platform claims remain separate.

`bash scripts/test.sh feature raw-request-media` verifies [required raw String/ByteArray requests](../docs/raw-request-media.md) with fresh Data/Plain × validation and baseline/allopts/lock outputs, fourJson, exact UTF-8/ISO-8859-1 and signed/empty bytes, JSON/form/raw priority, namespace aliases, callback override, validation preflight and fragmented SSE close/error/cancel/reuse. Full `http` registry invokes the same consumer; other platforms remain separate.

`bash scripts/test.sh feature tuple-field-defaults` verifies [scalar tuple own-field defaults](../docs/tuple-field-defaults.md), fresh Data/Plain × validation, baseline/allopts/lock,240fields/fourJson, independent helpers/nullable/temporal/required/Patch and HTTP/fragmented SSE. Validation profiles use explicit scalar items tails for the existing uniqueItems source policy; prefix-only enforcing uniqueness remains a Go negative. Node/byte/source and native failure preservation are separate; full `http` invokes the consumer and other targets remain open.

`bash scripts/kotlin/platforms.sh TARGET [data_class|plain_class]` retains the original catalog/model-increments consumers and adds [roadmap-increments](../samples/kotlin-platform-smoke/README.md) under validation+combined options. Its exact three-input Data/Plain profile passed fixed-version JVM/JS/Native runtime, metadata and Android AAR compilation; Android device/HTTPengine/fullfeature/minimumversion acceptance remain separate.

`bash scripts/kotlin/platforms.sh android-device [data_class|plain_class]` uses an explicitly selected `ANDROID_SERIAL` to run the current four-input common model consumer without an Activity. It audits the final min SDK23 APK manifest, refuses installed-package replacement, checks raw instrumentation results and removes the package installed this run. See [prerequisites and evidence scope](../samples/kotlin-platform-smoke/README.md#headless-android-device-execution). Offline SDK tools can be selected with `ANDROID_HOME`, `ADB`, `ANDROID_APKANALYZER` and `ANDROID_AAPT2`.

`python3 scripts/cli/host_probe.py --binary PATH --report REPORT.json [--compare OTHER.json]` executes a supplied local CLI against ten fixed JSON/YAML inputs in Data/Plain × validation profiles. It checks reports/bytes, file/stdin, no-op attributes, validation and invalid-input preservation; original structural-default validation-off rejections remain explicit. See [host requirements, recorded architecture details and evidence scope](../docs/cli-host-probes.md).

`bash scripts/test.sh feature composed-member-defaults` verifies Cartesian selected-constructor enum-own defaults and numeric initializer selection in four fresh Data/Plain × validation profiles, including optimized/locked output, HTTP, live SSE and Patch.

`bash scripts/test.sh feature allof-string-defaults` checks outer defaults over retained repeated plain String constraints: validation-on Data/Plain, baseline plus optimized/locked output, four Json configurations, width240, HTTP/Patch and live SSE.

`bash scripts/test.sh feature object-collection-factories` checks ordinary DTO collection root and wrapper factories in four Data/Plain × validation profiles, optimized/locked output, four Json configurations, width240/depth15 and typed HTTP/Patch/live SSE.

`bash scripts/test.sh feature unformatted-native-factories` checks unformatted Int/Double native enum root and alias factories in four profiles, four Json configurations, optimized/locked output, 240 members/slots, nullable controls and HTTP/Patch/live SSE.

`bash scripts/test.sh feature allof-numeric-defaults` verifies validation-on Data/Plain numeric allOf outer factories, baseline/optimized locked output, four Json configurations, Long precision/mapped IEEE/negative zero, width240, HTTP/Patch and fragmented UTF-8 SSE.

`bash scripts/test.sh feature tuple-root-defaults` checks direct scalar tuple component/alias factories in Data/Plain×validation, baseline/optimized locked output, four Json settings, temporal/Uuid/Long/IEEE, 240 fields, count/unique controls and HTTP/Patch/fragmented SSE.

`bash scripts/test.sh feature optional-form-fields` verifies optional flat form key omission versus empty/false/zero, independent body absence, Data/Plain×validation, optimized/locked output, four Json settings, 240 optional fields and HTTP/live SSE wire/validation/callback/cancellation.

`bash scripts/test.sh feature floating-form-fields` verifies required/optional Float/Double form text, negative zero/IEEE, opt-in checked bounds, body absence, Data/Plain×validation, optimized/locked output, 120Double/JVM240slots and HTTP/live UTF-8 SSE.

`bash scripts/test.sh feature enum-form-fields` verifies required/optional retained String/native enum forms, serialName/openness/mapped checks, Data/Plain×validation, optimized owner bindings, four Json settings, raw240 vocabularies and HTTP/live UTF-8 SSE.

`bash scripts/test.sh feature wide-deep-validation` checks 200×16 and240×12 List<Int> models with factories and repeated HTTP shapes in Data/Plain×all8 dedup/hoist/merge combinations. It retains all source via128KiB compilation inventory partitions, in-process Kotlin and existing heaps; asserts four Json settings/copy/distinct checks/HTTP/live SSE. Generic-signature feasibility and maximum Cartesian/cross-platform resources remain separate.

`bash scripts/test.sh feature standard-form-fields` verifies required/optional UUID/date/Instant form text, canonicalization, opt-in UUID checks, body absence, Data/Plain×validation, combined optimized/locked output, 240 mixed fields and HTTP/live UTF-8 SSE.

`bash scripts/test.sh feature allof-uuid-defaults` verifies validation-enabled UUID outer defaults with retained canonical-text constraints, original annotation/source identity, aliases/nested/list/map/required/Patch, Data/Plain baseline and optimized/locked output, four Json settings, 240 fields and HTTP/live SSE.

`bash scripts/test.sh feature jvm-signature-limits` verifies Data/Plain203×List16 baseline and combined optimization, fourJson/Data-copy/field202 validation, deterministic checks and204×16 rejection with old-output preservation under existing heap limits.

`bash scripts/test.sh feature jvm-copy-signatures` verifies feasible short-package Data and long-package Plain203×List16, baseline/combined optimization, fourJson/Data-copy/final-field checks, and structured long-package Data rejection. Native/source regressions cover original Pointers, all eight optimization combinations and old-output preservation.

`bash scripts/test.sh feature bridge-intersections` verifies finite internal root bridges to direct named ordinary DTO allOf operands, source/owner identity, Data/Plain×validation/baseline+allopts/locks, fourJson/defaults/nested DTO+enum/recursion/required/Patch/HTTP/SSE and239+1 constructor slots. Source/native probes enforce original cached depth/catalog/AST and failure preservation.

`bash scripts/test.sh feature nullable-bridge-intersections` verifies actual named ordinary target/alias/member null intent, Data/Plain×validation/baseline+allopts/locks, fourJson/defaults/nested types/recursion/Patch/HTTP-null/SSE-null/close/reuse and240 constructor slots. Source tests preserve exact original-node work charges before/after alias binding; core/native tests retain roles, Pointers and old output.

`bash scripts/test.sh feature internal-dto-intersections` verifies direct/finite-chain/root-bridge actual lexical DTO allOf operands, Data/Plain×validation with nullable/non-null baseline+allopts/locks, original owned recursion/nested defaults, fourJson/Patch/HTTP/SSE and239+1 slots. Source tests enforce exact shared16384work/8MiB capture boundaries, overlap charges, original AST/catalog/owners and missing/copy rejection.

`bash scripts/test.sh feature composed-dto-intersections` verifies unique actual ordinary nested DTOs under allOf/union owners, Data/Plain×validation with nullable/non-null baseline+allopts/locks, fourJson/canonical recursion/nested defaults/Patch/HTTP/SSE and239+1 slots. Source/core/native tests enforce actual owner/capture/cache identity, ambiguity/variant-root/hierarchy exclusions, original Pointers and old-output preservation.

`bash scripts/test.sh feature hierarchy-dto-intersections` verifies actual plain named-leaf nested DTO allOf, Data/Plain×validation with nullable/non-null baseline+allopts/locks, fourJson/canonical recursion/defaults/Patch/HTTP/SSE, no parent-kind leakage and239+1 slots. Source/core/native tests retain original owner/AST/Pointers/capture, reject missing/foreign/root/common/inline/composed-leaf roles and preserve old output.

`bash scripts/test.sh feature composed-leaf-intersections` verifies unique original nested ordinary DTO operands under named composed hierarchy leaves with fresh Kotlin API/JSON/Patch/HTTP/SSE profiles.

`bash scripts/test.sh feature allof-target-intersections` verifies nested ordinary allOf operands under direct ordinary object owners with fresh Kotlin API/JSON/Patch/HTTP/SSE profiles.

`bash scripts/test.sh feature composed-target-owners` verifies four original lexical owner roles with sixteen fresh Kotlin API/JSON/Patch/HTTP/SSE profiles and combined optimized/locked outputs.

`bash scripts/test.sh feature allof-directional-models` verifies ordinary allOf request/response projection with fresh Kotlin call sites, four Json settings, Patch/recursion/optimized HTTP/SSE and a240-field request constructor.

`bash scripts/test.sh feature dto-unique` verifies mapped scalar-field DTO uniqueness, constructor-aware defaults, named/inline/nullable keys, baseline/optimized locks, HTTP/SSE/Patch and independent240×240 Kotlin/resource profiles.

`bash scripts/test.sh feature dto-enum-unique` verifies the [direct enum-field DTO uniqueness profile](../docs/dto-enum-unique.md) in fresh Data/Plain baseline/validation and combined optimized/locked projects, including wide getters, defaults, JSON, Patch, HTTP and SSE.

`bash scripts/test.sh feature dto-const-unique` verifies [scalar const-body DTO uniqueness](../docs/dto-const-unique.md), including decoded actual body values, constructor defaults, wide serialized fields, JSON/Patch/HTTP/SSE and optimized/locked Data/Plain profiles.

The [four-input DTO getter platform probe](../docs/platform-dto-getters.md) adds direct DTO scalar/enum/const getters and records Data/Plain JVM/JS/Native runtime separately from metadata/Android AAR compilation. The fourth input has no device runtime evidence.

`bash scripts/test.sh feature dto-null-unique` checks [direct JsonNull DTO getters](../docs/dto-null-unique.md) in fresh Data/Plain baseline/validation and optimized/locked profiles, including actual null states, defaults, wide getters, Patch/HTTP/SSE and strict codec failure.

`bash scripts/test.sh feature dto-collection-unique` checks [inline scalar collection DTO getters](../docs/dto-collection-unique.md) in Data/Plain baseline/validation and optimized/locked output, with actual240 List constructor fields×240 referring arrays and an independent total16-edge key traversal. Defaults/Map-order/Float/UUID/Instant, JSON/Patch/HTTP/SSE and exact work/depth/source/native/fuzz checks retain separate evidence scopes.

The complete HTTP compilation inventory resolves dependencies against declared source symbols across package roots, including import aliases and quoted factory identifiers. It retains sealed families in one compilation, assigns every source exactly once and keeps the runtime registry intact; synthetic `org.example` package spelling is not a dependency boundary.

`bash scripts/test.sh feature null-const-object-defaults` checks [null const object-default members](../docs/null-const-object-defaults.md) in Data/Plain validation off/on and optimized/locked profiles, including fresh trees/factories/no inheritance/required/Patch, HTTP/SSE and an independent240-serialized-field input. Source/value budgets, native failures and bounded fuzz retain separate checks.

`bash scripts/test.sh feature split-locked-adoption` also exercises explicit-donor owner additions to an existing version4 inventory in every Data/Plain validation profile. Read-only reports, retained peer memberships/raw locks, file bytes/modes/mtimes, exact replay and ordinary configuration checks precede compilation/runtime assertions on the same Kotlin consumers.

`bash scripts/test.sh feature dto-builtin-collection-reference-unique` verifies named/internal/bridge builtin List/Map getters through fresh Data/Plain validation controls, optimized locks, 240×240 width, independent full checked depth, four Json settings and actual Patch/HTTP/SSE consumers. Original-source/type/check/default/mapped-key and native failure regressions live beside the implementation.

`bash scripts/test.sh feature dto-special-collection-reference-unique` extends the DTO collection reference profile to existing UUID/date/date-time leaves. It checks mapped default/equality semantics, actual named/internal/bridge sources, 240×240 UUID width, independent full Instant checked depth, Data/Plain validation controls, optimized locks, four Json settings and real Patch/HTTP/SSE consumers.

`bash scripts/test.sh feature dto-native-collection-reference-unique` verifies Int/Long/Float/Double/Boolean collection reference keys through fresh Data/Plain validation controls, native membership/default checks, original aliases/borrowed bridges, optimized locks, 240×240 width, independent checked depth, four Json settings and actual Patch/HTTP/SSE consumers. Existing unchecked borrowed-native refusal stays covered.

`bash scripts/test.sh feature dto-string-collection-reference-unique` verifies original plain String enum collection references, strict defaults, actual lexical enum ownership, named/internal/bridge/nullable paths, Data/Plain validation controls, merged enum classifiers, optimized locks, 240×240 width, depth16, four Json settings, Patch, HTTP and fragmented SSE.

`bash scripts/test.sh feature dto-collection-reference-leaves-unique` verifies scalar/plain enum references below ordinary DTO collection getter trees, inline/named/internal/bridge paths, original getter/check/source identity, strict own defaults, Data/Plain validation controls, actual merged enum classifiers, optimized locks, 240×240 width, depth16, four Json settings, Patch, HTTP and fragmented SSE.

`bash scripts/test.sh feature dto-allof-unique` verifies actual ordinary allOf constructors and inherited scalar/enum/const/reference-collection getters, original field/required/default and aggregate-check authority, Data/Plain validation controls, actual merged enums, optimized locks, allOf240×240 width/depth16, four Json settings, Patch, HTTP and fragmented SSE.

`bash scripts/test.sh feature allof-default-overrides` verifies validation-on Data/Plain ×base/optimized locked output for original homogeneous overrides with no earlier bounds, strict own/winning annotations, actual240-field factories, String/UUID/numeric/nullable/alias values, four Json settings, Patch, HTTP and fragmented SSE. Unchecked groups retain their tested refusal boundary.

`bash scripts/test.sh feature tuple-native-enum-references` runs validation-on Data/Plain projects with base and optimized locked outputs, original native5/alias/nullable/mixed/tail codecs, exact Long/Float mapping, independent factories, four Json settings, 240 fields, Patch and controlled HTTP/live SSE. Unchecked refusal controls are separate; PREPARE_ONLY does not compile or execute Kotlin.

`bash scripts/test.sh feature nullable-recursive-cover-edges` compiles/runs Data/Plain × validation off/on, baseline/optimized locked mutual covers, nullable graph aliases, four Json settings, JsonNull/240fields/optional unique checks, required/Patch and controlled HTTP/live SSE. PREPARE_ONLY is preparation only.

`bash scripts/test.sh feature false-component-roots` runs Data/Plain × validation off/on, unsuffixed and optimized/locked Dto variants, strict root/alias JSON-kind refusal, nullable alias null behavior, distinct descriptors and Serializer/Companion names. The complete HTTP route executes the standalone codec consumer; false HTTP roots are not admitted.

`bash scripts/test.sh feature literal-false-fields` verifies literal Boolean false DTO fields, strict absence, reserved annotation names and 240-field JVM probes.

`bash scripts/test.sh feature false-nominal-signatures` runs fresh known false-class descriptor controls for Data/Plain, including copy, strict absence and optimized output.

`bash scripts/test.sh feature literal-false-root-defaults` checks root-owned literal-false DTO default factories, strict member refusal, fresh state, nullable owners, controlled HTTP and 240-field/32KiB JVM probes.

`bash scripts/test.sh feature literal-false-dto-field-defaults` checks actual consuming-field defaults with literal/ref false target members, independent factories, required/count policy and fresh JVM resource probes.

`bash scripts/test.sh feature literal-false-collection-fields` checks actual array literal-false leaves in inline DTO collections, ordinary container presence, empty-container API/wire, original corpus and independent JVM resource probes.

`bash scripts/test.sh feature literal-false-collection-defaults` checks own outer literals, typed fresh initializers, required/nullable policy, complete metadata-named JSON data, HTTP and independent 240-field/32KiB JVM probes.

`bash scripts/test.sh feature false-generic-signatures` compiles baseline/optimized optional and required Data/Plain models at actual false-class generic signature boundaries, runs constructor/copy and four Json wire profiles, and rejects adjacent impossible inputs.

`bash scripts/test.sh feature literal-false-collection-root-defaults` checks component-own typed factories, independent field defaults, required/null policy, complete JSON literal identity, HTTP and independent 240-field/15-container/near32KiB probes.

`bash scripts/test.sh feature literal-false-collection-dto-field-defaults` checks consuming DTO own annotations, independent target factories/counts, required/nullable policy, complete literal identity, HTTP and independent 240-field/15-container/near32KiB probes.

`bash scripts/test.sh feature false-dto-owner-root-defaults` checks direct root-member DTO factories, independent target factories in either naming order, required/nullable/count policy, complete source/literal identity, HTTP and independent 240-field/15-container/exact32KiB probes.

`bash scripts/test.sh feature false-dto-owner-alias-defaults` checks pure alias root-member factories, nearest nullable/field overrides, immutable path/cache proof, shared capture budgets, HTTP and independent 240-field/15-container/46-alias/exact32KiB probes.

`bash scripts/test.sh feature pinned-structured-const` covers structured data bodies in union/hierarchy variants, same-kind state, scalar tags, references/inheritance/composition, optimized locks, 240 members and fresh Data/Plain × validation HTTP/live SSE with four JSON settings.

`bash scripts/test.sh feature discarded-structured-const` covers source-only validation of ignored extra-key and overridden pinned literals, original diagnostics, strict/ignored keys, actual body/dictionary controls, optimized locks, 240-member constructors and four-profile HTTP/live SSE contracts.
