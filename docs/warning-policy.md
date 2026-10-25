# CLI warning policy

Generation and validation accept warnings by default. CI can opt into rejection:

```bash
openapi-kt generate --input tests/fixtures/cli/warning-policy.json \
  --output build/generated/warning-policy --package-name org.example.warningpolicy \
  --deny-warnings --allow-warning W_ALL_OF_APPROXIMATION \
  --allow-warning W_JSON_ELEMENT_APPROXIMATION
openapi-kt validate --input tests/fixtures/cli/warning-policy.json --deny-warnings
```

`--deny-warnings` is a bare, non-repeatable flag on `generate` and `validate`, including `generate --check`. It is not a core generation option; the same policy applies to file and explicit stdin input. Repeat `--allow-warning CODE` (or `--allow-warning=CODE`) to allow exact warning codes; it requires `--deny-warnings`. Codes use ASCII `W_` followed by one or more uppercase letters, digits or underscores. No wildcard, comma list or regex is supported. Duplicate codes are harmless. Well-formed unused/unknown codes are permitted for forward-compatible policies, but never allow a different emitted code.

After successful in-memory generation, any unallowed warning produces exit **1**, JSON `status: "error"`, empty `files` and one `E_WARNING_DENIED` diagnostic per rejected warning in its original order and at its original Pointer. The safe message includes the emitted warning code. The `warnings` list still contains **all** original warnings, including allowed ones: this is acceptance policy, not suppression. JSON reports remain one stdout document with empty stderr; text reports write warnings/errors to stderr, without a success line.

Policy runs before output comparison/staging/writing, directory creation or lock acquisition. Opt-in name locking first reads previous owned state before core generation; a state-read failure retains its storage diagnostic rather than becoming warning denial. Rejection preserves old generated files, manifest, coordination files, handwritten files and modification times. It takes precedence over a potential check difference or output mutation/acquisition failure, not an earlier read-only state failure. Allowing every emitted warning produces the same artifacts as default generation and retains no-op/check behavior. A document without warnings succeeds under deny. Input/generation failures keep their original diagnostics, rather than being converted into warning-policy failures.

**Migration:** defaults, core artifacts and Kotlin APIs/wire are unchanged. Only explicit deny mode adds failure-on-warning. This is not full schema validation, severity customization or phase metrics. [Configured per-spec policy](configuration.md#configured-warning-policy) is also available. See [P1 diagnostics](p1-contract.md) and the [compatibility matrix](compatibility-matrix.md).

The [synthetic fixture](../tests/fixtures/cli/warning-policy.json), [parser/policy tests](../internal/cli/warning_policy_test.go) and [actual native CLI tests](../internal/cli/warning_policy_process_test.go) cover mixed/all/none allowlists, original locations, text/JSON streams, required flag grammar, repeated codes, default compatibility, check/no-op and rejection before output mutation/acquisition.

Defaults/specs may persist `deny_warnings` and `allow_warnings`; explicit false/empty replace defaults and invocation warning flags override configured policy for every generated source. See [the effective policy, migration and atomic-batch contract](configuration.md#configured-warning-policy).
