import hashlib
import importlib.util
import io
import json
from pathlib import Path
import tarfile
import tempfile
import unittest
import zipfile

ROOT = Path(__file__).resolve().parents[2]


def load(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


package = load('package_for_verify', ROOT / 'scripts/release/package.py')
verify = load('verify_package', ROOT / 'scripts/release/verify.py')


def write_archive_without_notices(path, version, target):
    prefix = 'openapi-kt-v' + version + '-' + target + '/'
    binary = 'openapi-kt.exe' if target.startswith('windows-') else 'openapi-kt'
    files = {name: b'Synthetic historical package data'
             for name in (binary, 'README.md', 'LICENSE', 'GO-LICENSE')}
    files['RELEASE.json'] = json.dumps({'version': 'v' + version, 'target': target}).encode()
    if path.suffix == '.zip':
        with zipfile.ZipFile(path, 'w') as archive:
            for name, data in files.items():
                archive.writestr(prefix + name, data)
    else:
        with tarfile.open(path, 'w:gz') as archive:
            for name, data in files.items():
                entry = tarfile.TarInfo(prefix + name)
                entry.mode = 0o755 if name == binary else 0o644
                entry.size = len(data)
                archive.addfile(entry, io.BytesIO(data))


class PackageVerificationTests(unittest.TestCase):
    def test_packaged_members_checksum_and_version(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            inputs = [root / name for name in ('binary', 'readme', 'license', 'go-license')]
            for p in inputs:
                p.write_bytes(b'Synthetic package verification data')
            inputs.append(ROOT / 'THIRD-PARTY-NOTICES')
            for target in package.TARGETS:
                for version in ('0.1.0-alpha.1', '0.2.0-alpha.1'):
                    with self.subTest(target=target, version=version):
                        path = package.archive(*inputs, 'v' + version, target, root / target)
                        digest = hashlib.sha256(path.read_bytes()).hexdigest()
                        files, binary = verify.inspect_archive(path, digest, version, target)
                        self.assertEqual(len(files), 10)
                        self.assertEqual(files[binary], inputs[0].read_bytes())
                        self.assertEqual(files['THIRD-PARTY-NOTICES'], (ROOT / 'THIRD-PARTY-NOTICES').read_bytes())
                        for checksum, expected_version in [('0' * 64, version), (digest, '0.3.0-alpha.1')]:
                            with self.assertRaises(ValueError):
                                verify.inspect_archive(path, checksum, expected_version, target)

    def test_missing_notices_are_only_allowed_for_the_historical_first_preview(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            for target in package.TARGETS:
                suffix = '.zip' if target.startswith('windows-') else '.tar.gz'
                path = root / ('synthetic-package' + suffix)
                for version in ('0.1.0-alpha.1', '0.1.0-alpha.2', '0.1.0', '0.2.0-alpha.1', '0.2.0'):
                    with self.subTest(target=target, version=version):
                        write_archive_without_notices(path, version, target)
                        digest = hashlib.sha256(path.read_bytes()).hexdigest()
                        if version == '0.1.0-alpha.1':
                            files, binary = verify.inspect_archive(path, digest, version, target)
                            self.assertEqual(set(files), {binary, 'README.md', 'LICENSE', 'GO-LICENSE', 'RELEASE.json'})
                        else:
                            with self.assertRaisesRegex(ValueError, 'Unexpected or duplicate archive members'):
                                verify.inspect_archive(path, digest, version, target)

    def test_tar_rejects_links_duplicates_and_unexpected_paths(self):
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp) / 'unsafe.tar.gz'
            prefix = 'openapi-kt-v0.2.0-alpha.1-darwin-arm64/'
            names = sorted(verify.MEMBERS | {'openapi-kt'})
            for fault in ('binary-link', 'notice-link', 'duplicate', 'traversal'):
                with tarfile.open(path, 'w:gz') as archive:
                    for name in names:
                        entry = tarfile.TarInfo(prefix + name)
                        entry.mode = 0o755
                        if fault == 'traversal' and name == 'THIRD-PARTY-NOTICES':
                            entry.name = '../THIRD-PARTY-NOTICES'
                        if (fault == 'binary-link' and name == 'openapi-kt') or (fault == 'notice-link' and name == 'THIRD-PARTY-NOTICES'):
                            entry.type = tarfile.SYMTYPE
                            entry.linkname = '/bin/sh'
                            archive.addfile(entry)
                        else:
                            data = b'{}'
                            entry.size = len(data)
                            archive.addfile(entry, io.BytesIO(data))
                        if fault == 'duplicate' and name == 'THIRD-PARTY-NOTICES':
                            archive.addfile(entry, io.BytesIO(b'{}'))
                with self.assertRaises(ValueError):
                    verify.inspect_archive(path, hashlib.sha256(path.read_bytes()).hexdigest(), '0.2.0-alpha.1', 'darwin-arm64')

    def test_zip_rejects_symlinked_duplicate_and_unexpected_notices(self):
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp) / 'unsafe.zip'
            prefix = 'openapi-kt-v0.2.0-alpha.1-windows-amd64/'
            names = sorted(verify.MEMBERS | {'openapi-kt.exe'})
            for fault in ('link', 'duplicate', 'traversal'):
                with zipfile.ZipFile(path, 'w') as archive:
                    for name in names:
                        entry = zipfile.ZipInfo(prefix + name)
                        entry.external_attr = 0o100644 << 16
                        if name == 'THIRD-PARTY-NOTICES':
                            if fault == 'link':
                                entry.external_attr = 0o120644 << 16
                            elif fault == 'traversal':
                                entry.filename = '../THIRD-PARTY-NOTICES'
                        archive.writestr(entry, b'{}')
                        if fault == 'duplicate' and name == 'THIRD-PARTY-NOTICES':
                            with self.assertWarnsRegex(UserWarning, 'Duplicate name'):
                                archive.writestr(entry, b'{}')
                with self.assertRaises(ValueError):
                    verify.inspect_archive(path, hashlib.sha256(path.read_bytes()).hexdigest(), '0.2.0-alpha.1', 'windows-amd64')


if __name__ == '__main__':
    unittest.main()
