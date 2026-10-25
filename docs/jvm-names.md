# Generated JVM function names

Generated Kotlin declarations whose escaped function names contain whitespace
receive a fully qualified `@kotlin.jvm.JvmName` annotation. This includes private
default/validation helpers and public escaped component factories such as
`` `SeedDto default`() ``. Kotlin declarations and calls retain those names.
The JVM method name is a deterministic ASCII `openapiKtJvm_` name derived from
the complete escaped name; existing source identifiers are reserved before a
collision suffix is chosen. Ordinary function declarations remain unchanged.
User string/default literals and comments are not rewritten.

This permits DEX compilation with min SDK 23: whitespace in method names is not
accepted by the older DEX format selected for that minimum. Raising the minimum
SDK is not required by this correction. Internal HTTP server-default holders
also use whitespace-free escaped class names. Their Kotlin references and output
file paths remain coordinated.

Kotlin source call spelling, constructors, JSON field names, serializer
ownership/descriptors, HTTP/SSE wire behavior and persisted manifest/name-lock
formats retain their contracts. Regenerated affected JVM methods do have new
binary names. Rebuild generated libraries and their compiled Kotlin consumers
together; do not mix an older compiled consumer with regenerated classes. Java
reflection or tooling that locates these methods by their old JVM spelling must
use the annotated name or Kotlin metadata. The server-default holder is internal;
its JVM class name also changes. Treat these binary-name differences as an
intentional compatibility change when choosing a release version.

The three-input [platform consumer](../samples/kotlin-platform-smoke/README.md)
has passed in Data and Plain styles after this correction: JVM, JS and macOS
ARM64 Native runtime with four JSON configurations; common metadata and Android
AAR compilation; min SDK 23 test APK DEX compilation; and headless instrumentation
on a physical Android API 34 device. API 34 execution does not establish API 23
runtime support, all model/HTTP feature roles, other devices or minimum dependency
versions. Those remain separate acceptance checks.
