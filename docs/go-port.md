# Go implementation

The generator uses an independent native Go implementation. The product remains an offline OpenAPI JSON → Kotlin generator; changing the implementation language does not change Kotlin call sites or JSON wire contracts.

## Current availability

The native Go implementation is the default core and CLI and has passed integrated behavior acceptance. Its pure core generates the accepted [model subset](model-semantics.md): DTOs, recursive local references, arrays, primitive aliases, open/native enums, scalar const projections/body defaults, nullable compositions, object allOf merges, inline primitive/object unions, primitive-union normalization, named root-kind/null unions, content versus closed-polymorphic object roots, and typed discriminator hierarchies. Kotlin names, source-located warnings, declaration order, caller-specific projection rules and resource guards have Go regression coverage.

Command handling and output transactions have native Go process and filesystem fault-injection coverage. Direct dictionaries are implemented; dictionary composition, opaque common-value narrowing, native single-enum inline body defaults and default-polymorphic fixed-singleton contexts remain deferred; changing implementation language does not widen the accepted subset.

An additive [HTTP subset](http-client.md) now uses the same core/CLI, with typed operation plans and fresh Ktor MockEngine verification. It does not imply full client/SSE support.

Rust production sources and Cargo manifests are retired. Builds, generation and Go tests do not execute Rust or require Cargo. There is no subprocess wrapper backend and no published release.

## Development checks

With Go 1.27.2 or newer installed for the current source checkout and a matching `GOROOT`:

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test ./...
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go vet ./...
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -race ./...
gofmt -l core internal cmd
```

The module vendors its pinned TOML and YAML parser dependencies; see [TOML notices](../THIRD-PARTY-NOTICES) and [YAML notices](../THIRD-PARTY-NOTICES.md). `GOTOOLCHAIN=local` prevents automatic toolchain downloads; `GOPROXY=off` prevents dependency downloads. The race detector requires a supported host compiler/toolchain. Go 1.27 requires macOS 13 Ventura or newer. Current Go 1.27.2 runtime checks cover macOS ARM64; all six release targets cross-compile, but the other five binaries have not been executed with this toolchain. The earlier preview's hosted source-CLI checks are recorded separately in the [README](../README.md#features-and-limits).

## Architecture

| Package | Responsibility | Status |
|---|---|---|
| `core` | Small in-memory public request/result/diagnostic API | `Generate` and `GenerateWithDiagnostics` implemented for the accepted model subset |
| `internal/document` | Ordered JSON tree, exact number tokens, format detection and input bounds | Implemented |
| `internal/naming` | Kotlin identifiers, string literals, enum spelling and JSON Pointers | Implemented |
| `internal/schema` | Reference catalog, normalization, analysis and typed declarations | Accepted model/scalar/union/normalization and bounded discriminator-graph analysis implemented |
| `internal/kotlin` | Deterministic Kotlin emission from typed declarations | Accepted model, alias, enum, union/null and typed hierarchy backends implemented |
| `internal/cli` / `cmd/openapi-kt` | Input loading, flags, diagnostics and exit codes | Native CLI; process contracts verified |
| `internal/output` | Manifest ownership, locking, staging, rollback and read-only checks | Native ownership/transaction implementation; host filesystem contracts verified |

Generation stays filesystem-independent and does not read environment variables, start subprocesses, or access the network. CLI IO and output mutation remain outside the core. Internal representations are not a stable public API.

The loader preserves property declaration order and distinguishes missing values from explicit null. JSON numbers retain exact tokens rather than being decoded through `float64`. Duplicate keys are errors, including duplicates in opaque example payloads. Invalid UTF-8 and unpaired escaped UTF-16 surrogates are rejected rather than replaced. The 8 MiB input bound and original decoder nesting budget are preserved; the separate 48-level schema traversal bound belongs to the analyzer.

## Core API

`Generate(GenerationRequest) (GenerationArtifacts, error)` returns files only. `GenerateWithDiagnostics(GenerationRequest) (GenerationReport, error)` additionally returns non-fatal warnings; their generated files are identical. Structured errors are `core.Diagnostics`, with stable code, safe message and optional JSON Pointer. A pointer to an empty string identifies the document root; an absent pointer is omitted from JSON.

`GenerateWithPhaseObserver(GenerationRequest, GenerationPhaseObserver)` adds synchronous caller-owned phase observation with identical artifacts and diagnostics. The core does not read clocks or OS metrics; the optional observer owns measurement. [CLI metrics](cli-metrics.md#core-boundaries-and-process-memory) document options/parse/analyze/transform/emit boundaries and error closure.

Requests provide document bytes, input format, format-detection source name and package/suffix options. Successful artifacts contain slash-separated relative paths and content. Errors return no partial files or successful warning report. Filesystem mutation is available only through the CLI, not these pure core functions.

## Verification coverage

Current Go generation checks all 154 corpus cases: 38 successes and 116 rejections, with 50 complete Kotlin snapshots. The 47 snapshots accepted at the language cutover remain unchanged; three direct dictionary cases now have new success snapshots. Passing this small corpus does not establish coverage of the broader model-semantics regression/runtime suites.

A fresh, test-only Kotlin exporter generates all 38 positive cases, 15 semantic fixtures in baseline/suffix configurations, all 11 multi-kind primitive combinations, and enum/constructor resource-boundary inputs through the actual Go core. The 13 original serializer runtime suites and the dictionary runtime suite pass with these Go-generated sources; three supplemental runtimes cover native aliases/const references/items, strict primitive-root null/ref/mixed-object behavior, and 240-value hierarchy enums with required single-enum constructor semantics. Resource checks include native/body/merged 240-slot models, boxed nullable and dictionary 240-field models, and 240-member open/merged/hierarchy enums. Compilation uses Kotlin 2.4.10, serialization 1.11.0, Gradle 9.3.1 and JDK 21 on macOS ARM64. The same complete input set also passes through the actual native Go CLI in a fresh Kotlin project: report values, file bytes, manifest hashes, read-only checks and no-op mtimes are compared with the in-memory API before compilation and all 17 runtime suites. These checks establish independent behavior acceptance; the product still does not claim full OpenAPI, Android/KMP or unexecuted host-platform support.

Set `OPENAPI_KT_TEST_EXPORT_HIERARCHY=1` and `OPENAPI_KT_TEST_EXPORT` to a **new** directory under the ignored `.pi/go-migration/` tree, then run `go test -count=1 -v ./core -run '^TestExportModelSmoke$'` to export the complete test project. Existing directories are rejected, and original Kotlin runtime sources are copied without rewriting expectations. The exporter is test-only, not a product CLI.

## CLI and output safety

Build the native CLI:

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go build -trimpath -o build/openapi-kt ./cmd/openapi-kt
build/openapi-kt validate --input tests/fixtures/catalog.json
build/openapi-kt generate --input tests/fixtures/catalog.json --output build/generated-go \
  --package-name org.example.catalog --diagnostic-format json
build/openapi-kt generate --input tests/fixtures/catalog.json --output build/generated-go \
  --package-name org.example.catalog --check
```

The CLI preserves `generate`/`validate`, input/diagnostic format flags, package/suffix options and exit codes 0–4 from the [CLI contract](p1-contract.md). JSON diagnostics are one stdout document without text on stderr; warning-free reports omit `warnings`. Input reads remain bounded at 8 MiB.

Actual Go binary tests run all 154 corpus cases and compare 50 reviewed snapshots, including the unchanged 47-snapshot cutover baseline. Process tests cover warnings, safe control-character rendering, distinct failure exits, absolute paths across working directories, ownership/stale cleanup, checks, locks and symlink guards. Filesystem fault injection verifies complete file bytes/mtimes/modes and tree restoration at commit boundaries, after file/manifest backup moves, and retained backups plus rejection of later checks/writes when rollback fails.

The manifest remains version 1 with sorted relative Kotlin paths and lowercase SHA-256 content hashes. Malformed/duplicate metadata, case collisions, handwritten destinations, modified owned files, symlinks within output paths and abandoned stages fail closed. Check mode creates no root or lock, acquires no writer lock and changes no file mtimes. Existing empty or missing owned files are distinguished correctly; only manifest-owned stale files may be removed.

Writers use cooperative, nonblocking OS locks: `flock` on Darwin/Linux/BSD hosts and `LockFileEx` on Windows. Unsupported lock hosts fail closed for writes. Staging/backups live inside the output root on the same filesystem; the manifest commits last. Recovery journals are evidence for manual inspection, **not** a power-loss or cross-file crash-atomicity guarantee. Locks and path checks are not a security boundary against hostile concurrent filesystem mutation. Platform compilation alone does not verify locking/rename behavior.

## Integrated harness

The default harness builds the native Go command and uses a new verification project. An explicit binary can bypass building without bypassing generation/checks:

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go build -trimpath -o .pi/openapi-kt-go ./cmd/openapi-kt
OPENAPI_KT_BINARY="$PWD/.pi/openapi-kt-go" bash scripts/test.sh models
```

This path needs Go, Python 3, an installed Gradle executable and JDK, with the Kotlin/serialization dependencies already cached. `GO` and `GRADLE` can select executable paths. Go and Gradle run offline. Each invocation retains a new repository-local verification project; it never compiles old generated output. The opt-in test exporter verifies that the executable is the native Go command, not a wrapper. The Python fixture tool accepts the same `OPENAPI_KT_BINARY` and an `OPENAPI_KT_SMOKE_BUILD` override restricted to the repository's smoke/verification directories.

## Go build baseline

A Go 1.26.5 macOS ARM64 baseline uses `go build -trimpath -o <binary> ./cmd/openapi-kt`, with default optimized compilation and debug information retained. One build with a new empty `GOCACHE` took **2.830 s**. Three subsequent builds using that populated cache took **0.0851 / 0.0678 / 0.0629 s** (median **0.0678 s**). Each result generated the catalog successfully and its manifest SHA-256 hashes were verified. All four binaries were byte-identical, **4,013,298 bytes**.

The boundary is Go process launch through successful build; output verification is excluded. “Cold” means an empty compiler build cache, not cold OS disk/page caches; no global cache was cleared. There is only one cold sample and three warm samples on one host, not a performance guarantee. This is not generator latency, peak-memory, Kotlin compilation-cost or Rust speedup evidence. No same-workload Rust comparison was performed, so no equivalence or speedup is established.

## Compatibility and migration

The accepted Kotlin model/call-site and wire contracts remain unchanged, including naming, source-located diagnostics, CLI flags/exit codes and manifest version 1. Existing owned outputs remain usable without resetting or deleting the manifest; unchanged generation keeps file mtimes. The implementation-language change does not add HTTP/SSE, Patch, YAML, downloads or a Gradle plugin.

Rust library consumers must migrate to the Go request/result API; this is an implementation-language API change, not a Kotlin API migration. Current offline source builds require Go 1.27.2 or newer rather than Rust/Cargo. Go consumers still using 1.26 must upgrade their toolchain; the existing `v0.1.0-alpha.1` tag and binary packages retain their original Go 1.26.0 build. This minimum-toolchain change does not change Kotlin call sites, wire behavior or manifest formats, and must be classified under the [versioning policy](versioning.md) before the next release. Product verification covers:

- The 154-case JSON corpus and 50 complete Kotlin snapshots, including the unchanged 47-snapshot cutover baseline, without bulk rewriting expectations.
- Ported semantic regressions, including warning codes/source pointers, recursive references, naming, nullability and resource limits.
- Generated Kotlin compilation and all existing serializer runtime contracts.
- Actual Go CLI process tests and output-transaction fault injection: preserve old output on failure, protect handwritten/modified files, remove only manifest-owned stale files, and keep `--check` read-only.
- Independent offline Go build, formatting, tests, vet and race checks.

Incomplete schema work is assessed separately from accepted behavior. A historical Rust test result is not evidence that the Go implementation passes, and Go unit tests alone do not establish Kotlin serialization correctness.

CLI flags, exit codes, generated naming, manifest version/hash format, and wire behavior retain the [P1 contract](p1-contract.md). An intentional difference requires a reviewed contract test and migration documentation. No performance or binary-size equivalence is promised without measurements on identical inputs and options.
