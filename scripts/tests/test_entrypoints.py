import json
import os
from pathlib import Path, PureWindowsPath
import shutil
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
RUNNER = ROOT / 'scripts/test.sh'
FEATURES = ('map-unique', 'uuid-unique', 'floating-unique', 'native-enum-unique', 'string-enum-unique', 'temporal-unique', 'nullable-composed-reference-members', 'parameter-styles', 'referenced-bytes-field-defaults', 'raw-response-media', 'directional-models', 'yaml-document', 'http-namespace-collisions', 'http-servers', 'allof-nominal-terminals', 'boolean-http', 'boolean-root-nullable', 'boolean-root', 'boolean-value', 'collection-owned-dto', 'composed-nested-owners', 'hierarchy-dispatch', 'hierarchy-fields', 'hierarchy-intrinsic-nullable', 'hierarchy-nested-owners', 'hierarchy-null-members', 'hierarchy-nullable-alias', 'hierarchy-reference-chains', 'internal-dto-chain', 'internal-dto', 'merge', 'named-dto-alias-chain', 'named-dto-chain', 'primitive-tuple', 'recursive-json', 'root-dto-bridge', 'root-internal-dto', 'union-allof', 'union-nominal-owners', 'enum-nominal-owners', 'valueclass-reference-chains', 'composed-union-terminals', 'temporal-reference-chains', 'builtin-reference-chains', 'root-borrowed-builtin', 'bigint-reference-chains', 'bytes-reference-chains', 'caller-native-reference-chains', 'raw-true-reference-chains', 'recursive-cover-reference-chains', 'named-collection-reference-chains', 'root-borrowed-collection', 'borrowed-collection-dto-leaves', 'borrowed-collection-enum-leaves', 'borrowed-collection-valueclass-leaves', 'borrowed-collection-allof-model-leaves', 'borrowed-collection-union-leaves', 'borrowed-collection-product-union-leaves', 'borrowed-collection-hierarchy-leaves', 'borrowed-collection-builtin-leaves', 'borrowed-collection-temporal-leaves', 'borrowed-collection-bigint-leaves', 'borrowed-collection-bytes-leaves', 'borrowed-collection-caller-native-leaves', 'borrowed-collection-raw-true-leaves', 'borrowed-collection-recursive-cover-leaves', 'union-common-dictionaries', 'nullable-allof-dto', 'union-common-nominals', 'union-inline-values', 'nullable-product-unions', 'nullable-object-members', 'nullable-object-reference-members', 'model-defaults', 'enum-defaults', 'json-unique', 'inline-errors', 'merge-exclusions', 'native-enum-validation', 'native-enum-consumers', 'native-enum-bounds', 'internal-native-enums', 'named-native-enums', 'nullable-native-enums', 'native-enum-null-members', 'string-enum-null-members', 'referenced-allof-bounds', 'referenced-allof-counts', 'explicit-null-literals', 'inferred-null-literals', 'null-only-defaults', 'structured-const-fields', 'structured-const-values', 'nullable-structured-const', 'primitive-array-defaults', 'checked-array-defaults', 'primitive-map-defaults', 'native-enum-collection-defaults', 'string-enum-collection-defaults', 'nested-builtin-collection-defaults', 'nested-temporal-collection-defaults', 'nested-string-enum-collection-defaults', 'nested-native-enum-collection-defaults', 'referenced-collection-field-defaults', 'string-enum-collection-bounds', 'referenced-scalar-field-defaults', 'referenced-temporal-scalar-field-defaults', 'referenced-temporal-collection-field-defaults', 'referenced-string-enum-collection-defaults', 'referenced-native-enum-collection-defaults', 'referenced-native-enum-scalar-defaults', 'referenced-string-enum-scalar-defaults', 'referenced-string-enum-null-member-defaults', 'nested-object-defaults', 'nested-object-collection-defaults', 'referenced-object-defaults', 'referenced-object-collection-defaults', 'union-field-defaults', 'raw-json-field-defaults', 'direct-nullable-component-counts', 'bytes-field-defaults', 'bytes-component-defaults', 'reference-wrapper-component-defaults', 'hierarchy-inline-members', 'nullable-component-defaults', 'raw-json-component-defaults')

def bash_executable():
    return os.environ.get('OPENAPI_KT_TEST_BASH') or shutil.which('bash') or 'bash'


def process_diagnostics(result, args):
    def clean(value):
        if isinstance(value, bytes):
            value = value.decode(errors='replace')
        text = str(value or '').replace('\x00', '')
        for root, label in ((ROOT, '<REPO>'), (Path.home(), '<HOME>')):
            text = text.replace(str(root), label).replace(root.as_posix(), label)
        return text[:2000]
    return {'args': args, 'bash_on_path': clean(shutil.which('bash')),
            'bash_from_ci': clean(os.environ.get('OPENAPI_KT_TEST_BASH')),
            'stdout': clean(result.stdout), 'stderr': clean(result.stderr)}


class EntrypointTests(unittest.TestCase):
    def test_list_contains_every_existing_feature_without_starting_tools(self):
        result = subprocess.run([bash_executable(), RUNNER.as_posix(), '--list'], cwd=ROOT.parent, capture_output=True, text=True, env=dict(os.environ, GO='must-not-execute'))
        self.assertEqual(result.returncode, 0, process_diagnostics(result, ['--list']))
        self.assertEqual(set(result.stdout.splitlines()), {'unit', 'models', 'http', 'features', 'all'} | {'feature ' + name for name in FEATURES})
        self.assertEqual({p.stem for p in (ROOT / 'scripts/kotlin/features').glob('*.sh')}, set(FEATURES))

    def test_bad_feature_and_extra_arguments_fail_before_execution(self):
        for args in [[], ['feature'], ['feature', '../http'], ['feature', 'missing'], ['feature', 'merge', 'extra'], ['unit', 'extra'], ['unknown']]:
            result = subprocess.run([bash_executable(), RUNNER.as_posix(), *args], cwd=ROOT.parent, capture_output=True, env=dict(os.environ, GO='must-not-execute'))
            self.assertEqual(result.returncode, 2, process_diagnostics(result, args))

    def test_explicit_bash_runs_same_list_and_empty_arguments(self):
        bash = os.environ.get('OPENAPI_KT_TEST_BASH') or shutil.which('bash')
        for args, expected in [(['--list'], 0), ([], 2)]:
            result = subprocess.run([bash, RUNNER.as_posix(), *args], cwd=ROOT.parent,
                                    capture_output=True, text=True,
                                    env=dict(os.environ, GO='must-not-execute'))
            self.assertEqual(result.returncode, expected, process_diagnostics(result, args))
            if args:
                self.assertEqual(set(result.stdout.splitlines()), {'unit', 'models', 'http', 'features', 'all'} | {'feature ' + name for name in FEATURES})

    def test_unit_passes_current_bash_to_python_tests(self):
        bash = bash_executable()
        (ROOT / 'build').mkdir(exist_ok=True)
        with tempfile.TemporaryDirectory(prefix='synthetic shell ', dir=ROOT / 'build') as directory:
            directory = Path(directory)
            go = directory / 'go.sh'
            python = directory / 'python.sh'
            output = directory / 'bash.txt'
            go.write_bytes(b'#!/usr/bin/env bash\nexit 0\n')
            python.write_bytes(b'#!/usr/bin/env bash\nprintf "%s" "${OPENAPI_KT_TEST_BASH:-}" > "$CALLS"\n')
            go.chmod(0o755)
            python.chmod(0o755)
            environment = dict(os.environ, GO=go.as_posix(), GOFMT=go.as_posix(),
                               PYTHON=python.as_posix(), CALLS=output.as_posix())
            environment.pop('OPENAPI_KT_TEST_BASH', None)
            result = subprocess.run([bash, RUNNER.as_posix(), 'unit'], cwd=directory,
                                    capture_output=True, text=True, env=environment)
            self.assertEqual(result.returncode, 0, process_diagnostics(result, ['unit']))
            expected = subprocess.run([bash, '-c', 'if command -v cygpath >/dev/null 2>&1; then cygpath -w "$BASH"; else printf "%s" "$BASH"; fi'],
                                      capture_output=True, text=True, check=True)
            self.assertEqual(output.read_text(), expected.stdout.strip())

    def test_failed_exit_includes_subprocess_stderr(self):
        result = subprocess.CompletedProcess([], 1, stdout='', stderr='synthetic launcher failed')
        with patch('subprocess.run', return_value=result):
            with self.assertRaisesRegex(AssertionError, 'synthetic launcher failed'):
                self.test_list_contains_every_existing_feature_without_starting_tools()

    def test_original_entrypoints_use_configured_bash_executable(self):
        bash = 'C:/synthetic shell/bin/bash.exe'
        output = '\n'.join(['unit', 'models', 'http', 'features', 'all'] + ['feature ' + name for name in FEATURES])
        with patch.dict(os.environ, {'OPENAPI_KT_TEST_BASH': bash}):
            with patch('subprocess.run', return_value=subprocess.CompletedProcess([], 0, stdout=output)) as run:
                self.test_list_contains_every_existing_feature_without_starting_tools()
            self.assertEqual(run.call_args.args[0], [bash, RUNNER.as_posix(), '--list'])
            with patch('subprocess.run', return_value=subprocess.CompletedProcess([], 2)) as run:
                self.test_bad_feature_and_extra_arguments_fail_before_execution()
            self.assertEqual(run.call_count, 7)
            for call in run.call_args_list:
                self.assertEqual(call.args[0][:2], [bash, RUNNER.as_posix()])
                self.assertEqual(call.kwargs['cwd'], ROOT.parent)
                self.assertEqual(call.kwargs['env']['GO'], 'must-not-execute')

    def test_unconfigured_entrypoint_qualifies_bash_from_path(self):
        bash = 'C:/synthetic shell/bin/bash.exe'
        output = '\n'.join(['unit', 'models', 'http', 'features', 'all'] + ['feature ' + name for name in FEATURES])
        with patch.dict(os.environ):
            os.environ.pop('OPENAPI_KT_TEST_BASH', None)
            with patch('shutil.which', return_value=bash), patch('subprocess.run', return_value=subprocess.CompletedProcess([], 0, stdout=output)) as run:
                self.test_list_contains_every_existing_feature_without_starting_tools()
            self.assertEqual(run.call_args.args[0], [bash, RUNNER.as_posix(), '--list'])

    def test_bash_script_argument_keeps_portable_windows_separators(self):
        runner = PureWindowsPath(r'C:\synthetic repo\scripts\test.sh')
        output = '\n'.join(['unit', 'models', 'http', 'features', 'all'] + ['feature ' + name for name in FEATURES])
        result = subprocess.CompletedProcess([], 0, stdout=output)
        with patch(__name__ + '.RUNNER', runner), patch('subprocess.run', return_value=result) as run:
            self.test_list_contains_every_existing_feature_without_starting_tools()
        self.assertEqual(run.call_args.args[0], [bash_executable(), 'C:/synthetic repo/scripts/test.sh', '--list'])
        self.assertEqual(run.call_args.kwargs['cwd'], ROOT.parent)
        self.assertEqual(run.call_args.kwargs['env']['GO'], 'must-not-execute')
        with patch(__name__ + '.RUNNER', runner), patch('subprocess.run', return_value=subprocess.CompletedProcess([], 2)) as run:
            self.test_bad_feature_and_extra_arguments_fail_before_execution()
        self.assertEqual(run.call_count, 7)
        for call in run.call_args_list:
            self.assertEqual(call.args[0][:2], [bash_executable(), 'C:/synthetic repo/scripts/test.sh'])
            self.assertEqual(call.kwargs['cwd'], ROOT.parent)
            self.assertEqual(call.kwargs['env']['GO'], 'must-not-execute')

    @unittest.skipIf(os.name == 'nt', 'POSIX executable shim; native Windows behavior is checked by CI')
    def test_nested_string_enum_prepare_only_generates_four_profiles_without_gradle(self):
        (ROOT / 'build').mkdir(exist_ok=True)
        with tempfile.TemporaryDirectory(dir=ROOT / 'build') as directory:
            directory = Path(directory)
            calls = directory / 'calls.jsonl'
            gradle_call = directory / 'gradle.txt'
            go = directory / 'native go'
            gradle = directory / 'forbidden gradle'
            cli_source = '''#!{python}
import json, os, sys
from pathlib import Path
with Path(os.environ['CALLS']).open('a') as output:
    output.write(json.dumps(sys.argv[1:]) + '\\n')
'''.format(python=sys.executable)
            go.write_text('#!' + sys.executable + '\nimport sys\nfrom pathlib import Path\n'
                          + 'target = Path(sys.argv[sys.argv.index("-o") + 1])\n'
                          + 'target.write_text(' + repr(cli_source) + ')\ntarget.chmod(0o755)\n')
            gradle.write_text('#!' + sys.executable + '\nimport os, sys\nfrom pathlib import Path\n'
                              + 'Path(os.environ["GRADLE_CALL"]).write_text("unexpected")\nsys.exit(91)\n')
            go.chmod(0o755)
            gradle.chmod(0o755)
            result = subprocess.run([bash_executable(), RUNNER.as_posix(), 'feature', 'nested-string-enum-collection-defaults'],
                                    cwd=directory, capture_output=True, text=True,
                                    env=dict(os.environ, GO=str(go), GRADLE=str(gradle), CALLS=str(calls),
                                             GRADLE_CALL=str(gradle_call), OPENAPI_KT_PREPARE_ONLY='1'))
            self.assertEqual(result.returncode, 0, process_diagnostics(result, ['feature', 'nested-string-enum-collection-defaults']))
            self.assertFalse(gradle_call.exists(), 'PREPARE_ONLY must never invoke Gradle')
            prepared = [line for line in result.stdout.splitlines() if line.startswith('PREPARED NESTED STRING ENUM COLLECTION DEFAULTS ')]
            self.assertEqual(len(prepared), 4)
            invocations = [json.loads(line) for line in calls.read_text().splitlines()]
            projects = [Path(line.split(': ', 1)[1]) for line in prepared]
            for project in projects:
                commands = [args for args in invocations if Path(args[args.index('--input') + 1]).parent == project]
                generated = [args for args in commands if '--check' not in args]
                checked = [args for args in commands if '--check' in args]
                self.assertEqual({Path(args[args.index('--input') + 1]).stem for args in generated},
                                 {'input', 'options', 'dedup', 'merge', 'slots', 'source', 'checked', 'bytes', 'vocabulary'})
                self.assertEqual(checked, [args + ['--check'] for args in generated])
                self.assertEqual(len(generated), 9)
                model_style, validation = project.name.rsplit('-', 1)
                for args in generated:
                    self.assertEqual(args[args.index('--model-style') + 1], model_style)
                    probe = Path(args[args.index('--input') + 1]).stem
                    self.assertEqual('--validation' in args, validation == 'true' or probe == 'checked')
                self.assertTrue((project / 'src/main/kotlin/org/example/contracts/NestedStringEnumCollectionDefaultsContracts.kt').is_file())
                self.assertTrue((project / 'src/main/kotlin/org/example/contracts/Main.kt').is_file())

    @unittest.skipIf(os.name == 'nt', 'POSIX executable shim; native Windows behavior is checked by CI')
    def test_all_feature_routes_resolve_repository_and_offline_backend(self):
        (ROOT / 'build').mkdir(exist_ok=True)
        with tempfile.TemporaryDirectory(dir=ROOT / 'build') as directory:
            directory = Path(directory)
            go = directory / 'native go'
            log = directory / 'calls.json'
            go.write_text('#!' + sys.executable + '\nimport json,os,sys\nfrom pathlib import Path\nPath(os.environ["CALLS"]).write_text(json.dumps({"cwd":os.getcwd(),"args":sys.argv[1:],"offline":[os.environ.get(k) for k in ("GOTOOLCHAIN","GOPROXY","GOSUMDB")]}))\nsys.exit(17)\n')
            go.chmod(0o755)
            for name in FEATURES:
                result = subprocess.run([bash_executable(), RUNNER.as_posix(), 'feature', name], cwd=directory, capture_output=True, env=dict(os.environ, GO=str(go), CALLS=str(log)))
                self.assertEqual(result.returncode, 17, name)
                call = json.loads(log.read_text())
                self.assertEqual(call['cwd'], str(ROOT))
                self.assertEqual(call['offline'], ['local', 'off', 'off'])
                self.assertEqual(call['args'][:3], ['build', '-trimpath', '-o'])
                self.assertEqual(call['args'][-1], './cmd/openapi-kt')


if __name__ == '__main__':
    unittest.main()
