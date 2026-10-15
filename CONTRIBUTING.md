# Contributing

The project has a model generator and an [initial HTTP subset](docs/http-client.md). Start with the [P1 contract](docs/p1-contract.md); the [development plan](docs/development-plan.md) and [compatibility matrix](docs/compatibility-matrix.md) describe the broader roadmap, not a claim of complete support.

## Proposing a change

For a feature or bug report, include:

- a small OpenAPI example;
- the expected Kotlin declaration or HTTP/JSON behavior;
- the relevant compatibility-matrix entries;
- any implications for generated names, serialization, or configuration.

Separate a design proposal from a claim that a capability is implemented. Update the corresponding documentation when a decision changes. Classify releases by the [versioning and compatibility policy](docs/versioning.md), including generated Kotlin call sites and wire behavior, not just source-text differences.

## Test fixtures

Keep shared inputs and expected Kotlin snapshots under `tests/fixtures/`; Go tests remain alongside their packages. The executable corpus inventory is `tests/fixtures/p1/cases.json`.

Use small, self-contained synthetic examples. Each fixture should exercise a named contract and include a counterexample where useful.

- Use `example.com` or `example.test` for example domains and `org.example.*` for Kotlin packages.
- Use a loopback server for HTTP tests rather than contacting external services.
- Keep fixture inputs, expected outputs, and generation options together or clearly linked.
- Record a seed and size parameters for generated benchmark datasets.
- Do not include credentials, personal data, environment files, or machine-specific paths.

Use explicit, per-symbol imports in new or modified Kotlin sources.

## Verification

Once the relevant implementation exists, verify changes at the appropriate levels:

1. Go unit and integration tests for parsing, normalization, analysis, and diagnostics.
2. Reviewed snapshots for generated file sets and declarations.
3. Kotlin compilation for generated source changes.
4. Runtime assertions for serialization, HTTP requests, SSE, and cancellation.
5. Bounded file/stdin/EOF/read-error and failure-injection tests for input/output transactions. Specification acquisition belongs to external tools; do not add a downloader/curl wrapper to the CLI. Removal/no-fetch regressions may use loopback servers only.

Require no paths from `gofmt -l core internal cmd`, and run `go vet ./...`, `go test ./...` and `go test -race ./...` with `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`. The installed Go 1.27.2-or-newer toolchain must have a matching `GOROOT`. The module pins and vendors `github.com/pelletier/go-toml/v2@v2.4.3` and `go.yaml.in/yaml/v3@v3.0.4`; ordinary offline builds use the included source and require no module-cache preparation. Use `bash scripts/test.sh models` for changed generated Kotlin: it builds the native Go command and generates a fresh compilation/runtime project, with Go/Gradle offline and Kotlin dependencies already cached. `GO`, `GRADLE` and an optional `OPENAPI_KT_BINARY` select existing executables. P1 routine DTO checks use Go tests and snapshots plus minimal Kotlin compilation; ordinary DTO round-trips are deferred, while later Patch/union/custom-serializer changes require runtime tests.

For HTTP emitter changes, also run `bash scripts/test.sh http` with cached Ktor dependencies; this builds fresh CLI output and executes MockEngine wire assertions.

Do not bulk-accept snapshot changes without checking the contract. List checks that were not run and why; unavailable tooling is not a passing result.

## Documentation

Write user-facing documentation about the product: its behavior, limitations, setup, and supported workflows. Keep local task journals, investigation transcripts, and machine-specific instructions outside versioned documentation.

Examples should use commands and APIs that exist. Clearly label proposed interfaces in design documents. Do not add installation instructions or support claims before they can be verified.

## Dependencies and attribution

Review dependency maintenance, target support, Go/toolchain requirements, and licenses. Preserve required copyright and license notices for third-party material. The project is licensed under [MIT](LICENSE); third-party material retains its original license and required notices. [THIRD-PARTY-NOTICES](THIRD-PARTY-NOTICES) preserves the TOML full license, and [THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md) identifies the YAML license/NOTICE and full Apache text. Include both attribution/license sets in new binary archives.

## Release checks

Before releasing, verify the supported build matrix, installation from the packaged artifacts, checksums, package contents, documentation, and dependency notices. Keep credentials and temporary build data out of source archives, binaries, logs, and test reports. Follow the [publication checks](docs/publication.md) for candidate scans, optional local policies, and the distinction between a public repository and a verified release package.
