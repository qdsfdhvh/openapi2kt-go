import importlib.util
from pathlib import Path
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('http_compilations', ROOT / 'scripts/kotlin/http_compilations.py')
compilations = importlib.util.module_from_spec(spec)
spec.loader.exec_module(compilations)


class HTTPCompilationTests(unittest.TestCase):
    def test_complete_inventory_preserves_recursive_packages_providers_and_shared_internal_helpers(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            sources = {
                'build/generated/a/Node.kt': 'package org.example.a\nclass Node(val next: org.example.b.Node?)',
                'build/generated/b/Node.kt': 'package org.example.b\nclass Node(val next: org.example.a.Node?, val native: org.example.provider.Value)',
                'src/main/kotlin/Value.kt': 'package org.example.provider\nclass Value',
                'src/main/kotlin/org/example/contracts/Engine.kt': 'package org.example.contracts\ninternal class TestEngine',
                'src/main/kotlin/org/example/contracts/One.kt': 'package org.example.contracts\nfun one() { TestEngine() }',
                'src/main/kotlin/org/example/contracts/Two.kt': 'package org.example.contracts\nfun two() { TestEngine() }',
                'src/main/kotlin/org/example/contracts/HttpContracts.kt': 'package org.example.contracts\nfun main() { one(); two() }',
            }
            for name, text in sources.items():
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(text)
            (root / 'build.gradle.kts').write_text('// declared compiler and dependencies\n')
            inventory = compilations.partition(root)
            self.assertEqual(sorted(sources), sorted(path for paths in inventory.values() for path in paths))
            libraries = [paths for name, paths in inventory.items() if 'Library' in name]
            contracts = [paths for name, paths in inventory.items() if 'Contract' in name]
            self.assertEqual(len(libraries), 1)
            self.assertEqual(len(libraries[0]), 3)
            self.assertEqual(len(contracts), 1)
            self.assertEqual(len(contracts[0]), 3)
            copied = list((root / 'build/http-compilations').rglob('*.kt'))
            self.assertCountEqual([p.read_text() for p in copied], list(sources.values()))
            self.assertEqual((root / 'build/http-compilations/main/HttpContracts.kt').read_text(), sources[inventory['main'][0]])
            build = (root / 'build.gradle.kts').read_text()
            self.assertIn('associateWith', build)
            self.assertIn('-Xfriend-paths=', build)
            self.assertIn('mustRunAfter', build)
            self.assertNotIn('jvmargs', build)
            with self.assertRaisesRegex(ValueError, 'fresh HTTP compilation plan'):
                compilations.partition(root)

    def test_sealed_family_is_not_split_by_acyclic_inheritance(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            sources = {
                'build/generated/Base.kt': 'package org.example.family\npublic sealed class Base',
                'build/generated/Child.kt': 'package org.example.family\npublic class Child() : org.example.family.Base()',
                'src/main/kotlin/org/example/contracts/HttpContracts.kt': 'package org.example.contracts\nfun main() {}',
            }
            for name, text in sources.items():
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(text)
            (root / 'build.gradle.kts').write_text('// declared build\n')
            previous = compilations.TARGET_BYTES
            try:
                compilations.TARGET_BYTES = 1
                inventory = compilations.partition(root)
            finally:
                compilations.TARGET_BYTES = previous
            libraries = [paths for name, paths in inventory.items() if 'Library' in name]
            self.assertEqual(len(libraries), 1)
            self.assertEqual(len(libraries[0]), 2)

    def test_internal_serializer_dependency_keeps_original_test_module_access(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            sources = {
                'build/generated/Serializer.kt': 'package org.example.codec\ninternal object Codec',
                'build/generated/Holder.kt': 'package org.example.model\nclass Holder(val value: org.example.codec.Codec)',
                'src/main/kotlin/org/example/contracts/HttpContracts.kt': 'package org.example.contracts\nfun main() {}',
            }
            for name, text in sources.items():
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(text)
            (root / 'build.gradle.kts').write_text('// declared build\n')
            previous = compilations.TARGET_BYTES
            try:
                compilations.TARGET_BYTES = 1
                compilations.partition(root)
            finally:
                compilations.TARGET_BYTES = previous
            build = (root / 'build.gradle.kts').read_text()
            self.assertIn('"httpLibrary001" to listOf("httpLibrary000")', build)
            self.assertIn('"-Xfriend-paths="', build)
            self.assertIn('httpInternalFriends[name].orEmpty()', build)

    def test_recursive_components_keep_their_original_dependency_order(self):
        edges = {'a': {'b'}, 'b': {'a', 'provider'}, 'provider': set(), 'client': {'a'}}
        self.assertEqual(compilations.dependency_components(edges), [['provider'], ['a', 'b'], ['client']])
        sizes = {name: compilations.TARGET_BYTES for name in edges}
        self.assertEqual(compilations.pack(compilations.dependency_components(edges), sizes, ordered=True), [['provider'], ['a', 'b'], ['client']])

    def test_packing_retains_an_indivisible_boundary_probe_above_the_target(self):
        groups = [['a'], ['b', 'c'], ['d']]
        sizes = {'a': compilations.TARGET_BYTES, 'b': compilations.TARGET_BYTES, 'c': 1, 'd': 1}
        self.assertEqual(compilations.pack(groups, sizes), [['a'], ['b', 'c'], ['d']])


if __name__ == '__main__':
    unittest.main()
