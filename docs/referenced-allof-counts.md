# Referenced and nested allOf whole-object counts

With `--validation`, an ordinary non-inherited DTO can retain `minProperties`/`maxProperties` from admitted object `allOf` references, named object aliases and nested object compositions. Every contribution constrains the same whole flattened JSON object. This extends the [flat inline whole-object count profile](model-semantics.md#flat-inline-allof-dto-property-counts) while keeping its existing serializer and presence contracts.

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Lower":{"properties":{"value":{"type":"string"}},"minProperties":1},
  "Alias":{"$ref":"#/components/schemas/Lower"},
  "Record":{"allOf":[
    {"$ref":"#/components/schemas/Alias"},
    {"type":"object","allOf":[{"properties":{"other":{"type":"boolean"}},"maxProperties":2}]}
  ]}
}}}
```

`Record` retains two optional constructor properties. Its JSON codec accepts whole objects containing one or two members. The lower source does not introduce a nested object or limit the final count to its own fields. Unknown fields and explicit nulls count once each before ordinary defaults and unknown-key handling. Outgoing counts use the actual automatic-serializer object under the caller's `encodeDefaults` and `explicitNulls`; Patch omission/null and always-encoded constants keep their existing behavior. A valid input can become invalid after re-encoding if unknown members were discarded or defaults change the output count.

The retained automatic `generatedSerializer()` delegate supplies the same descriptor, field order, primitive checks and Kotlin/wire API. Counts remain JSON-only. Constructors and DataClass `copy` are not whole-object count barriers; they keep existing primitive checks. PlainClass retains its existing API. No repair, hidden Json settings, extra fields, constructor slots or per-consumer count checks are added.

Count sources retain their original document nodes, Pointers and source depths in member-subtree order, followed by the object's own counts. Each source first passes the existing local grammar and interval rules: plain nonnegative Int-range integers only, with local reversal diagnosed at its own `maxProperties`. All local checks finish before cross-source consolidation. Consolidation then proceeds in source order, minimum before maximum, keeping the strongest bounds and first equal contribution; an empty cross-source interval points to the keyword that closes it. Later valid sources cannot repair malformed or reversed earlier sources. A malformed later source retains the existing local-before-fold diagnostic precedence.

An ordinary final model must have at least one constructor property. Count-bearing members must contribute a nonempty object shape through their own properties or admitted object composition. Count-only/empty members, AP-only dictionaries and standalone const-only models remain outside this DTO profile. Object references and aliases retain their existing source-role, cycle, nullability and expansion rules. Pure single-reference wrappers do not acquire a new object grammar. Nullable allOf component counts remain rejected at original count Pointers; [direct explicitly nullable ordinary root own counts](direct-nullable-component-counts.md) have a separate admission, and nested nullable DTOs retain their existing nullable codec and whole-object count behavior.

Union variants, hierarchy/common/inherited/typed shape planners and raw/stringified projections do not acquire count permissions. Ignored extras and discarded counted fields still reject checks they cannot retain. Inline HTTP root `allOf` projection remains unsupported. Existing [referenced/nested primitive conjunctions](referenced-allof-bounds.md) can coexist with whole-object counts, but enum/default/collection constraint aggregation and broader schema intersections remain separate.

Typed nominal HTTP bodies/results and SSE data use the same count codec. A valid constructor value can fail outbound encoding after the request callback; that request must not reach the engine. Existing primitive preflight remains earlier. Incoming HTTP counts run before defaults/unknown loss using the plugin's Json. SSE counts data only, with its separately supplied Json; invalid known data or delegate failures close a still-open fragmented input while preserving caller-client reuse. Arbitrary requestBuilder overrides remain caller-owned.

Only effective bounds enter declaration reuse and owned shape-lock identity. Equal effective schemas can share declarations despite different source Pointers, while changed owned counts fail an existing lock. Manifest/name-lock formats, first insertion position, winning fields, required union, nullable approximation and `W_ALL_OF_APPROXIMATION` remain unchanged.

**Migration:** eligible referenced/nested schemas that previously failed now generate when validation is enabled; invalid JSON input/output fails serialization. Default-off rejection and historical accepted output remain unchanged. F04/F06/F07/V05/M12 gain this bounded profile; general V06, nullable graphs and broader validation remain open. Limits remain source depth 48, expansion work 16,384, checked collection edges 16 and JVM serialized slots 240. Required non-null Long/Double use two slots, nullable values use one. Executed Kotlin/JVM evidence does not establish Android/KMP or other-host runtime support.

See the [fixture](../tests/fixtures/http/referenced-allof-counts.json), [core contracts](../core/referenced_allof_counts_test.go), [source/role guards](../core/referenced_allof_counts_guards_test.go), [resource boundaries](../core/referenced_allof_counts_resources_test.go), [reuse/lock checks](../core/referenced_allof_counts_optimization_test.go), [native CLI preservation](../internal/cli/referenced_allof_counts_process_test.go) and [Kotlin contracts](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedAllOfCountsContracts.kt). Run `bash scripts/test.sh feature referenced-allof-counts` for fresh Data/Plain compilation and four Json configurations per style, including defaults/unknown loss, const/Patch/null, non-JSON rejection, actual delegate parity and HTTP/live SSE behavior.
