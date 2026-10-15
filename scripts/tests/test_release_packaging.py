import contextlib
import importlib.util
import io
from pathlib import Path
import json
import os
import tarfile
import tempfile
import unittest
from unittest.mock import patch
import zipfile

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('release_package', ROOT / 'scripts/release/package.py')
release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(release)


class ReleasePackagingTests(unittest.TestCase):
    def test_version_and_platform_set_are_explicit(self):
        self.assertEqual(release.source_version(ROOT), '0.2.0-alpha.1')
        self.assertEqual(len(release.TARGETS), 6)

    def test_exact_allowlist_and_reproducible_archives(self):
        (ROOT / 'build').mkdir(exist_ok=True)
        with tempfile.TemporaryDirectory(dir=ROOT / 'build') as directory:
            root = Path(directory)
            inputs = [root / name for name in ('binary', 'README.md', 'LICENSE', 'GO-LICENSE')]
            for path in inputs:
                path.write_bytes(b'Synthetic packaging test, not a project license decision')
            inputs.append(ROOT / 'THIRD-PARTY-NOTICES')
            (root / 'private-note.txt').write_text('Must not be packaged')
            for target in ('linux-amd64', 'windows-arm64'):
                first = release.archive(*inputs, 'v0.1.0', target, root / 'one')
                second = release.archive(*inputs, 'v0.1.0', target, root / 'two')
                self.assertEqual(first.read_bytes(), second.read_bytes())
                expected = {'openapi-kt.exe' if target.startswith('windows') else 'openapi-kt', 'README.md', 'LICENSE', 'GO-LICENSE', 'THIRD-PARTY-NOTICES', 'RELEASE.json', 'THIRD-PARTY-NOTICES.md', 'YAML-LICENSE', 'YAML-NOTICE', 'APACHE-2.0.txt'}
                if target.startswith('windows'):
                    with zipfile.ZipFile(first) as archive:
                        names = archive.namelist()
                        notices = archive.read(next(name for name in names if name.endswith('/THIRD-PARTY-NOTICES')))
                        yaml_license = archive.read(next(name for name in names if name.endswith('/YAML-LICENSE')))
                else:
                    with tarfile.open(first) as archive:
                        names = archive.getnames()
                        notices = archive.extractfile(next(name for name in names if name.endswith('/THIRD-PARTY-NOTICES'))).read()
                        yaml_license = archive.extractfile(next(name for name in names if name.endswith('/YAML-LICENSE'))).read()
                        self.assertEqual(archive.getmembers()[0].uid, 0)
                self.assertEqual(notices, (ROOT / 'THIRD-PARTY-NOTICES').read_bytes())
                self.assertEqual(yaml_license, (ROOT / 'vendor/go.yaml.in/yaml/v3/LICENSE').read_bytes())
                self.assertEqual({Path(name).name for name in names}, expected)
                self.assertTrue(all(name.startswith(first.name.removesuffix('.tar.gz').removesuffix('.zip') + '/') for name in names))
                with self.assertRaises(ValueError):
                    release.archive(*inputs, 'v0.1.0', target, root / 'one')

    def test_preview_metadata_and_prefix_match_the_declared_version(self):
        tag = 'v' + release.source_version(ROOT)
        self.assertEqual(tag, 'v0.2.0-alpha.1')
        (ROOT / 'build').mkdir(exist_ok=True)
        with tempfile.TemporaryDirectory(dir=ROOT / 'build') as directory:
            root = Path(directory)
            inputs = [root / name for name in ('binary', 'readme', 'license', 'go-license', 'third-party-notices')]
            for path in inputs:
                path.write_bytes(b'Synthetic preview packaging input')
            for target in release.TARGETS:
                result = release.archive(*inputs, tag, target, root / target)
                prefix = 'openapi-kt-' + tag + '-' + target
                self.assertTrue(result.name.startswith(prefix + '.'))
                if target.startswith('windows-'):
                    with zipfile.ZipFile(result) as archive:
                        metadata = json.loads(archive.read(prefix + '/RELEASE.json'))
                else:
                    with tarfile.open(result) as archive:
                        metadata = json.load(archive.extractfile(prefix + '/RELEASE.json'))
                self.assertEqual(metadata['version'], tag)
                self.assertEqual(metadata['target'], target)

    def test_missing_license_bad_target_tag_and_machine_paths_fail_closed(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            inputs = [root / name for name in ('binary', 'readme', 'license', 'go-license', 'third-party-notices')]
            for path in inputs:
                path.write_text('Synthetic public data')
            for tag, target in [('v0.1.0;unsafe', 'linux-amd64'), ('v0.1.0', '../outside')]:
                with self.assertRaises(ValueError):
                    release.archive(*inputs, tag, target, root / 'output')
            for missing in (inputs[2], inputs[4]):
                with self.subTest(missing=missing.name):
                    missing.unlink()
                    with self.assertRaises(ValueError):
                        release.archive(*inputs, 'v0.1.0', 'linux-amd64', root / 'output')
                    self.assertFalse((root / 'output').exists())
                    missing.write_text('Synthetic public data')
            for unsafe in (inputs[0], inputs[4]):
                with self.subTest(unsafe=unsafe.name):
                    unsafe.write_text('/' + 'Users/example/private')
                    with self.assertRaises(ValueError):
                        release.archive(*inputs, 'v0.1.0', 'linux-amd64', root / 'output')
                    unsafe.write_text('Synthetic public data')

    def test_missing_required_notices_refuse_packaging_before_build(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name in ('LICENSE', 'THIRD-PARTY-NOTICES'):
                with self.subTest(missing=name):
                    for filename in ('LICENSE', 'THIRD-PARTY-NOTICES'):
                        (root / filename).write_text('Synthetic public data')
                    (root / name).unlink()
                    with patch.object(release, 'ROOT', root), \
                         patch.object(release, 'source_version', return_value='0.2.0-alpha.1'), \
                         patch('sys.argv', ['package.py', '--tag', 'v0.2.0-alpha.1', '--target', 'linux-amd64']), \
                         patch.object(release.subprocess, 'run') as go, \
                         contextlib.redirect_stdout(io.StringIO()):
                        self.assertEqual(release.main(), 1)
                    go.assert_not_called()
                    self.assertFalse((root / 'build').exists())

    @unittest.skipUnless(os.name == 'posix', 'Symlink setup requires a supported host')
    def test_symlinked_dependency_notice_is_refused(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            inputs = [root / name for name in ('binary', 'readme', 'license', 'go-license', 'third-party-notices')]
            for path in inputs[:-1]:
                path.write_text('Synthetic public data')
            inputs[-1].symlink_to(inputs[2])
            with self.assertRaises(ValueError):
                release.archive(*inputs, 'v0.2.0-alpha.1', 'linux-amd64', root / 'output')


if __name__ == '__main__':
    unittest.main()
