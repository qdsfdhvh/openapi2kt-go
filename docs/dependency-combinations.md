# Tested JVM dependency combinations

The generator's default consumer builds retain Kotlin2.4.10,
serialization1.11.0, Ktor3.5.2 and datetime0.7.1. Additional offline trials used the
same freshly generated synthetic source and consumer inventories:344 Kotlin
files in the complete models exporter and5813 generated/provider/consumer files
in the complete HTTP registry. They copied no compiled output and used no
metadata-version bypass, old UUID opt-in or dependency replacement.

| Compiler/plugin | Serialization | Ktor | Actual result |
|---|---|---|---|
|2.3.10|1.10.0|3.4.0|Models and HTTP compilation reject unannotated kotlin.uuid.Uuid uses requiring ExperimentalUuidApi; no runtime acceptance|
|2.4.0|1.10.0|not used|Complete models compilation and all17 JSON/API contract groups pass|
|2.4.0|1.10.0|3.4.0|Full HTTP source compilation passes; fragmented UTF-8 SSE leaf-character runtime assertion fails|
|2.4.0|1.11.0|3.5.2|Full HTTP/SSE registry compilation and runtime pass; datetime0.7.1|

The successful lower profiles resolve kotlin-stdlib2.4.0. The model profile
resolves serialization core/JSON1.10.0; the successful HTTP profile resolves
serialization1.11.0 and Ktor3.5.2, including their actual transitive dependencies.
[Exact resolved inventories](benchmarks/lower-dependency-combinations.json)
record these combinations and their evidence boundaries.

A consumer that asserts its runtime Kotlin version can supply the exact expected
version through `openapiKt.expectedKotlin`; its default remains2.4.10. That
assertion is retained when testing other profiles rather than deleted. All wire,
serializer, cancellation and failure assertions remain. The fixed default
HTTP registry also passed after this test-harness adjustment.

These are tested points on macOS ARM64/JVM with JDK21 and Gradle9.5.1, not a
promise that every intermediate or earlier version works or fails. The lower
Ktor trial remains a failed combination. Kotlin/serialization plugin version,
actual runtime dependencies and declared dependency versions are distinct; do
not report a requested downgrade as accepted if dependency resolution selected
a newer version. Other hosts/targets/engines and the full independent feature
matrix require separate evidence. Generation remains deterministic and offline.
