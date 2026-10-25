# Actual internal ordinary DTOs in allOf

AllOf operands can bind a direct schema-internal Pointer, finite internal ref
chain or existing root bridge to an actual ordinary constructor DTO retained
under a direct ordinary component owner. Supported original lexical paths use
the existing property/List-item/typed-Map-value model ownership profile. The
operand must have ordinary constructor fields and a unique real original owner;
source syntax, a borrowed copy or a validation scratch declaration cannot supply
that ownership.

```json
{"Owner":{"properties":{"payload":{"properties":{
 "id":{"type":"string"}},"required":["id"]}}},
 "Bridge":{"$ref":"#/components/schemas/Owner/properties/payload"},
 "Consumer":{"allOf":[{"$ref":"#/components/schemas/Bridge"},
                         {"properties":{"extra":{"type":"boolean"}}}]}}
```

The planner carries the original resolved schema node and Pointer. Named
operands retain their real catalog entry; internal operands keep no component
identity. Fields borrow their actual original source/default/check metadata,
required order and depths. The resulting Consumer follows existing later-field
allOf overrides and `W_ALL_OF_APPROXIMATION`, with its own constructor and nested
copies. Nominal property refs continue to use the original canonical owner;
self/mutual recursion stops there. Bridge constructor aliases and serializers
remain bound to the original owner. Target/member nearest nullable views and
allOf null-set AND follow the existing [null-intent](nullable-intersections.md)
and [original-ref discovery](nullable-bridge-intersections.md) contracts.

Existing ordinary DTO admission records original catalog/node snapshots and
checks final declarations for exactly one retained constructor at the original
lexical Pointer/name. Removing the original owner, substituting copied source or
catalog nodes, or retaining only a borrowed consumer fails with
`E_REF_UNSUPPORTED`. No declaration name is predicted from the ref string.
Source/model/Patch/default and enum openness remain those of the shared planner;
this is field borrowing, not a general intersection validator.

The new internal shape role captures original semantic nodes through the existing
source traversal. A separate document-wide capture budget, shared by analyzer
contexts and discarded uses, admits at most16,384 node/edge units and8MiB of
original snapshots. Repeated identical targets reuse their capture; a different
overlapping target is charged again, including its byte snapshot. Descriptive
metadata remains opaque during semantic traversal. Exceeding this capture budget
reports `E_MODEL_LIMIT`. Existing ordinary composition work16,384, source-depth48,
checked-depth16, JVMslots240, literal/default and input caps remain independent
and unchanged. Existing tuple/non-tuple source roles keep their prior budgets.

This profile excludes composed/union/hierarchy lexical owners, composed targets,
root/default annotations on the operand itself, whole-object counts, raw/empty,
const-only and dictionary-only targets, scratch additionalProperties owners,
ambiguous/discarded sources and new oneOf/hierarchy/SSE-envelope shape roles.
Defaults on the target's ordinary fields keep their existing grammar; this does
not grant inheritance of the target's own root default. Other graph roles and
platforms require independent evidence.

`bash scripts/test.sh feature internal-dto-intersections` generates fresh
Data/Plain×validation projects, each with nullable/non-null baseline and combined
dedup/hoist/merge/locked models, plus an independent239+1 constructor probe.
The [non-null](../tests/fixtures/http/internal-dto-intersections.json) and
[nullable](../tests/fixtures/http/nullable-internal-dto-intersections.json)
fixtures and consumers test canonical constructor/alias codecs, copied nested
enum/DTO defaults, nominal recursion to the original owner, four Json settings,
required/Patch, HTTP null/data and fragmented SSE null/data/error/closure/reuse.
Core/source tests retain three exact former refusals as positives, keep excluded
owners rejected, exercise escaped paths and combined width, verify original
AST/catalog/node/field identity and missing-owner rejection, and test exact
capture work/bytes, copied contexts, repeated and overlapping targets. Native
checks compare core bytes/readonly/no-op and preserve old output with original
failure Pointers. Bounded fuzz varies finite original chains, nullable intent,
validation and direct/bridge use while checking deterministic output.

**Migration:** formerly rejected eligible internal DTO allOf operands now
generate with existing ordinary DTO API/wire/approximation rules. Existing named
and bridge inputs, canonical owners, output manifests and name-lock formats
retain their contracts. General ambiguous/composed/recursive shape graphs and
the wider D01/D02 roadmap remain open.

[Unique composed-owned nested DTO intersections](composed-dto-intersections.md) extend D01/D02/F04/F07 to actual ordinary property/collection DTOs under allOf/union lexical owners, using original source carriers and existing deferred unique-owner proof. Variant roots/common fields/hierarchy/ambiguous/discarded/scratch/count/root-default roles stay separate. Canonical recursion, nullable/Patch/default/check/override/warning contracts and shared16384work/8MiB capture/source/checked/JVM240 limits remain; cache identity rejects foreign Pointers/catalogs. Existing API/wire/locks remain stable; wider graph/platform completion stays open.

[Named hierarchy leaf nested DTO intersections](hierarchy-dto-intersections.md) extend D01/D02/F04/F07 only to actual ordinary nested DTOs directly owned by named plain-source hierarchy leaves. Original catalog/Pointer and unique retained owner/capture proof remain; parent tags/common fields do not become nested payload fields or Consumer slots. Nullable/Patch/default/check/recursion/warnings, shared16384work/8MiB capture and source/checked/JVM240 caps hold. Root/common/inline/composed-leaf/count/root-default/ambiguous roles stay separate; existing dispatch/API/wire/state remain stable, and broader graph/platform completion stays open.

[Named composed hierarchy-leaf nested DTO intersections](composed-leaf-intersections.md) extend D01/D02/F04/F07 to actual ordinary property/collection DTOs under named ordinary allOf leaf sources. Original catalog/Pointer and unique retained owner/capture proof remain; parent tags/common fields do not become payload fields or Consumer slots. Nullable/Patch/default/check/recursion/warnings and shared16384work/8MiB/source48/checked16/JVM240 limits hold. Root/common/inline/ambiguous/discarded/count/root-default roles stay separate; existing dispatch/API/wire/locks remain stable. Broader graph/platform completion remains open.

[Nested allOf DTO intersection targets](allof-target-intersections.md) extend D01/D02/F04/F07 to actual unique nested ordinary allOf constructors under direct ordinary object owners. Original composed source/alias snapshots and final retained owner proof remain; shared16384work/8MiB capture, source48/checked16/JVM240 and default/value caps hold. Nullable/Patch/default/check/recursion/override/warning/API/wire/locks keep their contracts. Composed/union/hierarchy lexical owners, products/count/root-default targets and broader graph/platform completion remain separate.

[Composed intersection targets across lexical owners](composed-target-owners.md) extend D01/D02/F04/F07 to actual unique nested ordinary allOf constructors under allOf/union owners and named plain/allOf hierarchy leaves. Original source/alias snapshots, final unique owner/capture proof and parenttag isolation remain; shared16384work/8MiB/source48/checked16/JVM240 and default/value limits hold. Nullable/Patch/default/check/recursion/override/warning/API/wire/locks keep their contracts. Ambiguous/discarded/common/inline/product/count/root-default roles and broader graph/platform completion remain separate.
