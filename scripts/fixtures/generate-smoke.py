#!/usr/bin/env python3
"""Generate each successful JSON fixture for one shared Kotlin compilation."""
import json
import os
from pathlib import Path
import re
import shutil
import subprocess

ROOT = Path(__file__).resolve().parents[2]
FIXTURES = ROOT / "tests" / "fixtures" / "p1"
BUILD = ROOT / "samples" / "kotlin-smoke" / "build"


def generator_command():
    binary = os.environ.get("OPENAPI_KT_BINARY")
    if binary:
        return [binary]
    return [os.environ.get("GO", "go"), "run", "./cmd/openapi-kt"]


def main():
    build = Path(os.environ.get("OPENAPI_KT_SMOKE_BUILD", BUILD)).resolve()
    if not (build.is_relative_to((ROOT / ".pi" / "go-migration").resolve()) or build.is_relative_to(BUILD.resolve())):
        raise ValueError("Smoke output must stay in the repository verification/build directories")
    corpus = json.loads((FIXTURES / "cases.json").read_text())
    if corpus["version"] != 1:
        raise ValueError("Unsupported fixture corpus version")
    active = []
    for case in corpus["cases"]:
        if case["expect"]["kind"] != "success":
            continue
        case_id = case["id"]
        if not re.fullmatch(r"[A-Za-z0-9-]+", case_id) or case_id in active:
            raise ValueError("Invalid or duplicate fixture identifier")
        source = (FIXTURES / case["input"]).resolve()
        source.relative_to(FIXTURES.resolve())
        name = case.get("source_name", "input.json")
        if Path(name).name != name or name in (".", "..") or "\\" in name:
            raise ValueError("Fixture source name must be one filename")
        input_path = build / "fixture-inputs" / case_id / name
        input_path.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(source, input_path)
        output_path = build / "fixture-generated" / case_id
        subprocess.run(generator_command() + [
            "generate", "--input", str(input_path), "--output", str(output_path),
            "--package-name", case["package_name"], "--model-name-suffix", case["model_name_suffix"],
            "--input-format", case["input_format"],
        ], cwd=ROOT, check=True, env={**os.environ, "GOTOOLCHAIN": "local", "GOPROXY": "off", "GOSUMDB": "off"})
        active.append(case_id)
    # Compile only current cases, even if old build directories still exist.
    (build / "fixture-source-roots.txt").write_text("\n".join(active) + "\n")
    print(f"Generated {len(active)} positive fixture inputs for Kotlin compilation.")


if __name__ == "__main__":
    main()
