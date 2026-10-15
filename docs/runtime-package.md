# Shared runtime package

`core.GenerationOptions.RuntimePackage`, `generate|validate --runtime-package PACKAGE`, and [configuration](configuration.md) `runtime_package` select the package for the two existing generated utilities:

- `Patch` and `PatchSerializer` in `Patch.kt`;
- the internal field-only `LongAsStringSerializer` in its own file.

Empty/omitted selects the historical `<package_name>.models`. Explicitly selecting that same package produces identical sources. The option does not relocate model/HTTP/SSE declarations, caller-native types/codecs, API error helpers or per-file validation helpers, and it does not deduplicate models or enable name locks. Unused utilities are not emitted.

```toml
config_version = 1
[defaults]
output_dir = "build/generated"
runtime_package = "org.example.generated.runtime"
[[specs]]
name = "catalog"
input = "catalog.json"
package_name = "org.example.catalog"
[[specs]]
name = "inventory"
input = "inventory.json"
package_name = "org.example.inventory"
```

Precedence is builtins < defaults < spec < explicit CLI override. Empty strings replace a configured default. CLI overrides require one selected spec. Package syntax uses the same safe ASCII/reserved-word/path rules as model packages. Core validates before parsing document bytes; configuration validates every spec's effective settings before reading selected inputs. Direct-input CLI retains its read-then-core-option-validation ordering.

## Ownership and package scope

An explicit shared runtime package must have one declared, resolved output-root path across the complete inventory, including unselected specs and matching implicit default packages. Conflicting declarations fail with `E_CONFIG_RUNTIME_ROOTS` before generation. Differently spelled root aliases are not automatically merged. Legacy entirely implicit multi-root layouts retain their existing behavior; this does not certify combined compilation of duplicate implicit runtime classifiers across roots.

The pure core does not know other specs or output roots. It independently emits needed utility bytes. [Named ownership](output-ownership.md) combines identical paths/bytes into one file with multiple owners; incompatible bytes fail rather than picking one producer. Partial generation retains the other owners. Moving one spec's runtime package preserves the old utility while a peer still owns it. Full reconciliation removes utilities only after the last owner is gone. Config removal of an entire root still does not discover/clean that root.

Runtime classifier reservations apply to the actual package. `Patch`, `PatchSerializer` and `LongAsStringSerializer` model names in another package may coexist; a conflicting root/API/client/model classifier in the utility's actual package is rejected. Default package collisions retain their original diagnostics. Existing handwritten collisions and modified generated files remain output-safety errors.

## API and wire impact

Opting into another package intentionally changes imports/references and artifact paths for `Patch` and the bigint serializer annotation. There are no automatic old-package aliases or migration shims. Default options keep historical output; do not confuse package relocation with [ownership-version migration](ownership-migration.md).

`Patch.Absent`/`Null`/`Value`, generic identity, default omission and explicit-null behavior are unchanged. Bigint fields (including request Patch values) retain decimal-string codecs; native Long roots/list values remain JSON numbers. Utility bodies, serializer descriptors and visibility remain unchanged. In particular, the bigint utility remains `internal`: shared sources are intended to compile together in the same Kotlin module, not as an independently compiled public runtime library.

Verification surfaces: [core byte parity/eligibility/scoped collisions](../core/runtime_package_test.go), actual native [default/shared generation](../internal/cli/runtime_package_process_test.go), [owner lifecycle/root guards](../internal/cli/runtime_package_owners_process_test.go), and the fresh two-spec Kotlin [Patch/serde/HTTP/live SSE contract](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/RuntimePackageContracts.kt). These do not prove all platforms, arbitrary compiled-module layouts, hoisting/name-lock integration or cross-root crash atomicity.
