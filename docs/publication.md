# Publication checks

The repository and published packages must contain only product source, synthetic fixtures, public documentation and required attribution. Do not upload an entire development directory or assume that `.gitignore` removes files already tracked by Git.

## Candidate scan

From an initialized repository:

```bash
python3 scripts/publication/check.py
```

The scanner checks tracked files (including tracked-but-ignored files) and untracked, non-ignored candidates. It rejects local/build state in that candidate set, escaping symlinks, sensitive credential/key filenames, common key/token shapes and machine user paths. It does not scan Git history or establish that all possible secret formats have been found.

Before Git exists, use an explicitly limited file-only scan:

```bash
python3 scripts/publication/check.py --source-tree
```

This excludes local notes, local inputs, agent state and build outputs, including Go test executables (`*.test` and `*.test.exe`). Keep coverage/profiling outputs under `build/`; arbitrary `*.out` files are not excluded because they may be test fixtures. A clean result does **not** prove what a future commit or archive will contain. Never use this mode as evidence that previously committed files or history are safe.

An optional local JSON policy can add case-insensitive forbidden substrings for file contents, paths and symlink targets:

```bash
python3 scripts/publication/check.py --policy .pi/public-policy.json
```

The policy has a single `forbidden_terms` string array. Keep confidential policy values outside the published tree; the script does not print values, file contents or candidate paths. Reports identify findings by candidate number and category. No network or upload is performed.

## Public repository versus release readiness

Before making a repository public, review the exact commit/tree and history, ensure local inputs and notes were never staged, and preserve third-party license notices. The project uses the explicitly approved [MIT license](../LICENSE); third-party material retains its own licensing and required notices.

Before publishing a Go module or binary, additionally verify:

- package names, metadata, license and declared dependency versions;
- the exact packaged file list, not just the working tree;
- building, testing and installing from the package without external checkout dependencies;
- supported host-platform checks and dependency license/security review;
- documentation that clearly separates implemented behavior from planned features.

No release has been published. MIT licensing does not authorize changing repository visibility, pushing a release tag or publishing assets; those require an explicit operator decision. Follow the [versioning policy](versioning.md) as well as the package checks; these checks do not publish anything.
