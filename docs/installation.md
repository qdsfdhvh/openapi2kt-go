# Installation

`openapi-kt` is a native CLI. Running a binary requires neither Go, Java nor Gradle. Compiling its generated Kotlin sources requires a separate Kotlin/serialization/Ktor setup.

## Choose a version

The current source CLI identifies the **`0.2.0-alpha.1` candidate**, which has not been published. The [README](../README.md) describes this source checkout. See [scope and migration](releases/next-alpha.md) and the [candidate acceptance checklist](json-release-checklist.md) for its release boundaries.

The historical **`v0.1.0-alpha.1`** is a Draft Prerelease in a private repository, visible only to collaborators with write access. Its archives are unchanged and do not include later model/reference support or the new file/stdin input boundary. They may still include the former downloader; current source removes it. See the [first preview scope](releases/v0.1.0-alpha.1.md) and [input migration guide](local-input.md#configuration-and-migration).

## Install from source

The current checkout requires **Go 1.27.2 or newer** with a matching `GOROOT`. Go 1.27 builds require **macOS 13 Ventura or newer** on macOS. The existing `v0.1.0-alpha.1` tag and its archives retain their original Go 1.26.0 build; upgrading the source toolchain does not replace those packages.

The pinned TOML and YAML parsers are included in `vendor/`; install Go separately. From the local checkout, add `GOBIN` (or `$(go env GOPATH)/bin` when unset) to your `PATH`, then install offline:

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go install ./cmd/openapi-kt
openapi-kt --version
openapi-kt --help
```

`GOTOOLCHAIN=local` disables automatic toolchain downloads; `GOPROXY=off GOSUMDB=off` keeps installation offline using the included vendor source. [TOML notices](../THIRD-PARTY-NOTICES) and [YAML notices](../THIRD-PARTY-NOTICES.md) are included in new binary archives. Remote `go install …@version` installation is not available yet; see [release packaging](releases.md).

To keep the binary inside the checkout instead, follow the [README build command](../README.md#installation).

## Download a binary

1. Sign in to GitHub and open [Releases](https://github.com/qdsfdhvh/openapi2kt-go/releases). The `v0.1.0-alpha.1` Draft requires repository write access. If you cannot access it, use an authorized source checkout; an anonymous download link is not available.
2. Download **both** `SHA256SUMS` and the archive matching your OS/CPU into the same directory:

| Platform / CPU | Archive |
|---|---|
| macOS Intel | `openapi-kt-v0.1.0-alpha.1-darwin-amd64.tar.gz` |
| macOS Apple Silicon | `openapi-kt-v0.1.0-alpha.1-darwin-arm64.tar.gz` |
| Linux x86-64 | `openapi-kt-v0.1.0-alpha.1-linux-amd64.tar.gz` |
| Linux ARM64 | `openapi-kt-v0.1.0-alpha.1-linux-arm64.tar.gz` |
| Windows x64 | `openapi-kt-v0.1.0-alpha.1-windows-amd64.zip` |
| Windows ARM64 | `openapi-kt-v0.1.0-alpha.1-windows-arm64.zip` |

3. Verify the checksum **before** extracting or installing. Packages are unsigned; checksums verify file integrity, not a signature or complete security audit. Cross-built packages do not establish runtime support on every architecture.

### macOS / Linux

In the download directory, run this block in **Bash**. Set `target` to `darwin-amd64`, `darwin-arm64`, `linux-amd64` or `linux-arm64` as appropriate; it stops if verification fails:

```bash
set -euo pipefail
version=v0.1.0-alpha.1
target=darwin-arm64
archive="openapi-kt-${version}-${target}.tar.gz"
checksum=$(awk -v name="$archive" '$2 == name { print; count++ } END { if (count != 1) exit 1 }' SHA256SUMS)
if command -v sha256sum >/dev/null 2>&1; then
  printf '%s\n' "$checksum" | sha256sum -c -
else
  printf '%s\n' "$checksum" | shasum -a 256 -c -
fi
tar -xzf "$archive"
mkdir -p "$HOME/.local/bin"
install -m 755 "openapi-kt-${version}-${target}/openapi-kt" "$HOME/.local/bin/openapi-kt"
export PATH="$HOME/.local/bin:$PATH"
openapi-kt --version
```

For future terminals, add `export PATH="$HOME/.local/bin:$PATH"` to your shell startup file (for example `~/.zshrc` or `~/.bashrc`).

### Windows (PowerShell)

In the download directory, set `target` to `windows-amd64` or `windows-arm64`, then run:

```powershell
$ErrorActionPreference = 'Stop'
$version = 'v0.1.0-alpha.1'
$target = 'windows-amd64'
$archive = "openapi-kt-$version-$target.zip"
$pattern = '^([a-fA-F0-9]{64})  ' + [regex]::Escape($archive) + '$'
$entries = @(Get-Content -LiteralPath '.\SHA256SUMS' | Where-Object { $_ -match $pattern })
if ($entries.Count -ne 1) { throw 'Missing or duplicate checksum entry.' }
$expected = ($entries[0] -split '  ', 2)[0]
$actual = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash
if ($actual -ne $expected) { throw 'SHA-256 verification failed.' }
Expand-Archive -LiteralPath $archive -DestinationPath '.\openapi-kt-unpacked'
$bin = Join-Path $env:LOCALAPPDATA 'openapi-kt\bin'
New-Item -ItemType Directory -Force -Path $bin | Out-Null
Copy-Item -LiteralPath ".\openapi-kt-unpacked\openapi-kt-$version-$target\openapi-kt.exe" -Destination "$bin\openapi-kt.exe"
$env:Path = "$bin;$env:Path"
openapi-kt --version
```

For future terminals, add `%LOCALAPPDATA%\openapi-kt\bin` to your **user** `Path` in Windows Environment Variables settings.

## After installation

The historical preview prints `openapi-kt 0.1.0-alpha.1`; a build of the current source prints `openapi-kt 0.2.0-alpha.1`. Run `openapi-kt --help` to inspect your installed version's commands.

Continue with the [quick start](../README.md#quick-start), replacing `./build/openapi-kt` with `openapi-kt` when installed on `PATH`. Binary archives do not contain the repository's fixtures: use your own local OpenAPI JSON file as `--input`.
