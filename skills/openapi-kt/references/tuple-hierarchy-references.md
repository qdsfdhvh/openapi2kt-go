# Hierarchy references in positional arrays

OpenAPI 3.1 positional arrays accept references through existing pure named aliases to an actual retained hierarchy root, intermediate or leaf. This extends D01/D02/D03/D06 and F08/M03/M07/Q05. The named hierarchy contract still owns declarations, mapping/inferred/compound dispatch, inheritance, typed tag vocabularies and Unknown behavior. Tuple syntax does not grant general inline discriminator or internal hierarchy ownership.

Every original prefix and schema tail validates before projection. Uniform sources require the same original terminal and effective nullable intent; roots, intermediates and leaves retain distinct nominal identities. Different source/null views use the existing mixed `List<JsonElement>` projection. An explicit schema tail selects the actual element codec after all prefixes validate. Boolean tails retain List approximation: positions, exact length and false-tail maximum length are not enforced.

Actual canonical classifiers remain shared across root/parent/leaf references, operation JSON bodies/responses and SSE. Physical hierarchy classes and direct serializers remain non-null. Intrinsic/alias/consumer nullable references use Kotlin nullable adapters; JSON null becomes Kotlin null. There is no hierarchy Null subtype. Unknown remains a distinct non-null carrier at its selected level, and invalid known payloads still fail.

## Own defaults

Existing ordinary field, component-root and independent collection-alias outer annotations may use the selected hierarchy codec. Validation follows actual tag/conjunction rules through each selected level to a known concrete leaf. Closed inner levels use their actual subtype SerialName and the standard `type` discriminator. Unknown does not authorize a default.

The original selected leaf's declared/required fields and supported member literals validate through the shared object walker. Inherited/injected typed discriminator fields retain their actual vocabulary and body tags; direct leaf annotations may omit generated body constants, while supplied values must agree with the actual typed body. Nearest nullable intent controls null. Targets, intermediate aliases and child annotations are not inherited. Required and Patch annotations validate before initializer suppression; required arguments and `Patch.Absent` remain.

Root versus inner mapping routes are independent. A concrete leaf valid through its own serializer need not be reachable through an ancestor's mapping. Defaults must select a known leaf through the actual consuming route; no guessed classifier or expanded schema tree is used.

## Provenance and bounds

Retained proof covers original paths/nodes/catalog/alias hops, actual hierarchy/nominal origin, parent/child declarations, leaf identity and source, tag vocabularies/bodies, final consuming type/backing addresses and cached source work. Discarded hierarchy prefixes still require the original complete hierarchy validation. Same-byte source or declaration copies do not gain ownership. Literal const/default/enum payloads remain data-only.

A hierarchy classifier is a nominal recursion barrier. Erased consuming reference/alias hops remain in collection alias height caches; original hierarchy planning independently checks every dispatch source graph. Existing source48, cumulative16384-step/8MiB capture, branch/member240, default16384-node/32KiB, checked16 and JVM240 limits remain. Cycles, invalid required/default sources and over-budget work fail without partial artifacts or replacing existing output. Generation remains deterministic and offline.

## Verification and migration

`bash scripts/test.sh feature tuple-hierarchy-references` builds fresh Data/Plain × validation profiles with baseline/combined optimizations, nonempty naming locks and an independent240-field DTO. Runtime contracts exercise four Json settings, actual root/intermediate/leaf constructors, strict default factories, nullable/direct serializers, compound and closed-inner dispatch, Unknown at multiple levels, Patch, direct/shared HTTP and fragmented live SSE failure/closure/reuse. The fixture is part of the HTTP registry. Individual limits and JVM results do not prove simultaneous maxima or other platforms.

Previously rejected eligible tuple inputs gain canonical hierarchy APIs. **Intentional wire fix:** a content-dispatched ancestor with a closed sealed descendant now encodes through that actual child serializer. Previously, encoding a closed leaf through the ancestor omitted the inner class discriminator, so the ancestor could not decode its own output. The encoded object now retains the configured inner discriminator and actual subtype SerialName (for example, `type: "closedleft"`). Consumers comparing exact JSON or persisting this previously incomplete payload must accept the extra discriminator; regenerate related hierarchy files together. Constructor/classifier APIs, mapping dispatch and persisted output/name-lock formats are unchanged. General recursive/inline/multiple-parent hierarchy graphs, broader default roles, platforms and final candidate acceptance remain separate.
