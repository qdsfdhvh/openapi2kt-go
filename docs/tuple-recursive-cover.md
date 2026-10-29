# Recursive JSON covers in positional arrays

OpenAPI 3.1 positional members and schema tails may reference an existing named [complete recursive JSON-kind cover](recursive-json.md), including its accepted finite named/internal reference chains. Self-recursive and mutually recursive graphs retain their actual `JsonElement` codec. This extends D01/D03/D06 and F04/F08/F10/M03/M07/Q05; it does not turn a partial, constrained or inline union into a complete cover.

Each original source validates before selection. A uniform prefix retains the original cover and effective nullable view; different sources/views use the existing raw projection. An explicit schema tail selects its own codec after all prefixes validate. True/false tails retain their existing List approximation. Tuple positions, exact length and false-tail maximum length are not enforced; `W_TUPLE_APPROXIMATION` remains explicit. Every complete cover retains its original single `W_RECURSIVE_JSON_PROJECTION` warning and backing-codec numeric behavior.

Root aliases and collection APIs use the existing transparent backing, for example `List<JsonElement>` or `List<JsonElement?>`. Use ordinary lists and reified/backing serializers. A non-null `JsonElement` admits the `JsonNull` value. A nullable reference's JSON null decodes to Kotlin null; the nearest explicit alias/consumer intent still wins. Containers and elements retain independent nullability. HTTP JSON bodies/responses, request Patch and SSE use these same selected codecs.

## Outer defaults

Ordinary tuple fields, tuple component roots and eligible independent collection aliases may own outer annotations. A selected complete cover accepts every JSON value, including objects whose keys resemble schema keywords. The actual nullable element codec determines whether a null becomes `JsonNull` or Kotlin null. Factories and field helpers decode fresh outer collections through the retained codec. The generator does not promise arbitrary-precision or original-text number preservation beyond the caller's JsonElement codec.

Targets, intermediate references and positional elements acquire no new default role, and annotations are not inherited. Required/Patch annotations validate before initializer suppression; required arguments and `Patch.Absent` remain. Outer array kind, nullability, retained count checks and all existing value/byte limits apply. UniqueItems eligibility follows its original explicit items contract; a uniform prefix alone does not grant it.

## Source identity and limits

The tuple role borrows the original complete graph and its unique actual JSON export. It retains every original consuming/alias path, reachable cover/graph-alias source, canonical component, SCC height, nullable intent and final consuming element/backing address. Same-byte source replacements, a different scalar with the same Kotlin spelling and copied consuming type nodes do not acquire authority. Discarded prefixes retain graph validation and cached source work.

Incoming reference hops plus the complete graph's SCC suffix remain part of the 48-level source depth, including cached collection aliases. All reachable original graph sources share the tuple capture limit of 16,384 work steps and 8 MiB; repeated nodes reuse their immutable capture. Default 16,384-node/32 KiB, checked-edge16 and JVM240 limits remain. Over-budget or invalid input returns no partial artifacts and preserves existing output. Generation remains deterministic and offline.

## Verification and migration

`bash scripts/test.sh feature tuple-recursive-cover` builds fresh Data/Plain × validation consumers with baseline/combined optimizations, nonempty name locks and an independent240-field DTO. The runtime contract covers four Json settings, self/mutual covers, nullable/internal aliases, own defaults, backing-codec numeric parity, Patch, direct/shared HTTP and byte-fragmented live SSE closure/failure/reuse. The fixture and consumer also enter the complete HTTP gate. Source/depth/default boundaries and actual declaration/address mutations have Go tests; individual limits do not establish simultaneous maxima or other platforms.

Previously rejected eligible tuple references now generate through their existing JSON backing. Existing accepted APIs, wire and output/name-lock formats retain their contracts. General recursive schemas, constrained/inline covers, broader default roles, platform matrices and final candidate acceptance remain separate.
