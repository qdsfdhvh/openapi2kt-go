# False-reference members in object root defaults

An ordinary named shared object component may declare its own root default using its retained [false-reference fields](false-reference-fields.md). The literal walk uses the actual constructor getters and original field/reference/nullability sources.

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Denied":false,
  "Holder":{
    "properties":{"value":{"$ref":"#/components/schemas/Denied","nullable":true}},
    "default":{"value":null}
  }
}}}
```

With a `Dto` suffix, call the existing companion factory `HolderDto.default()`. It decodes canonical JSON through the actual DTO serializer and returns fresh object state. A reference or alias to `Holder` does not acquire that factory or annotation.

The original logical field intent controls supplied values. A nullable false field admits JSON null and rejects every non-null value. A non-null false field admits no supplied value, including null, so it can appear in an object annotation only by being absent when optional. Its generated strict absence serializer still omits the carrier with every Json setting. No false schema is removed or projected to raw JSON.

Literal required keys remain mandatory even for nullable fields or fields with their own defaults. `{}` therefore fails when the source requires `value`; an annotation must supply `{"value":null}` when that value is admitted. A nullable object root may independently have a null default, after its complete member source has validated. Existing field-own annotations remain independent and keep their constructor required/optional rules.

Existing count opt-in and literal validation remain. Factory decoding uses the established default Json configuration. Re-encoding the resulting DTO with a caller's settings can change emitted key counts: `explicitNulls=false` may omit null keys, and the existing count serializer rejects output below `minProperties`. A valid factory annotation does not bypass that wire check.

Permission belongs only to the original named object's own annotation. The planner requires an exact original root/source match, a unique retained ordinary model/property, immutable getter/default/type/source proof and the actual rejecting root codec. It creates no temporary DTO to grant permission. Logical nullability is resolved before optional presence wrapping.

The actual model factory and canonical initializer are separately bound to the original default literal, including member/item/Get identity for JSON keys resembling schema metadata. Removing or replacing the factory, changing a literal node or substituting a getter cannot retain authority through equal text. Source depth48, cumulative false-source work16,384, default value16,384 nodes/32 KiB, checked depth16, JVM slots240 and enum limits remain.

[Direct consuming DTO field defaults](false-dto-field-defaults.md) have a separate profile with no root annotations in its base profile; [independent target root annotations](false-dto-independent-defaults.md) have separate actual factory authority. Nested/child defaults, reference wrappers, allOf/union/hierarchy/directional/internal/inline owners and broader object graphs keep their separate boundaries. This profile adds no direct false HTTP-root role. Existing ordinary DTO JSON HTTP consumers retain their codec and presence policies.

Previously rejected eligible object annotations now emit factories. Existing accepted Kotlin/wire, naming-lock, configuration and ownership formats need no migration. Data/Plain and current dedup/hoist/merge options preserve actual factory and nominal codec identity. General defaults/Boolean graphs, platforms and final plan acceptance remain open.

[Fixture](../tests/fixtures/http/false-object-default-members.json), [core/value/role/budget contracts](../core/false_object_default_members_test.go), [actual factory/source guards](../internal/schema/false_object_default_members_test.go), [native lifecycle](../internal/cli/false_object_default_members_process_test.go) and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FalseObjectDefaultMembersContracts.kt) cover the profile. `bash scripts/test.sh feature false-object-default-members` generates Data/Plain × validation off/on, baseline and optimized locked output, four Json settings and controlled JSON HTTP. Separate 240-field factories cover strict optional absence, nullable members, required arguments and enabled wire counts. These JVM probes do not establish Android/KMP or combined maximum-resource acceptance.
