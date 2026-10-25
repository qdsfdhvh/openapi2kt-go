"""Run local CLI contracts and optionally compare another host's exact output."""

import argparse
import hashlib
import json
import platform
from pathlib import Path
import subprocess
import tempfile


CASES = [
    ('catalog', 'tests/fixtures/catalog.json'),
    ('structural-unique', 'tests/fixtures/http/map-unique.json'),
    ('allof-tuple', 'tests/fixtures/http/tuple-allof-references.json'),
    ('tuple-target-defaults', 'tests/fixtures/http/tuple-target-defaults.json'),
    ('tuple-field-defaults', 'tests/fixtures/http/tuple-field-defaults.json'),
    ('primitive-union-merge', 'tests/fixtures/http/merge-primitive-unions.json'),
    ('form', 'tests/fixtures/http/form-request-media.json'),
    ('raw', 'tests/fixtures/http/raw-request-media.json'),
    ('yaml-control', 'tests/fixtures/yaml/records.json'),
    ('yaml', 'tests/fixtures/yaml/records.yaml'),
]


def digest(data):
    return hashlib.sha256(data).hexdigest()


def inventory(root):
    return {path.relative_to(root).as_posix(): digest(path.read_bytes())
            for path in sorted(root.rglob('*')) if path.is_file()}


def attributes(root):
    return {path.relative_to(root).as_posix(): (path.stat().st_mode, path.stat().st_mtime_ns)
            for path in root.rglob('*') if path.is_file()}


def binary_format(binary):
    header = binary.read_bytes()[:32]
    if header.startswith(b'\x7fELF'):
        order = 'little' if header[5] == 1 else 'big'
        return 'ELF/' + {183: 'arm64', 62: 'amd64'}.get(int.from_bytes(header[18:20], order), 'other')
    if header[:4] == b'\xcf\xfa\xed\xfe':
        return 'MachO/' + {0x100000c: 'arm64', 0x1000007: 'amd64'}.get(int.from_bytes(header[4:8], 'little'), 'other')
    if header.startswith(b'MZ'):
        return 'PE'
    return 'unknown'


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--repo', type=Path, default=Path(__file__).resolve().parents[2])
    parser.add_argument('--report', type=Path, required=True)
    parser.add_argument('--compare', type=Path)
    args = parser.parse_args()
    binary, repo = args.binary.resolve(), args.repo.resolve()

    def run(command, *, expected=0, data=None):
        result = subprocess.run([str(binary), *command], input=data, capture_output=True, timeout=60)
        if result.returncode != expected or result.stderr:
            raise RuntimeError('CLI contract failed for ' + command[0] + ': ' + result.stdout.decode(errors='replace') + result.stderr.decode(errors='replace'))
        return result.stdout

    if b'openapi-kt' not in run(['--help']):
        raise RuntimeError('Missing CLI help')
    results = []
    with tempfile.TemporaryDirectory(prefix='openapi-kt-host-') as directory:
        scratch = Path(directory)
        for style in ('data_class', 'plain_class'):
            for checked in (False, True):
                peers = {}
                for name, relative in CASES:
                    input_path = repo / relative
                    input_format = 'yaml' if input_path.suffix == '.yaml' else 'json'
                    output = scratch / (name + '-' + style + '-' + str(checked))
                    flags = ['--package-name', 'org.example.hostprobe', '--model-name-suffix', 'Dto',
                             '--model-style', style, '--deduplicate-models', '--hoist-shared-shapes',
                             '--merge-identical-models', '--lock-hoisted-names', '--diagnostic-format=json']
                    if checked:
                        flags.append('--validation')
                    command = ['generate', '--input', str(input_path), '--output', str(output), *flags]
                    # This fixture's structural own defaults require the
                    # validation-on role. Keep the documented negative profile.
                    if name == 'structural-unique' and not checked:
                        report = json.loads(run(command, expected=1))
                        diagnostics = report.get('diagnostics', [])
                        if (len(diagnostics) != 1 or diagnostics[0]['code'] != 'E_SCHEMA_UNSUPPORTED'
                                or diagnostics[0].get('pointer') != '/components/schemas/Source/properties/rows/uniqueItems'
                                or output.exists()):
                            raise RuntimeError('Original structural-default negative contract changed')
                        streamed = json.loads(run(['generate', '--input', '-', '--output', str(output), *flags],
                                                  expected=1, data=input_path.read_bytes()))
                        if streamed != report or output.exists():
                            raise RuntimeError('Rejected file/stdin input differs or created output')
                        results.append({'name': name, 'input': relative, 'input_sha256': digest(input_path.read_bytes()),
                                        'model_style': style, 'validation': checked, 'report': report, 'files': {},
                                        'expected_rejection': True})
                        print(name, style, 'validation=False original rejection PASS', flush=True)
                        continue
                    report = json.loads(run(command))
                    before, attrs = inventory(output), attributes(output)
                    if json.loads(run(command + ['--check'])) != report or json.loads(run(command)) != report:
                        raise RuntimeError('Generate/check report drift')
                    if before != inventory(output) or attrs != attributes(output):
                        raise RuntimeError('No-op changed bytes, modes or mtimes')
                    run(['validate', '--input', str(input_path), '--diagnostic-format=json', *(['--validation'] if checked else [])])
                    invalid = scratch / ('invalid.' + input_format)
                    invalid.write_text('{')
                    run(['generate', '--input', str(invalid), '--output', str(output), *flags], expected=1)
                    if before != inventory(output) or attrs != attributes(output):
                        raise RuntimeError('Invalid input changed old output')
                    streamed = output.with_name(output.name + '-stdin')
                    stdin_report = json.loads(run(['generate', '--input', '-', '--input-format', input_format,
                                                  '--output', str(streamed), *flags], data=input_path.read_bytes()))
                    if stdin_report != report or inventory(streamed) != before:
                        raise RuntimeError('File/stdin report or output mismatch')
                    entry = {'name': name, 'input': relative, 'input_sha256': digest(input_path.read_bytes()),
                             'model_style': style, 'validation': checked, 'report': report, 'files': before}
                    peers[name] = entry
                    results.append(entry)
                    print(name, style, 'validation=' + str(checked), 'PASS', flush=True)
                if peers['yaml']['report'] != peers['yaml-control']['report'] or peers['yaml']['files'] != peers['yaml-control']['files']:
                    raise RuntimeError('JSON/YAML semantic parity failed')

    # Host and binary metadata deliberately stay outside the cross-host equality.
    result = {'host_system': platform.system(), 'interpreter_machine': platform.machine(),
              'binary_format': binary_format(binary), 'binary_sha256': digest(binary.read_bytes()),
              'cases': results, 'local_checks': ['generate/check/noop', 'file/stdin', 'validate',
                                               'invalid-input-preserves-bytes-modes-mtimes', 'JSON/YAML-parity'],
              'limits': 'ten fixed inputs; Data/Plain x validation; combined options; no full Go/Kotlin/HTTP engine or all-host acceptance'}
    if args.compare:
        if json.loads(args.compare.read_text())['cases'] != results:
            raise RuntimeError('Cross-host reports, input hashes or file hashes differ')
        result['cross_host_cases_equal'] = True
    args.report.parent.mkdir(parents=True, exist_ok=True)
    args.report.write_text(json.dumps(result, indent=2) + '\n')
    print('Host CLI contracts passed:', len(results), 'cases;', result['binary_format'])


if __name__ == '__main__':
    main()
