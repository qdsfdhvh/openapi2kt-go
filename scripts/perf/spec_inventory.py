#!/usr/bin/env python3
"""Local-only structural inventory and model probes; never downloads a specification."""
import argparse
from collections import Counter, defaultdict
import hashlib
import json
import math
from pathlib import Path
import re
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[2]
METADATA = {"title", "description", "example", "examples", "externalDocs", "$comment", "deprecated"}
MAP_SCHEMAS = {"properties", "patternProperties", "$defs", "definitions", "dependentSchemas"}
LIST_SCHEMAS = {"allOf", "anyOf", "oneOf", "prefixItems"}
SINGLE_SCHEMAS = {"items", "additionalProperties", "additionalItems", "contains", "not", "if", "then", "else", "propertyNames", "unevaluatedProperties", "unevaluatedItems", "contentSchema"}
METHODS = {"get", "post", "put", "delete", "patch", "head", "options", "trace", "query"}


def pointer(parent, key):
    return parent + "/" + str(key).replace("~", "~0").replace("/", "~1")


def schema_nodes(document):
    nodes = []

    def schema(value, location):
        nodes.append((location, value))
        if not isinstance(value, dict):
            return
        for key, child in value.items():
            p = pointer(location, key)
            if key in MAP_SCHEMAS and isinstance(child, dict):
                for name, item in child.items():
                    schema(item, pointer(p, name))
            elif key in LIST_SCHEMAS and isinstance(child, list):
                for index, item in enumerate(child):
                    schema(item, pointer(p, index))
            elif key in SINGLE_SCHEMAS and isinstance(child, (dict, bool)):
                schema(child, p)

    def containers(value, location):
        if isinstance(value, dict):
            for key, child in value.items():
                p = pointer(location, key)
                if location == "/components" and key == "schemas" and isinstance(child, dict):
                    for name, item in child.items():
                        schema(item, pointer(p, name))
                elif key == "schema":
                    schema(child, p)
                elif key not in METADATA | {"default", "const", "enum"}:
                    containers(child, p)
        elif isinstance(value, list):
            for index, item in enumerate(value):
                containers(item, pointer(location, index))

    containers(document, "")
    return nodes


def features(schema):
    if isinstance(schema, bool):
        return {"boolean-schema:" + str(schema).lower()}
    if not isinstance(schema, dict):
        return {"invalid-schema-slot:" + type(schema).__name__}
    result = {"keyword:" + key for key in schema if key not in METADATA}
    kind = schema.get("type")
    if isinstance(kind, str):
        result.add("type:" + kind)
    elif isinstance(kind, list) and all(isinstance(t, str) for t in kind):
        result.add("type-array:" + ",".join(sorted(kind)))
    if isinstance(schema.get("format"), str):
        result.add("format:" + schema["format"])
    return result


def operation_features(document):
    """Inventory wire-facing constructs without claiming P1 implements them."""
    counts = Counter()

    def resolve(value):
        seen = set()
        while isinstance(value, dict) and isinstance(value.get("$ref"), str):
            ref = value["$ref"]
            if not ref.startswith("#/") or ref in seen:
                break
            seen.add(ref)
            target = document
            try:
                for token in ref[2:].split("/"):
                    target = target[token.replace("~1", "/").replace("~0", "~")]
            except (KeyError, TypeError):
                break
            value = target
        return value if isinstance(value, dict) else {}

    for container in ["paths", "webhooks"]:
        for item in document.get(container, {}).values():
            item = resolve(item)
            for method, operation in item.items():
                if method.lower() not in METHODS or not isinstance(operation, dict):
                    continue
                counts["method:" + method.lower()] += 1
                if operation.get("security") or document.get("security"):
                    counts["security"] += 1
                parameters = [*item.get("parameters", []), *operation.get("parameters", [])]
                for parameter in parameters:
                    parameter = resolve(parameter)
                    location = str(parameter.get("in", "unknown"))
                    counts["parameter:" + location] += 1
                    counts["parameter-required:" + str(bool(parameter.get("required"))).lower()] += 1
                    for key in ["style", "explode", "allowReserved", "allowEmptyValue"]:
                        if key in parameter:
                            counts["parameter-" + key + ":" + str(parameter[key]).lower()] += 1
                body = resolve(operation.get("requestBody", {}))
                if body:
                    counts["body-required:" + str(bool(body.get("required"))).lower()] += 1
                    for mime in body.get("content", {}):
                        counts["body-media:" + mime.lower()] += 1
                for status, response in operation.get("responses", {}).items():
                    counts["response-status:" + str(status)] += 1
                    response = resolve(response)
                    for mime in response.get("content", {}):
                        counts["response-media:" + mime.lower()] += 1
                    if response.get("headers"):
                        counts["response-headers"] += 1
    return dict(sorted(counts.items()))


def semantic_copy(schema):
    """Drop only descriptive schema metadata; preserve names, order and semantic values."""
    if not isinstance(schema, dict):
        return schema
    result = {}
    for key, value in schema.items():
        if key in METADATA:
            continue
        if key in MAP_SCHEMAS and isinstance(value, dict):
            value = {k: semantic_copy(v) for k, v in value.items()}
        elif key in LIST_SCHEMAS and isinstance(value, list):
            value = [semantic_copy(v) for v in value]
        elif key in SINGLE_SCHEMAS and isinstance(value, (dict, bool)):
            value = semantic_copy(value)
        result[key] = value
    return result


def encode(value):
    return json.dumps(value, ensure_ascii=False, separators=(",", ":")).encode()


def fixture_features():
    coverage = defaultdict(lambda: {"success": set(), "error": set()})
    corpus = json.loads((ROOT / "tests/fixtures/p1/cases.json").read_text())
    for case in corpus["cases"]:
        if case["expect"].get("code") == "E_JSON_INVALID":
            continue  # Syntax failures do not exercise any schema feature.
        try:
            document = json.loads((ROOT / "tests/fixtures/p1" / case["input"]).read_bytes())
        except (ValueError, UnicodeError):
            continue
        for location, schema in schema_nodes(document):
            for feature in features(schema):
                kind = case["expect"]["kind"]
                if kind == "error":
                    error_pointer = case["expect"]["pointer"]
                    family, value = feature.split(":", 1)
                    key = value if family == "keyword" else "format" if family == "format" else "type"
                    if error_pointer not in (location, pointer(location, key)):
                        continue  # An earlier unrelated failure is not feature coverage.
                coverage[feature][kind].add(case["id"])
    return {key: {kind: sorted(ids) for kind, ids in value.items()} for key, value in coverage.items()}


def dependency_closure(schema, components):
    required = {}
    pending = [schema]
    while pending:
        value = pending.pop()
        for _, node in schema_nodes({"schema": value}):
            ref = node.get("$ref") if isinstance(node, dict) else None
            if not isinstance(ref, str) or not ref.startswith("#/components/schemas/"):
                continue
            encoded = ref.removeprefix("#/components/schemas/")
            if "/" in encoded or "%" in encoded or re.search(r"~(?:[^01]|$)", encoded):
                continue
            name = encoded.replace("~1", "/").replace("~0", "~")
            if name in components and name not in required:
                required[name] = semantic_copy(components[name])
                pending.append(components[name])
    return required


def run_cli(binary, input_path, *, output=None, input_format="json"):
    command = [str(binary), "generate" if output else "validate", "--input", str(input_path), "--input-format", input_format, "--package-name", "org.example.localmodels", "--diagnostic-format", "json"]
    if output:
        command += ["--output", str(output)]
    result = subprocess.run(command, capture_output=True, check=False)
    if result.returncode not in (0, 1):
        raise RuntimeError("Unexpected CLI exit during a model probe: " + str(result.returncode))
    report = json.loads(result.stdout)
    if result.stderr:
        raise RuntimeError("Unexpected stderr in JSON diagnostic mode")
    return {"exit_code": result.returncode, "report": report}


def load_document(raw, suffix):
    def unique_pairs(pairs):
        result = {}
        for key, value in pairs:
            if key in result:
                raise ValueError("Duplicate JSON key in local input")
            result[key] = value
        return result

    def invalid_constant(_):
        raise ValueError("Non-finite JSON number in local input")

    def finite_float(value):
        number = float(value)
        if not math.isfinite(number):
            raise ValueError("Out-of-range JSON number in local input")
        return number

    try:
        return json.loads(raw, object_pairs_hook=unique_pairs, parse_constant=invalid_constant, parse_float=finite_float), "json"
    except (json.JSONDecodeError, UnicodeError):
        if suffix.lower() not in (".yaml", ".yml"):
            raise ValueError("Invalid JSON input; no YAML fallback for JSON files") from None
        import yaml  # Optional local conversion, not a claim of CLI YAML support.
        return yaml.safe_load(raw), "yaml-converted-locally"


def inspect(label, source, binary, out, corpus):
    raw = source.read_bytes()
    source_hash = hashlib.sha256(raw).hexdigest()
    document, source_format = load_document(raw, source.suffix)
    if not isinstance(document, dict):
        raise ValueError("Specification root must be an object")
    converted = out / (label + ".json")
    # This derivative is only for JSON model probes; retain original byte-size evidence.
    converted.write_bytes(encode(document))
    full = run_cli(binary, source, input_format="json" if source_format == "json" else "auto")
    converted_result = run_cli(binary, converted) if source_format != "json" else None
    nodes = schema_nodes(document)
    counts = Counter()
    representatives = {}
    for location, schema in nodes:
        counts.update(features(schema))
        canonical = semantic_copy(schema)
        digest = hashlib.sha256(encode(canonical)).hexdigest()
        entry = representatives.setdefault(digest, {"schema": canonical, "pointers": []})
        entry["pointers"].append(location)
    components = document.get("components", {}).get("schemas", {})
    probe_results = []
    codes = Counter()
    accepted_components = {}
    with tempfile.TemporaryDirectory(prefix="probes-", dir=out) as temp:
        probe_path = Path(temp) / "input.json"
        for index, (digest, entry) in enumerate(representatives.items()):
            dependencies = dependency_closure(entry["schema"], components)
            probe_name = "LocalProbeRoot"
            while probe_name in dependencies:
                probe_name += "Extra"
            dependencies[probe_name] = {"type": "object", "properties": {"value": entry["schema"]}, "required": ["value"]}
            probe = {"openapi": "3.1.0", "components": {"schemas": dependencies}}
            probe_path.write_bytes(encode(probe))
            result = run_cli(binary, probe_path)
            diagnostics = result["report"]["diagnostics"]
            code = diagnostics[0]["code"] if diagnostics else "OK"
            codes[code] += 1
            probe_results.append({"schema_sha256": digest, "source_pointers": entry["pointers"], "exit_code": result["exit_code"], "diagnostics": diagnostics})
            if index % 100 == 0:
                print(json.dumps({"progress": {"spec": label, "probes": index + 1, "total": len(representatives)}}), flush=True)
        # Preserve original component definitions and their full reference closure.
        component_results = {}
        for name, schema in components.items():
            definitions = dependency_closure(schema, components)
            definitions[name] = semantic_copy(schema)
            probe_path.write_bytes(encode({"openapi": "3.1.0", "components": {"schemas": definitions}}))
            result = run_cli(binary, probe_path)
            component_results[name] = result
            if result["exit_code"] == 0:
                accepted_components.update(definitions)
    subset = None
    if accepted_components:
        subset_input = out / (label + "-supported-components.json")
        subset_input.write_bytes(encode({"openapi": "3.1.0", "components": {"schemas": accepted_components}}))
        subset = {"input": str(subset_input), "component_count": len(accepted_components), "generation": run_cli(binary, subset_input, output=out / (label + "-models"))}
    coverage = {}
    for feature, occurrences in sorted(counts.items()):
        matches = corpus.get(feature, {"success": [], "error": []})
        coverage[feature] = {"occurrences": occurrences, "fixture_matches": matches, "status": "positive-fixture-exists" if matches["success"] else "rejection-fixture-only" if matches["error"] else "missing-fixture"}
    paths = document.get("paths", {})
    operations = sum(1 for path in paths.values() if isinstance(path, dict) for method in path if method.lower() in METHODS)
    if hashlib.sha256(source.read_bytes()).hexdigest() != source_hash:
        raise RuntimeError("Source changed during inspection; rerun against a stable input")
    wire_features = operation_features(document)
    report = {"label": label, "source": str(source), "binary_sha256": hashlib.sha256(binary.read_bytes()).hexdigest(), "source_sha256": source_hash, "source_bytes": len(raw), "source_format": source_format,
              "openapi": document.get("openapi"), "path_count": len(paths), "operation_count": operations, "component_count": len(components),
              "schema_occurrences": len(nodes), "distinct_semantic_probes": len(representatives), "whole_document": full, "locally_converted_json": converted_result,
              "probe_outcomes": dict(codes), "feature_coverage": coverage, "schema_probes": probe_results,
              "operation_features": {key: {"occurrences": count, "status": "deferred-not-wire-tested"} for key, count in wire_features.items()},
              "component_probes": component_results, "supported_component_subset": subset,
              "limits": ["Probes normalize the document envelope to OpenAPI 3.1 and place each schema in a required model property, importing its local component dependencies.",
                         "Probe acceptance is not request/response wire validation or whole-document compatibility.",
                         "Feature matches are atomic fixture associations, not proof that every feature combination or API operation is supported.",
                         "No unsupported schema keyword is removed; only descriptive metadata is omitted when grouping probes."]}
    (out / (label + "-report.json")).write_text(json.dumps(report, indent=2, ensure_ascii=False) + "\n")
    summary = {key: report[key] for key in ["label", "source_bytes", "source_format", "openapi", "path_count", "operation_count", "component_count", "schema_occurrences", "distinct_semantic_probes", "probe_outcomes"]}
    summary["whole_document_exit"] = full["exit_code"]
    summary["missing_fixture_features"] = [key for key, value in coverage.items() if value["status"] == "missing-fixture"]
    summary["generatable_components"] = len(accepted_components)
    summary["deferred_operation_feature_families"] = len(wire_features)
    print(json.dumps(summary), flush=True)
    return summary


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--spec", action="append", required=True, help="Local label=path; repeat for multiple inputs")
    parser.add_argument("--binary", type=Path, default=ROOT / "build/openapi-kt")
    parser.add_argument("--output", type=Path, default=ROOT / "local/spec-validation")
    args = parser.parse_args()
    out = args.output.resolve()
    if not any(out.is_relative_to(ROOT / prefix) for prefix in ["local", "private", ".pi"]):
        parser.error("Reports containing local names must stay under ignored local/, private/ or .pi/")
    out.mkdir(parents=True, exist_ok=True)
    corpus = fixture_features()
    summaries = []
    labels = set()
    for item in args.spec:
        label, source = item.split("=", 1)
        if not re.fullmatch(r"[A-Za-z0-9_-]+", label) or label in labels:
            parser.error("Labels must be unique simple identifiers")
        labels.add(label)
        summaries.append(inspect(label, Path(source).resolve(), args.binary.resolve(), out, corpus))
    (out / "summary.json").write_text(json.dumps(summaries, indent=2) + "\n")


if __name__ == "__main__":
    main()
