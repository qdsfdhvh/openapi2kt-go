# Local file and stdin input

The generator consumes OpenAPI JSON or [bounded YAML](yaml-input.md) from a local file or **one standard-input stream**. Specification acquisition, authentication, proxies, redirects, retries and input-file replacement belong to external tools such as curl. The CLI makes no network requests, runs no downloader subprocess and does not fetch external `$ref` targets. Generated Ktor clients are unaffected: their HTTP/SSE behavior belongs to the caller's Kotlin application.

```sh
openapi-kt generate --input openapi.json --output generated --package-name org.example.api
openapi-kt validate --input openapi.json

# Bash: observe failures from both the external producer and the generator.
set -o pipefail
curl -fsSL https://api.example.test/openapi.json |
  openapi-kt generate --input - --output generated --package-name org.example.api
```

The external command above is an example, not a command launched or a transport policy enforced by the generator. curl is not a dependency for local file/stdin generation. Supply authentication through your existing secret-management procedure, not credentials embedded in documented URLs or committed scripts.

## Stream contract

- `--input -` and `--input=-` select stdin on direct `generate`, `validate` and `generate --check`; `--input` remains required. There is no implicit stdin input, background service, incremental emission or URL input mode.
- The CLI reads the selected stream until EOF before analysis/output. At most **8 MiB plus one detection byte** is read; exceeding the existing 8 MiB document limit yields exit 1 / `E_INPUT_LIMIT`. It need not drain an oversized producer to EOF. Whitespace and metadata count toward the limit. An open producer may block until EOF; producer cancellation and shell pipeline lifecycle are the caller's responsibility.
- Auto format treats stdin as extensionless JSON. Explicit `--input-format json` works; YAML stdin requires `--input-format yaml` and follows the [YAML safety profile](yaml-input.md). Empty, truncated, duplicate-key, trailing-data or malformed JSON follows the same safe parser diagnostics as files. A stream read error yields exit 4 / `E_INPUT_IO` without echoing input or raw reader errors.
- Stdin is caller-owned: it is read only when selected and is not closed by the CLI. File input ignores stdin. Help, version and argument errors do not read it. `./-` denotes an ordinary file literally named `-`, not stdin.
- Stream and file inputs use the same pure generation core, flags, warnings, naming, Kotlin bytes and output ownership. Read metrics count actual consumed bytes, including the size-limit detection byte; no networking phase is introduced. Stream failures and warning-policy rejection preserve existing generated/handwritten output. `--check` retains its read-only and exit-3 difference contract, including name-locked output.

A consumer cannot learn an external producer's eventual exit status from stdin alone. Shell `pipefail` reports upstream failure but cannot undo generation if a producer emits a complete valid document and subsequently fails. When that distinction matters, fetch to a separately managed temporary file, check the producer's success, then invoke the generator with that file. Input acquisition/atomic replacement is not a generator guarantee.

## Configuration and migration

Local `--config FILE [--spec NAME]` generation/validation remains supported. Configuration accepts local file paths only; `input = '-'` is rejected with `E_CONFIG_INVALID`, even in defaults or an unselected entry. Use a direct `--input -` command for a single stream, or `input = './-'` for a literal dash filename. No stdin marker or replay support is provided in configured specs.

**Breaking CLI/configuration change in the current source:** `download`, `help download` and download-specific flags are removed; invoking the removed command returns usage exit 2 / `E_USAGE`, without reading stdin, accessing URLs or replacing inputs. `[specs.download]`, URL/header metadata and former header/environment fields are no longer accepted; old download tables fail with `E_CONFIG_UNKNOWN_FIELD` rather than being ignored. Remove these tables and fetch specifications externally before using existing local configurations. [TOML 1.1 syntax](configuration.md#supported-toml-language), including inline tables for existing `defaults`/`specs`, is supported; grammar support does not restore download metadata or expand the closed schema.

The old `v0.1.0-alpha.1` artifacts are unchanged and may still contain the former downloader; this contract describes the current source. CLI version, config version 1 and manifest/name-lock versions are not reset by this change. A future release must classify this removal as a breaking CLI/configuration change under the [version policy](versioning.md), without rewriting historical tags/assets. Generated Kotlin APIs and JSON/HTTP/SSE wire behavior are unchanged.

[Native file/pipe/EOF/error/removal contracts](../internal/cli/local_input_process_test.go) define the verification surface. Verification status and executed platforms are reported separately; this contract is not a cross-platform or complete OpenAPI-validation claim.
