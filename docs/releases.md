# Binary packaging and GitHub CI

The checked-in workflows prepare preview binaries; they are not evidence that hosted CI has run or that a release exists.

## Workflows

- [CI](../.github/workflows/ci.yml): pull requests, `main` pushes and manual runs. The first gate runs only Go format, vet, ordinary tests and race tests on Linux, macOS and Windows. The Kotlin generation/check jobs depend on the entire Go gate succeeding; they then run source installation, native CLI Kotlin generation and read-only `--check`, followed by Python tooling tests, native archive packaging/isolated installation with file/stdin checks, and the candidate scan. Generation/check does not compile or execute Kotlin; the model and HTTP/SSE compilation/runtime suites remain separate local verification tools. The candidate scan is not a complete privacy/history audit.
- [Release binaries](../.github/workflows/release.yml): a `v*` tag push, or a manual run selected on a tag. All CI jobs must pass first. Six cross-built targets produce `.tar.gz` (macOS/Linux) or `.zip` (Windows) archives, then `SHA256SUMS` and a **draft prerelease**. A branch-selected manual run tests but does not package or publish.

Targets: `darwin-amd64`, `darwin-arm64`, `linux-amd64`, `linux-arm64`, `windows-amd64`, `windows-arm64`. Cross-building an archive does not establish runtime behavior on that target. No stable or platform-complete release claim is implied.

Actions are pinned to commit SHAs. Default permissions are read-only; only the draft-upload job can write repository contents. Checkout does not retain Git credentials. Existing releases and existing local archives are not overwritten.

## Versions

Use the [versioning and compatibility policy](versioning.md) to choose the next version and preview stage. The source candidate is `0.2.0-alpha.1`; the historical first preview is `v0.1.0-alpha.1`, with its own [scope and limitations](releases/v0.1.0-alpha.1.md). Draft is a GitHub review state; Prerelease is a maturity flag. Neither grants permission to publish or proves that the complete roadmap has passed.

## Candidate acceptance

The [JSON release checklist](json-release-checklist.md) freezes the current source candidate to the existing bounded JSON profiles, macOS ARM64 CLI and Kotlin/JVM, at alpha maturity. Candidate-wide acceptance must bind those profiles, migrations, tools and source together; historical alpha CI cannot substitute for it. The [next-alpha scope and migration guide](releases/next-alpha.md) classifies the additions, downloader removal and increased Go minimum as a MINOR alpha (`0.2.0-alpha.1`) for preparation. The CLI version and version assertions now identify `0.2.0-alpha.1`; candidate archives must be rebuilt from that source. These documents do not authorize Git writes, uploads or publication.

## Before the first tag

1. Approve the exact sanitized source tree and reachable history before pushing. Do not upload local specs, logs, archives, credentials or heap dumps.
2. Include the approved [MIT license](../LICENSE) as a regular `LICENSE` file and preserve required third-party notices in [THIRD-PARTY-NOTICES](../THIRD-PARTY-NOTICES). Packaging refuses to run without both regular files; CI does not choose or change the license.
3. Match the tag to the CLI version declared in `internal/cli/arguments.go` (currently `v0.2.0-alpha.1`). Update version contracts together when changing it.
4. Obtain permission to push the source/tag. Review hosted CI results and the draft assets/notes before publishing the draft.

The only archive members are the binary, `README.md`, `LICENSE`, the Go runtime `GO-LICENSE`, `THIRD-PARTY-NOTICES`, `RELEASE.json`, `THIRD-PARTY-NOTICES.md`, `YAML-LICENSE`, `YAML-NOTICE` and `APACHE-2.0.txt`. The notices preserve go-toml MIT and YAML MIT/Apache attribution and full license texts. Historical alpha archives are unchanged. Builds disable CGO and trim source paths; package input files must be regular files and machine user paths are rejected. These guards are not proof that every possible secret format has been found. Packages are unsigned.

## Local packaging

With the approved license present and Go 1.27.2 or newer installed for the current checkout (the existing `v0.1.0-alpha.1` tag retains Go 1.26.0):

```bash
python3 scripts/release/package.py --tag v0.2.0-alpha.1 --target darwin-arm64
```

Outputs stay under ignored `build/`, `artifacts/` or `dist/`. The script only builds/packages; it never uploads, creates tags or changes Git state.

Use the emitted SHA-256 to verify and exercise an archive on its actual target host:

```bash
python3 scripts/release/verify.py \
  build/release/openapi-kt-v0.2.0-alpha.1-darwin-arm64.tar.gz \
  --sha256 "$ARCHIVE_SHA256" --version 0.2.0-alpha.1 --work-dir build/package-check
```

The verifier checks the exact ten regular members, metadata, checksum and executable version. Archive inspection also accepts unchanged five/six-member `v0.1.0-alpha.1` packages; new versions require both dependency notice/license sets. CLI smoke checks target the current source candidate's file/stdin contract. The verifier installs into an isolated temporary directory with an empty `PATH`, then checks help, file/stdin validation and generation, generated-source parity, read-only check, no-op mtimes and failure preservation. It uses an embedded synthetic specification, so no fixture checkout or Go/Gradle installation is required by the packaged CLI. Run it separately on each declared native host; cross-built archives cannot substitute for those results. The release workflow does this automatically: the `verify-native` job checks each published archive on a matching hosted runner (`darwin-arm64`, `darwin-amd64`, `linux-amd64`, `linux-arm64`, `windows-amd64`) before a draft is created. The remaining target (`windows-arm64`) stay byte-verified unless someone runs the verifier on that host. See [publication checks](publication.md) for repository and release boundaries.

## Installing a binary

Follow the [installation guide](installation.md#download-a-binary) for platform selection, checksum verification, extraction and user-local `PATH` setup. The current `v0.1.0-alpha.1` is a private-repository Draft Prerelease; viewing a Draft requires repository write access. It is not an anonymous public download or a stable release. Go and Gradle are not required to run the binary. After installation, run:

```bash
openapi-kt --version
openapi-kt --help
```
