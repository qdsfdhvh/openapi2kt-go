# Next bounded JSON alpha: scope and migration

The next preview keeps the [frozen bounded JSON scope](../json-release-checklist.md): current documented model/reference/nullable/composition/validation profiles, local configuration and output ownership, and the existing Ktor HTTP/SSE subset. Candidate verification targets macOS ARM64 for the CLI and Kotlin/JVM 17 bytecode running on JDK 21. General reference graphs, Android/KMP and other host/engine combinations remain on the roadmap.

The source CLI now declares **`0.2.0-alpha.1`** under the [version policy](../versioning.md). This identifies the local candidate; no new tag or release is implied. The new input/reference profiles, removal of previously supported downloader interfaces and increased minimum Go toolchain require a MINOR change before 1.0. Freezing the candidate does not satisfy beta/rc admission by itself.

## Migrate from the historical `v0.1.0-alpha.1`

| Surface | Required action |
|---|---|
| Go source builds and `core` consumers | Use Go **1.27.2 or newer** with its matching `GOROOT`. Go 1.27 macOS builds require macOS 13 or newer. Running a packaged native CLI does not require an installed Go toolchain |
| Former `openapi-kt download` / `help download` | Acquire the specification with an external tool, confirm acquisition succeeded, then call `validate` or `generate --input FILE`. Removed commands return usage exit 2 / `E_USAGE` without fetching or consuming stdin |
| Former `[specs.download]`, URL/header metadata | Remove downloader tables and metadata from TOML. Keep local `input` paths; unknown downloader fields fail with `E_CONFIG_UNKNOWN_FIELD` rather than being ignored |
| One direct streamed document | Explicitly use `--input -` for `generate`, `validate` or `generate --check`. The input must reach EOF and fit the existing 8 MiB limit. Configured specs remain local-file-only; use `input = './-'` for a literal file named `-` |
| Producer failures | Observe the producer's exit status. A valid complete JSON stream followed by producer failure cannot undo successful generation. When success must precede generation, first acquire a local file successfully |
| New DTO reference profiles | Eligible existing ordinary DTO owners and named/internal alias bridges preserve their canonical physical constructor/serializer identity. Access nested classifiers through their canonical owner, as documented in the [DTO reference contract](../internal-dto-references.md), rather than assuming Kotlin typealias provides nested classifier access |
| Persisted configuration/output/name locks | Preserve the existing configuration and manifest/name-lock versions and state. This release classification does not authorize deleting locks, resetting ownership, renaming owners or moving roots to bypass a diagnostic |
| Generated Kotlin and wire behavior | Downloader removal and the Go minimum upgrade preserve existing generated APIs and serialization/HTTP/SSE behavior. Opt-in optimization migrations continue to follow their individual contracts; unsupported inputs retain diagnostics |

The [local-input contract](../local-input.md) defines EOF, bounded reads, failure preservation and exact migration behavior. Existing `v0.1.0-alpha.1` packages retain their original Go toolchain and may still contain the downloader; they are not updated by changes in the working source.

## Acceptance and packaging boundaries

Local source acceptance runs the entire `scripts/test.sh all` profile set and bounded fuzz targets against one frozen source tree. Generated Kotlin is compiled and run from fresh output, with existing compiler options and resource limits; no claims extend to the full option Cartesian product or wide-by-deep maximum combinations.

Local package preflight verifies the macOS ARM64 archive, SHA-256, the exact five members (`openapi-kt`, `README.md`, `LICENSE`, `GO-LICENSE`, `RELEASE.json`) and installation/help/file/stdin/validate/generate/check without fixture or Go/Gradle dependencies from another checkout. Build archives with `--tag v0.2.0-alpha.1` and verify them with the [isolated package check](../releases.md#local-packaging). An archive is a local candidate until the complete acceptance and publication gates pass. Do not replace or distribute it as the historical release.

A new release still needs exact Git tree/history and package review, dependency-security review, version identity consistency and explicit Git/tag/upload/publication authorization. Packages are unsigned. Candidate scans and checksums do not constitute a complete vulnerability audit. Keep local logs, hashes, machine paths and task records outside public documentation.
