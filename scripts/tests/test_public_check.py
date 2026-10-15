import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

SCRIPT = Path(__file__).resolve().parents[1] / "publication" / "check.py"
spec = importlib.util.spec_from_file_location("public_check", SCRIPT)
public_check = importlib.util.module_from_spec(spec)
spec.loader.exec_module(public_check)


class PublicationChecks(unittest.TestCase):
    def test_terms_match_case_insensitive_substrings_in_contents_and_paths(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "BrandPrivateBrand.txt").write_text("A PRIVATEBRAND variant")
            paths, scope = public_check.candidates(root, True)
            report = public_check.scan(root, paths, ["privatebrand"])
            self.assertEqual(scope, "source-tree-only")
            self.assertEqual(report["findings"][0]["categories"], ["forbidden-term-1", "forbidden-term-1-path"])
            self.assertNotIn("privatebrand", json.dumps(report).lower())

    def test_sensitive_filenames_cannot_hide_unrecognized_secret_formats(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            paths = [Path(".env"), Path("credentials.json"), Path("key.pem"), Path(".env.example")]
            for path in paths:
                (root / path).write_text("Synthetic placeholder")
            report = public_check.scan(root, paths, [])
            self.assertEqual(len(report["findings"]), 3)
            self.assertTrue(all(f["categories"] == ["sensitive-file-name"] for f in report["findings"]))

    def test_file_only_mode_excludes_local_notes_and_build_outputs(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name in [".pi/state.txt", "local/input.json", "target/cache.txt", "samples/build/generated.txt", "task_plan.md"]:
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text("privatebrand")
            (root / "README.md").write_text("Public documentation")
            paths, _ = public_check.candidates(root, True)
            self.assertEqual(paths, [Path("README.md")])
            self.assertEqual(public_check.scan(root, paths, ["privatebrand"])["findings"], [])

    def test_go_test_binaries_are_local_but_fixture_out_files_are_public(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            binaries = [Path("core.test"), Path("internal/core.test.exe")]
            public = [Path("tests/fixtures/expected.out"), Path("internal/core_test.go")]
            for path in binaries + public:
                (root / path).parent.mkdir(parents=True, exist_ok=True)
                (root / path).write_text("Synthetic data")
            paths, _ = public_check.candidates(root, True)
            self.assertEqual(paths, sorted(public))
            report = public_check.scan(root, binaries, [])
            self.assertEqual(report["files_checked"], 0)
            self.assertTrue(all(f["categories"] == ["local-only-candidate"] for f in report["findings"]))
            self.assertEqual(len(report["findings"]), 2)

    def test_explicit_candidates_cannot_include_tracked_local_state(self):
        with tempfile.TemporaryDirectory() as directory:
            report = public_check.scan(Path(directory), [Path(".pi/state.json"), Path("task_plan.md")], [])
            self.assertEqual(len(report["findings"]), 2)
            self.assertTrue(all(f["categories"] == ["local-only-candidate"] for f in report["findings"]))
            self.assertEqual(report["files_checked"], 0)

    def test_common_secret_and_machine_path_shapes_are_detected_without_echo(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            marker = "-----BEGIN " + "PRIVATE KEY-----"
            (root / "sample.txt").write_text(marker + "\n" + "AKIA" + "A" * 16 + "\n" + "/" + "Users/example/file")
            report = public_check.scan(root, [Path("sample.txt")], [])
            self.assertEqual(report["findings"][0]["categories"], ["machine-user-path", "private-key", "provider-token"])
            self.assertNotIn(marker, json.dumps(report))

    @unittest.skipUnless(os.name == "posix", "Symlink setup requires a supported host")
    def test_symlink_escape_and_local_target_are_rejected(self):
        with tempfile.TemporaryDirectory() as directory, tempfile.TemporaryDirectory() as outside:
            root = Path(directory)
            (root / "escape").symlink_to(outside, target_is_directory=True)
            (root / "local").mkdir()
            (root / "alias").symlink_to(root / "local", target_is_directory=True)
            report = public_check.scan(root, [Path("escape"), Path("alias")], [])
            self.assertEqual(report["findings"][0]["categories"], ["symlink-escape"])
            self.assertEqual(report["findings"][1]["categories"], ["local-only-symlink"])

    def test_missing_git_requires_explicit_file_only_mode(self):
        with tempfile.TemporaryDirectory() as directory:
            result = subprocess.run([sys.executable, str(SCRIPT), "--root", directory], capture_output=True, text=True)
            self.assertEqual(result.returncode, 2)
            self.assertEqual(json.loads(result.stdout)["status"], "error")
            self.assertEqual(result.stderr, "")

    def test_policy_is_local_and_errors_do_not_echo_its_values(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "README.md").write_text("PRIVATEBRAND suffix")
            (root / ".pi").mkdir()
            policy = root / ".pi/policy.json"
            policy.write_text(json.dumps({"forbidden_terms": ["privatebrand"]}))
            command = [sys.executable, str(SCRIPT), "--root", directory, "--source-tree", "--policy", str(policy)]
            result = subprocess.run(command, capture_output=True, text=True)
            self.assertEqual(result.returncode, 1)
            self.assertNotIn("privatebrand", result.stdout.lower())
            self.assertEqual(json.loads(result.stdout)["forbidden_term_count"], 1)
            policy.write_text(json.dumps({"forbidden_terms": ["privatebrand"], "unexpected": "privatebrand"}))
            result = subprocess.run(command, capture_output=True, text=True)
            self.assertEqual(result.returncode, 2)
            self.assertNotIn("privatebrand", result.stdout.lower())


if __name__ == "__main__":
    unittest.main()
