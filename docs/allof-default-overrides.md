# Plain primitive overrides in allOf defaults

With validation, existing ordinary allOf default walks accept complete homogeneous primitive contribution groups whose earlier occurrences have no bounds. The last occurrence supplies the actual constructor field, nullable view, own default and checks. String, UUID, Int, Long, Float and Double use the existing matching kind/format/nullability signature. Earlier annotations are validated against their own original schemas before override; a later bound does not retroactively constrain an earlier occurrence's own default.

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Record":{"allOf":[
    {"properties":{"value":{"type":"integer","default":0}}},
    {"properties":{"value":{"type":"integer","minimum":1,"maximum":3,"default":2}}}
  ],"default":{}}
}}}
```

The factory returns a Record whose value is 2. A supplied outer value 0 rejects against the winning minimum; an invalid earlier annotation still rejects at its original Pointer. Omitted members use the actual winning initializer. Required/Patch suppression follows validation of every own annotation, and alias factories remain independent without default inheritance.

This role is available wherever the existing ordinary allOf default walk is already admitted: component factories, ordinary field-own annotations, object members, finite List/typed Map values and their supported named/internal/reference paths. It adds no new type, reference or enclosing annotation permission. Unchecked repeated groups, Boolean/temporal/enum/native override/valueclass/ref/composed/heterogeneous contribution groups and earlier bound-bearing profiles retain separate contracts. Earlier bound-bearing groups continue to use their existing complete aggregation rules.

Literal-only planning proves the complete original contribution order, winner and discarded occurrence identities, then reuses original selected endpoint/factor nodes for annotation checks. It does not change model aggregation, emitted validation or helper hints. Actual constructors retain only their existing winning checks when earlier contributions have none. Input nodes, exact numeric/IEEE and UUID rules, original value Pointers and all source/depth/work/value/default/member/JVM budgets remain.

[Core and original role/mode controls](../core/allof_default_overrides_test.go), [original sources, complete groups and shared work](../internal/schema/allof_default_overrides_test.go), [native parity and failure preservation](../internal/cli/allof_default_overrides_process_test.go) and [bounded deterministic fuzz](../core/allof_default_overrides_fuzz_test.go) cover this role. `bash scripts/test.sh feature allof-default-overrides` runs validation-on Data/Plain projects with base and optimized locked output, factories with 240 fields, String/UUID/numeric/nullable/alias/default checks, four Json settings, Patch, HTTP and fragmented SSE. Validation-off refusals are checked separately; these projects do not claim unchecked default support.

**Migration:** eligible outer/default inputs previously rejected and now generate. Existing accepted APIs, constructor checks, wire behavior, output ownership and lock formats retain their contracts. General default/composition/ownership graphs, platforms and final candidate acceptance remain open.
