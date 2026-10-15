# Development tooling

`bash scripts/test.sh feature referenced-string-enum-null-member-defaults` generates fresh Data/Plain × validation consumers for existing referenced scalar/finite collection field-own defaults with explicitly nullable String enum null-member vocabularies. Four Json settings cover actual classifiers/nullable aliases, null versus `"null"`, strict annotations/open runtime, fresh trees, narrowed leaf rejection, required/Patch/copy, controlled HTTP and finite SSE, optimized/merged owner identity and independent 240-field/raw240-vocabulary probes. `OPENAPI_KT_PREPARE_ONLY=1` is native preparation only; `http` also generates/checks the fixture and calls its consumer.

`bash scripts/test.sh feature referenced-string-enum-scalar-defaults` runs fresh Data/Plain × validation profiles for plain String/typeless enum references' own scalar field defaults. It checks actual lexical types/aliases, file-wide helper reuse, escaping, four Json settings, required/Patch, strict annotations and open unknown Strings, optimization/locks, controlled HTTP and finite SSE decoding/reuse, plus independent 240-field and 240-member compilation probes. Native preparation with `OPENAPI_KT_PREPARE_ONLY=1` is not Kotlin execution evidence. The fixture and consumer also run in `http`.

`bash scripts/test.sh feature referenced-native-enum-scalar-defaults` runs fresh Data/Plain × validation profiles for native numeric/Boolean enum references' own scalar field defaults. It checks literal/alias ABI, minimum values, strict annotation vocabulary, four Json settings, required/Patch, checked internal aliases, optimization/locks, controlled HTTP and finite SSE decoding/reuse, plus independent 240-field and 240-member compilation probes. `OPENAPI_KT_PREPARE_ONLY=1` provides native generation/check evidence only. The fixture and consumer also run through `http`.

`bash scripts/test.sh feature nested-native-enum-collection-defaults` 是[直接嵌套 native enum own-default](../docs/nested-native-enum-collection-defaults.md)的消费验证入口，覆盖 ordinary direct inline finite List/typed Map、既有五种 native enum terminal、Data/Plain × validation、四 Json presence/default/check、actual optimizer aliases/locks、typed HTTP 与 fragmented live SSE。`bash scripts/test.sh models` 经 `TestExportModelSmoke` 导出普通模型并通过 `EnumContextContracts.main` 扩展既有 `runEnumContextContracts`，不新增模型 runtime suite；`bash scripts/test.sh http` 生成/check 完整 fixture，既有 `runHttpContracts` 在 `HttpContracts` suspend registry 中直接调用完整 HTTP 消费者 body。原 source48/checked16/slots240/default32KiB caps 不变。`OPENAPI_KT_PREPARE_ONLY=1` 仅 native generation/check 和项目准备，不是 Kotlin compilation/runtime PASS；Go/native 回归另用 `go test ./internal/cli -run '^TestNestedNativeEnumCollectionDefaults'`。平台、统一候选及发布验收独立。

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

Hosted CI first runs a Go-only format/vet/test/race gate. Only after all Go jobs pass does it run native Kotlin generation/check and Python tooling checks. The local `unit` command still runs both Go and Python checks; its coverage is unchanged. Hosted generation/check does not replace the `models`/`http` Kotlin compilation/runtime suites.

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
