"""Backend routing tests; these do not establish generator or Kotlin support."""
import importlib.util
from pathlib import Path
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("fixture_smoke", Path(__file__).resolve().parents[1] / "fixtures" / "generate-smoke.py")
smoke = importlib.util.module_from_spec(spec)
spec.loader.exec_module(smoke)


class SmokeBackendTests(unittest.TestCase):
    def test_fixture_layout_is_repository_relative_not_working_directory_relative(self):
        self.assertEqual(smoke.FIXTURES, smoke.ROOT / "tests" / "fixtures" / "p1")
        self.assertTrue((smoke.FIXTURES / "cases.json").is_file())
        self.assertTrue((smoke.ROOT / "go.mod").is_file())

    def test_native_binary_path_is_one_argument_and_does_not_use_cargo(self):
        with patch.dict("os.environ", {"OPENAPI_KT_BINARY": ".pi/native command", "CARGO": "must-not-run"}, clear=True):
            self.assertEqual(smoke.generator_command(), [".pi/native command"])

    def test_default_backend_runs_native_go_without_cargo(self):
        with patch.dict("os.environ", {"GO": "native-go", "CARGO": "must-not-run"}, clear=True):
            self.assertEqual(smoke.generator_command(), ["native-go", "run", "./cmd/openapi-kt"])

    def test_override_cannot_escape_repository_build_area(self):
        with patch.dict("os.environ", {"OPENAPI_KT_SMOKE_BUILD": "/outside-build"}, clear=True):
            with self.assertRaisesRegex(ValueError, "repository"):
                smoke.main()


if __name__ == "__main__":
    unittest.main()
