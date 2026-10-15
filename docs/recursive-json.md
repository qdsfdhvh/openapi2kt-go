# Recursive JSON-value roots

An OpenAPI 3.1 named component can project a **connected graph of complete JSON-kind unions** to `kotlinx.serialization.json.JsonElement`. Self-recursion and mutual recursion use the same carrier. This is an F10 profile for complete kind covers, not a general recursive schema resolver or JSON Schema validator.

```json
{
  "openapi": "3.1.0",
  "components": {
    "schemas": {
      "Value": {
        "oneOf": [
          {"type": "null"},
          {"type": "string"},
          {"type": "boolean"},
          {"type": "number"},
          {"type": "array", "items": {"$ref": "#/components/schemas/Value"}},
          {"type": "object", "additionalProperties": {"$ref": "#/components/schemas/Value"}}
        ]
      }
    }
  }
}
```

`anyOf` is also accepted. The six branches must occur exactly once, in any order. `number` already includes integer values; an additional integer branch is not admitted. Array items and object values must reference a **named complete kind cover**, directly or through finite annotation-only named `$ref` chains, including correctly escaped JSON Pointer segments. They may target the same root, different mutually recursive roots, or a downstream complete cover. Every reachable root must satisfy this grammar; an ordinary DTO, partial cover or transparent reference-only cycle cannot supply a JSON-value branch. Standard supported annotations are allowed. Bounds, formats, enum/const pins, defaults, native/value-class extensions, nullable keys and other semantic keywords are not discarded to obtain this projection.

With suffix `Dto`, the root emits:

```kotlin
typealias ValueDto = kotlinx.serialization.json.JsonElement
```

Pure root aliases retain canonical backing and original source hops. They do not create wrapper classes, constructors or companion serializers. Use the backing or reified codec. Root-alias `nullable: true` exports the canonical alias with `?`; absent intent inherits the nearest alias and explicit false narrows the Kotlin carrier again. Typed consumer intent takes precedence. A narrowed non-null JsonElement still admits the JsonNull value. These rules match [nullable literal-true aliases](boolean-roots.md#nullable-root-aliases). Intermediate reference nodes inside the recursive array/map graph allow annotations only; nullable siblings on those graph edges remain diagnosed.

The non-null Kotlin carrier contains `JsonNull` as a JSON value. Explicit nullable **consumers** retain the existing `JsonElement?` policy: JSON null can decode to Kotlin null; optional fields retain their default/presence policy. Required nullable constructor arguments do not guarantee a wire key with `explicitNulls=false`. Request-only `Patch.Value(JsonNull)` and `Patch.Null` share a wire representation; nullable Patch decoding yields `Patch.Null`.

Typed fields, lists/maps, JSON HTTP bodies/responses and existing SSE data use the same carrier. No generated positional/native union wrapper or extra input/response DTO is introduced for the root. SSE envelopes and nominal allOf/union/hierarchy shape borrowing do not gain support.

[Typed reference chains to named complete covers](recursive-cover-reference-chains.md) now have a separate source-identity contract, including finite internal Pointers to reference-only leaves and component-root bridges. Incoming hops add to the original graph suffix; this does not admit inline cover nodes or nominal shape borrowing.

## Codec and numeric boundaries

Generation preserves a JSON value carrier rather than converting the branches into generated Long/Double wrappers. **Encoding still uses the caller's kotlinx.serialization JsonElement codec.** This is not a lossless arbitrary-precision number or original-text codec.

In the tested serialization 1.11.0 configuration, `1e99` encodes as `1.0E99`; values beyond unsigned 64-bit range can also lose precision, for example `18446744073709551617` encodes as `1.8446744073709552E19`. These are backing-codec behaviors, not additional generated alias conversions. Applications requiring exact arbitrary numeric lexemes or precision must establish a separate codec/wire policy; this projection does not promise it.

## Diagnostics, limits and migration

Each original complete-cover root emits `W_RECURSIVE_JSON_PROJECTION`, in deterministic declaration order. It explains the JSON carrier and absence of a general schema/oneOf match-count validator. Existing warning denial/allowlisting applies before output mutation.

Source validation retains every original root and branch/reference node. The graph is analyzed as strongly connected components without recursively duplicating shared descendants. A recursive group contributes two source edges per complete-cover member (branch and reference leaf) and one per transparent `$ref` node; its longest outgoing group height is added. A self-recursive root therefore retains height 2, a mutually recursive pair height 4, and a pair referring onward to a self-recursive root height 6. This conservative finite bound is independent of entry root and branch order. It remains part of the existing 48-level source budget after projection; cached aliases and consumer containers cannot erase it. Traversal has the existing 16,384-work ceiling. Other resource limits and validation policies are unchanged.

**Migration:** the exact profile above previously failed with `E_SCHEMA_UNSUPPORTED`. Ordinary recursive DTOs remain ordinary DTOs; a single self-reference or partial kind cover is not sufficient to erase a union. Mutually recursive complete-cover roots previously failed and now export transparent JsonElement aliases, with one projection warning per root. Existing self-recursive output and wire behavior remain unchanged. Finite named reference chains and root nullable aliases now have the same Kotlin carrier/nullability behavior as literal-true aliases; previously rejected nullable root-alias inputs now generate deterministic nullable/non-null aliases. OpenAPI 3.0, internal schema Pointer targets outside the separate named-cover reference-chain contract, inline covers, constrained covers, nullable recursive graph-edge siblings and broader nullable/composition graphs remain outside this profile.

[Self-recursive fixture](../tests/fixtures/http/recursive-json.json), [mutual fixture](../tests/fixtures/http/mutual-recursive-json.json), [mutual core](../core/mutual_recursive_json_test.go), [transparent/nullable graph fixture](../tests/fixtures/http/recursive-json-reference-graph.json), [graph core](../core/recursive_json_reference_graph_test.go), [graph native lifecycle](../internal/cli/recursive_json_reference_graph_process_test.go), [graph SCC/depth guards](../internal/schema/mutual_recursive_json_test.go), [mutual native lifecycle](../internal/cli/mutual_recursive_json_process_test.go), [core](../core/recursive_json_test.go), [source/order guards](../core/recursive_json_source_test.go), [resource guards](../core/recursive_json_guards_test.go), [native lifecycle](../internal/cli/recursive_json_process_test.go) and [fresh Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/RecursiveJSONContracts.kt) cover the bounded contract. Kotlin runtime evidence uses generated Data/Plain classes with both validation settings and four Json configurations, including backing-codec comparisons, JsonNull, constructor/Data copy, Patch, controlled HTTP and fragmented still-open SSE closure/reuse. Independent source/method limits are Go boundary contracts, not boundary Kotlin runtime, platform or Cartesian proofs.
