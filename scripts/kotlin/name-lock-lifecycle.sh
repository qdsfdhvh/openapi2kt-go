#!/usr/bin/env bash
# Standalone D09 direct/version3 name-lock lifecycle and Kotlin wire contracts.
# --prepare-only performs native CLI/byte checks and stages projects without JVMs.
set -euo pipefail
# shellcheck source=scripts/kotlin/common.sh
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
prepare_only=false
if [[ $# -eq 1 && "$1" == --prepare-only ]]; then
  prepare_only=true
elif [[ $# -ne 0 ]]; then
  printf 'Usage: bash scripts/kotlin/name-lock-lifecycle.sh [--prepare-only]\n' >&2
  exit 2
fi
mkdir -p build
workspace="$(mktemp -d "$root/build/name-lock-lifecycle-XXXXXX")"
# Always build this checkout; OPENAPI_KT_BINARY must not select stale output.
openapi_kt_build_cli "$workspace/openapi-kt"
python3 - "$root" "$workspace" <<'PY'
from pathlib import Path
import hashlib
import json
import re
import shutil
import stat
import subprocess
import sys

root, workspace = map(Path, sys.argv[1:])
cli = workspace / "openapi-kt"
fixture = root / "tests/fixtures/name-lock-lifecycle"
package = "org.example.namelock"
kotlin_dir = package.replace(".", "/")
profile = {"package": package, "suffix": "Dto", "qualified": True}
metadata = ".openapi-kt-names.json"
manifest_name = ".openapi-kt-manifest.json"


def require(condition, message):
    if not condition:
        raise AssertionError(message)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def tree(directory):
    # Include directory entries, modes and mtimes: --check must not create a
    # lock/stage or rewrite bytes/metadata, even to the same content.
    result = {}
    for path in [directory, *sorted(directory.rglob("*"))]:
        info = path.lstat()
        require(not path.is_symlink(), f"Unexpected symlink: {path}")
        result[path.relative_to(directory).as_posix()] = (
            stat.S_IFMT(info.st_mode), stat.S_IMODE(info.st_mode), info.st_mtime_ns,
            sha(path.read_bytes()) if path.is_file() else None,
        )
    return result


def kotlin(directory):
    return {p.relative_to(directory).as_posix(): p.read_bytes()
            for p in sorted(directory.rglob("*.kt"))}


def invoke(output, input_name, style, label, *, locked=True, action=None,
           check=False, expected=0):
    args = [str(cli), "generate", "--input", str(fixture / input_name),
            "--output", str(output), "--package-name", package,
            "--model-name-suffix", "Dto", "--model-style", style,
            "--hoist-shared-shapes", "--qualify-hoisted-names",
            "--deduplicate-models", f"--lock-hoisted-names={str(locked).lower()}",
            "--diagnostic-format=json"]
    if action:
        args += ["--name-lock-action", action]
    if check:
        args += ["--check"]
    prior = tree(output) if check else None
    result = subprocess.run(args, capture_output=True, check=False)
    (workspace / f"{style}-{label}.json").write_bytes(result.stdout)
    require(result.returncode == expected,
            f"{label}: expected exit {expected}, got {result.returncode}\n"
            f"{result.stdout.decode()}\n{result.stderr.decode()}")
    report = json.loads(result.stdout)
    require(report["status"] == ("different" if expected == 3 else "ok")
            and not report["diagnostics"], f"{label}: unexpected result {report}")
    if check:
        require(tree(output) == prior, f"{label}: --check modified output")
    if action:
        require(len(report.get("name_lock_migrations", [])) == 1,
                f"{label}: missing single direct migration report")
    else:
        require("name_lock_migrations" not in report,
                f"{label}: ordinary report gained migration metadata")
    return report


def locked_state(output):
    files = kotlin(output)
    manifest = json.loads((output / manifest_name).read_bytes())
    state_bytes = (output / metadata).read_bytes()
    state = json.loads(state_bytes)
    require(manifest == {"version": 3, "files": {p: sha(b) for p, b in files.items()},
                         "name_lock": sha(state_bytes)}, "Invalid version3/hash ownership")
    require(state["version"] == 1 and state["profile"] == profile,
            "Lifecycle changed state version/package/suffix/qualification")
    require({e["role"] for e in state["entries"]} == {"ModeDto", "ResponseDto"}
            and len(state["entries"]) == 2, "Fixture did not hoist enum and owned DTO tree")
    return state


def symbols(state):
    return [{"role": e["role"], "shape": e["shape"],
             "classifier": f"{package}.models.{e['name']}",
             "descriptor": f"{package}.models.{e['name']}",
             "occurrences": e["occurrences"]} for e in state["entries"]]


def occurrences(inventory):
    result = {}
    for symbol in inventory:
        require(re.fullmatch(r"[0-9a-f]{64}", symbol["shape"]) is not None,
                "Invalid shape fingerprint")
        require(symbol["occurrences"] == sorted(set(symbol["occurrences"])),
                "Unsorted/duplicate occurrences")
        for occurrence in symbol["occurrences"]:
            key = (symbol["role"], occurrence)
            require(key not in result, "Repeated role/occurrence")
            result[key] = symbol
    return result


def verify_migration(report, action, before_state, after_state, before, after):
    migration = report["name_lock_migrations"][0]
    require(migration["action"] == action and not migration.get("spec"),
            "Wrong direct action/owner")
    require(migration["before"] == symbols(before_state)
            and migration["after"] == symbols(after_state),
            "Reported symbols differ from persisted/fresh allocations")
    require(isinstance(migration["impact"], str) and migration["impact"].strip(),
            "Missing migration impact")
    old, new = occurrences(migration["before"]), occurrences(migration["after"])
    changes = []
    for role, pointer in sorted(old.keys() | new.keys()):
        b, a = old.get((role, pointer), {}), new.get((role, pointer), {})
        if all(b.get(field) == a.get(field) for field in ("classifier", "descriptor", "shape")):
            continue
        change = {"role": role, "occurrence": pointer,
                  "shape_changed": b.get("shape") != a.get("shape")}
        for field, value in (("before", b.get("classifier")), ("after", a.get("classifier")),
                             ("descriptor_before", b.get("descriptor")),
                             ("descriptor_after", a.get("descriptor"))):
            if value:
                change[field] = value
        changes.append(change)
    require(migration["changes"] == changes, "Per-role/occurrence/descriptor changes differ")
    files = []
    for path in sorted(before.keys() | after.keys()):
        b, a = before.get(path), after.get(path)
        if b == a:
            continue
        change = {"path": path}
        if b is not None:
            change["before"] = sha(b)
        if a is not None:
            change["after"] = sha(a)
        files.append(change)
    require(migration["files"] == files, "Reported files/hashes differ from actual generated bytes")
    return migration


def review_sources(output, inventory, expanded, style):
    files = kotlin(output)
    by_role = {s["role"]: s for s in inventory}
    for role, symbol in by_role.items():
        classifier = symbol["classifier"]
        require(classifier == symbol["descriptor"], "Unexpected merged canonical descriptor")
        name = classifier.rsplit(".", 1)[1]
        require(re.fullmatch(r"[A-Za-z][A-Za-z0-9.]*", classifier) is not None,
                "Unsafe reported classifier")
        path = kotlin_dir + "/models/" + name + ".kt"
        require(path in files, f"Missing canonical file {path}")
        source = files[path].decode()
        declaration = ("sealed class " if role == "ModeDto" else
                       "class " if style == "plain_class" else "data class ") + name
        require("package " + package + ".models" in source and declaration in source,
                f"Unexpected canonical declaration {path}")
        for operation in (["Aardvark"] if expanded else []) + ["Beta", "Gamma"]:
            alias = operation + role
            target = kotlin_dir + "/api/models/" + alias + ".kt"
            require(target in files and ("public typealias " + alias + " = " + classifier)
                    in files[target].decode(), f"Wrong root alias target {target}")
    response = files[kotlin_dir + "/models/" + by_role["ResponseDto"]["classifier"].rsplit(".", 1)[1] + ".kt"].decode()
    require("class Child" in response and "class Detail" in response
            and "sealed class Status" in response, "Owned nested tree was not emitted")
    require(any(p.startswith(kotlin_dir + "/api/") and "suspend fun beta(" in b.decode()
                and "suspend fun gamma(" in b.decode() for p, b in files.items()),
            "Actual HTTP client methods are missing")


def stage_runtime(project, inventory, phase, style, expanded):
    dest = project / "src/main/kotlin/org/example/contracts"
    dest.mkdir(parents=True)
    sample = root / "samples/kotlin-http-smoke"
    for name in ("build.gradle.kts", "settings.gradle.kts"):
        shutil.copy2(sample / name, project / name)
    shutil.copy2(sample / "src/main/kotlin/org/example/contracts/NameLockLifecycleContracts.kt", dest)
    by_role = {s["role"]: s for s in inventory}
    response, mode = by_role["ResponseDto"], by_role["ModeDto"]
    extra_imports = ""
    extra_checks = ""
    extra_response = ""
    if expanded:
        extra_imports = f"import {package}.api.models.AardvarkResponseDto\nimport {package}.api.models.AardvarkModeDto\n"
        extra_checks = f'''    check(AardvarkResponseDto(id = "record", child = child()) == BetaResponseDto(id = "record", child = child()))
    check(AardvarkResponseDto.serializer().descriptor.serialName == "{response['descriptor']}")
    check(AardvarkModeDto.fromSerialName("turbo") == BetaModeDto.fromSerialName("turbo"))
    check(AardvarkModeDto.serializer().descriptor.serialName == "{mode['descriptor']}")
'''
        extra_response = ", api.aardvark(AardvarkModeDto.fromSerialName(selected.serialName))"
    data_checks = "" if style == "plain_class" else '''    check(BetaResponseDto(id = "record", child = child()).copy(id = "copy").id == "copy")
'''
    # Report-selected imports are compile-time canonical nested call sites. Never
    # edit/replace identifiers in generated Kotlin to make a consumer compile.
    main = f'''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlinx.coroutines.runBlocking
import {package}.NameLockLifecycleApiClient
import {package}.api.models.BetaResponseDto
import {package}.api.models.GammaResponseDto
import {package}.api.models.BetaModeDto
import {package}.api.models.GammaModeDto
import {response['classifier']} as SharedResponse
import {mode['classifier']} as SharedMode
{extra_imports}
private fun child() = SharedResponse.Child(
    status = SharedResponse.Child.Status.fromSerialName("future"),
    detail = SharedResponse.Child.Detail(enabled = true),
)

fun main() = runBlocking {{
    check(SharedMode.fromSerialName("fast") === SharedMode.FAST)
    check(SharedResponse.Child.Status.fromSerialName("ready") === SharedResponse.Child.Status.READY)
{extra_checks}{data_checks}    nameLockLifecycleContracts(
        phase = "{phase}", plain = {str(style == 'plain_class').lower()},
        probe = NameLockLifecycleProbe(
            response = {{ BetaResponseDto(id = "record", child = child()) }},
            peerResponse = {{ GammaResponseDto(id = "record", child = child()) }},
            responseCodec = BetaResponseDto.serializer(),
            peerResponseCodec = GammaResponseDto.serializer(),
            canonicalResponseCodec = SharedResponse.serializer(),
            responseClasses = listOf(BetaResponseDto::class.java, GammaResponseDto::class.java, SharedResponse::class.java),
            responseDescriptor = "{response['descriptor']}",
            nestedCodecs = listOf(
                SharedResponse.Child.serializer() to "{response['descriptor']}.Child",
                SharedResponse.Child.Status.serializer() to "{response['descriptor']}.Child.Status",
                SharedResponse.Child.Detail.serializer() to "{response['descriptor']}.Child.Detail",
            ),
            modeFromSerialName = {{ BetaModeDto.fromSerialName(it) }},
            peerModeFromSerialName = {{ GammaModeDto.fromSerialName(it) }},
            modeSerialName = {{ it.serialName }},
            modeCodec = BetaModeDto.serializer(), peerModeCodec = GammaModeDto.serializer(),
            modeClasses = listOf(BetaModeDto::class.java, GammaModeDto::class.java, SharedMode::class.java),
            modeDescriptor = "{mode['descriptor']}",
            paths = listOf("/beta", "/gamma"{', "/aardvark"' if expanded else ''}),
            invoke = {{ http, selected ->
                val api = NameLockLifecycleApiClient(http, "http://example.test/").default
                listOf(api.beta(selected), api.gamma(GammaModeDto.fromSerialName(selected.serialName)){extra_response})
            }},
        ),
    )
}}
'''
    (dest / "Main.kt").write_text(main)
    with (project / "build.gradle.kts").open("a") as build:
        build.write('''
tasks.register<JavaExec>("runNameLockLifecycleContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')


# The before/after fixtures differ only by adding the lexical earlier operation.
before_doc = json.loads((fixture / "kotlin-before.json").read_bytes())
after_doc = json.loads((fixture / "kotlin-after.json").read_bytes())
require(before_doc["paths"] == {p: value for p, value in after_doc["paths"].items()
                                 if p != "/aardvark"}, "Fixture drift outside new occurrence")
require(before_doc["paths"]["/beta"]["get"]["responses"] ==
        after_doc["paths"]["/aardvark"]["get"]["responses"], "New response changed shape")

for style in ("data_class", "plain_class"):
    output = workspace / (style + "-live")
    baseline = workspace / (style + "-baseline")
    reset_project = workspace / (style + "-reset")
    fresh = workspace / (style + "-fresh-control")
    invoke(output, "kotlin-before.json", style, "baseline")
    old_state = locked_state(output)
    old_files = kotlin(output)
    require({e["name"] for e in old_state["entries"]} == {"BetaResponseDto", "BetaModeDto"}
            and all(len(e["occurrences"]) == 2 for e in old_state["entries"]),
            "Baseline did not allocate qualified Beta names")
    review_sources(output, symbols(old_state), False, style)
    # Snapshot fresh baseline before adding the new operation; it is compiled in
    # its own project so old consumers are never compiled against reset sources.
    shutil.copytree(output, baseline / "build/generated")
    invoke(output, "kotlin-after.json", style, "expanded-locked")
    expanded_state = locked_state(output)
    expanded_files = kotlin(output)
    require([(e["role"], e["shape"], e["name"]) for e in old_state["entries"]] ==
            [(e["role"], e["shape"], e["name"]) for e in expanded_state["entries"]]
            and all(len(e["occurrences"]) == 3 for e in expanded_state["entries"]),
            "Ordinary locked expansion renamed/reallocated the retained shape")
    for path, data in old_files.items():
        if path.startswith(kotlin_dir + "/models/"):
            require(expanded_files[path] == data, f"Expansion changed retained shared bytes: {path}")
    review_sources(output, symbols(expanded_state), True, style)
    invoke(output, "kotlin-after.json", style, "expanded-locked-check", check=True)

    # Independent fresh allocation is the expected reset source/state oracle.
    invoke(fresh, "kotlin-after.json", style, "fresh-control")
    fresh_state, fresh_files = locked_state(fresh), kotlin(fresh)
    require({e["name"] for e in fresh_state["entries"]} == {"AardvarkResponseDto", "AardvarkModeDto"},
            "Fresh reset oracle did not select the lexical earlier representative")
    preview = invoke(output, "kotlin-after.json", style, "reset-check", action="reset", check=True, expected=3)
    reset_report = verify_migration(preview, "reset", expanded_state, fresh_state, expanded_files, fresh_files)
    require(len(reset_report["changes"]) == 6 and
            all(not change["shape_changed"] for change in reset_report["changes"]),
            "Reset should rename both roles for all three unchanged occurrences")
    applied = invoke(output, "kotlin-after.json", style, "reset", action="reset")
    require(applied["name_lock_migrations"][0] == reset_report, "Reset apply differs from readonly report")
    require(kotlin(output) == fresh_files and (output / metadata).read_bytes() == (fresh / metadata).read_bytes(),
            "Reset source/state bytes differ from independent fresh generation")
    require(locked_state(output) == fresh_state, "Reset did not preserve version3 ownership")
    review_sources(output, reset_report["after"], True, style)
    invoke(output, "kotlin-after.json", style, "ordinary-after-reset-check", check=True)
    deterministic = invoke(output, "kotlin-after.json", style, "reset-deterministic-check", action="reset", check=True)
    verify_migration(deterministic, "reset", fresh_state, fresh_state, fresh_files, fresh_files)
    shutil.copytree(output, reset_project / "build/generated")

    disable_preview = invoke(output, "kotlin-after.json", style, "disable-check", locked=False,
                             action="disable", check=True, expected=3)
    disable_report = verify_migration(disable_preview, "disable", fresh_state, fresh_state, fresh_files, fresh_files)
    require(not disable_report["changes"] and not disable_report["files"],
            "Disable after reset must change metadata only")
    require("Future generation" in disable_report["impact"], "Disable omitted future allocation impact")
    pre_disable = tree(output)
    disabled = invoke(output, "kotlin-after.json", style, "disable", locked=False, action="disable")
    require(disabled["name_lock_migrations"][0] == disable_report, "Disable apply differs from readonly report")
    require(not (output / metadata).exists() and kotlin(output) == fresh_files,
            "Disable retained owned state or changed Kotlin bytes")
    require(json.loads((output / manifest_name).read_bytes()) ==
            {"version": 1, "files": {p: sha(b) for p, b in fresh_files.items()}},
            "Disable did not commit ordinary version1 ownership")
    post_disable = tree(output)
    require(set(pre_disable) - set(post_disable) == {metadata}
            and not set(post_disable) - set(pre_disable), "Disable removed/created unexpected paths")
    require(all(pre_disable[p] == post_disable[p] for p in post_disable
                if p not in (".", manifest_name)), "Metadata-only disable rewrote Kotlin/other entries")
    invoke(output, "kotlin-after.json", style, "ordinary-unlocked-check", locked=False, check=True)
    stage_runtime(baseline, reset_report["before"], "baseline", style, False)
    stage_runtime(reset_project, reset_report["after"], "reset", style, True)
    print(f"NATIVE NAME-LOCK LIFECYCLE {style} PASS: reviewed generated bytes, reports, readonly checks, metadata-only disable", flush=True)
PY
for style in data_class plain_class; do
  for phase in baseline reset; do
    project="$workspace/$style-$phase"
    if [[ "$prepare_only" == true ]]; then
      printf 'JVM CHECK DEFERRED: %q --offline --no-daemon --no-build-cache --rerun-tasks -p %q runNameLockLifecycleContracts\n' "${GRADLE:-gradle}" "$project"
    else
      "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runNameLockLifecycleContracts
      printf 'FRESH NAME-LOCK LIFECYCLE %s %s PASS: %s\n' "$style" "$phase" "$project"
    fi
  done
done
printf 'Fresh native Go CLI name-lock lifecycle projects: %s\n' "$workspace"
