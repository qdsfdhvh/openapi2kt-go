# Versioning and compatibility

openapi2kt-go uses Semantic Versioning for the generator product, not for the input OpenAPI specification or the Kotlin toolchain.

## Version identity

Project versions have the form `MAJOR.MINOR.PATCH`, optionally followed by `-alpha.N`, `-beta.N` or `-rc.N`. The three version numbers are non-negative integers without leading zeroes. `N` is a positive integer starting at 1, without leading zeroes. Stage names are lowercase. Build metadata (`+...`) is not used.

For example:

| Surface | Example |
|---|---|
| CLI version in `internal/cli/arguments.go` | `0.2.0-alpha.1` |
| Proposed Git tag and release name | `v0.2.0-alpha.1` |
| Archive prefix | `openapi-kt-v0.2.0-alpha.1-<os>-<arch>` |
| `RELEASE.json` version | `v0.2.0-alpha.1` |

All six targets use the same version and source revision. A release tag must equal `v` plus the declared CLI version. Update version assertions and version-dependent documentation together; rebuild and reverify packages when the version changes.

The source CLI declares `0.2.0-alpha.1`, following the [next-alpha scope and migration](releases/next-alpha.md). Tag `v0.2.0-alpha.1` was published as a prerelease from that source, producing the six CLI archives and `SHA256SUMS` listed in the [installation guide](installation.md). Later changes merged into `main` are not part of that tag, so the CLI version alone does not identify which sources a checkout contains; use the commit or tag. The historical first preview `v0.1.0-alpha.1` retains its [scope and limitations](releases/v0.1.0-alpha.1.md). Neither version claims that the complete roadmap is finished.

## What compatibility means

Judge compatibility against the documented supported inputs, profiles and defaults, including:

- Generated Kotlin public names, packages, constructors, property and method types, nullability, defaults, nested classifier access and serializer ownership.
- JSON and HTTP/SSE wire behavior: field names, absent/null/value handling, enum and union dispatch, parameter encoding, payloads and cancellation.
- CLI commands, flags, exit codes, diagnostic codes, configuration semantics and the public Go `core` API.
- Output ownership, persisted manifest/name-lock/configuration formats, migration paths and the declared minimum toolchain/platform requirements.

Generated text differences alone are not necessarily breaking changes. Conversely, compiling successfully does not establish call-site or wire compatibility. Internal Go packages are not a public API.

Persisted-format versions are separate from the product version. Change them only when format or interpretation changes require it, and document compatibility or migration explicitly; a product version bump does not permit silently resetting existing state.

## Choosing the next version

| Change | Before `1.0.0` | From `1.0.0` onward |
|---|---|---|
| Compatible correction within an already-supported contract | PATCH | PATCH |
| New supported input/profile or backward-compatible capability | MINOR; reset PATCH to 0 | MINOR; reset PATCH to 0 |
| Breaking Kotlin, wire, CLI/core, storage or minimum-toolchain change | MINOR; reset PATCH to 0 | MAJOR; reset MINOR and PATCH to 0 |

Use the highest-impact change in a release. Before 1.0 the API is still developing, but breaking changes still require a contract test, a release note and migration guidance; they are not hidden in PATCH releases.

A correction to previously uncompilable output or rejected input within an already-promised contract may be PATCH if existing working call sites and wire behavior remain compatible. A change to previously usable output or accepted supported input is reviewed as breaking even if described as a bug fix. New OpenAPI support outside the promised scope is MINOR, not a disguised bug fix.

Documentation, CI and development-tool changes alone do not require a release. If bundled into a release, classify them by their effect on the supported product contracts. A behavior-preserving optimization may be PATCH, but a speedup claim still needs measurements and compatibility evidence.

Examples after a hypothetical `0.1.0` release:

- Fix output locking without changing public contracts: `0.1.1`.
- Add a supported reference profile with existing behavior preserved: `0.2.0`.
- Rename existing generated methods or change their payload encoding: `0.2.0` with a migration note, not `0.1.1`.

## Preview stages

| Stage | Admission and intended use |
|---|---|
| `alpha.N` | Supported slices are usable, but scope and APIs may still evolve. Publish explicit known limitations; do not call it the complete first version. |
| `beta.N` | The declared release scope and intended APIs/wire are frozen, and the announced profile matrix passes. Focus on integration feedback and corrections. |
| `rc.N` | No planned features or compatibility changes remain for this release. Package contents, checksums, installation, supported host targets and release-blocking tests pass. |
| No suffix | The declared release scope and release checks are accepted. A `0.x` release still does not promise 1.x API stability. |

For a fixed target version, increment the stage number (`alpha.1` → `alpha.2`); start a new stage at 1 (`alpha.2` → `beta.1` → `rc.1`). Change the target version when the intended scope or compatibility level changes, resetting the stage number. Numbers are not dates, commit counts or CI run IDs.

`1.0.0` requires the [complete first-version definition of done](development-plan.md#84-首个完整版本-definition-of-done), or an explicitly approved and documented scope adjustment. It does not imply full OpenAPI/JSON Schema support, YAML, Android or KMP unless those contracts have actually been verified.

## Release discipline

- Alpha, beta and RC releases are marked **Prerelease** in GitHub. **Draft** is a separate review state, not a maturity level or a substitute for the version suffix.
- The current workflow creates draft prereleases for every eligible tag; a suffix-free tag does not automatically publish a final release. Final promotion requires a separate review and operator approval.
- Published versions and uploaded tags/assets are immutable: do not rewrite or reuse them. Publish a new version for a correction.
- Release notes identify supported profiles, additions/fixes, breaking changes and migrations, actual toolchains/host checks, known limitations and unsigned-package status. Cross-builds are not host runtime proof; Go generation/check is not Kotlin compilation/runtime proof.
- Repository visibility remains unchanged. Selecting MIT or approving a version number is not authorization to make the repository public, push a tag or publish a release.

Current packaging enforces tag/CLI-version equality and output safety. It accepts a broader prerelease syntax than the project policy above; stage admission, compatibility classification and stage-number rules still require review, not an assumed automated check. See [binary packaging and release checks](releases.md).
