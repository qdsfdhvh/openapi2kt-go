#!/usr/bin/env python3
"""Build a source-path-trimmed CLI archive from a fixed file allowlist; never upload."""
import argparse
import gzip
import hashlib
import io
import json
import os
from pathlib import Path
import re
import subprocess
import tarfile
import zipfile

ROOT = Path(__file__).resolve().parents[2]
TARGETS = ('darwin-amd64', 'darwin-arm64', 'linux-amd64', 'linux-arm64', 'windows-amd64', 'windows-arm64')


def source_version(root):
    text = (root / 'internal/cli/arguments.go').read_text()
    match = re.search(r'^const version = "([0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?)"$', text, re.M)
    if not match:
        raise ValueError('Cannot determine the declared CLI version')
    return match.group(1)


def payload(path):
    if path.is_symlink() or not path.is_file():
        raise ValueError('Package inputs must be regular files, not symlinks')
    return path.read_bytes()


def archive(binary, readme, license_file, go_license, third_party_notices, tag, target, output):
    if target not in TARGETS or not re.fullmatch(r'v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?', tag):
        raise ValueError('Invalid release tag or target')
    files = {
        'openapi-kt.exe' if target.startswith('windows-') else 'openapi-kt': payload(binary),
        'README.md': payload(readme), 'LICENSE': payload(license_file), 'GO-LICENSE': payload(go_license),
        'THIRD-PARTY-NOTICES': payload(third_party_notices),
        'THIRD-PARTY-NOTICES.md': payload(ROOT / 'THIRD-PARTY-NOTICES.md'),
        'YAML-LICENSE': payload(ROOT / 'vendor/go.yaml.in/yaml/v3/LICENSE'),
        'YAML-NOTICE': payload(ROOT / 'vendor/go.yaml.in/yaml/v3/NOTICE'),
        'APACHE-2.0.txt': payload(ROOT / 'third-party/APACHE-2.0.txt'),
        'RELEASE.json': (json.dumps({'version': tag, 'target': target, 'runtime_validation': 'See CI results; cross-built targets are not runtime proof'}, sort_keys=True) + '\n').encode(),
    }
    # Runtime privacy check on exactly the packaged bytes; not a complete secret detector.
    if any(re.search(rb'/' rb'Users/|[A-Z]:\\Users\\|/home/[^/\s]+/', data) for data in files.values()):
        raise ValueError('A package input contains a machine user path')
    name = 'openapi-kt-' + tag + '-' + target
    destination = output / (name + ('.zip' if target.startswith('windows-') else '.tar.gz'))
    if destination.exists():
        raise ValueError('Refusing to overwrite an existing package')
    stream = io.BytesIO()
    if target.startswith('windows-'):
        with zipfile.ZipFile(stream, 'w', compression=zipfile.ZIP_DEFLATED) as package:
            for filename, data in sorted(files.items()):
                info = zipfile.ZipInfo(name + '/' + filename, date_time=(1980, 1, 1, 0, 0, 0))
                info.compress_type = zipfile.ZIP_DEFLATED
                info.external_attr = (0o755 if filename.startswith('openapi-kt') else 0o644) << 16
                package.writestr(info, data)
    else:
        with gzip.GzipFile(fileobj=stream, mode='wb', mtime=0) as compressed:
            with tarfile.open(fileobj=compressed, mode='w') as package:
                for filename, data in sorted(files.items()):
                    info = tarfile.TarInfo(name + '/' + filename)
                    info.size = len(data)
                    info.mode = 0o755 if filename == 'openapi-kt' else 0o644
                    info.mtime = 0
                    package.addfile(info, io.BytesIO(data))
    output.mkdir(parents=True, exist_ok=True)
    with destination.open('xb') as file:
        file.write(stream.getvalue())
    return destination


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--tag', required=True)
    parser.add_argument('--target', choices=TARGETS, required=True)
    parser.add_argument('--output', type=Path, default=ROOT / 'build/release')
    args = parser.parse_args()
    try:
        if args.tag != 'v' + source_version(ROOT):
            raise ValueError('Release tag must match the declared CLI version')
        output = args.output.resolve()
        if not any(output.is_relative_to(ROOT / name) for name in ('build', 'artifacts', 'dist')):
            raise ValueError('Packages belong in an ignored build/artifacts/dist directory')
        # Fail before building when required license notices are missing or unsafe.
        payload(ROOT / 'LICENSE')
        payload(ROOT / 'THIRD-PARTY-NOTICES')
        environment = dict(os.environ, GOOS=args.target.split('-')[0], GOARCH=args.target.split('-')[1], CGO_ENABLED='0', GOTOOLCHAIN='local', GOPROXY='off', GOSUMDB='off', GOFLAGS='', GOWORK='off')
        environment.pop('GOROOT', None)
        go = os.environ.get('GO', 'go')
        go_root = subprocess.run([go, 'env', 'GOROOT'], cwd=ROOT, env=environment, capture_output=True, text=True, check=True).stdout.strip()
        output.mkdir(parents=True, exist_ok=True)
        binary = output / ('openapi-kt-' + args.target + ('.exe' if args.target.startswith('windows-') else ''))
        subprocess.run([go, 'build', '-trimpath', '-ldflags=-s -w', '-o', str(binary), './cmd/openapi-kt'], cwd=ROOT, env=environment, check=True)
        result = archive(binary, ROOT / 'README.md', ROOT / 'LICENSE', Path(go_root) / 'LICENSE', ROOT / 'THIRD-PARTY-NOTICES', args.tag, args.target, output)
        print(result.name)
        print('sha256 ' + hashlib.sha256(result.read_bytes()).hexdigest())
        return 0
    except (OSError, ValueError, subprocess.SubprocessError):
        print('Packaging refused: check license, version, target, safe output and regular-file inputs.')
        return 1


if __name__ == '__main__':
    raise SystemExit(main())
