"""Unit checks for coverage extraction and benchmark correctness, not product support claims."""
import copy
import importlib.util
import tempfile
from pathlib import Path
import unittest

def load_tool(name):
    path = Path(__file__).resolve().parents[1] / "perf" / (name + ".py")
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


benchmark = load_tool("benchmark")
inventory = load_tool("spec_inventory")


class SchemaInventoryTests(unittest.TestCase):
    def test_json_load_never_hides_duplicate_keys_nonfinite_values_or_yaml(self):
        for raw in [b'{"key":1,"key":2}', b'{"value":NaN}', b'{"value":1e999}', b'openapi: 3.1.0']:
            with self.assertRaises(ValueError):
                inventory.load_document(raw, ".json")

    def test_examples_defaults_and_field_names_are_not_schema_containers(self):
        document = {"components": {"schemas": {"Record": {"type": "object", "properties": {
            "description": {"type": "string", "default": {"schema": {"oneOf": []}}},
            "value": {"type": "string", "example": {"schema": {"$ref": "outside.json"}}},
        }, "example": {"schema": {"type": "integer"}}}}}}
        nodes = inventory.schema_nodes(document)
        self.assertEqual([p for p, _ in nodes], ["/components/schemas/Record", "/components/schemas/Record/properties/description", "/components/schemas/Record/properties/value"])
        self.assertNotIn("keyword:oneOf", set().union(*(inventory.features(v) for _, v in nodes)))

    def test_all_schema_children_and_parameter_slots_are_visited(self):
        document = {"components": {"schemas": {"Record": {"allOf": [
            {"type": "object", "additionalProperties": False, "propertyNames": {"type": "string"}},
            {"type": "array", "prefixItems": [True, {"type": "integer"}], "items": False},
        ]}}}, "paths": {"/items": {"get": {"parameters": [{"schema": {"type": "string"}}]}}}}
        nodes = inventory.schema_nodes(document)
        self.assertEqual(len(nodes), 9)
        self.assertIn("/paths/~1items/get/parameters/0/schema", dict(nodes))
        self.assertEqual(sum(isinstance(value, bool) for _, value in nodes), 3)

    def test_operation_inventory_records_wire_features_separately(self):
        operation = {
            "parameters": [{"in": "query", "style": "form", "explode": True}],
            "requestBody": {"required": True, "content": {"application/json": {"schema": {"type": "string"}}}},
            "responses": {"200": {"content": {"text/event-stream": {"schema": {"type": "string"}}}}},
        }
        document = {"paths": {"/events": {"post": operation}}}
        features = inventory.operation_features(document)
        self.assertEqual(features["method:post"], 1)
        self.assertEqual(features["parameter-explode:true"], 1)
        self.assertEqual(features["body-required:true"], 1)
        self.assertEqual(features["response-media:text/event-stream"], 1)

    def test_semantic_copy_preserves_wire_keys_order_and_literal_payloads(self):
        original = {"type": "object", "description": "text", "properties": {
            "description": {"type": "string", "description": "field note"},
            "value": {"const": {"description": "literal data"}},
        }, "default": {"description": "literal default"}}
        before = copy.deepcopy(original)
        projected = inventory.semantic_copy(original)
        self.assertNotIn("description", projected)
        self.assertEqual(list(projected["properties"]), ["description", "value"])
        self.assertEqual(projected["properties"]["value"]["const"], {"description": "literal data"})
        self.assertEqual(projected["default"], {"description": "literal default"})
        self.assertEqual(original, before)

    def test_dependency_closure_handles_cycles_escapes_and_ignores_example_refs(self):
        components = {
            "A/B~": {"type": "object", "properties": {"next": {"$ref": "#/components/schemas/Other"}}, "example": {"$ref": "#/components/schemas/Unused"}},
            "Other": {"type": "object", "properties": {"next": {"$ref": "#/components/schemas/A~1B~0"}}},
            "Unused": {"type": "string"},
        }
        closure = inventory.dependency_closure({"$ref": "#/components/schemas/A~1B~0"}, components)
        self.assertEqual(set(closure), {"A/B~", "Other"})
        self.assertEqual(inventory.dependency_closure({"$ref": "outside.json#/Other"}, components), {})

    def test_coverage_does_not_count_unreached_features_in_syntax_failures(self):
        coverage = inventory.fixture_features()
        self.assertNotIn("duplicate-property", coverage["type:integer"]["error"])
        self.assertIn("format-uri", coverage["format:uri"]["error"])
        self.assertIn("scalars-30", coverage["type:integer"]["success"])


class BenchmarkTests(unittest.TestCase):
    def test_synthetic_workloads_are_deterministic_and_have_valid_reference_targets(self):
        first = benchmark.synthetic(10, 16, 17)
        self.assertEqual(first, benchmark.synthetic(10, 16, 17))
        self.assertNotEqual(first, benchmark.synthetic(10, 16, 18))
        schemas = first["components"]["schemas"]
        self.assertEqual(len(schemas), 10)
        for _, node in inventory.schema_nodes(first):
            if isinstance(node, dict) and "$ref" in node:
                self.assertIn(node["$ref"].removeprefix("#/components/schemas/"), schemas)

    def test_manifest_verification_rejects_modified_content(self):
        import hashlib
        import json
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            (root / "A.kt").write_text("old")
            manifest = {"version": 1, "files": {"A.kt": hashlib.sha256(b"old").hexdigest()}}
            (root / ".openapi-kt-manifest.json").write_text(json.dumps(manifest))
            summary, baseline = benchmark.verify_tree(root, 1)
            self.assertEqual(summary["source_bytes"], 3)
            benchmark.verify_tree(root, 1, baseline)
            (root / "A.kt").write_text("new")
            with self.assertRaises(RuntimeError):
                benchmark.verify_tree(root, 1)


if __name__ == "__main__":
    unittest.main()
