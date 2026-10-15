#!/usr/bin/env python3
"""Correctness-checked local CLI latency, RSS and binary-size baselines."""
import argparse
import hashlib
import json
import math
from pathlib import Path
import platform
import random
import re
import shutil
import statistics
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[2]


def synthetic(models, fields, seed):
    rng = random.Random(seed)
    schemas = {}
    for index in range(models):
        properties = {}
        required = []
        for column in range(fields):
            name = f"field{column:02}"
            choice = (index + column) % 8
            schema = [
                {"type": "string"}, {"type": "integer", "format": "int64"},
                {"type": "number", "format": "double"}, {"type": "boolean"},
                {"type": "array", "items": {"type": ["string", "null"]}},
                {"$ref": f"#/components/schemas/Model{(index - 1) % models:05}"},
                {"type": "object", "properties": {"value": {"type": "string"}, "enabled": {"type": "boolean"}}},
                {"type": "string", "format": "uuid"},
            ][choice]
            properties[name] = schema
            if rng.randrange(2):
                required.append(name)
        schemas[f"Model{index:05}"] = {"type": "object", "properties": properties, "required": required}
    return {"openapi": "3.1.0", "info": {"title": "Synthetic model benchmark", "version": "1"}, "paths": {}, "components": {"schemas": schemas}}


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def command(binary, source, output, mode):
    args = [str(binary), "validate" if mode == "validate" else "generate", "--input", str(source), "--input-format", "json", "--package-name", "org.example.benchmark", "--diagnostic-format", "json"]
    if mode != "validate":
        args += ["--output", str(output)]
        if mode == "check":
            args.append("--check")
    return args


def execute(args, expected_files):
    start = time.perf_counter_ns()
    completed = subprocess.run(args, capture_output=True, check=False)
    milliseconds = (time.perf_counter_ns() - start) / 1_000_000
    if completed.returncode != 0:
        raise RuntimeError("A failed command is not a generation benchmark: " + completed.stdout.decode(errors="replace"))
    report = json.loads(completed.stdout)
    if report["status"] != "ok" or report["diagnostics"] or len(report["files"]) != expected_files:
        raise RuntimeError("Benchmark output count/status did not match the workload")
    return milliseconds, report


def verify_tree(output, expected_files, baseline=None):
    manifest_bytes = (output / ".openapi-kt-manifest.json").read_bytes()
    manifest = json.loads(manifest_bytes)
    if manifest["version"] != 1 or len(manifest["files"]) != expected_files:
        raise RuntimeError("Incorrect output manifest")
    actual_paths = {p.relative_to(output).as_posix() for p in output.rglob("*.kt")}
    if actual_paths != set(manifest["files"]):
        raise RuntimeError("Generated Kotlin files differ from the manifest inventory")
    fingerprint = hashlib.sha256()
    total = 0
    states = {}
    for relative, expected in sorted(manifest["files"].items()):
        if Path(relative).is_absolute() or ".." in Path(relative).parts:
            raise RuntimeError("Unsafe generated path")
        file = output / relative
        data = file.read_bytes()
        actual = hashlib.sha256(data).hexdigest()
        if actual != expected:
            raise RuntimeError("Generated content does not match its manifest")
        fingerprint.update(relative.encode() + b"\0" + data + b"\0")
        total += len(data)
        states[relative] = (actual, file.stat().st_mtime_ns)
    states[".openapi-kt-manifest.json"] = (hashlib.sha256(manifest_bytes).hexdigest(), (output / ".openapi-kt-manifest.json").stat().st_mtime_ns)
    if baseline is not None and states != baseline:
        raise RuntimeError("No-op/check modified output contents or mtimes")
    return {"source_bytes": total, "files": expected_files, "fingerprint_sha256": fingerprint.hexdigest()}, states


def measure_rss(args, expected_files):
    if platform.system() not in ("Darwin", "Linux") or not Path("/usr/bin/time").exists():
        return None
    flag = "-l" if platform.system() == "Darwin" else "-v"
    completed = subprocess.run(["/usr/bin/time", flag, *args], capture_output=True, check=False)
    if completed.returncode != 0:
        raise RuntimeError("RSS run failed")
    report = json.loads(completed.stdout)
    if report["status"] != "ok" or len(report["files"]) != expected_files:
        raise RuntimeError("Invalid RSS workload")
    text = completed.stderr.decode(errors="replace")
    if platform.system() == "Darwin":
        match = re.search(r"(\d+)\s+maximum resident set size", text)
        return int(match.group(1)) if match else None
    match = re.search(r"Maximum resident set size \(kbytes\):\s*(\d+)", text)
    return int(match.group(1)) * 1024 if match else None


def workload(binary, source, root, iterations, warmup):
    document = json.loads(source.read_bytes())
    expected_files = len(document["components"]["schemas"])
    if not expected_files:
        raise ValueError("Benchmark needs non-empty model input")
    persistent = root / "persistent"
    execute(command(binary, source, persistent, "generate"), expected_files)
    output_summary, baseline = verify_tree(persistent, expected_files)
    result = {"input": str(source), "input_sha256": digest(source), "input_bytes": source.stat().st_size, "components": expected_files, "output": output_summary, "modes": {}}
    for mode in ["validate", "clean", "no-op", "check"]:
        samples = []
        for iteration in range(warmup + iterations):
            with tempfile.TemporaryDirectory(prefix="fresh-", dir=root) as fresh:
                output = Path(fresh) / "output" if mode == "clean" else persistent
                elapsed, _ = execute(command(binary, source, output, mode), expected_files)
                if mode != "validate":
                    summary, _ = verify_tree(output, expected_files, baseline if mode in ("no-op", "check") else None)
                    if summary != output_summary:
                        raise RuntimeError("Repeated generation is not deterministic")
                if iteration >= warmup:
                    samples.append(elapsed)
        with tempfile.TemporaryDirectory(prefix="rss-", dir=root) as fresh:
            output = Path(fresh) / "output" if mode == "clean" else persistent
            rss = measure_rss(command(binary, source, output, mode), expected_files)
            if rss is not None and mode != "validate":
                verify_tree(output, expected_files, baseline if mode in ("no-op", "check") else None)
        ordered = sorted(samples)
        result["modes"][mode] = {"samples_ms": samples, "median_ms": statistics.median(samples), "p95_ms": ordered[max(0, math.ceil(len(ordered) * .95) - 1)], "min_ms": min(samples), "max_ms": max(samples), "peak_rss_bytes_single_run": rss}
        print(json.dumps({"progress": {"input": source.name, "mode": mode, "median_ms": statistics.median(samples)}}), flush=True)
    if digest(source) != result["input_sha256"]:
        raise RuntimeError("Benchmark input changed while measuring")
    return result


def binary_sizes(binary, out):
    result = {"bytes": binary.stat().st_size, "sha256": digest(binary), "stripped_copy": None}
    strip = shutil.which("strip")
    if strip and platform.system() in ("Darwin", "Linux"):
        copy = out / "openapi-kt-stripped"
        shutil.copy2(binary, copy)
        flags = ["-S", "-x"] if platform.system() == "Darwin" else ["--strip-unneeded"]
        subprocess.run([strip, *flags, str(copy)], check=True, capture_output=True)
        # Stripping must leave an executable that still generates actual models.
        seed = out / "strip-check.json"
        seed.write_text(json.dumps(synthetic(2, 4, 0)))
        target = out / "strip-check-output"
        execute(command(copy, seed, target, "generate"), 2)
        verify_tree(target, 2)
        result["stripped_copy"] = {"bytes": copy.stat().st_size, "sha256": digest(copy), "flags": flags}
    return result


def version(command):
    if not shutil.which(command[0]):
        return None
    completed = subprocess.run(command, capture_output=True, text=True, check=False)
    return completed.stdout.strip() if completed.returncode == 0 else None


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, default=ROOT / "build/openapi-kt")
    parser.add_argument("--output", type=Path, default=ROOT / "local/benchmarks")
    parser.add_argument("--models", type=int, nargs="*", default=[100, 1000, 6000])
    parser.add_argument("--fields", type=int, default=16)
    parser.add_argument("--seed", type=int, default=17)
    parser.add_argument("--iterations", type=int, default=7)
    parser.add_argument("--warmup", type=int, default=2)
    parser.add_argument("--input", type=Path, action="append", default=[])
    args = parser.parse_args()
    if args.iterations < 1 or args.warmup < 0 or args.fields < 1 or args.fields > 100 or any(n < 1 for n in args.models):
        parser.error("Use positive bounded dimensions and at least one measured iteration")
    output = args.output.resolve()
    if not any(output.is_relative_to(ROOT / prefix) for prefix in ["local", "private", ".pi"]):
        parser.error("Raw benchmark results belong under ignored local/, private/ or .pi/")
    output.mkdir(parents=True, exist_ok=True)
    run = Path(tempfile.mkdtemp(prefix="run-", dir=output))
    binary = args.binary.resolve()
    sources = []
    for count in args.models:
        path = run / f"synthetic-{count}.json"
        path.write_text(json.dumps(synthetic(count, args.fields, args.seed), separators=(",", ":")))
        sources.append(path)
    sources.extend(p.resolve() for p in args.input)
    report = {"host": {"os": platform.system(), "release": platform.release(), "architecture": platform.machine(), "cpu": version(["sysctl", "-n", "machdep.cpu.brand_string"]) if platform.system() == "Darwin" else platform.processor()},
              "toolchain": {"go": version(["go", "version"]), "python": platform.python_version(), "binary_version": version([str(binary), "--version"])},
              "profile": "caller-built Go binary; optimization/trimpath/stripping flags are not inferred", "binary": binary_sizes(binary, run), "seed": args.seed,
              "fields_per_synthetic_component": args.fields, "warmup_per_mode": args.warmup, "iterations_per_mode": args.iterations,
              "timing_boundary": "subprocess launch through captured CLI completion; verification and cleanup excluded",
              "cache_policy": "warm OS cache; no OS cache flushing; clean means a new output directory, not cold disk",
              "rss_policy": "one additional successful /usr/bin/time run per mode; not a distribution",
              "mode_notes": {"validate": "read/parse/analyze/emit plus process startup, no output commit; not pure-core timing", "clean": "new output directory", "no-op": "existing identical output, hashes and ownership checked", "check": "read-only comparison"},
              "workloads": []}
    for index, source in enumerate(sources):
        case = run / f"workload-{index}"
        case.mkdir()
        report["workloads"].append(workload(binary, source, case, args.iterations, args.warmup))
        (run / "report.json").write_text(json.dumps(report, indent=2) + "\n")
    print("Report: " + str(run / "report.json"), flush=True)


if __name__ == "__main__":
    main()
