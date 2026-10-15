# YAML document consumer

`bash scripts/test.sh feature yaml-document` builds a native CLI and generates
fresh sources from [equivalent JSON/YAML inputs](../../tests/fixtures/yaml/),
checks identical files, manifests, reports and read-only checks, then compiles
and runs this consumer with Data/Plain × validation on/off. Four Json
encodeDefaults/explicitNulls combinations exercise required fields, defaults,
open enum unknown values, lists/nulls, Patch missing/null/value and MockEngine
GET query/PATCH body/response behavior. Dependencies/toolchain come from the
existing Kotlin HTTP smoke setup and run offline; the script retains its
isolated project in ignored `build/` for inspection. No heap/cap change is used.

This consumer verifies the [bounded YAML profile](../../docs/yaml-input.md) and
the shared existing emitter; it is not full YAML or platform acceptance.
