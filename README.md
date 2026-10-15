# openapi2kt-go

Generate Kotlin **kotlinx.serialization models** and a **Ktor HTTP/SSE client** from OpenAPI JSON or [bounded YAML](docs/yaml-input.md) with a native Go CLI, `openapi-kt`.

Generation runs offline from a local file or stdin. The generated Kotlin sources belong in your application; running the generator itself requires no JVM or Gradle.

> **Alpha:** the current source candidate is `0.2.0-alpha.1` and has not been published. It supports a documented subset of OpenAPI 3.0/3.1. The older `v0.1.0-alpha.1` Draft Prerelease has a different feature set; see [versions and installation](docs/installation.md) and [migration notes](docs/releases/next-alpha.md).

## Installation

From a local source checkout, build with **Go 1.27.2 or newer** and a matching `GOROOT`. On macOS, Go 1.27 requires macOS 13 Ventura or newer.

The TOML and YAML parsers are pinned in `go.mod`/`go.sum` and included in `vendor/`. With an existing Go toolchain, build offline:

```bash
mkdir -p build
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go build -o build/openapi-kt ./cmd/openapi-kt
./build/openapi-kt --version
```

The offline build downloads nothing; [TOML notices](THIRD-PARTY-NOTICES) and [YAML notices](THIRD-PARTY-NOTICES.md) preserve required attribution. The examples below use this local binary. For installation on `PATH`, binary archives, checksums and Windows instructions, see the [installation guide](docs/installation.md).

## Quick start

Generate models from the included [catalog example](tests/fixtures/catalog.json):

```bash
./build/openapi-kt generate \
  --input tests/fixtures/catalog.json \
  --output build/generated/catalog \
  --package-name org.example.catalog
```

Replace `--input` with your own `openapi.json`, `openapi.yaml` or `openapi.yml` to generate from your specification. The catalog example contains models only; its main output files are:

```text
build/generated/catalog/
├── .openapi-kt-manifest.json
└── org/example/catalog/models/
    ├── Item.kt
    └── Payload.kt
```

The example reports `W_JSON_ELEMENT_APPROXIMATION` for its unconstrained object schemas; generation succeeds with those values represented as `JsonElement`.

Keep `.openapi-kt-manifest.json`: it tracks which files the generator owns. Regeneration updates those files and removes obsolete owned files while preserving handwritten files. See [output safety](docs/output-ownership.md) for conflict and failure handling.

Documents with supported operations also generate API classes and an aggregate Ktor client. Try [the HTTP example](tests/fixtures/http/basic.json) in a separate output directory, then follow the [HTTP client guide](docs/http-client.md).

### Validate and check

Check whether an input fits the generator's supported subset:

```bash
./build/openapi-kt validate --input tests/fixtures/catalog.json
```

Compare the expected output with files on disk without writing; useful in CI:

```bash
./build/openapi-kt generate \
  --input tests/fixtures/catalog.json \
  --output build/generated/catalog \
  --package-name org.example.catalog \
  --check
```

`validate` is a generation compatibility check, not a complete OpenAPI validator. Direct commands also accept `--input -` for one stdin stream, up to 8 MiB. Auto stdin is JSON; YAML stdin requires `--input-format yaml`. See the [YAML profile](docs/yaml-input.md) for scalars, anchors and safety boundaries. Fetch specifications with your own tooling before generation; see [file and stdin input](docs/local-input.md).

### Common options

| Option | Purpose |
|---|---|
| `--model-name-suffix Dto` | Add a suffix to component names and references |
| `--api-client-name CatalogClient` | Override the generated client name |
| `--validation` | Generate supported model and collection checks |
| `--diagnostic-format json` | Emit machine-readable diagnostics |
| `--deny-warnings` | Reject generation warnings before writing output |
| `--config openapi-kt.toml` | Generate from a local configuration instead of `--input` |

Run `./build/openapi-kt generate --help` for all flags. For multiple specifications, per-spec options and output directories, use [local configuration](docs/configuration.md).

## Using the generated Kotlin

Add the output directory to your Kotlin source set and enable the Kotlin serialization compiler plugin with `kotlinx-serialization-json`. HTTP clients additionally need Ktor and a caller-configured `HttpClient`; SSE uses Ktor's SSE plugin. Some scalar formats need additional dependencies, described in [special scalars](docs/special-scalars.md).

The repository includes [model](samples/kotlin-smoke/README.md) and [HTTP/SSE](samples/kotlin-http-smoke/README.md) sample projects with dependency configuration and compilation/runtime checks. There is no Gradle generator plugin.

## Features and limits

| Area | Current scope |
|---|---|
| Models | Objects, scalars, enums, lists, dictionaries, nullable values and supported defaults |
| References and composition | Local references, recursive DTOs, aliases, and documented `allOf` / `oneOf` / `anyOf` and discriminator profiles |
| HTTP | Ktor clients for supported parameters, request bodies, responses and declared error models |
| SSE | Typed event flows for supported event schemas |
| Generation | Deterministic output, diagnostics, optional validation, read-only checks and manifest-owned files |

**Not yet supported:** full YAML (merge keys, tag directives, non-string keys and non-JSON tags), external references, arbitrary composition/reference graphs, or the complete OpenAPI HTTP/SSE surface. Unsupported constructs produce diagnostics; supported approximations may produce warnings. The [compatibility matrix](docs/compatibility-matrix.md) defines the exact boundaries, and [remaining work](docs/remaining-work.md) tracks open areas.

Historical `v0.1.0-alpha.1` hosted CLI checks covered Linux, macOS and Windows. Generated Kotlin compilation/runtime evidence is separate and covers macOS ARM64/JVM; Android/KMP, other binary architectures and minimum dependency versions require separate acceptance. See the [platform probes](samples/kotlin-platform-smoke/README.md) and [candidate acceptance checklist](docs/json-release-checklist.md).

## Documentation

| I want to… | Start here |
|---|---|
| Install or upgrade | [Installation](docs/installation.md) · [Next alpha migration](docs/releases/next-alpha.md) |
| Configure inputs and output directories | [File/stdin input](docs/local-input.md) · [Configuration](docs/configuration.md) |
| Understand generated types and JSON behavior | [Model semantics](docs/model-semantics.md) · [Request presence and Patch](docs/request-presence.md) |
| 使用嵌套 numeric/Boolean enum 集合字段默认值 | [直接嵌套 native enum 默认值](docs/nested-native-enum-collection-defaults.md)：保留 native scalar wire、严格原 default 词表和 required/Patch；只扩张 direct inline outer-own 角色 |
| Use a generated client | [HTTP client](docs/http-client.md) · [SSE](docs/sse.md) |
| Handle warnings or output conflicts | [Warning policy](docs/warning-policy.md) · [Output ownership](docs/output-ownership.md) |
| Check feature coverage | [Compatibility matrix](docs/compatibility-matrix.md) · [Remaining work](docs/remaining-work.md) |
| Integrate an AI coding agent | [Product usage skill](skills/README.md) |
| Contribute or prepare a release | [Contributing](CONTRIBUTING.md) · [Development plan](docs/development-plan.md) · [Release checks](docs/releases.md) · [Versioning](docs/versioning.md) |

## Development

Run the Go tests offline with the vendored dependencies:

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test ./...
```

See [development tooling](scripts/README.md) for the full Go checks and generated Kotlin model/HTTP suites, and [CONTRIBUTING.md](CONTRIBUTING.md) for fixture and verification requirements.

## License

[MIT](LICENSE). Third-party material retains its own license and required [TOML](THIRD-PARTY-NOTICES) / [YAML](THIRD-PARTY-NOTICES.md) notices. New binary archives include both dependencies' attribution/license texts and the Go runtime `GO-LICENSE`.
