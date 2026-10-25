#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/reference-wrapper-component-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for validation in off on; do
    project="$workspace/$style-$validation"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferenceWrapperComponentDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/WrapperOwnedDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/WrapperChainNullabilityDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ObjectEnumOwnDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ObjectNativeEnumOwnDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    cp tests/fixtures/http/reference-wrapper-component-defaults.json "$project/input.json"
    flags=()
    [[ "$validation" == off ]] || flags+=(--validation)
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.refwrapper --api-client-name RefWrapper --model-name-suffix Dto --model-style "$style" "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.refwrapper --api-client-name RefWrapper --model-name-suffix Dto --model-style "$style" "${flags[@]}" --check
    "$workspace/openapi-kt" generate --input tests/fixtures/http/wrapper-chain-nullability-defaults.json --output "$project/build/generated-limits/wrapper-chain" --package-name org.example.wrapperchain --model-name-suffix Dto --model-style "$style" "${flags[@]}"
    "$workspace/openapi-kt" generate --input tests/fixtures/http/wrapper-chain-nullability-defaults.json --output "$project/build/generated-limits/wrapper-chain" --package-name org.example.wrapperchain --model-name-suffix Dto --model-style "$style" "${flags[@]}" --check
    "$workspace/openapi-kt" generate --input tests/fixtures/http/wrapper-owned-defaults.json --output "$project/build/generated-limits/wrapper-owned" --package-name org.example.owneddefaults --model-name-suffix Dto --model-style "$style" "${flags[@]}"
    "$workspace/openapi-kt" generate --input tests/fixtures/http/wrapper-owned-defaults.json --output "$project/build/generated-limits/wrapper-owned" --package-name org.example.owneddefaults --model-name-suffix Dto --model-style "$style" "${flags[@]}" --check
    if [[ "$validation" == on ]]; then
      cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/UUIDComponentDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
      "$workspace/openapi-kt" generate --input tests/fixtures/http/uuid-component-defaults.json --output "$project/build/generated-checked/uuid" --package-name org.example.uuiddefaults --model-name-suffix Dto --model-style "$style" --validation
      "$workspace/openapi-kt" generate --input tests/fixtures/http/uuid-component-defaults.json --output "$project/build/generated-checked/uuid" --package-name org.example.uuiddefaults --model-name-suffix Dto --model-style "$style" --validation --check
      cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NumericComponentDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
      "$workspace/openapi-kt" generate --input tests/fixtures/http/numeric-component-defaults.json --output "$project/build/generated-checked/numeric" --package-name org.example.numericdefaults --model-name-suffix Dto --model-style "$style" --validation
      "$workspace/openapi-kt" generate --input tests/fixtures/http/numeric-component-defaults.json --output "$project/build/generated-checked/numeric" --package-name org.example.numericdefaults --model-name-suffix Dto --model-style "$style" --validation --check
      cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ConstrainedComponentDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
      "$workspace/openapi-kt" generate --input tests/fixtures/http/constrained-component-defaults.json --output "$project/build/generated-checked/bounded" --package-name org.example.boundeddefaults --model-name-suffix Dto --model-style "$style" --validation
      "$workspace/openapi-kt" generate --input tests/fixtures/http/constrained-component-defaults.json --output "$project/build/generated-checked/bounded" --package-name org.example.boundeddefaults --model-name-suffix Dto --model-style "$style" --validation --check
      cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullableScalarWrapperDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
      "$workspace/openapi-kt" generate --input tests/fixtures/http/nullable-scalar-wrapper-defaults.json --output "$project/build/generated-checked/scalars" --package-name org.example.refwrapper --model-name-suffix Dto --model-style "$style" --validation
      "$workspace/openapi-kt" generate --input tests/fixtures/http/nullable-scalar-wrapper-defaults.json --output "$project/build/generated-checked/scalars" --package-name org.example.refwrapper --model-name-suffix Dto --model-style "$style" --validation --check
    fi
    for check in generate check; do
      check_flags=(); [[ "$check" == generate ]] || check_flags=(--check)
      "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated/optimized" --package-name org.example.optimizedrefwrapper --api-client-name RefWrapper --model-name-suffix Dto --model-style "$style" "${flags[@]}" --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names "${check_flags[@]}"
    done
    python3 - "$project" "$style" "$validation" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1])
consumer=(p/'src/main/kotlin/org/example/contracts/ObjectEnumOwnDefaultsContracts.kt').read_text()
(p/'src/main/kotlin/org/example/contracts/OptimizedObjectEnumOwnDefaultsContracts.kt').write_text(consumer.replace('org.example.refwrapper','org.example.optimizedrefwrapper').replace('objectEnumOwnDefaultsContracts','optimizedObjectEnumOwnDefaultsContracts'))
consumer=(p/'src/main/kotlin/org/example/contracts/ObjectNativeEnumOwnDefaultsContracts.kt').read_text()
(p/'src/main/kotlin/org/example/contracts/OptimizedObjectNativeEnumOwnDefaultsContracts.kt').write_text(consumer.replace('org.example.refwrapper','org.example.optimizedrefwrapper').replace('objectNativeEnumOwnDefaultsContracts','optimizedObjectNativeEnumOwnDefaultsContracts'))
copy='' if sys.argv[2]=='plain_class' else '''
 val value=org.example.refwrapper.models.HolderDto()
 check(value.copy(mode=org.example.refwrapper.models.ModeDto.B).mode==org.example.refwrapper.models.ModeDto.B)
 check(value.copy().rows==null)
 val adopted=value.copy(nested=org.example.refwrapper.models.`WrappedNestedDto default`(), raw=org.example.refwrapper.models.`WrappedRawDto default`())
 check(adopted.nested?.get(0)?.get("a")==2)
 check(adopted.copy(nested=null, raw=null).nested==null)
 check(adopted.copy(nested=null, raw=null).raw==null)
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking {
'''+copy+(' nullableScalarWrapperDefaultsContracts()\n constrainedComponentDefaultsContracts()\n numericComponentDefaultsContracts()\n uuidComponentDefaultsContracts()\n' if sys.argv[3]=='on' else '')+''' wrapperOwnedDefaultsContracts()
 wrapperChainNullabilityDefaultsContracts()
 referenceWrapperComponentDefaultsContracts()
 for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
  optimizedObjectEnumOwnDefaultsContracts(kotlinx.serialization.json.Json { encodeDefaults=defaults; explicitNulls=nulls })
  optimizedObjectNativeEnumOwnDefaultsContracts(kotlinx.serialization.json.Json { encodeDefaults=defaults; explicitNulls=nulls })
 }
 }
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
kotlin.sourceSets.main { kotlin.srcDir(layout.buildDirectory.dir("generated-checked")) }
tasks.register<JavaExec>("runReferenceWrapperComponentDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runReferenceWrapperComponentDefaultsContracts
    echo "FRESH REFERENCE WRAPPER DEFAULTS PROFILE $style-$validation PASS: $project"
  done
done
echo "Fresh reference wrapper default native projects: $workspace"
