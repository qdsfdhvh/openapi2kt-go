#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/http-smoke-XXXXXX")"
cp -R samples/kotlin-http-smoke "$workspace/project"
openapi_kt_build_cli "$workspace/openapi-kt"
"$workspace/openapi-kt" generate --input tests/fixtures/http/schema-collection-owned-dtos.json \
  --output "$workspace/project/build/generated-limits/collection-owned-dtos" --package-name org.example.collectiondtos --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/schema-internal-dtos.json \
  --output "$workspace/project/build/generated-limits/schema-internal-dtos" --package-name org.example.internaldtos --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-internal-dto-aliases.json \
  --output "$workspace/project/build/generated-root-internal" --package-name org.example.rootinternaldtos --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-internal-dto-aliases.json \
  --output "$workspace/project/build/generated-root-internal" --package-name org.example.rootinternaldtos --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/schema-internal-dto-alias-chains.json \
  --output "$workspace/project/build/generated-dto-alias-chains" --package-name org.example.dtoaliaschains --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/schema-internal-dto-alias-chains.json \
  --output "$workspace/project/build/generated-dto-alias-chains" --package-name org.example.dtoaliaschains --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-named-dto-bridges.json \
  --output "$workspace/project/build/generated-root-dto-bridges" --package-name org.example.rootdtobridges --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-named-dto-bridges.json \
  --output "$workspace/project/build/generated-root-dto-bridges" --package-name org.example.rootdtobridges --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/basic.json \
  --output "$workspace/project/build/generated" --package-name org.example.http --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/basic.json \
  --output "$workspace/project/build/generated" --package-name org.example.http --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/collections.json \
  --output "$workspace/project/build/generated-collections" --package-name org.example.collections
"$workspace/openapi-kt" generate --input tests/fixtures/http/bodies.json \
  --output "$workspace/project/build/generated-bodies" --package-name org.example.bodies --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/references.json \
  --output "$workspace/project/build/generated-references" --package-name org.example.references --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/basic.json \
  --output "$workspace/project/build/generated-options" --package-name org.example.options \
  --api-client-name ChosenClient --exclude-api WidgetsApi
"$workspace/openapi-kt" generate --input tests/fixtures/http/operation-models.json \
  --output "$workspace/project/build/generated-operation-models" --package-name org.example.operationtypes --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/operation-collisions.json \
  --output "$workspace/project/build/generated-operation-collisions" --package-name org.example.operationcollisions --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/operation-models.json \
  --output "$workspace/project/build/generated-operation-excluded" --package-name org.example.operationexcluded \
  --model-name-suffix Dto --exclude-api DefaultApi --api-client-name NoOperationsClient
"$workspace/openapi-kt" generate --input tests/fixtures/http/request-presence.json \
  --output "$workspace/project/build/generated-request-presence" --package-name org.example.presence --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/request-presence-collisions.json \
  --output "$workspace/project/build/generated-presence-collisions" --package-name org.example.presencecollisions --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/request-presence.json \
  --output "$workspace/project/build/generated-presence-excluded" --package-name org.example.presenceexcluded --exclude-api DefaultApi
"$workspace/openapi-kt" generate --input tests/fixtures/http/request-unions.json \
  --output "$workspace/project/build/generated-request-unions" --package-name org.example.requestunions --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/deep-object.json \
  --output "$workspace/project/build/generated-deep-object" --package-name org.example.deepquery --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/deep-bindings.json \
  --output "$workspace/project/build/generated-deep-bindings" --package-name org.example.deepbindings --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/errors.json \
  --output "$workspace/project/build/generated-errors" --package-name org.example.errors --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/projections.json \
  --output "$workspace/project/build/generated-projections" --package-name org.example.projections --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/response-kinds.json \
  --output "$workspace/project/build/generated-response-kinds" --package-name org.example.responsekinds --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/structural-values.json \
  --output "$workspace/project/build/generated-structural-values" --package-name org.example.structural --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/nullable-request-unions.json \
  --output "$workspace/project/build/generated-nullable-unions" --package-name org.example.nullableunions --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/json-kind-request-unions.json \
  --output "$workspace/project/build/generated-request-kinds" --package-name org.example.requestkinds --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/json-kind-request-unions.json \
  --output "$workspace/project/build/generated-request-kinds-excluded" --package-name org.example.requestkindsexcluded --exclude-api DefaultApi
"$workspace/openapi-kt" generate --input tests/fixtures/http/operation-scalar-unions.json \
  --output "$workspace/project/build/generated-operation-scalar-unions" --package-name org.example.scalarunions --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/empty-response-unions.json \
  --output "$workspace/project/build/generated-empty-response-unions" --package-name org.example.emptyresponses --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/request-field-unions.json \
  --output "$workspace/project/build/generated-request-field-unions" --package-name org.example.requestfields --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/parameter-values.json \
  --output "$workspace/project/build/generated-parameter-values" --package-name org.example.parametervalues --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/special-scalars.json \
  --output "$workspace/project/build/generated-special-scalars" --package-name org.example.specialscalars --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/special-scalar-values.json \
  --output "$workspace/project/build/generated-special-scalar-values" --package-name org.example.specialvalues --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/native-enum-values.json \
  --output "$workspace/project/build/generated-native-enum-values" --package-name org.example.nativeenums --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/sse.json \
  --output "$workspace/project/build/generated-sse" --package-name org.example.streaming --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/value-classes.json \
  --output "$workspace/project/build/generated-value-classes" --package-name org.example.valueclasses --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/value-constraints.json \
  --output "$workspace/project/build/generated-constraints" --package-name org.example.constraints --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/dto-constraints.json \
  --output "$workspace/project/build/generated-dto-constraints" --package-name org.example.dtochecks --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/patterns.json \
  --output "$workspace/project/build/generated-patterns" --package-name org.example.patterns --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/alias-constraints.json \
  --output "$workspace/project/build/generated-alias-constraints" --package-name org.example.aliaschecks --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/element-constraints.json \
  --output "$workspace/project/build/generated-element-constraints" --package-name org.example.elementchecks --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/numeric-constraints.json \
  --output "$workspace/project/build/generated-numeric-constraints" --package-name org.example.numericchecks --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/numeric-legacy-constraints.json \
  --output "$workspace/project/build/generated-numeric-legacy-constraints" --package-name org.example.numericlegacy --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/collection-counts.json \
  --output "$workspace/project/build/generated-collection-counts" --package-name org.example.countchecks --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/collection-unique.json \
  --output "$workspace/project/build/generated-collection-unique" --package-name org.example.uniquechecks --model-name-suffix Dto --validation
# Counted ordinary DTOs share their nominal serializer across serde/HTTP/SSE.
"$workspace/openapi-kt" generate --input tests/fixtures/http/dto-property-counts.json \
  --output "$workspace/project/build/generated-limits/dto-property-counts" --package-name org.example.dtocounts --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/dto-property-counts-encode-limit.json \
  --output "$workspace/project/build/generated-limits/dto-property-counts-encode-limit" --package-name org.example.dtocountlimit --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/allof-primitive-bounds-supported-envelopes.json \
  --output "$workspace/project/build/generated-limits/allof-primitive-bounds" --package-name org.example.aggregation --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/allof-dto-property-counts-qualified-envelopes.json \
  --output "$workspace/project/build/generated-limits/allof-dto-property-counts" --package-name org.example.allofcounts --model-name-suffix Dto --validation
# Borrowed terminal scalar refs use the same typed serde/HTTP/SSE consumers.
"$workspace/openapi-kt" generate --input tests/fixtures/http/schema-internal-scalar-pointers-qualified.json \
  --output "$workspace/project/build/generated-limits/schema-internal-scalar-pointers" --package-name org.example.pointers --model-name-suffix Dto --validation
# Transparent internal scalar chains preserve terminal checks and request presence.
"$workspace/openapi-kt" generate --input tests/fixtures/http/schema-internal-scalar-chains.json \
  --output "$workspace/project/build/generated-limits/schema-internal-scalar-chains" --package-name org.example.chains --model-name-suffix Dto --validation
# The unchanged historical array-component input now emits a plain native alias.
"$workspace/openapi-kt" generate --input tests/fixtures/p1/invalid/array-component.json \
  --output "$workspace/project/build/generated-limits/array-component-original" --package-name org.example.arrayoriginal
# A 46-edge root plus model-field/ref hops reaches the cumulative48 boundary.
python3 - "$workspace/alias-consumer-depth.json" <<'PYALIASDEPTH'
import json
import sys
alias = {'type': 'string'}
for edge in range(46):
    alias = {'type': 'array', 'items': alias}
doc = {'openapi': '3.0.3', 'components': {'schemas': {
    'ZAlias': alias,
    'Consumer': {'required': ['v'], 'properties': {'v': {'$ref': '#/components/schemas/ZAlias'}}},
}}}
with open(sys.argv[1], 'w') as output:
    json.dump(doc, output)
PYALIASDEPTH
"$workspace/openapi-kt" generate --input "$workspace/alias-consumer-depth.json" \
  --output "$workspace/project/build/generated-limits/alias-consumer-depth" --package-name org.example.aliasdepth --model-name-suffix Dto
# A normalized nullable union branch still contributes original schema work.
python3 - "$workspace/alias-union-depth.json" <<'PYUNIONDEPTH'
import json
import sys
alias = {'anyOf': [{'type': 'string'}, {'type': 'null'}]}
for edge in range(45):
    alias = {'type': 'array', 'items': alias}
doc = {'openapi': '3.0.3', 'components': {'schemas': {
    'ZAlias': alias,
    'Consumer': {'required': ['v'], 'properties': {'v': {'$ref': '#/components/schemas/ZAlias'}}},
}}}
with open(sys.argv[1], 'w') as output:
    json.dump(doc, output)
PYUNIONDEPTH
"$workspace/openapi-kt" generate --input "$workspace/alias-union-depth.json" \
  --output "$workspace/project/build/generated-limits/alias-union-depth" --package-name org.example.aliasuniondepth --model-name-suffix Dto
# Transparent aliases share JVM erasure; nominal DTO/valueclasses stay distinct.
"$workspace/openapi-kt" generate --input tests/fixtures/http/alias-method-erasure.json \
  --output "$workspace/project/build/generated-limits/alias-method-erasure" --package-name org.example.aliasmethods --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/alias-method-erasure.json \
  --output "$workspace/project/build/generated-limits/alias-method-erasure" --package-name org.example.aliasmethods --model-name-suffix Dto --validation --check
# Named array components reuse native collection aliases and top-level peers.
"$workspace/openapi-kt" generate --input tests/fixtures/http/array-components.json \
  --output "$workspace/project/build/generated-limits/array-components" --package-name org.example.arraycomponents --model-name-suffix Dto --validation
python3 - "$workspace/array-component-probes.json" <<'PYARRAYCOMPONENT'
import json
import sys

def ref():
    return {'$ref': '#/components/schemas/ZValues'}

field = ref()
for edge in range(15):
    if edge % 2 == 0:
        field = {'type': 'array', 'items': field, 'minItems': 1}
    else:
        field = {'type': 'object', 'additionalProperties': field, 'minProperties': 1}
schemas = {
    'ZValues': {'type': 'array', 'minItems': 1, 'maxItems': 3, 'uniqueItems': True,
                'items': {'type': ['string', 'null'], 'minLength': 2, 'maxLength': 3, 'pattern': '^[a-z]+$'}},
    'AWide': {'properties': {f'p{i}': ref() for i in range(240)}},
    'ADeep': {'properties': {'v': field}},
}
with open(sys.argv[1], 'w') as f:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': schemas}}, f, indent=2)
PYARRAYCOMPONENT
"$workspace/openapi-kt" generate --input "$workspace/array-component-probes.json" \
  --output "$workspace/project/build/generated-limits/array-component-probes" --package-name org.example.arraycomponentprobes --model-name-suffix Dto --validation
# Pure ordinary-object root aliases share canonical DTO identity and serializers.
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-object-aliases.json \
  --output "$workspace/project/build/generated-limits/root-object-aliases" --package-name org.example.rootaliases --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-object-aliases.json \
  --output "$workspace/project/build/generated-limits/root-object-aliases" --package-name org.example.rootaliases --model-name-suffix Dto --check
python3 - "$workspace/root-object-alias-probes.json" <<'PYROOTOBJECT'
import json
import sys
schemas = {
    'CheckedAlias': {'$ref': '#/components/schemas/ZRecord'},
    'ComposedAlias': {'$ref': '#/components/schemas/ZComposed'},
    'ZComposed': {'allOf': [
        {'properties': {'id': {'type': 'string', 'minLength': 1}}, 'required': ['id']},
        {'properties': {'flag': {'type': 'boolean'}}},
    ]},
    'ZRecord': {'properties': {'id': {'type': 'string', 'minLength': 1}}, 'required': ['id']},
    'AWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/CheckedAlias'} for i in range(240)}},
    'ADeep': {'properties': {'v': {'$ref': '#/components/schemas/A45'}}, 'required': ['v']},
}
for hop in range(46):
    target = 'ZRecord' if hop == 0 else f'A{hop-1:02d}'
    schemas[f'A{hop:02d}'] = {'$ref': f'#/components/schemas/{target}'}
with open(sys.argv[1], 'w') as output:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': schemas}}, output)
PYROOTOBJECT
"$workspace/openapi-kt" generate --input "$workspace/root-object-alias-probes.json" \
  --output "$workspace/project/build/generated-limits/root-object-alias-probes" --package-name org.example.rootaliasprobes --model-name-suffix Dto --validation
# Intrinsic nullable enum roots preserve physical vocabulary and inherited typed intent.
python3 - "$workspace/intrinsic-nullable-enum-probes.json" <<'PYINTRINSICENUM'
import json
import sys
probes = {
    'Alias': {'$ref': '#/components/schemas/ZStatus'},
    'ZStatus': {'nullable': True, 'enum': ['ready', 'done']},
    'AWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/Alias'} for i in range(240)}},
    'ANear': {'properties': {'v': {'$ref': '#/components/schemas/A45'}}, 'required': ['v']},
    'ANarrow': {'properties': {'v': {'$ref': '#/components/schemas/A45', 'nullable': False}}, 'required': ['v']},
}
for hop in range(46):
    previous = 'ZStatus' if hop == 0 else f'A{hop-1:02d}'
    probes[f'A{hop:02d}'] = {'$ref': f'#/components/schemas/{previous}'}
with open(sys.argv[1], 'w') as target:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': probes}}, target)
PYINTRINSICENUM
"$workspace/openapi-kt" generate --input tests/fixtures/http/intrinsic-nullable-enums.json \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-enums" --package-name org.example.intrinsicnullableenums --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/intrinsic-nullable-enums.json \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-enums" --package-name org.example.intrinsicnullableenums --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input "$workspace/intrinsic-nullable-enum-probes.json" \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-enum-probes" --package-name org.example.intrinsicnullableenumprobes --model-name-suffix Dto
# Intrinsic nullable primitive wrappers retain the canonical constructors and boxed slots.
python3 - "$workspace/intrinsic-nullable-value-class-probes.json" <<'PYINTRINSICVALUECLASS'
import json
import sys
probes = {
    'LNullable': {'$ref': '#/components/schemas/ZCount', 'nullable': True},
    'ZCount': {'type': 'integer', 'format': 'int64', 'nullable': True, 'x-kotlin-value-class': True, 'minimum': 1, 'maximum': 5},
    'ZName': {'type': 'string', 'nullable': True, 'x-kotlin-value-class': True, 'minLength': 1, 'maxLength': 8},
    'AWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/LNullable'} for i in range(240)}},
    'AUnboxed': {'properties': {f'p{i}': {'$ref': '#/components/schemas/LNullable', 'nullable': False} for i in range(120)}, 'required': [f'p{i}' for i in range(120)]},
    'ANear': {'properties': {'v': {'$ref': '#/components/schemas/A45'}}, 'required': ['v']},
    'ANarrow': {'properties': {'v': {'$ref': '#/components/schemas/A45', 'nullable': False}}, 'required': ['v']},
}
for hop in range(46):
    previous = 'ZName' if hop == 0 else f'A{hop-1:02d}'
    probes[f'A{hop:02d}'] = {'$ref': f'#/components/schemas/{previous}'}
paths = {}
for name, width, narrow, stream in [('boxed', 240, False, False), ('narrow', 120, True, False), ('boxedStream', 239, False, True), ('narrowStream', 119, True, True)]:
    params = [{'name': f'p{i}', 'in': 'query', 'required': True, 'schema': {'$ref': '#/components/schemas/LNullable', **({'nullable': False} if narrow else {})}} for i in range(width)]
    response = {'200': {'content': {'text/event-stream': {'schema': {'properties': {}}}}}} if stream else {'204': {}}
    paths['/'+name] = {'get': {'operationId': name, 'parameters': params, 'responses': response}}
with open(sys.argv[1], 'w') as target:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': probes}, 'paths': paths}, target)
PYINTRINSICVALUECLASS
"$workspace/openapi-kt" generate --input tests/fixtures/http/intrinsic-nullable-value-classes.json \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-value-classes" --package-name org.example.intrinsicnullablevalueclasses --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/intrinsic-nullable-value-classes.json \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-value-classes" --package-name org.example.intrinsicnullablevalueclasses --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input "$workspace/intrinsic-nullable-value-class-probes.json" \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-value-class-probes" --package-name org.example.intrinsicnullablevalueclassprobes --model-name-suffix Dto
# Intrinsic nullable temporal roots shares canonical Instant/LocalDate scalar codecs.
python3 - "$workspace/intrinsic-nullable-temporal-probes.json" <<'PYINTRINSICTEMPORAL'
import json
import sys
probes = {
    'A': {'$ref': '#/components/schemas/ZStamp', 'nullable': True},
    'ZStamp': {'type': 'string', 'format': 'date-time', 'nullable': True},
    'AWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/A'} for i in range(240)}},
    'ANear': {'properties': {'v': {'$ref': '#/components/schemas/A45'}}, 'required': ['v']},
    'ANarrow': {'properties': {'v': {'$ref': '#/components/schemas/A45', 'nullable': False}}, 'required': ['v']},
}
for hop in range(46):
    previous = 'ZStamp' if hop == 0 else f'A{hop-1:02d}'
    probes[f'A{hop:02d}'] = {'$ref': f'#/components/schemas/{previous}'}
with open(sys.argv[1], 'w') as target:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': probes}}, target)
PYINTRINSICTEMPORAL
"$workspace/openapi-kt" generate --input tests/fixtures/http/intrinsic-nullable-temporals.json \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-temporals" --package-name org.example.intrinsicnullabletemporals --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/intrinsic-nullable-temporals.json \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-temporals" --package-name org.example.intrinsicnullabletemporals --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input "$workspace/intrinsic-nullable-temporal-probes.json" \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-temporal-probes" --package-name org.example.intrinsicnullabletemporalprobes --model-name-suffix Dto
# Intrinsic nullable bigint roots export nullable public Long aliases without globalizing the field codec.
python3 - "$workspace/intrinsic-nullable-bigint-probes.json" <<'PYINTRINSICBIGINT'
import json
import sys
probes = {
    'A': {'$ref': '#/components/schemas/ZStamp'},
    'ZStamp': {'type': 'string', 'nullable': True, 'x-native-type': 'bigint'},
    'AWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/A'} for i in range(240)}, 'required': [f'p{i}' for i in range(240)]},
    'ANear': {'properties': {'v': {'$ref': '#/components/schemas/A45'}}, 'required': ['v']},
    'ANarrow': {'properties': {'v': {'$ref': '#/components/schemas/A45', 'nullable': False}}, 'required': ['v']},
}
for hop in range(46):
    previous = 'ZStamp' if hop == 0 else f'A{hop-1:02d}'
    probes[f'A{hop:02d}'] = {'$ref': f'#/components/schemas/{previous}'}
probes['ARequired'] = {'properties': {f'p{i}': {'$ref': '#/components/schemas/A', 'nullable': False} for i in range(120)}, 'required': [f'p{i}' for i in range(120)]}
paths = {}
for name,limit,narrow,stream in [('Boxed',240,False,False),('Narrow',120,True,False),('EventsBoxed',239,False,True),('EventsNarrow',119,True,True)]:
    params = [{'name': f'p{i}', 'in': 'query', 'required': True, 'schema': {'$ref': '#/components/schemas/A', **({'nullable': False} if narrow else {})}} for i in range(limit)]
    responses = {'200': {'content': {'text/event-stream': {'schema': {'properties': {}}}}}} if stream else {'204': {}}
    paths['/'+name] = {'get': {'operationId': name, 'parameters': params, 'responses': responses}}

with open(sys.argv[1], 'w') as target:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': probes}, 'paths': paths}, target)
PYINTRINSICBIGINT
"$workspace/openapi-kt" generate --input tests/fixtures/http/intrinsic-nullable-bigints.json \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-bigints" --package-name org.example.intrinsicnullablebigints --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/intrinsic-nullable-bigints.json \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-bigints" --package-name org.example.intrinsicnullablebigints --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input "$workspace/intrinsic-nullable-bigint-probes.json" \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-bigint-probes" --package-name org.example.intrinsicnullablebigintprobes --model-name-suffix Dto
# Nullable bigint root intent retains field string codecs versus numeric root/list/map codecs.
python3 - "$workspace/root-nullable-bigint-probes.json" <<'PYNULLABLEBIGINT'
import json
import sys
probes = {
    'A': {'$ref': '#/components/schemas/ZStamp', 'nullable': True},
    'ZStamp': {'type': 'string', 'x-native-type': 'bigint'},
    'AWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/A'} for i in range(240)}, 'required': [f'p{i}' for i in range(240)]},
    'ANear': {'properties': {'v': {'$ref': '#/components/schemas/A45'}}, 'required': ['v']},
    'ANarrow': {'properties': {'v': {'$ref': '#/components/schemas/A45', 'nullable': False}}, 'required': ['v']},
}
for hop in range(46):
    previous = 'ZStamp' if hop == 0 else f'A{hop-1:02d}'
    probes[f'A{hop:02d}'] = {'$ref': f'#/components/schemas/{previous}'}
probes['A00']['nullable'] = True
probes['ARequired'] = {'properties': {f'p{i}': {'$ref': '#/components/schemas/A', 'nullable': False} for i in range(120)}, 'required': [f'p{i}' for i in range(120)]}
paths = {}
for name,limit,narrow,stream in [('Boxed',240,False,False),('Narrow',120,True,False),('EventsBoxed',239,False,True),('EventsNarrow',119,True,True)]:
    params = [{'name': f'p{i}', 'in': 'query', 'required': True, 'schema': {'$ref': '#/components/schemas/A', **({'nullable': False} if narrow else {})}} for i in range(limit)]
    responses = {'200': {'content': {'text/event-stream': {'schema': {'properties': {}}}}}} if stream else {'204': {}}
    paths['/'+name] = {'get': {'operationId': name, 'parameters': params, 'responses': responses}}

with open(sys.argv[1], 'w') as target:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': probes}, 'paths': paths}, target)
PYNULLABLEBIGINT
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-nullable-bigint-aliases.json \
  --output "$workspace/project/build/generated-limits/root-nullable-bigint-aliases" --package-name org.example.rootnullablebigintaliases --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-nullable-bigint-aliases.json \
  --output "$workspace/project/build/generated-limits/root-nullable-bigint-aliases" --package-name org.example.rootnullablebigintaliases --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input "$workspace/root-nullable-bigint-probes.json" \
  --output "$workspace/project/build/generated-limits/root-nullable-bigint-probes" --package-name org.example.rootnullablebigintprobes --model-name-suffix Dto
# Raw nullable aliases retain JsonElement backing, without nominal barriers.
python3 - "$workspace/root-nullable-raw-probes.json" <<'PYNULLABLERAW'
import json
import sys
probes = {
    'A': {'$ref': '#/components/schemas/ZRaw', 'nullable': True},
    'ZRaw': {'type': 'object'},
    'AWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/A'} for i in range(240)}, 'required': [f'p{i}' for i in range(240)]},
    'ANear': {'properties': {'v': {'$ref': '#/components/schemas/A45'}}, 'required': ['v']},
    'ANarrow': {'properties': {'v': {'$ref': '#/components/schemas/A45', 'nullable': False}}, 'required': ['v']},
}
for hop in range(46):
    previous = 'ZRaw' if hop == 0 else f'A{hop-1:02d}'
    probes[f'A{hop:02d}'] = {'$ref': f'#/components/schemas/{previous}'}
probes['A00']['nullable'] = True
paths = {}
for name,limit,stream in [('Boxed',240,False),('Events',239,True)]:
    params = [{'name': f'p{i}', 'in': 'query', 'required': True, 'schema': {'$ref': '#/components/schemas/A'}} for i in range(limit)]
    responses = {'200': {'content': {'text/event-stream': {'schema': {'properties': {}}}}}} if stream else {'204': {}}
    paths['/'+name] = {'get': {'operationId': name, 'parameters': params, 'responses': responses}}
with open(sys.argv[1], 'w') as target:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': probes}, 'paths': paths}, target)
PYNULLABLERAW
"$workspace/openapi-kt" generate --input "$workspace/root-nullable-raw-probes.json" \
  --output "$workspace/project/build/generated-limits/root-nullable-raw-probes" --package-name org.example.rootnullablerawprobes --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-nullable-raw-aliases.json \
  --output "$workspace/project/build/generated-limits/root-nullable-raw" --package-name org.example.rootnullableraw --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-nullable-raw-aliases.json \
  --output "$workspace/project/build/generated-limits/root-nullable-raw" --package-name org.example.rootnullableraw --model-name-suffix Dto --validation --check
# Intrinsic raw nullable roots reuse transparent JsonElement backing and source work.
python3 - "$workspace/intrinsic-nullable-raw-probes.json" <<'PYINTRINSICRAW'
import json
import sys
probes = {
    'A': {'$ref': '#/components/schemas/ZRaw'},
    'ZRaw': {'type': 'object', 'nullable': True},
    'AWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/A'} for i in range(240)}, 'required': [f'p{i}' for i in range(240)]},
    'ANear': {'properties': {'v': {'$ref': '#/components/schemas/A45'}}, 'required': ['v']},
    'ANarrow': {'properties': {'v': {'$ref': '#/components/schemas/A45', 'nullable': False}}, 'required': ['v']},
}
for hop in range(46):
    previous = 'ZRaw' if hop == 0 else f'A{hop-1:02d}'
    probes[f'A{hop:02d}'] = {'$ref': f'#/components/schemas/{previous}'}
paths = {}
for name,limit,stream in [('Boxed',240,False),('Events',239,True)]:
    params = [{'name': f'p{i}', 'in': 'query', 'required': True, 'schema': {'$ref': '#/components/schemas/A'}} for i in range(limit)]
    responses = {'200': {'content': {'text/event-stream': {'schema': {'properties': {}}}}}} if stream else {'204': {}}
    paths['/'+name] = {'get': {'operationId': name, 'parameters': params, 'responses': responses}}
with open(sys.argv[1], 'w') as target:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': probes}, 'paths': paths}, target)
PYINTRINSICRAW
"$workspace/openapi-kt" generate --input "$workspace/intrinsic-nullable-raw-probes.json" \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-raw-probes" --package-name org.example.intrinsicnullablerawprobes --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/intrinsic-nullable-raw.json \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-raw" --package-name org.example.intrinsicnullableraw --model-name-suffix Dto --api-client-name IntrinsicNullableRawApiClient --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/intrinsic-nullable-raw.json \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-raw" --package-name org.example.intrinsicnullableraw --model-name-suffix Dto --api-client-name IntrinsicNullableRawApiClient --validation --check
# Intrinsic builtin primitives retain native nullable codecs and opt-in checks.
python3 - "$workspace/intrinsic-nullable-primitive-probes.json" <<'PYINTRINSICPRIMITIVE'
import json, sys
ref = lambda n, **kw: {'$ref': '#/components/schemas/'+n, **kw}
probes = {'ZLong': {'type':['integer','null'], 'format':'int64'},
          'ZString': {'type':'string', 'nullable':True},
          'ZBoundString': {'type':['string','null'], 'minLength':2, 'maxLength':4},
          'ZBoundLong': {'type':['integer','null'], 'format':'int64', 'minimum':1, 'maximum':3},
          'A': ref('ZLong'), 'False': ref('ZLong', nullable=False),
          'Wide': {'properties':{f'p{i}':ref('A') for i in range(240)}, 'required':[f'p{i}' for i in range(240)]},
          'Narrow': {'properties':{f'p{i}':ref('False') for i in range(120)}, 'required':[f'p{i}' for i in range(120)]},
          'Near': {'properties':{'v':ref('A45')},'required':['v']},
          'BoundHolder': {'properties':{'s':ref('ZBoundString'),'l':ref('ZBoundLong'),'narrow':ref('ZBoundLong',nullable=False),'items':{'type':'array','items':ref('ZBoundString')},'values':{'type':'object','additionalProperties':ref('ZBoundLong')}},'required':['s','l','narrow','items','values']}}
for hop in range(46): probes[f'A{hop:02d}']=ref('ZString' if hop==0 else f'A{hop-1:02d}')
checked=ref('ZBoundString')
for _ in range(16): checked={'type':'array','items':checked}
probes['Checked']={'properties':{'v':checked},'required':['v']}
paths={}
for name,limit,stream,narrow in [('Boxed',240,False,False),('Longs',120,False,True),('Events',239,True,False),('LongEvents',119,True,True)]:
    params=[{'name':f'p{i}','in':'query','required':True,'schema':ref('False' if narrow else 'A')} for i in range(limit)]
    response={'200':{'content':{'text/event-stream':{'schema':{'properties':{}}}}}} if stream else {'204':{}}
    paths['/'+name]={'get':{'operationId':name,'parameters':params,'responses':response}}
for name in ['String','Long']:
    content={'application/json':{'schema':ref('ZBound'+name)}}
    paths['/bound'+name]={'post':{'operationId':'bound'+name,'requestBody':{'required':True,'content':content},'responses':{'200':{'content':content}}}}
request={'type':'object','properties':{'s':ref('ZBoundString'),'l':ref('ZBoundLong'),'required':ref('ZBoundLong')},'required':['required']}
paths['/patch']={'post':{'operationId':'patch','requestBody':{'required':True,'content':{'application/json':{'schema':request}}},'responses':{'204':{}}}}
event={'type':'object','properties':{'event':{'type':'string','const':'bounded'},'data':ref('ZBoundLong')},'required':['data']}
paths['/bound-events']={'get':{'operationId':'watchBounded','responses':{'200':{'content':{'text/event-stream':{'schema':event}}}}}}
with open(sys.argv[1],'w') as out: json.dump({'openapi':'3.1.0','components':{'schemas':probes},'paths':paths},out)
PYINTRINSICPRIMITIVE
"$workspace/openapi-kt" generate --input "$workspace/intrinsic-nullable-primitive-probes.json" \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-primitive-probes" --package-name org.example.intrinsicprimitiveprobes --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/intrinsic-nullable-primitives.json \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-primitives" --package-name org.example.intrinsicprimitives --model-name-suffix Dto --api-client-name IntrinsicPrimitiveApiClient --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/intrinsic-nullable-primitives.json \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-primitives" --package-name org.example.intrinsicprimitives --model-name-suffix Dto --api-client-name IntrinsicPrimitiveApiClient --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/nullable-int-responses.json \
  --output "$workspace/project/build/generated-limits/nullable-int-responses" --package-name org.example.nullableint
python3 - "$workspace/intrinsic-nullable-primitive-probes.json" "$workspace/root-nullable-primitive-probes.json" <<'PYROOTNULLPRIMITIVE'
import json, sys
with open(sys.argv[1]) as f: doc = json.load(f)
probes = doc['components']['schemas']
for name in ['ZLong', 'ZString', 'ZBoundString', 'ZBoundLong']:
    root = probes[name]
    root.pop('nullable', None)
    if isinstance(root.get('type'), list): root['type'] = next(v for v in root['type'] if v != 'null')
    probes['Base'+name] = root
    probes[name] = {'$ref':'#/components/schemas/Base'+name, 'nullable':True}
# The added own-nullable root alias is one real source edge: keep total 48,
# not the old 46 consumer hops plus an extra root alias.
probes.pop('A45')
probes['Near']['properties']['v']['$ref'] = '#/components/schemas/A44'
probes['LegacyIntermediate'] = {'$ref':'#/components/schemas/BaseZBoundString'}
probes['LegacyNullable'] = {'$ref':'#/components/schemas/LegacyIntermediate', 'nullable':True}
probes['LegacyHolder'] = {'properties':{'v':{'$ref':'#/components/schemas/LegacyNullable'}, 'narrow':{'$ref':'#/components/schemas/LegacyNullable', 'nullable':False}}, 'required':['v','narrow']}
with open(sys.argv[2], 'w') as f: json.dump(doc, f)
PYROOTNULLPRIMITIVE
"$workspace/openapi-kt" generate --input "$workspace/root-nullable-primitive-probes.json" \
  --output "$workspace/project/build/generated-limits/root-nullable-primitive-probes" --package-name org.example.rootnullableprimitiveprobes --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-nullable-primitives.json \
  --output "$workspace/project/build/generated-limits/root-nullable-primitives" --package-name org.example.rootnullableprimitives --model-name-suffix Dto --api-client-name RootNullablePrimitiveApiClient --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-nullable-primitives.json \
  --output "$workspace/project/build/generated-limits/root-nullable-primitives" --package-name org.example.rootnullableprimitives --model-name-suffix Dto --api-client-name RootNullablePrimitiveApiClient --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/merge-models.json \
  --output "$workspace/project/build/generated-limits/merge-models" --package-name org.example.merge --model-name-suffix Dto --validation --merge-identical-models
"$workspace/openapi-kt" generate --input tests/fixtures/http/merge-probes.json \
  --output "$workspace/project/build/generated-limits/merge-probes" --package-name org.example.mergeprobes --model-name-suffix Dto --validation --merge-identical-models
"$workspace/openapi-kt" generate --input tests/fixtures/http/merge-models.json \
  --output "$workspace/project/build/generated-limits/merge-models" --package-name org.example.merge --model-name-suffix Dto --validation --merge-identical-models --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/primitive-tuples.json \
  --output "$workspace/project/build/generated-limits/primitive-tuples" --package-name org.example.tuples --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/primitive-tuples.json \
  --output "$workspace/project/build/generated-limits/primitive-tuples" --package-name org.example.tuples --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/boolean-values.json \
  --output "$workspace/project/build/generated-limits/boolean-values" --package-name org.example.booleanvalues --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/boolean-values.json \
  --output "$workspace/project/build/generated-limits/boolean-values" --package-name org.example.booleanvalues --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/boolean-roots.json \
  --output "$workspace/project/build/generated-limits/boolean-roots" --package-name org.example.booleanroots --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/boolean-roots.json \
  --output "$workspace/project/build/generated-limits/boolean-roots" --package-name org.example.booleanroots --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/boolean-root-nullable.json \
  --output "$workspace/project/build/generated-limits/boolean-root-nullable" --package-name org.example.booleanrootnullable --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/boolean-root-nullable.json \
  --output "$workspace/project/build/generated-limits/boolean-root-nullable" --package-name org.example.booleanrootnullable --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/boolean-http.json \
  --output "$workspace/project/build/generated-limits/boolean-http" --package-name org.example.booleanhttp --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/boolean-http.json \
  --output "$workspace/project/build/generated-limits/boolean-http" --package-name org.example.booleanhttp --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/recursive-json.json \
  --output "$workspace/project/build/generated-limits/recursive-json" --package-name org.example.recursivejson --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/recursive-json.json \
  --output "$workspace/project/build/generated-limits/recursive-json" --package-name org.example.recursivejson --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/union-allof.json \
  --output "$workspace/project/build/generated-limits/union-allof" --package-name org.example.unionallof --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/union-allof.json \
  --output "$workspace/project/build/generated-limits/union-allof" --package-name org.example.unionallof --model-name-suffix Dto --validation --check
# Additional ordinary classifiers may keep reserved utility names in a different package.
python3 - "$root/tests/fixtures/http/shared-runtime.json" "$workspace/runtime-scoped.json" <<'PYRUNTIMESCOPE'
import json, sys
with open(sys.argv[1]) as f: doc = json.load(f)
for name in ['Patch', 'PatchSerializer', 'LongAsStringSerializer']:
    doc['components']['schemas'][name] = {'properties': {'id': {'type': 'string'}}}
with open(sys.argv[2], 'w') as f: json.dump(doc, f)
PYRUNTIMESCOPE
"$workspace/openapi-kt" generate --input "$workspace/runtime-scoped.json" \
  --output "$workspace/project/build/generated-limits/runtime-scoped" --package-name org.example.runtimecollisions \
  --runtime-package org.example.scoped.runtime --validation
# Two named specs use one actual shared runtime source set in the same Kotlin module.
cat > "$workspace/shared-runtime.toml" <<TOML
config_version=1
[defaults]
input='$root/tests/fixtures/http/shared-runtime.json'
output_dir='$workspace/project/build/generated-limits/shared-runtime'
model_name_suffix='Dto'
validation=true
runtime_package='org.example.shared.runtime'
[[specs]]
name='alpha'
package_name='org.example.runtimealpha'
[[specs]]
name='beta'
package_name='org.example.runtimebeta'
TOML
"$workspace/openapi-kt" generate --config "$workspace/shared-runtime.toml"
"$workspace/openapi-kt" generate --config "$workspace/shared-runtime.toml" --check
"$workspace/openapi-kt" generate --config "$workspace/shared-runtime.toml" --spec alpha --check
# Opt-in plain consumers: independent data control retains copy/componentN.
"$workspace/openapi-kt" generate --input tests/fixtures/model-semantics/plain-style-baseline.json --output "$workspace/project/build/generated-limits/plain-data" --package-name org.example.plainprobe --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/model-semantics/plain-style-baseline.json --output "$workspace/project/build/generated-limits/plain-actual" --package-name org.example.actualplain --model-name-suffix Dto --model-style plain_class
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-union-aliases.json --output "$workspace/project/build/generated-limits/plain-union" --package-name org.example.plainunion --model-name-suffix Dto --validation --model-style plain_class
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-hierarchy-aliases.json --output "$workspace/project/build/generated-limits/plain-hierarchy" --package-name org.example.plainhierarchy --model-name-suffix Dto --validation --model-style plain_class
"$workspace/openapi-kt" generate --input tests/fixtures/http/shared-runtime.json --output "$workspace/project/build/generated-limits/plain-runtime" --package-name org.example.plainruntime --model-name-suffix Dto --validation --model-style plain_class
"$workspace/openapi-kt" generate --input tests/fixtures/model-semantics/plain-style-arrays.json --output "$workspace/project/build/generated-limits/plain-arrays-data" --package-name org.example.dataarrays --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/model-semantics/plain-style-arrays.json --output "$workspace/project/build/generated-limits/plain-arrays" --package-name org.example.plainarrays --model-name-suffix Dto --validation --model-style plain_class
"$workspace/openapi-kt" generate --input tests/fixtures/http/deduplicate-models.json --output "$workspace/project/build/generated-limits/deduplicate" --package-name org.example.dedup --model-name-suffix Dto --validation --deduplicate-models
"$workspace/openapi-kt" generate --input tests/fixtures/http/deduplicate-models.json --output "$workspace/project/build/generated-limits/deduplicate-plain" --package-name org.example.dedupplain --model-name-suffix Dto --validation --deduplicate-models --model-style plain_class
"$workspace/openapi-kt" generate --input tests/fixtures/http/operation-native-dependency.json --output "$workspace/project/build/generated-limits/operation-dependency" --package-name org.example.dependencies --model-name-suffix Dto --validation --deduplicate-models --hoist-shared-shapes --model-style plain_class
"$workspace/openapi-kt" generate --input tests/fixtures/http/deduplicate-models.json --output "$workspace/project/build/generated-limits/hoist" --package-name org.example.hoist --model-name-suffix Dto --validation --hoist-shared-shapes --deduplicate-models
"$workspace/openapi-kt" generate --input tests/fixtures/http/deduplicate-models.json --output "$workspace/project/build/generated-limits/hoist-plain" --package-name org.example.hoistplain --model-name-suffix Dto --validation --hoist-shared-shapes --deduplicate-models --model-style plain_class
"$workspace/openapi-kt" generate --input tests/fixtures/http/deduplicate-models.json --output "$workspace/project/build/generated-limits/hoist-qualified" --package-name org.example.qualifiedhoist --model-name-suffix Dto --validation --hoist-shared-shapes --qualify-hoisted-names --deduplicate-models
"$workspace/openapi-kt" generate --input tests/fixtures/http/deduplicate-models.json --output "$workspace/project/build/generated-limits/hoist-qualified-plain" --package-name org.example.qualifiedhoistplain --model-name-suffix Dto --validation --hoist-shared-shapes --qualify-hoisted-names --deduplicate-models --model-style plain_class
"$workspace/openapi-kt" generate --input tests/fixtures/http/hoist-reservations.json --output "$workspace/project/build/generated-limits/hoist-reservations" --package-name org.example.hoistreservations --validation --hoist-shared-shapes --deduplicate-models
"$workspace/openapi-kt" generate --input tests/fixtures/http/hoist-reservations.json --output "$workspace/project/build/generated-limits/hoist-reservations-moved" --package-name org.example.hoistreservationsmoved --runtime-package org.example.hoistreservationssupport --validation --hoist-shared-shapes --deduplicate-models --model-style plain_class
"$workspace/openapi-kt" generate --input tests/fixtures/http/hoist-reservations.json --output "$workspace/project/build/generated-limits/hoist-reservations-qualified" --package-name org.example.hoistreservationsqualified --validation --hoist-shared-shapes --deduplicate-models --qualify-hoisted-names
# Independent widths/depths are not a Cartesian resource guarantee.
python3 - "$workspace/plain-limits.json" <<'PYPLAINLIMITS'
from pathlib import Path
import json,sys
schemas={}
for n,nullable in [(240,True),(120,False)]:
    props={f'f{i:03}':{'type':'integer','format':'int64','nullable':nullable} for i in range(n)}
    schemas[f'Wide{n}']={'type':'object','properties':props,'required':list(props)}
schemas['ZRecord']={'type':'object','properties':{'id':{'type':'string'}},'required':['id']}
for i in range(48):
    schemas[f'A{i:02}']={'$ref':'#/components/schemas/'+('ZRecord' if i==0 else f'A{i-1:02}')}
leaf={'type':'string','minLength':1}
for _ in range(16):
    leaf={'type':'array','items':leaf}
schemas['Deep']={'type':'object','properties':{'values':leaf},'required':['values']}
Path(sys.argv[1]).write_text(json.dumps({'openapi':'3.1.0','info':{'title':'Plain Independent Limits','version':'1'},'paths':{},'components':{'schemas':schemas}}))
PYPLAINLIMITS
"$workspace/openapi-kt" generate --input "$workspace/plain-limits.json" --output "$workspace/project/build/generated-limits/plain-caps" --package-name org.example.plainlimits --model-name-suffix Dto --validation --model-style plain_class
# Intrinsic caller-native roots export nullable provider aliases without codec copies.
python3 - "$workspace/root-nullable-caller-native-probes.json" "$workspace/intrinsic-nullable-caller-native-probes.json" <<'PYINTRINSICCALLERNATIVE'
import json
import sys
# The owned-alias probe is generated later; build the intrinsic profile directly.
probes = {
    'A': {'$ref': '#/components/schemas/ZMoney'},
    'ZMoney': {'x-native-type': 'org.example.scalarvalues.Money', 'nullable': True},
    'AWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/A'} for i in range(240)}, 'required': [f'p{i}' for i in range(240)]},
    'ANear': {'properties': {'v': {'$ref': '#/components/schemas/A45'}}, 'required': ['v']},
    'ANarrow': {'properties': {'v': {'$ref': '#/components/schemas/A45', 'nullable': False}}, 'required': ['v']},
    'LongMaybe': {'$ref': '#/components/schemas/ZLong'},
    'ZLong': {'x-native-type': 'kotlin.Long', 'type': ['integer','null']},
    'LongNarrow': {'$ref': '#/components/schemas/LongMaybe', 'nullable': False},
    'LongWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/LongNarrow'} for i in range(120)}, 'required': [f'p{i}' for i in range(120)]},
}
for hop in range(46):
    previous = 'ZMoney' if hop == 0 else f'A{hop-1:02d}'
    probes[f'A{hop:02d}'] = {'$ref': f'#/components/schemas/{previous}'}
paths = {}
for name,limit,stream,target in [('Boxed',240,False,'A'),('Events',239,True,'A'),('Longs',120,False,'LongNarrow'),('LongEvents',119,True,'LongNarrow')]:
    params = [{'name': f'p{i}', 'in': 'query', 'required': True, 'schema': {'$ref': '#/components/schemas/'+target}} for i in range(limit)]
    responses = {'200': {'content': {'text/event-stream': {'schema': {'properties': {}}}}}} if stream else {'204': {}}
    paths['/'+name] = {'get': {'operationId': name, 'parameters': params, 'responses': responses}}
with open(sys.argv[2], 'w') as target:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': probes}, 'paths': paths}, target)
PYINTRINSICCALLERNATIVE
"$workspace/openapi-kt" generate --input "$workspace/intrinsic-nullable-caller-native-probes.json" \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-caller-native-probes" --package-name org.example.intrinsicnullablecallerprobes --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/intrinsic-nullable-caller-natives.json \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-caller-native" --package-name org.example.intrinsicnullablecallernative --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/intrinsic-nullable-caller-natives.json \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-caller-native" --package-name org.example.intrinsicnullablecallernative --model-name-suffix Dto --validation --check
# Caller-provided scalar classifier/serializer ownership survives nullable root bindings.
python3 - "$workspace/root-nullable-caller-native-probes.json" <<'PYCALLERNATIVE'
import json
import sys
probes = {
    'A': {'$ref': '#/components/schemas/ZMoney', 'nullable': True},
    'ZMoney': {'x-native-type': 'org.example.scalarvalues.Money'},
    'AWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/A'} for i in range(240)}, 'required': [f'p{i}' for i in range(240)]},
    'ANear': {'properties': {'v': {'$ref': '#/components/schemas/A45'}}, 'required': ['v']},
    'ANarrow': {'properties': {'v': {'$ref': '#/components/schemas/A45', 'nullable': False}}, 'required': ['v']},
    'LongMaybe': {'$ref': '#/components/schemas/ZLong', 'nullable': True},
    'ZLong': {'x-native-type': 'kotlin.Long'},
    'LongNarrow': {'$ref': '#/components/schemas/LongMaybe', 'nullable': False},
    'LongWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/LongNarrow'} for i in range(120)}, 'required': [f'p{i}' for i in range(120)]},
}
for hop in range(46):
    previous = 'ZMoney' if hop == 0 else f'A{hop-1:02d}'
    probes[f'A{hop:02d}'] = {'$ref': f'#/components/schemas/{previous}'}
probes['A00']['nullable'] = True
paths = {}
for name,limit,stream,target in [('Boxed',240,False,'A'),('Events',239,True,'A'),('Longs',120,False,'LongNarrow'),('LongEvents',119,True,'LongNarrow')]:
    params = [{'name': f'p{i}', 'in': 'query', 'required': True, 'schema': {'$ref': '#/components/schemas/'+target}} for i in range(limit)]
    responses = {'200': {'content': {'text/event-stream': {'schema': {'properties': {}}}}}} if stream else {'204': {}}
    paths['/'+name] = {'get': {'operationId': name, 'parameters': params, 'responses': responses}}
with open(sys.argv[1], 'w') as target:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': probes}, 'paths': paths}, target)
PYCALLERNATIVE
"$workspace/openapi-kt" generate --input "$workspace/root-nullable-caller-native-probes.json" \
  --output "$workspace/project/build/generated-limits/root-nullable-caller-native-probes" --package-name org.example.rootnullablecallerprobes --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-nullable-caller-native-aliases.json \
  --output "$workspace/project/build/generated-limits/root-nullable-caller-native" --package-name org.example.rootnullablecallernative --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-nullable-caller-native-aliases.json \
  --output "$workspace/project/build/generated-limits/root-nullable-caller-native" --package-name org.example.rootnullablecallernative --model-name-suffix Dto --validation --check
# Canonical native JSON classifiers and builtin JSON plans share method erasure.
"$workspace/openapi-kt" generate --input tests/fixtures/http/native-json-erasure.json \
  --output "$workspace/project/build/generated-limits/native-json-erasure" --package-name org.example.nativejsonerasure --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/native-json-erasure.json \
  --output "$workspace/project/build/generated-limits/native-json-erasure" --package-name org.example.nativejsonerasure --model-name-suffix Dto --check
# Intrinsic nullable byte roots export nullable Long-free byte-array aliases; raw HTTP projection stays.
python3 - "$workspace/intrinsic-nullable-bytes-probes.json" <<'PYINTRINSICBYTES'
import json
import sys
probes = {
    'A': {'$ref': '#/components/schemas/ZStamp'},
    'ZStamp': {'type': 'string', 'nullable': True, 'format': 'byte'},
    'AWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/A'} for i in range(240)}},
    'ANear': {'properties': {'v': {'$ref': '#/components/schemas/A45'}}, 'required': ['v']},
    'ANarrow': {'properties': {'v': {'$ref': '#/components/schemas/A45', 'nullable': False}}, 'required': ['v']},
}
for hop in range(46):
    previous = 'ZStamp' if hop == 0 else f'A{hop-1:02d}'
    probes[f'A{hop:02d}'] = {'$ref': f'#/components/schemas/{previous}'}
probes['AWide']['required'] = [f'p{i}' for i in range(240)]
paths = {}
for name,limit,stream in [('Boxed',240,False),('Events',239,True)]:
    params = [{'name': f'p{i}', 'in': 'query', 'required': True, 'schema': {'$ref': '#/components/schemas/A'}} for i in range(limit)]
    responses = {'200': {'content': {'text/event-stream': {'schema': {'properties': {}}}}}} if stream else {'204': {}}
    paths['/'+name] = {'get': {'operationId': name, 'parameters': params, 'responses': responses}}

with open(sys.argv[1], 'w') as target:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': probes}, 'paths': paths}, target)
PYINTRINSICBYTES
"$workspace/openapi-kt" generate --input tests/fixtures/http/intrinsic-nullable-bytes.json \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-bytes" --package-name org.example.intrinsicnullablebytes --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/intrinsic-nullable-bytes.json \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-bytes" --package-name org.example.intrinsicnullablebytes --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input "$workspace/intrinsic-nullable-bytes-probes.json" \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-bytes-probes" --package-name org.example.intrinsicnullablebytesprobes --model-name-suffix Dto
# Nullable byte root aliases preserve model/SSE JSON arrays versus raw Ktor roots.
python3 - "$workspace/root-nullable-bytes-probes.json" <<'PYNULLABLEBYTES'
import json
import sys
probes = {
    'A': {'$ref': '#/components/schemas/ZStamp', 'nullable': True},
    'ZStamp': {'type': 'string', 'format': 'byte'},
    'AWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/A'} for i in range(240)}},
    'ANear': {'properties': {'v': {'$ref': '#/components/schemas/A45'}}, 'required': ['v']},
    'ANarrow': {'properties': {'v': {'$ref': '#/components/schemas/A45', 'nullable': False}}, 'required': ['v']},
}
for hop in range(46):
    previous = 'ZStamp' if hop == 0 else f'A{hop-1:02d}'
    probes[f'A{hop:02d}'] = {'$ref': f'#/components/schemas/{previous}'}
probes['A00']['nullable'] = True
probes['AWide']['required'] = [f'p{i}' for i in range(240)]
paths = {}
for name,limit,stream in [('Boxed',240,False),('Events',239,True)]:
    params = [{'name': f'p{i}', 'in': 'query', 'required': True, 'schema': {'$ref': '#/components/schemas/A'}} for i in range(limit)]
    responses = {'200': {'content': {'text/event-stream': {'schema': {'properties': {}}}}}} if stream else {'204': {}}
    paths['/'+name] = {'get': {'operationId': name, 'parameters': params, 'responses': responses}}

with open(sys.argv[1], 'w') as target:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': probes}, 'paths': paths}, target)
PYNULLABLEBYTES
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-nullable-bytes-aliases.json \
  --output "$workspace/project/build/generated-limits/root-nullable-bytes-aliases" --package-name org.example.rootnullablebytesaliases --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-nullable-bytes-aliases.json \
  --output "$workspace/project/build/generated-limits/root-nullable-bytes-aliases" --package-name org.example.rootnullablebytesaliases --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input "$workspace/root-nullable-bytes-probes.json" \
  --output "$workspace/project/build/generated-limits/root-nullable-bytes-probes" --package-name org.example.rootnullablebytesprobes --model-name-suffix Dto
# Nullable temporal root intent shares canonical Instant/LocalDate scalar codecs.
python3 - "$workspace/root-nullable-temporal-probes.json" <<'PYNULLABLETEMPORAL'
import json
import sys
probes = {
    'A': {'$ref': '#/components/schemas/ZStamp', 'nullable': True},
    'ZStamp': {'type': 'string', 'format': 'date-time'},
    'AWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/A'} for i in range(240)}},
    'ANear': {'properties': {'v': {'$ref': '#/components/schemas/A45'}}, 'required': ['v']},
    'ANarrow': {'properties': {'v': {'$ref': '#/components/schemas/A45', 'nullable': False}}, 'required': ['v']},
}
for hop in range(46):
    previous = 'ZStamp' if hop == 0 else f'A{hop-1:02d}'
    probes[f'A{hop:02d}'] = {'$ref': f'#/components/schemas/{previous}'}
probes['A00']['nullable'] = True
with open(sys.argv[1], 'w') as target:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': probes}}, target)
PYNULLABLETEMPORAL
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-nullable-temporal-aliases.json \
  --output "$workspace/project/build/generated-limits/root-nullable-temporal-aliases" --package-name org.example.rootnullabletemporalaliases --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-nullable-temporal-aliases.json \
  --output "$workspace/project/build/generated-limits/root-nullable-temporal-aliases" --package-name org.example.rootnullabletemporalaliases --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input "$workspace/root-nullable-temporal-probes.json" \
  --output "$workspace/project/build/generated-limits/root-nullable-temporal-probes" --package-name org.example.rootnullabletemporalprobes --model-name-suffix Dto
# Nullable primitive wrapper aliases retain the canonical constructors and boxed slots.
python3 - "$workspace/root-nullable-value-class-probes.json" <<'PYNULLABLEVALUECLASS'
import json
import sys
probes = {
    'LNullable': {'$ref': '#/components/schemas/ZCount', 'nullable': True},
    'ZCount': {'type': 'integer', 'format': 'int64', 'x-kotlin-value-class': True, 'minimum': 1, 'maximum': 5},
    'ZName': {'type': 'string', 'x-kotlin-value-class': True, 'minLength': 1, 'maxLength': 8},
    'AWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/LNullable'} for i in range(240)}},
    'AUnboxed': {'properties': {f'p{i}': {'$ref': '#/components/schemas/LNullable', 'nullable': False} for i in range(120)}, 'required': [f'p{i}' for i in range(120)]},
    'ANear': {'properties': {'v': {'$ref': '#/components/schemas/A45'}}, 'required': ['v']},
    'ANarrow': {'properties': {'v': {'$ref': '#/components/schemas/A45', 'nullable': False}}, 'required': ['v']},
}
for hop in range(46):
    previous = 'ZName' if hop == 0 else f'A{hop-1:02d}'
    probes[f'A{hop:02d}'] = {'$ref': f'#/components/schemas/{previous}'}
probes['A00']['nullable'] = True
paths = {}
for name, width, narrow, stream in [('boxed', 240, False, False), ('narrow', 120, True, False), ('boxedStream', 239, False, True), ('narrowStream', 119, True, True)]:
    params = [{'name': f'p{i}', 'in': 'query', 'required': True, 'schema': {'$ref': '#/components/schemas/LNullable', **({'nullable': False} if narrow else {})}} for i in range(width)]
    response = {'200': {'content': {'text/event-stream': {'schema': {'properties': {}}}}}} if stream else {'204': {}}
    paths['/'+name] = {'get': {'operationId': name, 'parameters': params, 'responses': response}}
with open(sys.argv[1], 'w') as target:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': probes}, 'paths': paths}, target)
PYNULLABLEVALUECLASS
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-nullable-value-class-aliases.json \
  --output "$workspace/project/build/generated-limits/root-nullable-value-class-aliases" --package-name org.example.rootnullablevalueclassaliases --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-nullable-value-class-aliases.json \
  --output "$workspace/project/build/generated-limits/root-nullable-value-class-aliases" --package-name org.example.rootnullablevalueclassaliases --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input "$workspace/root-nullable-value-class-probes.json" \
  --output "$workspace/project/build/generated-limits/root-nullable-value-class-probes" --package-name org.example.rootnullablevalueclassprobes --model-name-suffix Dto
# Nullable plain String enum aliases retain canonical vocabulary and nullable root intent.
python3 - "$workspace/root-nullable-enum-probes.json" <<'PYNULLABLEENUM'
import json
import sys
probes = {
    'Alias': {'$ref': '#/components/schemas/ZStatus', 'nullable': True},
    'ZStatus': {'type': 'string', 'enum': ['ready', 'done']},
    'AWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/Alias'} for i in range(240)}},
    'ANear': {'properties': {'v': {'$ref': '#/components/schemas/A45'}}, 'required': ['v']},
    'ANarrow': {'properties': {'v': {'$ref': '#/components/schemas/A45', 'nullable': False}}, 'required': ['v']},
}
for hop in range(46):
    previous = 'ZStatus' if hop == 0 else f'A{hop-1:02d}'
    probes[f'A{hop:02d}'] = dict({'$ref': f'#/components/schemas/{previous}'}, **({'nullable': True} if hop == 0 else {}))
with open(sys.argv[1], 'w') as target:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': probes}}, target)
PYNULLABLEENUM
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-nullable-enum-aliases.json \
  --output "$workspace/project/build/generated-limits/root-nullable-enum-aliases" --package-name org.example.rootnullableenumaliases --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-nullable-enum-aliases.json \
  --output "$workspace/project/build/generated-limits/root-nullable-enum-aliases" --package-name org.example.rootnullableenumaliases --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input "$workspace/root-nullable-enum-probes.json" \
  --output "$workspace/project/build/generated-limits/root-nullable-enum-probes" --package-name org.example.rootnullableenumprobes --model-name-suffix Dto
# Intrinsic nullable DTO roots retain the physical classifier and inherited typed intent.
python3 - "$workspace/intrinsic-nullable-dtos.json" "$workspace/intrinsic-nullable-probes.json" <<'PYINTRINSICNULLABLE'
import json
import sys
with open('tests/fixtures/http/intrinsic-nullable-dtos.json') as source:
    doc = json.load(source)
doc['components']['schemas']['ZRecord']['properties']['id']['minLength'] = 1
with open(sys.argv[1], 'w') as target:
    json.dump(doc, target)
probes = {
    'Alias': {'$ref': '#/components/schemas/ZRecord'},
    'ZRecord': {'nullable': True, 'properties': {'id': {'type': 'string', 'minLength': 1}}, 'required': ['id']},
    'AWide': {'properties': {f'p{i}': dict({'$ref': '#/components/schemas/Alias'}, **({'nullable': False} if i % 2 else {})) for i in range(240)}, 'required': [f'p{i}' for i in range(240)]},
    'ANear': {'properties': {'v': {'$ref': '#/components/schemas/A45'}}, 'required': ['v']},
    'ANarrow': {'properties': {'v': {'$ref': '#/components/schemas/A45', 'nullable': False}}, 'required': ['v']},
}
for hop in range(46):
    previous = 'ZRecord' if hop == 0 else f'A{hop-1:02d}'
    probes[f'A{hop:02d}'] = {'$ref': f'#/components/schemas/{previous}'}
leaf = {'type': 'string', 'minLength': 1}
for _ in range(16):
    leaf = {'type': 'array', 'items': leaf}
probes['ADeepChecked'] = {'nullable': True, 'properties': {'checked': leaf, 'child': {'$ref': '#/components/schemas/ADeepChecked'}}, 'required': ['checked']}
with open(sys.argv[2], 'w') as target:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': probes}}, target)
PYINTRINSICNULLABLE
"$workspace/openapi-kt" generate --input "$workspace/intrinsic-nullable-dtos.json" \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-dtos" --package-name org.example.intrinsicnullabledtos --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input "$workspace/intrinsic-nullable-dtos.json" \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-dtos" --package-name org.example.intrinsicnullabledtos --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input "$workspace/intrinsic-nullable-probes.json" \
  --output "$workspace/project/build/generated-limits/intrinsic-nullable-probes" --package-name org.example.intrinsicnullableprobes --model-name-suffix Dto --validation
# Ordinary DTO bindings keep explicit alias nullability separate from canonical classes.
python3 - "$workspace/root-nullable-object-aliases.json" "$workspace/root-nullable-object-probes.json" <<'PYROOTNULLABLEOBJECT'
import json
import sys
with open('tests/fixtures/http/root-nullable-object-aliases.json') as source:
    doc = json.load(source)
doc['components']['schemas']['ZRecord']['properties']['id']['minLength'] = 1
with open(sys.argv[1], 'w') as target:
    json.dump(doc, target)
probes = {
    'Alias': {'$ref': '#/components/schemas/ZRecord', 'nullable': True},
    'ZRecord': {'properties': {'id': {'type': 'string', 'minLength': 1}}, 'required': ['id']},
    'AWide': {'properties': {f'p{i}': dict({'$ref': '#/components/schemas/Alias'}, **({'nullable': False} if i % 2 else {})) for i in range(240)}, 'required': [f'p{i}' for i in range(240)]},
    'ANear': {'properties': {'v': {'$ref': '#/components/schemas/A45'}}, 'required': ['v']},
    'ANarrow': {'properties': {'v': {'$ref': '#/components/schemas/A45', 'nullable': False}}, 'required': ['v']},
}
for hop in range(46):
    previous = 'ZRecord' if hop == 0 else f'A{hop-1:02d}'
    probes[f'A{hop:02d}'] = dict({'$ref': f'#/components/schemas/{previous}'}, **({'nullable': True} if hop == 0 else {}))
with open(sys.argv[2], 'w') as target:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': probes}}, target)
PYROOTNULLABLEOBJECT
"$workspace/openapi-kt" generate --input "$workspace/root-nullable-object-aliases.json" \
  --output "$workspace/project/build/generated-limits/root-nullable-object-aliases" --package-name org.example.rootnullableobjectaliases --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input "$workspace/root-nullable-object-aliases.json" \
  --output "$workspace/project/build/generated-limits/root-nullable-object-aliases" --package-name org.example.rootnullableobjectaliases --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input "$workspace/root-nullable-object-probes.json" \
  --output "$workspace/project/build/generated-limits/root-nullable-object-probes" --package-name org.example.rootnullableobjectprobes --model-name-suffix Dto --validation
# Raw empty projections use JsonElement backing, never nominal DTO identity.
python3 - "$workspace/root-raw-probes.json" <<'PYROOTRAWALIAS'
import json
import sys
schemas = {
    'Alias': {'$ref': '#/components/schemas/ZRaw'},
    'ZRaw': {'type': 'object'},
    'AWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/Alias'} for i in range(240)}, 'required': [f'p{i}' for i in range(240)]},
    'ANear': {'properties': {'v': {'$ref': '#/components/schemas/A45'}}, 'required': ['v']},
}
for hop in range(46):
    previous = 'ZRaw' if hop == 0 else f'A{hop-1:02d}'
    schemas[f'A{hop:02d}'] = {'$ref': f'#/components/schemas/{previous}'}
with open(sys.argv[1], 'w') as target:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': schemas}}, target)
PYROOTRAWALIAS
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-raw-aliases.json \
  --output "$workspace/project/build/generated-limits/root-raw-aliases" --package-name org.example.rootrawaliases --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-raw-aliases.json \
  --output "$workspace/project/build/generated-limits/root-raw-aliases" --package-name org.example.rootrawaliases --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input "$workspace/root-raw-probes.json" \
  --output "$workspace/project/build/generated-limits/root-raw-probes" --package-name org.example.rootrawprobes --model-name-suffix Dto
# Hierarchy bindings preserve one canonical graph, top-level leaf/tag owners.
python3 - "$workspace/root-hierarchy-aliases.json" "$workspace/root-hierarchy-probes.json" "$workspace/root-hierarchy-backends.json" <<'PYROOTHIERARCHYALIAS'
import json
import sys
with open('tests/fixtures/http/root-hierarchy-aliases.json') as source:
    doc = json.load(source)
doc['components']['schemas']['ZFlux']['properties']['note']['minLength'] = 1
with open(sys.argv[1], 'w') as target:
    json.dump(doc, target)
probes = {
    'Alias': {'$ref': '#/components/schemas/ZRoot'},
    'ZRoot': {'oneOf': [{'$ref': '#/components/schemas/Leaf'}, {'$ref': '#/components/schemas/Other'}], 'discriminator': {'propertyName': 'tag'}},
    'Leaf': {'properties': {'tag': {'type': 'string'}, 'id': {'type': 'string'}}, 'required': ['id']},
    'Other': {'properties': {'tag': {'type': 'string'}, 'active': {'type': 'boolean'}}, 'required': ['active']},
    'AWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/Alias'} for i in range(240)}},
    'ANear': {'properties': {'v': {'$ref': '#/components/schemas/A45'}}, 'required': ['v']},
}
for hop in range(46):
    previous = 'ZRoot' if hop == 0 else f'A{hop-1:02d}'
    probes[f'A{hop:02d}'] = {'$ref': f'#/components/schemas/{previous}'}
with open(sys.argv[2], 'w') as target:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': probes}}, target)
# Reuse original accepted inferred/default/four-level graph, alias each graph
# reference consistently without rewriting metadata or implicit wire keys.
with open('tests/fixtures/model-semantics/hierarchy-unions.json') as source:
    backend = json.load(source)
catalog = backend['components']['schemas']
originals = list(catalog)
for name in originals:
    catalog['Alias' + name] = {'$ref': '#/components/schemas/' + name}
for name in originals:
    node = catalog[name]
    for keyword in ('oneOf', 'anyOf'):
        for branch in node.get(keyword, []):
            reference = branch['$ref']
            branch['$ref'] = reference.replace('#/components/schemas/', '#/components/schemas/Alias', 1)
    for wire, target in node.get('discriminator', {}).get('mapping', {}).items():
        mapped = target.replace('#/components/schemas/', '', 1)
        node['discriminator']['mapping'][wire] = 'Alias' + mapped
with open(sys.argv[3], 'w') as target:
    json.dump(backend, target)
PYROOTHIERARCHYALIAS
"$workspace/openapi-kt" generate --input "$workspace/root-hierarchy-aliases.json" \
  --output "$workspace/project/build/generated-limits/root-hierarchy-aliases" --package-name org.example.roothierarchyaliases --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input "$workspace/root-hierarchy-aliases.json" \
  --output "$workspace/project/build/generated-limits/root-hierarchy-aliases" --package-name org.example.roothierarchyaliases --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input "$workspace/root-hierarchy-probes.json" \
  --output "$workspace/project/build/generated-limits/root-hierarchy-probes" --package-name org.example.roothierarchyprobes --model-name-suffix Dto
"$workspace/openapi-kt" generate --input "$workspace/root-hierarchy-backends.json" \
  --output "$workspace/project/build/generated-limits/root-hierarchy-backends" --package-name org.example.hierarchybindingbackends --model-name-suffix Dto
# Non-discriminator union aliases retain canonical variants/backends/null carriers.
python3 - "$workspace/root-union-aliases.json" "$workspace/root-union-probes.json" <<'PYROOTUNIONALIAS'
import json
import sys
with open('tests/fixtures/http/root-union-aliases.json') as source:
    doc = json.load(source)
doc['components']['schemas']['ZNote']['properties']['text']['minLength'] = 1
with open(sys.argv[1], 'w') as target:
    json.dump(doc, target)
probes = {
    'Alias': {'$ref': '#/components/schemas/ZKind'},
    'ZKind': {'oneOf': [{'type': 'string'}, {'type': 'boolean'}, {'type': 'null'}]},
    'AWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/Alias'} for i in range(240)}},
    'ANear': {'properties': {'v': {'$ref': '#/components/schemas/A45'}}, 'required': ['v']},
}
for hop in range(46):
    previous = 'ZKind' if hop == 0 else f'A{hop-1:02d}'
    probes[f'A{hop:02d}'] = {'$ref': f'#/components/schemas/{previous}'}
with open(sys.argv[2], 'w') as target:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': probes}}, target)
PYROOTUNIONALIAS
"$workspace/openapi-kt" generate --input "$workspace/root-union-aliases.json" \
  --output "$workspace/project/build/generated-limits/root-union-aliases" --package-name org.example.rootunionaliases --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input "$workspace/root-union-aliases.json" \
  --output "$workspace/project/build/generated-limits/root-union-aliases" --package-name org.example.rootunionaliases --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input "$workspace/root-union-probes.json" \
  --output "$workspace/project/build/generated-limits/root-union-probes" --package-name org.example.rootunionprobes --model-name-suffix Dto
# Special root bindings share original wrapper/vocabulary/native/const contracts.
python3 - "$workspace/root-special-aliases.json" "$workspace/root-special-probes.json" <<'PYROOTSPECIAL'
import json
import sys
with open('tests/fixtures/http/root-special-aliases.json') as source:
    doc = json.load(source)
catalog = doc['components']['schemas']
catalog['ZId']['minLength'] = 1
catalog['ZLongId']['minimum'] = 0
with open(sys.argv[1], 'w') as target:
    json.dump(doc, target)
probes = {
    'Alias': {'$ref': '#/components/schemas/ZLong'},
    'ZLong': {'type': 'integer', 'format': 'int64', 'x-kotlin-value-class': True, 'minimum': 0},
    'AWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/Alias'} for i in range(120)}, 'required': [f'p{i}' for i in range(120)]},
    'ABoxed': {'properties': {f'p{i}': {'$ref': '#/components/schemas/Alias', 'nullable': True} for i in range(240)}},
    'ZConst': {'type': 'integer', 'const': 7},
    'ANear': {'properties': {'pin': {'$ref': '#/components/schemas/A45'}}},
}
for hop in range(46):
    previous = 'ZConst' if hop == 0 else f'A{hop-1:02d}'
    probes[f'A{hop:02d}'] = {'$ref': f'#/components/schemas/{previous}'}
with open(sys.argv[2], 'w') as target:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': probes}}, target)
PYROOTSPECIAL
"$workspace/openapi-kt" generate --input "$workspace/root-special-aliases.json" \
  --output "$workspace/project/build/generated-limits/root-special-aliases" --package-name org.example.rootspecial --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input "$workspace/root-special-aliases.json" \
  --output "$workspace/project/build/generated-limits/root-special-aliases" --package-name org.example.rootspecial --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input "$workspace/root-special-probes.json" \
  --output "$workspace/project/build/generated-limits/root-special-probes" --package-name org.example.rootspecialprobes --model-name-suffix Dto --validation
# Root collection bindings keep canonical targets, checked leaves and nullable views.
python3 - "$workspace/root-collection-aliases.json" "$workspace/root-collection-probes.json" <<'PYROOTCOLLECTION'
import json
import sys
with open('tests/fixtures/http/root-collection-aliases.json') as source:
    doc = json.load(source)
catalog = doc['components']['schemas']
leaf = {'type': ['string', 'null'], 'minLength': 2, 'maxLength': 3, 'pattern': '^[a-z]+$'}
catalog['ZValues'].update(minItems=1, maxItems=3, uniqueItems=True, items=leaf)
catalog['ZMap'].update(minProperties=1, maxProperties=3, additionalProperties=leaf)
catalog['ZOpen'].update(minProperties=1, maxProperties=3)
catalog['ZItems']['items']['minLength'] = 2
catalog['ZRows']['items']['properties']['id']['minLength'] = 1
with open(sys.argv[1], 'w') as target:
    json.dump(doc, target)
probes = {
    'ListAlias': {'$ref': '#/components/schemas/ZList'},
    'MapAlias': {'$ref': '#/components/schemas/ZMap'},
    'CountAlias': {'$ref': '#/components/schemas/ZCount'},
    'ZList': {'type': ['array', 'null'], 'minItems': 1, 'maxItems': 3, 'uniqueItems': True, 'items': leaf},
    'ZMap': {'type': 'object', 'nullable': True, 'minProperties': 1, 'maxProperties': 3, 'additionalProperties': leaf},
    'ZCount': {'type': 'object', 'nullable': True, 'minProperties': 1, 'maxProperties': 3, 'additionalProperties': {'type': 'boolean'}},
    'AWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/' + ('ListAlias' if i % 2 == 0 else 'MapAlias'), 'nullable': i % 3 == 0} for i in range(240)}},
    'ACountWide': {'properties': {f'p{i}': {'$ref': '#/components/schemas/CountAlias', 'nullable': False} for i in range(240)}},
    'ZPlain': {'type': 'array', 'items': {'type': 'string'}},
    'ANear': {'properties': {'v': {'$ref': '#/components/schemas/A44'}}, 'required': ['v']},
}
for hop in range(45):
    previous = 'ZPlain' if hop == 0 else f'A{hop-1:02d}'
    probes[f'A{hop:02d}'] = {'$ref': f'#/components/schemas/{previous}'}
field = {'$ref': '#/components/schemas/ListAlias', 'nullable': False}
for edge in range(15):
    field = {'type': 'array', 'minItems': 1, 'items': field}
probes['ADeep'] = {'properties': {'v': field}, 'required': ['v']}
with open(sys.argv[2], 'w') as target:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': probes}}, target)
PYROOTCOLLECTION
"$workspace/openapi-kt" generate --input "$workspace/root-collection-aliases.json" \
  --output "$workspace/project/build/generated-limits/root-collection-aliases" --package-name org.example.rootcollections --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input "$workspace/root-collection-aliases.json" \
  --output "$workspace/project/build/generated-limits/root-collection-aliases" --package-name org.example.rootcollections --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input "$workspace/root-collection-probes.json" \
  --output "$workspace/project/build/generated-limits/root-collection-probes" --package-name org.example.rootcollectionprobes --model-name-suffix Dto --validation
# Nullable roots retain native aliases; false consumers render qualified backing views.
"$workspace/openapi-kt" generate --input tests/fixtures/http/nullable-array-components.json \
  --output "$workspace/project/build/generated-limits/nullable-array-components" --package-name org.example.nullablearrays --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/nullable-array-components.json \
  --output "$workspace/project/build/generated-limits/nullable-array-components" --package-name org.example.nullablearrays --model-name-suffix Dto --validation --check
python3 - "$workspace/nullable-array-probes.json" <<'PYNULLABLEARRAY'
import json
import sys

def ref(narrow=False):
    value = {'$ref': '#/components/schemas/ZValues'}
    if narrow:
        value['nullable'] = False
    return value

field = ref(True)
for edge in range(15):
    field = {'type': 'array', 'items': field, 'minItems': 1}
schemas = {
    'ZValues': {'type': ['array', 'null'], 'minItems': 1, 'maxItems': 3, 'uniqueItems': True,
                'items': {'type': ['string', 'null'], 'minLength': 2, 'pattern': '^[a-z]+$'}},
    'AWide': {'properties': {f'p{i}': ref(i % 2 == 0) for i in range(240)}},
    'ADeep': {'properties': {'v': field}},
}
with open(sys.argv[1], 'w') as output:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': schemas}}, output)
PYNULLABLEARRAY
"$workspace/openapi-kt" generate --input "$workspace/nullable-array-probes.json" \
  --output "$workspace/project/build/generated-limits/nullable-array-probes" --package-name org.example.nullablearrayprobes --model-name-suffix Dto --validation
# Internal collection reference chains share typed Patch/HTTP/live SSE checks.
"$workspace/openapi-kt" generate --input tests/fixtures/http/schema-internal-collection-chains.json \
  --output "$workspace/project/build/generated-limits/schema-internal-collection-chains" --package-name org.example.collectionchains --model-name-suffix Dto --validation
# Independent width/depth probes exercise reference admission, not a Cartesian maximum.
python3 - "$workspace/collection-chain-probes.json" <<'PYCOLLECTIONCHAIN'
import json
import sys

def ref(key, **overrides):
    return {'$ref': '#/components/schemas/Source/properties/' + key, **overrides}

targets = {
    'leaf': {'type': ['string', 'null'], 'minLength': 2, 'maxLength': 3, 'pattern': '^[a-z]+$'},
    'list': {'type': ['array', 'null'], 'minItems': 1, 'maxItems': 3, 'uniqueItems': True, 'items': ref('leaf')},
    'hop': ref('list'),
}
next_key = 'leaf'
for edge in range(16):
    key = f'edge{edge}'
    if edge % 2 == 0:
        targets[key] = {'type': ['array', 'null'], 'minItems': 1, 'maxItems': 3, 'items': ref(next_key)}
        if edge == 0:
            targets[key]['uniqueItems'] = True
    else:
        targets[key] = {'type': ['object', 'null'], 'minProperties': 1, 'maxProperties': 3,
                        'additionalProperties': ref(next_key)}
    next_key = key
schemas = {
    'Source': {'properties': targets},
    'Wide': {'properties': {f'p{i}': ref('hop') for i in range(240)}},
    'Deep': {'properties': {'v': ref(next_key)}},
}
with open(sys.argv[1], 'w') as f:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': schemas}}, f, indent=2)
PYCOLLECTIONCHAIN
"$workspace/openapi-kt" generate --input "$workspace/collection-chain-probes.json" \
  --output "$workspace/project/build/generated-limits/collection-chain-probes" --package-name org.example.collectionchainprobes --model-name-suffix Dto --validation
# Borrowed finite collection refs: independent small runtime and wide/deep probes.
"$workspace/openapi-kt" generate --input tests/fixtures/http/schema-internal-collection-pointers.json \
  --output "$workspace/project/build/generated-limits/schema-internal-collection-pointers" --package-name org.example.collectionpointers --model-name-suffix Dto --validation
python3 - "$workspace/collection-pointer-probes.json" <<'PYCOLLECTION'
import copy
import json
import sys

def ref(key, **overrides):
    return {'$ref': '#/components/schemas/Source/properties/' + key, **overrides}

leaves = {
    'string': {'type': ['string', 'null'], 'minLength': 2, 'maxLength': 3, 'pattern': '^[a-z]+$'},
    'long': {'type': ['integer', 'null'], 'format': 'int64', 'minimum': -2, 'maximum': 10,
             'exclusiveMinimum': 0, 'exclusiveMaximum': 8, 'multipleOf': 2},
    'double': {'type': ['number', 'null'], 'format': 'double', 'minimum': -2, 'maximum': 10,
               'exclusiveMinimum': 0, 'exclusiveMaximum': 8, 'multipleOf': 2},
    'float': {'type': ['number', 'null'], 'format': 'float', 'minimum': -2, 'maximum': 10,
              'exclusiveMinimum': 0, 'exclusiveMaximum': 8, 'multipleOf': 2},
}
targets = copy.deepcopy(leaves)
for key in leaves:
    targets[key + 'List'] = {'type': ['array', 'null'], 'items': ref(key), 'minItems': 1, 'maxItems': 3}
    if key in ('string', 'long'):
        targets[key + 'List']['uniqueItems'] = True
    targets[key + 'Map'] = {'type': ['object', 'null'], 'additionalProperties': copy.deepcopy(leaves[key]),
                            'minProperties': 1, 'maxProperties': 3}
targets['countList'] = {'type': ['array', 'null'], 'items': {'type': ['string', 'null']}, 'minItems': 1, 'maxItems': 3, 'uniqueItems': False}
targets['countMap'] = {'type': ['object', 'null'], 'additionalProperties': {'type': ['string', 'null']}, 'minProperties': 1, 'maxProperties': 3}
for checked, name in [(True, 'deepFull'), (False, 'deepCount')]:
    tree = copy.deepcopy(leaves['string'] if checked else {'type': ['string', 'null']})
    for edge in range(16):
        if edge % 2 == 0:
            tree = {'type': ['array', 'null'], 'items': tree, 'minItems': 1, 'maxItems': 3}
            if edge == 0 and checked:
                tree['uniqueItems'] = True
        else:
            tree = {'type': ['object', 'null'], 'additionalProperties': tree, 'minProperties': 1, 'maxProperties': 3}
    targets[name] = tree
schemas = {'Source': {'properties': targets},
    'FullRuleProbe': {'properties': {key: ref(key) for key in ('stringList', 'stringMap', 'longList', 'doubleMap', 'floatList')}},
    'CountOnlyProbe': {'properties': {'list': ref('countList'), 'map': ref('countMap')}},
    'CountCodecProbe': {'properties': {'a': ref('stringList', nullable=False), 'b': ref('stringList')},
                       'required': ['a'], 'minProperties': 1, 'maxProperties': 1},
    'FullRuleWide': {'properties': {f'p{i}': ref(('string', 'long', 'double', 'float')[i % 4] + ('List' if i % 2 == 0 else 'Map')) for i in range(240)}},
    'CountOnlyWide': {'properties': {f'p{i}': ref('countList' if i % 2 == 0 else 'countMap') for i in range(240)}},
    'DeepFull': {'properties': {'v': ref('deepFull')}},
    'DeepCount': {'properties': {'v': ref('deepCount')}},
}
paths = {'/codec': {'post': {'operationId': 'codec', 'requestBody': {'required': True, 'content': {
    'application/json': {'schema': {'$ref': '#/components/schemas/CountCodecProbe'}}}}, 'responses': {'204': {}}}}}
with open(sys.argv[1], 'w') as f:
    json.dump({'openapi': '3.1.0', 'components': {'schemas': schemas}, 'paths': paths}, f, indent=2)
PYCOLLECTION
"$workspace/openapi-kt" generate --input "$workspace/collection-pointer-probes.json" \
  --output "$workspace/project/build/generated-limits/collection-pointer-probes" --package-name org.example.collectionpointerprobes --model-name-suffix Dto --validation
# Synthetic JVM slot boundary: 120 non-null Long parameters consume 240 slots.
python3 - "$workspace/limits.json" <<'PY'
import json
import sys
params = [{"name": f"p{i}", "in": "query", "required": True,
           "schema": {"type": "integer", "format": "int64"}} for i in range(120)]
paths = {"/wide": {"get": {"parameters": params, "responses": {"204": {}}}}}
paths["/bodywide"] = {"post": {"parameters": params[:-1], "responses": {"204": {}},
    "requestBody": {"required": True, "content": {"application/json": {
        "schema": {"type": "integer", "format": "int64"}}}}}}
# SSE Flow captures parameters; compile its wide closure and event dispatch cap.
paths["/ssewide"] = {"get": {"parameters": params[:-1], "responses": {"200": {
    "content": {"text/event-stream": {"schema": {"oneOf": [
        {"type": "object", "properties": {"event": {"type": "string", "const": f"v{i}"}}}
        for i in range(240)
    ]}}}
}}}}
# A request-only Patch<Long> field is one boxed JVM slot, not a wide Long slot.
paths["/presencewide"] = {"post": {"responses": {"204": {}}, "requestBody": {
    "required": True, "content": {"application/json": {"schema": {"type": "object", "properties": {
        f"p{i}": {"type": ["integer", "null"], "format": "int64"} for i in range(240)
    }}}}}}}
# Root request kind unions validate a wide object branch and the 240-branch cap.
paths["/kindwide"] = {"post": {"responses": {"204": {}}, "requestBody": {
    "required": True, "content": {"application/json": {"schema": {"anyOf": [
        {"type": "string"}, {"type": "object", "properties": {
            f"p{i}": {"type": "integer", "format": "int64"} for i in range(120)
        }, "required": [f"p{i}" for i in range(120)]}
    ]}}}}}}
paths["/kindbranches"] = {"post": {"responses": {"204": {}}, "requestBody": {
    "required": True, "content": {"application/json": {"schema": {
        "oneOf": [{"type": "boolean"}] + [{"type": "string"} for _ in range(239)]
    }}}}}}
for param in ["a", "b"]:
    paths[f"/kindfallback/{{{param}}}"] = {"post": {"responses": {"204": {}}, "parameters": [
        {"name": param, "in": "path", "required": True, "schema": {"type": "string"}}
    ], "requestBody": {"required": True, "content": {"application/json": {"schema": {
        "anyOf": [{"type": "string"}, {"type": "boolean"}]
    }}}}}}
# Borrowed nullable response views still compile full-width models and final fallback names.
paths["/emptyresponsewide"] = {"get": {"responses": {"200": {"content": {"application/json": {"schema": {
    "oneOf": [{}, {"type": "object", "properties": {
        f"p{i}": {"type": "integer", "format": "int64"} for i in range(120)
    }, "required": [f"p{i}" for i in range(120)]}]
}}}}}}}
for param in ["a", "b"]:
    paths[f"/emptyfallback/{{{param}}}"] = {"get": {"parameters": [
        {"name": param, "in": "path", "required": True, "schema": {"type": "string"}}
    ], "responses": {"200": {"content": {"application/json": {"schema": {"anyOf": [
        {}, {"type": "object", "properties": {"id": {"type": "string"}}}
    ]}}}}}}}
# Raw request-field unions keep boxed Patch slots, not payload wrapper fields.
field_union = {"oneOf": [{"type": "string"}, {"type": "boolean"}, {"type": "null"}]}
paths["/rawfieldwide"] = {"post": {"responses": {"204": {}}, "requestBody": {
    "required": True, "content": {"application/json": {"schema": {"type": "object", "properties": {
        f"p{i}": field_union for i in range(240)
    }}}}}}}
paths["/rawfieldbranches"] = {"post": {"responses": {"204": {}}, "requestBody": {
    "required": True, "content": {"application/json": {"schema": {"type": "object", "properties": {
        "payload": {"oneOf": [{"type": "boolean"}] + [{"type": "string"} for _ in range(239)]}
    }}}}}}}
for param in ["a", "b"]:
    paths[f"/rawfieldfallback/{{{param}}}"] = {"post": {"responses": {"204": {}}, "parameters": [
        {"name": param, "in": "path", "required": True, "schema": {"type": "string"}}
    ], "requestBody": {"required": True, "content": {"application/json": {"schema": {
        "type": "object", "properties": {"payload": field_union}
    }}}}}}
# Nullable scalar signatures erase alike; reserve the natural `same2` name.
for path, name in [("/a", "same"), ("/b", "same"), ("/c", "same2")]:
    paths[path] = {"get": {"operationId": name, "responses": {"204": {}}}}
# Inline value classes can unbox Long; verify primary/serialization constructors
# and suspend/Flow capture boundaries, not only nominal source references.
vc_ref = {"$ref": "#/components/schemas/VCounter"}
vc_params = [{"name": f"p{i}", "in": "query", "required": True, "schema": vc_ref} for i in range(120)]
paths["/vcwide"] = {"get": {"parameters": vc_params, "responses": {"204": {}}}}
paths["/vcssewide"] = {"get": {"parameters": vc_params[:-1], "responses": {"200": {
    "content": {"text/event-stream": {"schema": {"type": "object", "required": ["data"], "properties": {"data": vc_ref}}}}
}}}}
paths["/vcfloatwide"] = {"get": {"parameters": [{"name": f"p{i}", "in": "query", "required": True,
    "schema": {"$ref": "#/components/schemas/VRatio"}} for i in range(120)], "responses": {"204": {}}}}
# Inline element validators add no constructor slots. Deep nested checks must
# compile at the checked-collection limit (16 edges); graph depth stays48.
deep_element = {"type": "string", "minLength": 1}
for _ in range(16):
    deep_element = {"type": "array", "items": deep_element}
deep_count = {"type": "array", "minItems": 1, "maxItems": 2, "items": {"type": "string"}}
for _ in range(15):
    deep_count = {"type": "array", "items": deep_count}
deep_unique = {"type": "array", "uniqueItems": True, "items": {"type": ["boolean", "null"]}}
for _ in range(15):
    deep_unique = {"type": "array", "items": deep_unique}
schemas = {
    # Private codec symbols must not shadow class/property/nested declarations.
    "OpenapiKtPropertyCountSerializer": {"type": "object", "minProperties": 0,
        "properties": {"v": {"type": "string"}}},
    "PropertyCountShadow": {"type": "object", "minProperties": 0,
        "properties": {"openapiKtPropertyCountSerializer": {"type": "object", "minProperties": 0,
            "properties": {"v": {"type": "string"}}}}},
    "PropertyCountPropertyShadow": {"type": "object", "minProperties": 0,
        "properties": {"openapiKtPropertyCountSerializer": {"type": "string"}}},
    # Compilation-only boundaries for the opt-in ordinary DTO codec.
    "PropertyCountWide": {"type": "object", "minProperties": 0, "maxProperties": 120,
        "required": [f"p{i}" for i in range(120)], "properties": {
            f"p{i}": {"type": "integer", "format": "int64"} for i in range(120)}},
    "PropertyCountBoxed": {"type": "object", "minProperties": 0, "maxProperties": 240,
        "properties": {f"p{i}": {"type": ["integer", "null"], "format": "int64"} for i in range(240)}},
    "UniqueDeep": {"type": "object", "properties": {"v": deep_unique}},
    "UniqueWide": {"type": "object", "properties": {
        f"p{i}": {"type": "array", "uniqueItems": True, "items": {"type": ["integer", "null"], "format": "int64"}} for i in range(240)}},
    "CountDeep": {"type": "object", "properties": {"v": deep_count}},
    "CountWide": {"type": "object", "properties": {
        f"p{i}": {"type": "array", "minItems": 1, "maxItems": 2, "items": {"type": "integer"}} for i in range(240)}},
    "ElementDeep": {"type": "object", "properties": {"v": deep_element}},
    "ElementWide": {"type": "object", "properties": {
        f"p{i}": {"type": "array", "items": {"type": "integer", "format": "int64", "minimum": -2, "maximum": 2}} for i in range(240)}},
    "AliasBound": {"type": "integer", "format": "int64", "minimum": -2, "maximum": 2},
    "AliasLink": {"$ref": "#/components/schemas/AliasBound"},
    "AliasWide": {"type": "object", "required": [f"p{i}" for i in range(120)], "properties": {
        f"p{i}": {"$ref": "#/components/schemas/AliasLink"} for i in range(120)}},
    "AliasBoxed": {"type": "object", "properties": {
        f"p{i}": {"$ref": "#/components/schemas/AliasLink", "nullable": True} for i in range(240)}},
    "ValidatedWide": {"type": "object", "required": [f"p{i}" for i in range(120)], "properties": {
        f"p{i}": {"type": "integer", "format": "int64", "minimum": 0, "maximum": 1} for i in range(120)}},
    "ValidatedBoxed": {"type": "object", "properties": {
        f"p{i}": {"type": ["number", "null"], "minimum": 0, "maximum": 1} for i in range(240)}},
    "VRatio": {"type": "number", "format": "double", "x-kotlin-value-class": True},
    "VDoubleWide": {"type": "object", "required": [f"p{i}" for i in range(120)],
        "properties": {f"p{i}": {"$ref": "#/components/schemas/VRatio"} for i in range(120)}},
    "VCounter": {"type": "integer", "format": "int64", "x-kotlin-value-class": True},
    "VWide": {"type": "object", "required": [f"p{i}" for i in range(120)], "properties": {f"p{i}": vc_ref for i in range(120)}},
    "VBoxed": {"type": "object", "properties": {f"p{i}": {**vc_ref, "nullable": True} for i in range(240)}}
}
# Flat inline allOf aggregates retain fixed validation slots at constructor limits.
def aggregate_wide(n, boxed):
    first, second = {}, {}
    for i in range(n):
        kind, fmt = ('integer', 'int64') if i % 2 == 0 else ('number', 'double')
        if boxed and i % 3 == 0:
            first[f'p{i}'] = {'type': ['string', 'null'], 'minLength': 2, 'pattern': '^[a-z]+$'}
            second[f'p{i}'] = {'type': ['string', 'null'], 'maxLength': 3, 'pattern': '^[a-z]+$'}
        else:
            ty = [kind, 'null'] if boxed else kind
            first[f'p{i}'] = {'type': ty, 'format': fmt, 'minimum': 0, 'exclusiveMinimum': 0}
            second[f'p{i}'] = {'type': ty, 'format': fmt, 'maximum': 8, 'exclusiveMaximum': 8, 'multipleOf': 2}
    return {'allOf': [{'type': 'object', 'required': list(first), 'properties': first},
                      {'type': 'object', 'properties': second}]}
schemas['AggregateFloatRemainder'] = {'allOf': [
    {'properties': {'value': {'type': 'number', 'format': 'float', 'minimum': 0}}},
    {'properties': {'value': {'type': 'number', 'format': 'float', 'multipleOf': 0.1}}}
]}
schemas['AggregateHelperShadow'] = {'allOf': [
    {'properties': {'v': {'type': 'integer', 'minimum': 0}}},
    {'properties': {'v': {'type': 'integer', 'maximum': 2}}}
], 'properties': {
    'openapiKtValidatePrimitive': {'type': 'string'},
    'openapiKtValidatePrimitive2': {'type': 'object', 'properties': {'id': {'type': 'string'}}},
    'openapiKtValidatePrimitive3': {'type': 'string', 'const': 'c'},
    'kotlin': {'type': 'string'}
}}
schemas['AggregateWide'] = aggregate_wide(120, False)
schemas['AggregateBoxed'] = aggregate_wide(240, True)
# Independent composed-count DTO boundaries keep every prior primitive category,
# factor and String rule, with the same compiler/plugins/heap settings.
for name, n, boxed in [('AllOfCountWide', 120, False), ('AllOfCountBoxed', 240, True)]:
    model = aggregate_wide(n, boxed)
    model['allOf'][0]['minProperties'] = 0
    model['allOf'][1]['maxProperties'] = n
    schemas[name] = model
# Real composed codec/helper allocation against class, property and nested names.
for name, props in [
    ('AllOfCountHelperShadow', {'openapiKtValidatePrimitive': {'type': 'string'},
        'openapiKtPropertyCountSerializer': {'type': 'string'}}),
    ('AllOfCountNestedShadow', {'openapiKtPropertyCountSerializer': {
        'allOf': [{'minProperties': 0, 'properties': {'v': {'type': 'string'}}}]}}),
    ('AllOfCountOpenapiKtPropertyCountSerializer', {'v': {'type': 'string'}})
]:
    schemas[name] = {'allOf': [
        {'minProperties': 0, 'properties': {'value': {'type': 'integer', 'minimum': 0}}},
        {'properties': {'value': {'type': 'integer', 'maximum': 2}}}
    ], 'properties': props}
# Keep original decimal tokens exact, especially the distinct Double endpoints.
# Inclusive source maxima are tighter, but both mapped strict checks must survive.
schemas['AggregateUpperTies'] = {'type': 'object', 'allOf': [
    {'type': 'object', 'required': ['floatUpper', 'doubleUpper'], 'properties': {
        'floatUpper': {'type': 'number', 'format': 'float', 'maximum': '__upperFloatMaximum__'},
        'doubleUpper': {'type': 'number', 'format': 'double', 'maximum': '__upperDoubleMaximum__'}
    }},
    {'type': 'object', 'properties': {
        'floatUpper': {'type': 'number', 'format': 'float', 'exclusiveMaximum': '__upperFloatExclusive__'},
        'doubleUpper': {'type': 'number', 'format': 'double', 'exclusiveMaximum': '__upperDoubleExclusive__'}
    }}
]}
# Additive terminal-ref boundaries; all prior wide rules above remain intact.
pointer_source = {'type': 'object', 'properties': {
    'long': {'type': 'integer', 'format': 'int64', 'minimum': 0, 'exclusiveMinimum': 0,
        'maximum': 8, 'exclusiveMaximum': 8, 'multipleOf': 2},
    'double': {'type': 'number', 'format': 'double', 'minimum': 0, 'exclusiveMinimum': 0,
        'maximum': 8, 'exclusiveMaximum': 8, 'multipleOf': 2},
    'string': {'type': 'string', 'minLength': 2, 'maxLength': 3, 'pattern': '^[a-z]+$'}
}}
schemas['PointerSource'] = pointer_source
for name, n, boxed in [('PointerWide', 120, False), ('PointerBoxed', 240, True)]:
    props = {}
    for i in range(n):
        key = ('string' if boxed and i % 3 == 0 else 'long' if i % 2 == 0 else 'double')
        props[f'p{i}'] = {'$ref': '#/components/schemas/PointerSource/properties/' + key}
        if boxed:
            props[f'p{i}']['nullable'] = True
    schemas[name] = {'type': 'object', 'properties': props, 'minProperties': 0, 'maxProperties': n}
    if not boxed:
        schemas[name]['required'] = list(props)
pointer_string = {'$ref': '#/components/schemas/PointerSource/properties/string'}
schemas['PointerEncodeLimit'] = {'type': 'object', 'required': ['value'],
    'minProperties': 1, 'maxProperties': 1, 'properties': {'value': pointer_string,
        'note': {**pointer_string, 'nullable': True}}}
paths['/pointercount'] = {'post': {'operationId': 'pointerCount', 'responses': {'204': {}},
    'requestBody': {'required': True, 'content': {'application/json': {
        'schema': {'$ref': '#/components/schemas/PointerEncodeLimit'}}}}}}
source = json.dumps({"openapi": "3.1.0", "components": {"schemas": schemas}, "paths": paths})
for token, number in {
    '__upperFloatMaximum__': '0.300000001',
    '__upperFloatExclusive__': '0.300000004',
    '__upperDoubleMaximum__': '0.300000000000000001',
    '__upperDoubleExclusive__': '0.300000000000000004'
}.items():
    source = source.replace(json.dumps(token), number)
with open(sys.argv[1], "w", encoding="utf-8") as output:
    output.write(source)
PY
"$workspace/openapi-kt" generate --input "$workspace/limits.json" \
  --output "$workspace/project/build/generated-limits" --package-name org.example.http.limits --validation
# Small full-rule typed-helper consumers are independent of the immutable wide
# dataset above; do not replace or simplify any original boundary input.
python3 - "$workspace/limits.json" "$workspace/pointer-helper-probes.json" <<'PY'
import json
import sys
with open(sys.argv[1], encoding='utf-8') as source:
    target = json.load(source)['components']['schemas']['PointerSource']
props = target['properties']
props['floatTie'] = {'type': 'number', 'format': 'float', 'minimum': 0.300000001,
    'exclusiveMinimum': 0.300000004, 'maximum': 1}
props['doubleTie'] = {'type': 'number', 'format': 'double', 'minimum': '__probeDoubleMinimum__',
    'exclusiveMinimum': '__probeDoubleExclusive__', 'maximum': 1}
for kind in ['float', 'double']:
    props[kind + 'Remainder'] = {'type': 'number', 'format': kind, 'minimum': 0,
        'exclusiveMinimum': 0, 'maximum': 1, 'exclusiveMaximum': 1, 'multipleOf': 0.1}
def ref(key, **kw):
    return {'$ref': '#/components/schemas/PointerSource/properties/' + key, **kw}
collection = {'type': 'array', 'items': ref('long'), 'minItems': 1, 'maxItems': 3, 'uniqueItems': True}
mapping = {'type': 'object', 'properties': {}, 'additionalProperties': ref('double'),
    'minProperties': 1, 'maxProperties': 2}
fields = {'s': ref('string'), 'n': ref('long'), 'd': ref('double'),
    **{key: ref(key) for key in ['floatTie', 'doubleTie', 'floatRemainder', 'doubleRemainder']},
    'list': collection, 'map': mapping, 'alias': {'$ref': '#/components/schemas/Alias'}}
probe = {'type': 'object', 'required': ['s', 'n', 'd'], 'minProperties': 3,
    'maxProperties': 10, 'properties': fields}
request = {'type': 'object', 'properties': {'n': ref('long', nullable=True),
    'list': {**collection, 'nullable': True}, 'map': {**mapping, 'nullable': True}}}
envelope = {'type': 'object', 'required': ['data'], 'properties': {
    'event': {'type': 'string', 'const': 'probe'}, 'data': {'$ref': '#/components/schemas/Probe'}}}
helper_events = {'get': {'operationId': 'helperWatch', 'responses': {'200': {
    'content': {'text/event-stream': {'schema': envelope}}}}}}
doc = {'openapi': '3.1.0', 'components': {'schemas': {'PointerSource': target,
    'Probe': probe, 'Alias': {'type': 'object', 'properties': {}, 'additionalProperties': ref('long')}}},
    'paths': {'/probe': {'post': {'operationId': 'probe', 'requestBody': {'required': True,
        'content': {'application/json': {'schema': request}}}, 'responses': {'204': {}}}},
        '/helper-events': helper_events}}
text = json.dumps(doc)
for tag, token in {'__probeDoubleMinimum__': '0.300000000000000001',
    '__probeDoubleExclusive__': '0.300000000000000004'}.items():
    text = text.replace(json.dumps(tag), token)
with open(sys.argv[2], 'w', encoding='utf-8') as output:
    output.write(text)
PY
"$workspace/openapi-kt" generate --input "$workspace/pointer-helper-probes.json" \
  --output "$workspace/project/build/generated-limits/pointer-helper-probes" --package-name org.example.pointerhelpers --model-name-suffix Dto --validation
# Independent borrowed-wrapper presence and genuinely nullable borrowed leaves.
# Keep the original pointer helper probes and full-rule boundaries untouched.
python3 - "$workspace/pointer-presence-null-probes.json" <<'PYFOLLOWUP'
import sys
with open(sys.argv[1], 'w', encoding='utf-8') as output:
    output.write('{"openapi":"3.1.0","info":{"title":"Pointer Followup","version":"1"},"paths":{"/submit":{"post":{"operationId":"submit","requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"object","required":["requiredNull","requiredFalse"],"properties":{"inherited":{"allOf":[{"$ref":"#/components/schemas/Source/properties/nullable"}]},"memberTrue":{"allOf":[{"$ref":"#/components/schemas/Source/properties/plain","nullable":true}]},"memberFalse":{"allOf":[{"$ref":"#/components/schemas/Source/properties/nullable","nullable":false}]},"annotationTrue":{"allOf":[{"$ref":"#/components/schemas/Source/properties/nullable","nullable":false},{"nullable":true}]},"annotationFalse":{"allOf":[{"$ref":"#/components/schemas/Source/properties/plain","nullable":true},{"nullable":false}]},"outerTrue":{"allOf":[{"$ref":"#/components/schemas/Source/properties/nullable","nullable":false}],"nullable":true},"outerFalse":{"allOf":[{"$ref":"#/components/schemas/Source/properties/plain","nullable":true}],"nullable":false},"requiredNull":{"allOf":[{"$ref":"#/components/schemas/Source/properties/nullable"}]},"requiredFalse":{"allOf":[{"$ref":"#/components/schemas/Source/properties/nullable","nullable":false}]},"named":{"allOf":[{"$ref":"#/components/schemas/Named","nullable":true}]},"nested":{"type":"object","properties":{"inherited":{"allOf":[{"$ref":"#/components/schemas/Source/properties/nullable"}]},"memberTrue":{"allOf":[{"$ref":"#/components/schemas/Source/properties/plain","nullable":true}]}}},"patchList":{"type":"array","minItems":1,"uniqueItems":true,"items":{"$ref":"#/components/schemas/Source/properties/nullable"},"nullable":true},"patchMap":{"type":"object","minProperties":1,"additionalProperties":{"$ref":"#/components/schemas/Source/properties/nullable"},"nullable":true}}}}}},"responses":{"204":{"description":"Done"}}}}},"components":{"schemas":{"Named":{"type":"string"},"Source":{"type":"object","properties":{"nullable":{"type":"string","nullable":true,"minLength":2,"maxLength":3,"pattern":"^[a-z]+$"},"plain":{"type":"string","minLength":2,"maxLength":3,"pattern":"^[a-z]+$"},"number":{"type":["integer","null"],"format":"int64","minimum":0,"exclusiveMinimum":0,"maximum":8,"exclusiveMaximum":8,"multipleOf":2}}},"Leaves":{"type":"object","required":["list","map","numbers","nullableList","nullableMap","nonNullList","nonNullMap"],"properties":{"list":{"type":"array","minItems":1,"uniqueItems":true,"items":{"$ref":"#/components/schemas/Source/properties/nullable"}},"map":{"type":"object","minProperties":1,"additionalProperties":{"$ref":"#/components/schemas/Source/properties/nullable"}},"numbers":{"type":"array","minItems":1,"items":{"$ref":"#/components/schemas/Source/properties/number"}},"nullableList":{"type":"array","minItems":1,"uniqueItems":true,"items":{"$ref":"#/components/schemas/Source/properties/nullable"},"nullable":true},"nullableMap":{"type":"object","minProperties":1,"additionalProperties":{"$ref":"#/components/schemas/Source/properties/nullable"},"nullable":true},"nonNullList":{"type":"array","minItems":1,"items":{"$ref":"#/components/schemas/Source/properties/nullable","nullable":false}},"nonNullMap":{"type":"object","minProperties":1,"additionalProperties":{"$ref":"#/components/schemas/Source/properties/nullable","nullable":false}}}},"Shared":{"type":"object","properties":{"inherited":{"allOf":[{"$ref":"#/components/schemas/Source/properties/nullable"}]},"memberTrue":{"allOf":[{"$ref":"#/components/schemas/Source/properties/plain","nullable":true}]}}}}}}\n')
PYFOLLOWUP
"$workspace/openapi-kt" generate --input "$workspace/pointer-presence-null-probes.json" \
  --output "$workspace/project/build/generated-limits/pointer-presence-null-probes" --package-name org.example.pointerfollowup --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-fields.json \
  --output "$workspace/project/build/generated-limits/hierarchy-fields" --package-name org.example.hierarchyfields --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-fields.json \
  --output "$workspace/project/build/generated-limits/hierarchy-fields" --package-name org.example.hierarchyfields --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-dispatch.json \
  --output "$workspace/project/build/generated-limits/hierarchy-dispatch" --package-name org.example.hierarchydispatch --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-dispatch.json \
  --output "$workspace/project/build/generated-limits/hierarchy-dispatch" --package-name org.example.hierarchydispatch --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-nullable-aliases.json \
  --output "$workspace/project/build/generated-limits/hierarchy-nullable-aliases" --package-name org.example.hierarchynullable --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-nullable-aliases.json \
  --output "$workspace/project/build/generated-limits/hierarchy-nullable-aliases" --package-name org.example.hierarchynullable --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-intrinsic-nullable.json \
  --output "$workspace/project/build/generated-limits/hierarchy-intrinsic-nullable" --package-name org.example.hierarchyintrinsic --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-intrinsic-nullable.json \
  --output "$workspace/project/build/generated-limits/hierarchy-intrinsic-nullable" --package-name org.example.hierarchyintrinsic --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-reference-chains.json \
  --output "$workspace/project/build/generated-limits/hierarchy-reference-chains" --package-name org.example.hierarchychains --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-reference-chains.json \
  --output "$workspace/project/build/generated-limits/hierarchy-reference-chains" --package-name org.example.hierarchychains --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-null-members.json \
  --output "$workspace/project/build/generated-limits/hierarchy-null-members" --package-name org.example.hierarchymembers --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-null-members.json \
  --output "$workspace/project/build/generated-limits/hierarchy-null-members" --package-name org.example.hierarchymembers --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-nested-owners.json \
  --output "$workspace/project/build/generated-limits/hierarchy-nested-owners" --package-name org.example.hierarchynested --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-nested-owners.json \
  --output "$workspace/project/build/generated-limits/hierarchy-nested-owners" --package-name org.example.hierarchynested --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/composed-nested-owners.json \
  --output "$workspace/project/build/generated-limits/composed-nested-owners" --package-name org.example.composedowners --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/composed-nested-owners.json \
  --output "$workspace/project/build/generated-limits/composed-nested-owners" --package-name org.example.composedowners --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/allof-nominal-terminals.json \
  --output "$workspace/project/build/generated-limits/allof-nominal-terminals" --package-name org.example.allofterminals --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/allof-nominal-terminals.json \
  --output "$workspace/project/build/generated-limits/allof-nominal-terminals" --package-name org.example.allofterminals --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/union-nominal-owners.json \
  --output "$workspace/project/build/generated-limits/union-nominal-owners" --package-name org.example.unionowners --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/union-nominal-owners.json \
  --output "$workspace/project/build/generated-limits/union-nominal-owners" --package-name org.example.unionowners --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/enum-nominal-owners.json \
  --output "$workspace/project/build/generated-limits/enum-nominal-owners" --package-name org.example.enumowners --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/enum-nominal-owners.json \
  --output "$workspace/project/build/generated-limits/enum-nominal-owners" --package-name org.example.enumowners --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/valueclass-reference-chains.json \
  --output "$workspace/project/build/generated-limits/valueclass-reference-chains" --package-name org.example.wrapperchains --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/valueclass-reference-chains.json \
  --output "$workspace/project/build/generated-limits/valueclass-reference-chains" --package-name org.example.wrapperchains --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/composed-union-terminals.json \
  --output "$workspace/project/build/generated-limits/composed-union-terminals" --package-name org.example.productterminals --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/composed-union-terminals.json \
  --output "$workspace/project/build/generated-limits/composed-union-terminals" --package-name org.example.productterminals --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/temporal-reference-chains.json \
  --output "$workspace/project/build/generated-limits/temporal-reference-chains" --package-name org.example.temporalchains --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/temporal-reference-chains.json \
  --output "$workspace/project/build/generated-limits/temporal-reference-chains" --package-name org.example.temporalchains --model-name-suffix Dto --validation --check
# Named builtin chain consumers share original scalar codecs/checks with fields.
python3 - "$workspace/builtin-reference-chains-checked.json" <<'PYBUILTINCHAINS'
from pathlib import Path
import json,sys
d=json.loads(Path('tests/fixtures/http/builtin-reference-chains.json').read_text())
s=d['components']['schemas']
s['Text'].update(minLength=2,pattern='^[a-z]+$')
s['Count'].update(minimum=2,multipleOf=2)
s['Wide'].update(minimum=2)
s['Ratio'].update(minimum=0,maximum=10)
s['Measure'].update(minimum=0,maximum=10)
Path(sys.argv[1]).write_text(json.dumps(d,indent=2)+'\n')
PYBUILTINCHAINS
"$workspace/openapi-kt" generate --input "$workspace/builtin-reference-chains-checked.json" \
  --output "$workspace/project/build/generated-limits/builtin-reference-chains" --package-name org.example.builtinchains --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input "$workspace/builtin-reference-chains-checked.json" \
  --output "$workspace/project/build/generated-limits/builtin-reference-chains" --package-name org.example.builtinchains --model-name-suffix Dto --validation --check
python3 - "$workspace/root-borrowed-builtin-checked.json" <<'PYBORROWEDROOT'
from pathlib import Path
import json,sys
d=json.loads(Path('tests/fixtures/http/root-borrowed-builtin.json').read_text())
s=d['components']['schemas']['Source']['properties']
s['text'].update(minLength=2,pattern='^[a-z]+$')
s['count'].update(minimum=2,multipleOf=2)
s['wide'].update(minimum=2)
s['ratio'].update(minimum=0,maximum=10)
s['measure'].update(minimum=0,maximum=10)
Path(sys.argv[1]).write_text(json.dumps(d,indent=2)+'\n')
PYBORROWEDROOT
"$workspace/openapi-kt" generate --input "$workspace/root-borrowed-builtin-checked.json" \
  --output "$workspace/project/build/generated-limits/root-borrowed-builtin" --package-name org.example.borrowedbuiltin --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input "$workspace/root-borrowed-builtin-checked.json" \
  --output "$workspace/project/build/generated-limits/root-borrowed-builtin" --package-name org.example.borrowedbuiltin --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/bigint-reference-chains.json \
  --output "$workspace/project/build/generated-limits/bigint-reference-chains" --package-name org.example.bigintchains --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/bigint-reference-chains.json \
  --output "$workspace/project/build/generated-limits/bigint-reference-chains" --package-name org.example.bigintchains --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/bytes-reference-chains.json \
  --output "$workspace/project/build/generated-limits/bytes-reference-chains" --package-name org.example.byteschains --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/bytes-reference-chains.json \
  --output "$workspace/project/build/generated-limits/bytes-reference-chains" --package-name org.example.byteschains --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/caller-native-reference-chains.json \
  --output "$workspace/project/build/generated-limits/caller-native-reference-chains" --package-name org.example.nativechains --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/caller-native-reference-chains.json \
  --output "$workspace/project/build/generated-limits/caller-native-reference-chains" --package-name org.example.nativechains --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/raw-true-reference-chains.json \
  --output "$workspace/project/build/generated-limits/raw-true-reference-chains" --package-name org.example.rawtruechains --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/raw-true-reference-chains.json \
  --output "$workspace/project/build/generated-limits/raw-true-reference-chains" --package-name org.example.rawtruechains --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/recursive-cover-reference-chains.json \
  --output "$workspace/project/build/generated-limits/recursive-cover-reference-chains" --package-name org.example.coverchains --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/recursive-cover-reference-chains.json \
  --output "$workspace/project/build/generated-limits/recursive-cover-reference-chains" --package-name org.example.coverchains --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/named-collection-reference-chains.json \
  --output "$workspace/project/build/generated-limits/named-collection-reference-chains" --package-name org.example.namedcollectionchains --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/named-collection-reference-chains.json \
  --output "$workspace/project/build/generated-limits/named-collection-reference-chains" --package-name org.example.namedcollectionchains --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-borrowed-collection.json \
  --output "$workspace/project/build/generated-limits/root-borrowed-collection" --package-name org.example.borrowedcollectionroots --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/root-borrowed-collection.json \
  --output "$workspace/project/build/generated-limits/root-borrowed-collection" --package-name org.example.borrowedcollectionroots --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-dto-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-dto-leaves" --package-name org.example.borroweddtoleaves --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-dto-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-dto-leaves" --package-name org.example.borroweddtoleaves --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-enum-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-enum-leaves" --package-name org.example.borrowedenumleaves --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-enum-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-enum-leaves" --package-name org.example.borrowedenumleaves --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-valueclass-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-valueclass-leaves" --package-name org.example.borrowedvalueclassleaves --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-valueclass-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-valueclass-leaves" --package-name org.example.borrowedvalueclassleaves --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-allof-model-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-allof-model-leaves" --package-name org.example.borrowedallofleaves --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-allof-model-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-allof-model-leaves" --package-name org.example.borrowedallofleaves --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-union-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-union-leaves" --package-name org.example.borrowedunionleaves --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-union-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-union-leaves" --package-name org.example.borrowedunionleaves --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-product-union-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-product-union-leaves" --package-name org.example.borrowedproductunionleaves --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-product-union-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-product-union-leaves" --package-name org.example.borrowedproductunionleaves --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-hierarchy-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-hierarchy-leaves" --package-name org.example.borrowedhierarchyleaves --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-hierarchy-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-hierarchy-leaves" --package-name org.example.borrowedhierarchyleaves --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-builtin-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-builtin-leaves" --package-name org.example.borrowedbuiltinleaves --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-builtin-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-builtin-leaves" --package-name org.example.borrowedbuiltinleaves --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-temporal-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-temporal-leaves" --package-name org.example.borrowedtemporalleaves --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-temporal-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-temporal-leaves" --package-name org.example.borrowedtemporalleaves --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-bigint-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-bigint-leaves" --package-name org.example.borrowedbigintleaves --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-bigint-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-bigint-leaves" --package-name org.example.borrowedbigintleaves --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-bytes-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-bytes-leaves" --package-name org.example.borrowedbytesleaves --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-bytes-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-bytes-leaves" --package-name org.example.borrowedbytesleaves --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-caller-native-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-caller-native-leaves" --package-name org.example.borrowedcallernativeleaves --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-caller-native-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-caller-native-leaves" --package-name org.example.borrowedcallernativeleaves --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-raw-true-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-raw-true-leaves" --package-name org.example.borrowedrawtrueleaves --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-raw-true-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-raw-true-leaves" --package-name org.example.borrowedrawtrueleaves --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-recursive-cover-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-recursive-cover-leaves" --package-name org.example.borrowedrecursivecoverleaves --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/borrowed-collection-recursive-cover-leaves.json \
  --output "$workspace/project/build/generated-limits/borrowed-collection-recursive-cover-leaves" --package-name org.example.borrowedrecursivecoverleaves --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/union-common-dictionaries.json \
  --output "$workspace/project/build/generated-limits/union-common-dictionaries" --package-name org.example.commondictionaries --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/union-common-dictionaries.json \
  --output "$workspace/project/build/generated-limits/union-common-dictionaries" --package-name org.example.commondictionaries --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/nullable-allof-dtos.json \
  --output "$workspace/project/build/generated-limits/nullable-allof-dtos" --package-name org.example.nullableallof --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/nullable-allof-dtos.json \
  --output "$workspace/project/build/generated-limits/nullable-allof-dtos" --package-name org.example.nullableallof --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/union-common-nominals.json \
  --output "$workspace/project/build/generated-limits/union-common-nominals" --package-name org.example.commonnominals --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/union-common-nominals.json \
  --output "$workspace/project/build/generated-limits/union-common-nominals" --package-name org.example.commonnominals --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/union-inline-values.json \
  --output "$workspace/project/build/generated-limits/union-inline-values" --package-name org.example.inlinevalues --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/union-inline-values.json \
  --output "$workspace/project/build/generated-limits/union-inline-values" --package-name org.example.inlinevalues --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/nullable-product-unions.json \
  --output "$workspace/project/build/generated-limits/nullable-product-unions" --package-name org.example.nullableproducts --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/nullable-product-unions.json \
  --output "$workspace/project/build/generated-limits/nullable-product-unions" --package-name org.example.nullableproducts --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/nullable-object-members.json \
  --output "$workspace/project/build/generated-limits/nullable-object-members" --package-name org.example.nullablemembers --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/nullable-object-members.json \
  --output "$workspace/project/build/generated-limits/nullable-object-members" --package-name org.example.nullablemembers --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/nullable-object-reference-members.json \
  --output "$workspace/project/build/generated-limits/nullable-object-reference-members" --package-name org.example.nullablerefmembers --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/nullable-object-reference-members.json \
  --output "$workspace/project/build/generated-limits/nullable-object-reference-members" --package-name org.example.nullablerefmembers --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/model-defaults.json \
  --output "$workspace/project/build/generated-limits/model-defaults" --package-name org.example.defaults --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/model-defaults.json \
  --output "$workspace/project/build/generated-limits/model-defaults" --package-name org.example.defaults --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/enum-defaults.json \
  --output "$workspace/project/build/generated-limits/enum-defaults" --package-name org.example.enumdefaults --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/enum-defaults.json \
  --output "$workspace/project/build/generated-limits/enum-defaults" --package-name org.example.enumdefaults --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/json-unique.json \
  --output "$workspace/project/build/generated-limits/json-unique" --package-name org.example.jsonunique --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/json-unique.json \
  --output "$workspace/project/build/generated-limits/json-unique" --package-name org.example.jsonunique --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/inline-errors.json \
  --output "$workspace/project/build/generated-limits/inline-errors" --package-name org.example.inlineerrors --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/inline-errors.json \
  --output "$workspace/project/build/generated-limits/inline-errors" --package-name org.example.inlineerrors --model-name-suffix Dto --check
exclusion_flags=(--merge-identical-models --validation --hoist-shared-shapes --deduplicate-models --lock-hoisted-names --qualify-hoisted-names --runtime-package org.example.exclusions.runtime --exclude-merge-model-pattern 'org\.example\.exclusions\.models\.ZetaDto' --exclude-merge-model-suffix DtoDto)
"$workspace/openapi-kt" generate --input tests/fixtures/http/merge-exclusions.json \
  --output "$workspace/project/build/generated-limits/merge-exclusions" --package-name org.example.exclusions --model-name-suffix Dto "${exclusion_flags[@]}"
"$workspace/openapi-kt" generate --input tests/fixtures/http/merge-exclusions.json \
  --output "$workspace/project/build/generated-limits/merge-exclusions" --package-name org.example.exclusions --model-name-suffix Dto "${exclusion_flags[@]}" --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/native-enum-validation.json \
  --output "$workspace/project/build/generated-limits/native-enum-validation" --package-name org.example.nativeenumchecks --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/native-enum-validation.json \
  --output "$workspace/project/build/generated-limits/native-enum-validation" --package-name org.example.nativeenumchecks --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/native-enum-consumers.json \
  --output "$workspace/project/build/generated-limits/native-enum-consumers" --package-name org.example.nativeenumconsumers --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/native-enum-consumers.json \
  --output "$workspace/project/build/generated-limits/native-enum-consumers" --package-name org.example.nativeenumconsumers --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/native-enum-bounds.json \
  --output "$workspace/project/build/generated-limits/native-enum-bounds" --package-name org.example.nativeenumbounds --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/native-enum-bounds.json \
  --output "$workspace/project/build/generated-limits/native-enum-bounds" --package-name org.example.nativeenumbounds --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/internal-native-enums.json \
  --output "$workspace/project/build/generated-limits/internal-native-enums" --package-name org.example.internalnativeenums --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/internal-native-enums.json \
  --output "$workspace/project/build/generated-limits/internal-native-enums" --package-name org.example.internalnativeenums --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/named-native-enums.json \
  --output "$workspace/project/build/generated-limits/named-native-enums" --package-name org.example.namednativeenums --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/named-native-enums.json \
  --output "$workspace/project/build/generated-limits/named-native-enums" --package-name org.example.namednativeenums --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/nullable-native-enums.json \
  --output "$workspace/project/build/generated-limits/nullable-native-enums" --package-name org.example.nullablenativeenums --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/nullable-native-enums.json \
  --output "$workspace/project/build/generated-limits/nullable-native-enums" --package-name org.example.nullablenativeenums --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/native-enum-null-members.json \
  --output "$workspace/project/build/generated-limits/native-enum-null-members" --package-name org.example.nativeenumnullmembers --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/native-enum-null-members.json \
  --output "$workspace/project/build/generated-limits/native-enum-null-members" --package-name org.example.nativeenumnullmembers --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/string-enum-null-members.json \
  --output "$workspace/project/build/generated-limits/string-enum-null-members" --package-name org.example.stringenumnullmembers --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/string-enum-null-members.json \
  --output "$workspace/project/build/generated-limits/string-enum-null-members" --package-name org.example.stringenumnullmembers --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-allof-bounds.json \
  --output "$workspace/project/build/generated-limits/referenced-allof-bounds" --package-name org.example.referencedbounds --api-client-name ReferencedBounds --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-allof-bounds.json \
  --output "$workspace/project/build/generated-limits/referenced-allof-bounds" --package-name org.example.referencedbounds --api-client-name ReferencedBounds --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-allof-counts.json \
  --output "$workspace/project/build/generated-limits/referenced-allof-counts" --package-name org.example.referencedcounts --api-client-name ReferencedCounts --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-allof-counts.json \
  --output "$workspace/project/build/generated-limits/referenced-allof-counts" --package-name org.example.referencedcounts --api-client-name ReferencedCounts --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/null-only-defaults.json \
  --output "$workspace/project/build/generated-limits/explicit-null-literals" --package-name org.example.nullliterals --api-client-name NullLiterals --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/null-only-defaults.json \
  --output "$workspace/project/build/generated-limits/explicit-null-literals" --package-name org.example.nullliterals --api-client-name NullLiterals --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/structured-const-fields.json \
  --output "$workspace/project/build/generated-limits/structured-const-fields" --package-name org.example.structuredconst --api-client-name StructuredConsts --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/structured-const-fields.json \
  --output "$workspace/project/build/generated-limits/structured-const-fields" --package-name org.example.structuredconst --api-client-name StructuredConsts --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/structured-const-values.json \
  --output "$workspace/project/build/generated-limits/structured-const-values" --package-name org.example.structuredvalues --api-client-name StructuredValues --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/structured-const-values.json \
  --output "$workspace/project/build/generated-limits/structured-const-values" --package-name org.example.structuredvalues --api-client-name StructuredValues --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/nullable-structured-const.json \
  --output "$workspace/project/build/generated-limits/nullable-structured-const" --package-name org.example.nullablestructured --api-client-name NullableStructured --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/nullable-structured-const.json \
  --output "$workspace/project/build/generated-limits/nullable-structured-const" --package-name org.example.nullablestructured --api-client-name NullableStructured --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/primitive-array-defaults.json \
  --output "$workspace/project/build/generated-limits/primitive-array-defaults" --package-name org.example.arraydefaults --api-client-name ArrayDefaults --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/primitive-array-defaults.json \
  --output "$workspace/project/build/generated-limits/primitive-array-defaults" --package-name org.example.arraydefaults --api-client-name ArrayDefaults --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/checked-array-defaults.json \
  --output "$workspace/project/build/generated-limits/checked-array-defaults" --package-name org.example.checkedarrays --api-client-name CheckedArrays --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/checked-array-defaults.json \
  --output "$workspace/project/build/generated-limits/checked-array-defaults" --package-name org.example.checkedarrays --api-client-name CheckedArrays --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/checked-map-defaults.json \
  --output "$workspace/project/build/generated-limits/primitive-map-defaults" --package-name org.example.mapdefaults --api-client-name MapDefaults --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/checked-map-defaults.json \
  --output "$workspace/project/build/generated-limits/primitive-map-defaults" --package-name org.example.mapdefaults --api-client-name MapDefaults --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/native-enum-collection-defaults.json \
  --output "$workspace/project/build/generated-limits/native-enum-collection-defaults" --package-name org.example.enumcollections --api-client-name EnumCollections --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/native-enum-collection-defaults.json \
  --output "$workspace/project/build/generated-limits/native-enum-collection-defaults" --package-name org.example.enumcollections --api-client-name EnumCollections --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/string-enum-collection-bounds.json \
  --output "$workspace/project/build/generated-limits/string-enum-collection-bounds" --package-name org.example.stringenumbounds --api-client-name StringEnumBounds --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/string-enum-collection-bounds.json \
  --output "$workspace/project/build/generated-limits/string-enum-collection-bounds" --package-name org.example.stringenumbounds --api-client-name StringEnumBounds --model-name-suffix Dto --validation --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/string-enum-collection-defaults.json \
  --output "$workspace/project/build/generated-limits/string-enum-collection-defaults" --package-name org.example.stringenumcollections --api-client-name StringEnumCollections --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/string-enum-collection-defaults.json \
  --output "$workspace/project/build/generated-limits/string-enum-collection-defaults" --package-name org.example.stringenumcollections --api-client-name StringEnumCollections --model-name-suffix Dto --check
for prefix in org frame input requestBuilder value client basePath json httpClient shadows; do
  "$workspace/openapi-kt" generate --input tests/fixtures/http/namespace-collisions.json \
    --output "$workspace/project/build/generated-limits/http-namespaces-$prefix" --package-name "$prefix.example.namespaces" --model-name-suffix Dto
  "$workspace/openapi-kt" generate --input tests/fixtures/http/namespace-collisions.json \
    --output "$workspace/project/build/generated-limits/http-namespaces-$prefix" --package-name "$prefix.example.namespaces" --model-name-suffix Dto --check
done
"$workspace/openapi-kt" generate --input tests/fixtures/http/servers.json \
  --output "$workspace/project/build/generated-limits/http-servers" --package-name org.example.servers --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/servers.json \
  --output "$workspace/project/build/generated-limits/http-servers" --package-name org.example.servers --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/nested-builtin-collection-defaults.json \
  --output "$workspace/project/build/generated-limits/nested-builtin-collection-defaults" --package-name org.example.nesteddefaults --api-client-name NestedDefaults --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/nested-builtin-collection-defaults.json \
  --output "$workspace/project/build/generated-limits/nested-builtin-collection-defaults" --package-name org.example.nesteddefaults --api-client-name NestedDefaults --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/nested-temporal-collection-defaults.json \
  --output "$workspace/project/build/generated-limits/nested-temporal-collection-defaults" --package-name org.example.nestedtemporal --api-client-name NestedTemporal --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/nested-temporal-collection-defaults.json \
  --output "$workspace/project/build/generated-limits/nested-temporal-collection-defaults" --package-name org.example.nestedtemporal --api-client-name NestedTemporal --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/nested-string-enum-collection-defaults.json \
  --output "$workspace/project/build/generated-limits/nested-string-enum-collection-defaults" --package-name org.example.nestedstringenum --api-client-name NestedStringEnum --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/nested-string-enum-collection-defaults.json \
  --output "$workspace/project/build/generated-limits/nested-string-enum-collection-defaults" --package-name org.example.nestedstringenum --api-client-name NestedStringEnum --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/nested-native-enum-collection-defaults.json \
  --output "$workspace/project/build/generated-limits/nested-native-enum-collection-defaults" --package-name org.example.nestednativeenum --api-client-name NestedNativeEnum --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/nested-native-enum-collection-defaults.json \
  --output "$workspace/project/build/generated-limits/nested-native-enum-collection-defaults" --package-name org.example.nestednativeenum --api-client-name NestedNativeEnum --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-collection-field-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-collection-field-defaults" --package-name org.example.referenceddefaults --api-client-name ReferencedDefaults --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-collection-field-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-collection-field-defaults" --package-name org.example.referenceddefaults --api-client-name ReferencedDefaults --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-scalar-field-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-scalar-field-defaults" --package-name org.example.refscalars --api-client-name ReferencedScalars --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-scalar-field-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-scalar-field-defaults" --package-name org.example.refscalars --api-client-name ReferencedScalars --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-temporal-scalar-field-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-temporal-scalar-field-defaults" --package-name org.example.reftemporal --api-client-name ReferencedTemporal --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-temporal-scalar-field-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-temporal-scalar-field-defaults" --package-name org.example.reftemporal --api-client-name ReferencedTemporal --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-temporal-collection-field-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-temporal-collection-field-defaults" --package-name org.example.referencedtemporaldefaults --api-client-name ReferencedTemporalDefaults --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-temporal-collection-field-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-temporal-collection-field-defaults" --package-name org.example.referencedtemporaldefaults --api-client-name ReferencedTemporalDefaults --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-string-enum-collection-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-string-enum-collection-defaults" --package-name org.example.refstringdefaults --api-client-name RefStringDefaults --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-string-enum-collection-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-string-enum-collection-defaults" --package-name org.example.refstringdefaults --api-client-name RefStringDefaults --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-native-enum-collection-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-native-enum-collection-defaults" --package-name org.example.refnativedefaults --api-client-name RefNativeDefaults --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-native-enum-collection-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-native-enum-collection-defaults" --package-name org.example.refnativedefaults --api-client-name RefNativeDefaults --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-native-enum-scalar-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-native-enum-scalar-defaults" --package-name org.example.refnativescalar --api-client-name RefNativeScalar --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-native-enum-scalar-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-native-enum-scalar-defaults" --package-name org.example.refnativescalar --api-client-name RefNativeScalar --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-string-enum-scalar-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-string-enum-scalar-defaults" --package-name org.example.refstringscalar --api-client-name RefStringScalar --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-string-enum-scalar-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-string-enum-scalar-defaults" --package-name org.example.refstringscalar --api-client-name RefStringScalar --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-string-enum-null-member-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-string-enum-null-member-defaults" --package-name org.example.refnullstrings --api-client-name RefNullStrings --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-string-enum-null-member-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-string-enum-null-member-defaults" --package-name org.example.refnullstrings --api-client-name RefNullStrings --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/direct-nullable-component-counts.json \
  --output "$workspace/project/build/generated-limits/direct-nullable-component-counts" --package-name org.example.directcounts --api-client-name DirectCounts --model-name-suffix Dto --validation
"$workspace/openapi-kt" generate --input tests/fixtures/http/direct-nullable-component-counts.json \
  --output "$workspace/project/build/generated-limits/direct-nullable-component-counts" --package-name org.example.directcounts --api-client-name DirectCounts --model-name-suffix Dto --validation --check
for contract in bytes-field-defaults bytes-component-defaults reference-wrapper-component-defaults hierarchy-inline-members; do
  case "$contract" in
    bytes-field-defaults) pkg=org.example.bytesfielddefaults; client=BytesFieldDefaults ;;
    bytes-component-defaults) pkg=org.example.bytescomponentdefaults; client=BytesComponentDefaults ;;
    reference-wrapper-component-defaults) pkg=org.example.refwrapper; client=RefWrapper ;;
    hierarchy-inline-members) pkg=org.example.inlinehier; client=InlineHier ;;
  esac
  "$workspace/openapi-kt" generate --input "tests/fixtures/http/$contract.json" \
    --output "$workspace/project/build/generated-limits/$contract" --package-name "$pkg" --api-client-name "$client" --model-name-suffix Dto
  "$workspace/openapi-kt" generate --input "tests/fixtures/http/$contract.json" \
    --output "$workspace/project/build/generated-limits/$contract" --package-name "$pkg" --api-client-name "$client" --model-name-suffix Dto --check
done
"$workspace/openapi-kt" generate --input tests/fixtures/http/nested-object-defaults.json \
  --output "$workspace/project/build/generated-limits/nested-object-defaults" --package-name org.example.nestedobject --api-client-name NestedObject --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/nested-object-defaults.json \
  --output "$workspace/project/build/generated-limits/nested-object-defaults" --package-name org.example.nestedobject --api-client-name NestedObject --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-object-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-object-defaults" --package-name org.example.refobject --api-client-name RefObject --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-object-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-object-defaults" --package-name org.example.refobject --api-client-name RefObject --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/nested-object-collection-defaults.json \
  --output "$workspace/project/build/generated-limits/nested-object-collection-defaults" --package-name org.example.colobject --api-client-name ColObject --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/nested-object-collection-defaults.json \
  --output "$workspace/project/build/generated-limits/nested-object-collection-defaults" --package-name org.example.colobject --api-client-name ColObject --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-object-collection-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-object-collection-defaults" --package-name org.example.refcol --api-client-name RefCol --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/referenced-object-collection-defaults.json \
  --output "$workspace/project/build/generated-limits/referenced-object-collection-defaults" --package-name org.example.refcol --api-client-name RefCol --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/union-field-defaults.json \
  --output "$workspace/project/build/generated-limits/union-field-defaults" --package-name org.example.uniondef --api-client-name UnionDef --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/union-field-defaults.json \
  --output "$workspace/project/build/generated-limits/union-field-defaults" --package-name org.example.uniondef --api-client-name UnionDef --model-name-suffix Dto --check
"$workspace/openapi-kt" generate --input tests/fixtures/http/raw-json-field-defaults.json \
  --output "$workspace/project/build/generated-limits/raw-json-field-defaults" --package-name org.example.rawjson --api-client-name RawJson --model-name-suffix Dto
"$workspace/openapi-kt" generate --input tests/fixtures/http/raw-json-field-defaults.json \
  --output "$workspace/project/build/generated-limits/raw-json-field-defaults" --package-name org.example.rawjson --api-client-name RawJson --model-name-suffix Dto --check
if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
  echo "PREPARED HTTP native generation/check: $workspace/project"
  exit 0
fi
"${GRADLE:-gradle}" --offline --no-daemon -p "$workspace/project" runHttpContracts "$@"
printf 'Fresh native Go CLI HTTP project: %s\n' "$workspace/project"
