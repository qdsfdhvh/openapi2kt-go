# Inferred null literals

Plain `{"const":null}` and `{"enum":[null]}` schemas may omit `type`. They use the same `kotlinx.serialization.json.JsonNull` carrier, generated declarations and codecs as their explicit `type:"null"` counterparts. Both validation settings and Data/Plain model styles support this profile.

```json
{"Holder":{"properties":{
  "fixed":{"const":null},
  "requiredNull":{"enum":[null]},
  "optionalNull":{"enum":[null]}
},"required":["requiredNull"]}}
```

`fixed` is an always-encoded JsonNull body property. `requiredNull` remains a constructor argument; `optionalNull` retains the ordinary `JsonNull? = null` policy. Scalar component aliases, typed List/Map leaves, nested DTOs, supported object shapes and object-union property pins retain the original canonical ownership. Inference reads the actual const/enum node; it does not create a synthetic type key, replacement schema or enum declaration.

## Nullability and wire behavior

Inferring a JsonNull value does not infer Kotlin nullability. Explicit `nullable` intent and existing optional-field policy continue to select a nullable view. JsonNull, Kotlin null and absence remain distinct Kotlin states. A nullable JsonNull codec decodes wire null into Kotlin null. Optional nullable request fields retain Patch.Absent; Patch.Value(JsonNull) and Patch.Null both encode wire null and decode as Patch.Null. See the [explicit null contract](explicit-null-literals.md) for these existing presence and codec rules.

Named typed HTTP payloads/results and known SSE data use the JsonNull codec and reject wrong scalar/object/array values. Direct inline null success schemas retain the existing Unit projection, whose reader ignores body contents. Query values retain the existing `"null"` text representation. Object-union const-only singleton dispatch keeps its documented token approximation; this profile does not implement exact oneOf match counts or new tag validation.

## Grammar and limits

Inference applies only when `type` is absent and the plain literal is a null const or a nonempty enum containing only actual JSON null members. A vocabulary must still contain exactly one unique member. Duplicate nulls fail at their original index; the raw 240-member cap precedes uniqueness. An empty/non-array or mixed enum keeps its existing planner and diagnostics. In particular, explicitly nullable mixed String vocabularies keep their open String classifier, and the String token `"null"` remains a String.

Explicit types take precedence. Native, format and value-class profiles are not inferred; unknown keys, malformed metadata and retained constraints still fail their original grammar. [Null-only DTO field defaults](null-only-defaults.md) have separate field-only admission; root/ref/leaf and const defaults remain excluded. Nullable component roots, `type:["null"]`, null-only composition branches and structured const values remain outside this profile. A null literal used as a oneOf/anyOf branch is rejected at the original const/enum keyword rather than silently becoming a nullable branch. Supported object variants may still contain null literal properties.

Discarded fields and fixed-object extra schemas must pass source validation. Source depth 48, checked collection edges 16, serialized/operation slots 240 and composition work 16,384 remain unchanged. Inferring a type neither inserts nor erases a source hop. Independent width/depth probes do not promise the Cartesian maximum or additional platforms.

## Compatibility and verification

These formerly rejected plain typeless null inputs become accepted. Existing accepted inputs keep their generated Kotlin APIs, wire behavior, warnings and name-lock fingerprints. Equivalent inferred/explicit owned shapes replay the same locks; changing a const's actual domain still fails shape identity checks. There is no manifest or lock protocol migration.

The [synthetic fixture](../tests/fixtures/http/inferred-null-literals.json), [core equivalence tests](../core/inferred_null_literals_test.go), [grammar/source guards](../core/inferred_null_literals_guards_test.go), [original-node tests](../internal/schema/inferred_null_literals_test.go) and [native preservation tests](../internal/cli/inferred_null_literals_process_test.go) cover the new source profile. Existing null resource, optimizer/lock and fuzz tests also include inferred sources.

Run `bash scripts/test.sh feature inferred-null-literals` for fresh Data/Plain projects with validation off/on. Each project compiles the existing [null literal consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ExplicitNullLiteralsContracts.kt) against the inferred fixture and exercises all four `encodeDefaults`/`explicitNulls` settings, Patch, canonical constructors, HTTP and live SSE closure/cancellation/client reuse. The dedicated inferred feature route supplies separate inferred-source compilation and runtime evidence. The ordinary `http` route uses the [null-only default fixture](null-only-defaults.md) and retains the existing null consumer. General default/const/composition support and Android/KMP acceptance remain separate work.
