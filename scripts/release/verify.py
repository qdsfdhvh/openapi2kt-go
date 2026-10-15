#!/usr/bin/env python3
"""Verify a native release archive and exercise its CLI without a source checkout."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import subprocess
import tarfile
import tempfile
import zipfile

MEMBERS = {'README.md', 'LICENSE', 'GO-LICENSE', 'THIRD-PARTY-NOTICES', 'RELEASE.json',
           'THIRD-PARTY-NOTICES.md', 'YAML-LICENSE', 'YAML-NOTICE', 'APACHE-2.0.txt'}
# The unchanged first preview predates the external TOML dependency.
LEGACY_VERSIONS_WITHOUT_NOTICES = {'0.1.0-alpha.1'}
MAX_MEMBER_BYTES = 64 * 1024 * 1024


def native_target():
    system = {'Darwin': 'darwin', 'Linux': 'linux', 'Windows': 'windows'}.get(platform.system())
    arch = {'arm64': 'arm64', 'aarch64': 'arm64', 'x86_64': 'amd64', 'AMD64': 'amd64'}.get(platform.machine())
    if not system or not arch:
        raise ValueError('Unsupported native host')
    return system + '-' + arch


def inspect_archive(path, checksum, version, target):
    if hashlib.sha256(path.read_bytes()).hexdigest() != checksum:
        raise ValueError('Archive checksum mismatch')
    prefix = 'openapi-kt-v' + version + '-' + target
    binary_name = 'openapi-kt.exe' if target.startswith('windows-') else 'openapi-kt'
    expected = {prefix + '/' + name for name in MEMBERS | {binary_name}}
    allowed_member_sets = [expected]
    if version in LEGACY_VERSIONS_WITHOUT_NOTICES:
        legacy = {prefix + '/' + name for name in {'README.md', 'LICENSE', 'GO-LICENSE', 'RELEASE.json', binary_name}}
        allowed_member_sets.extend([legacy, legacy | {prefix + '/THIRD-PARTY-NOTICES'}])
    contents = {}
    if path.suffix == '.zip':
        with zipfile.ZipFile(path) as archive:
            entries = archive.infolist()
            if not any(len(entries) == len(members) and {e.filename for e in entries} == members
                       for members in allowed_member_sets):
                raise ValueError('Unexpected or duplicate archive members')
            for entry in entries:
                mode = entry.external_attr >> 16
                if entry.is_dir() or (mode & 0o170000) not in (0, 0o100000) or entry.file_size > MAX_MEMBER_BYTES:
                    raise ValueError('Unsafe archive member')
                contents[Path(entry.filename).name] = archive.read(entry)
    else:
        with tarfile.open(path, 'r:gz') as archive:
            entries = archive.getmembers()
            if not any(len(entries) == len(members) and {e.name for e in entries} == members
                       for members in allowed_member_sets):
                raise ValueError('Unexpected or duplicate archive members')
            for entry in entries:
                if not entry.isfile() or entry.size > MAX_MEMBER_BYTES:
                    raise ValueError('Unsafe archive member')
                if Path(entry.name).name == binary_name and not entry.mode & 0o111:
                    raise ValueError('Binary is not executable')
                contents[Path(entry.name).name] = archive.extractfile(entry).read()
    metadata = json.loads(contents['RELEASE.json'])
    if metadata.get('version') != 'v' + version or metadata.get('target') != target:
        raise ValueError('Release metadata mismatch')
    return contents, binary_name


def smoke(contents, binary_name, version, directory):
    binary = directory / binary_name
    binary.write_bytes(contents[binary_name])
    binary.chmod(0o755)
    # No Go, Gradle, fixture checkout or globally installed CLI is available here.
    environment = dict(os.environ, PATH='')
    def run(*args, data=None, expected=0):
        result = subprocess.run([str(binary), *args], cwd=directory, env=environment,
                                input=data, capture_output=True, timeout=30)
        if result.returncode != expected:
            raise ValueError('Packaged CLI check failed: ' + args[0])
        return result.stdout
    if run('--version').decode().strip() != 'openapi-kt ' + version:
        raise ValueError('Binary version mismatch')
    run('--help')
    spec = json.dumps({'openapi': '3.1.0', 'info': {'title': 'Package smoke', 'version': '1'},
                      'paths': {}, 'components': {'schemas': {'Item': {'type': 'object',
                      'required': ['id'], 'properties': {'id': {'type': 'string'}}}}}}).encode()
    (directory / 'input.json').write_bytes(spec)
    run('validate', '--input', 'input.json')
    run('validate', '--input', '-', data=spec)
    def generate(source, output, *extra, data=None, expected=0):
        return run('generate', '--input', source, '--output', output,
                   '--package-name', 'org.example.install', *extra, data=data, expected=expected)
    generate('input.json', 'file')
    generate('-', 'stdin', data=spec)
    def snapshot(name):
        return {str(p.relative_to(directory / name)): (p.read_bytes(), p.stat().st_mtime_ns)
                for p in (directory / name).rglob('*') if p.is_file()}
    before = snapshot('file')
    generated = {p: b for p, (b, _) in before.items() if p.endswith('.kt')}
    if not generated or generated != {p: b for p, (b, _) in snapshot('stdin').items() if p.endswith('.kt')}:
        raise ValueError('File/stdin generated sources differ')
    generate('input.json', 'file', '--check')
    generate('input.json', 'file')
    generate('-', 'stdin', '--check', data=spec)
    generate('-', 'file', data=b'{', expected=1)
    if snapshot('file') != before:
        raise ValueError('Check, no-op or failure changed installed output')
    return ['version', 'help', 'file-validate', 'stdin-validate', 'file-generate',
            'stdin-generate', 'file-stdin-parity', 'check', 'no-op-mtime', 'failure-preservation']


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('archive', type=Path)
    parser.add_argument('--sha256', required=True)
    parser.add_argument('--version', required=True, help='Expected CLI version without v')
    parser.add_argument('--work-dir', type=Path, required=True, help='Ignored directory for isolated installation')
    args = parser.parse_args()
    try:
        target = native_target()
        contents, binary = inspect_archive(args.archive, args.sha256, args.version, target)
        args.work_dir.mkdir(parents=True, exist_ok=True)
        with tempfile.TemporaryDirectory(prefix='package-install-', dir=args.work_dir.resolve()) as directory:
            checks = smoke(contents, binary, args.version, Path(directory))
        print(json.dumps({'status': 'passed', 'version': args.version, 'target': target,
                          'archive_sha256': args.sha256, 'binary_sha256': hashlib.sha256(contents[binary]).hexdigest(),
                          'members': sorted(contents), 'checks': checks}, indent=2))
        return 0
    except (OSError, ValueError, subprocess.SubprocessError, tarfile.TarError, zipfile.BadZipFile):
        print(json.dumps({'status': 'failed', 'message': 'Package verification failed; check checksum, members, native target, version and CLI behavior.'}))
        return 1


if __name__ == '__main__':
    raise SystemExit(main())
