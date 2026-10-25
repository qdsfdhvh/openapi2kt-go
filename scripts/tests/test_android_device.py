import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch


SPEC = importlib.util.spec_from_file_location('android_device', Path(__file__).resolve().parents[1] / 'kotlin/android_device.py')
RUNNER = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(RUNNER)
PACKAGE = 'org.example.openapikt.roadmapprobe.data'
MANIFEST = '''<manifest xmlns:android="http://schemas.android.com/apk/res/android" package="%s">
<uses-sdk android:minSdkVersion="23"/>
<instrumentation android:name="org.example.platform.RoadmapProbeInstrumentation" android:targetPackage="%s"/>
<application/></manifest>''' % (PACKAGE, PACKAGE)
PASSED = 'INSTRUMENTATION_RESULT: roadmap_result=passed\nINSTRUMENTATION_CODE: -1\n'


class AndroidDeviceLifecycleTests(unittest.TestCase):
    def invoke(self, directory, *, existing=False, manifest=MANIFEST, output=PASSED, timeout=False):
        apk = Path(directory) / 'probe.apk'
        apk.write_bytes(b'synthetic APK; analyzer is mocked')
        calls = []

        def run(command):
            calls.append(command)
            if command[0] == 'analyzer':
                return SimpleNamespace(returncode=0, stdout=manifest)
            action = command[3:]
            result = ''
            if action == ['get-state']:
                result = 'device\n'
            elif action[:3] == ['shell', 'pm', 'path']:
                result = 'package:/already-installed.apk\n' if existing else ''
            elif action[:2] == ['shell', 'getprop']:
                result = '34\n'
            elif action[0] in ('install', 'uninstall'):
                result = 'Success\n'
            elif action[:3] == ['shell', 'am', 'instrument']:
                self.assertIn('-r', action)
                if timeout:
                    raise subprocess.TimeoutExpired(command, 180)
                result = output
            else:
                self.fail('unexpected device operation')
            return SimpleNamespace(returncode=0, stdout=result, stderr='')

        environment = {'ANDROID_SERIAL': 'synthetic-device', 'ADB': 'adb', 'ANDROID_APKANALYZER': 'analyzer'}
        with patch.dict(os.environ, environment), patch.object(sys, 'argv', ['android_device.py', '--apk', str(apk), '--package', PACKAGE]), patch.object(RUNNER, 'run', side_effect=run), patch('builtins.print'):
            try:
                RUNNER.main()
            finally:
                self.calls = calls

    def test_existing_package_is_never_replaced_or_removed(self):
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaisesRegex(RuntimeError, 'Refusing to replace'):
                self.invoke(directory, existing=True)
            self.assertFalse(any(c[3:4] in (['install'], ['uninstall']) for c in self.calls))

    def test_apk_with_activity_or_permission_never_reaches_device(self):
        for element in ('activity', 'uses-permission-sdk-23'):
            with self.subTest(element=element), tempfile.TemporaryDirectory() as directory:
                with self.assertRaisesRegex(RuntimeError, 'no UI components or permissions'):
                    self.invoke(directory, manifest=MANIFEST.replace('<application/>', '<%s/>' % element))
                self.assertEqual(len(self.calls), 1)

    def test_wrong_result_code_and_timeout_still_remove_only_probe(self):
        for timeout in (False, True):
            with self.subTest(timeout=timeout), tempfile.TemporaryDirectory() as directory:
                with self.assertRaises((RuntimeError, subprocess.TimeoutExpired)):
                    self.invoke(directory, output=PASSED.replace('-1\n', '-10\n'), timeout=timeout)
                self.assertEqual(self.calls[-1][3:], ['uninstall', PACKAGE])
                report = json.loads((Path(directory) / 'android-device-result.json').read_text())
                self.assertTrue(report['uninstalled'])

    def test_success_has_exact_runtime_result_hash_and_cleanup(self):
        with tempfile.TemporaryDirectory() as directory:
            self.invoke(directory)
            report = json.loads((Path(directory) / 'android-device-result.json').read_text())
            self.assertTrue(report['passed'] and report['uninstalled'])
            self.assertEqual(report['device_api'], '34')
            self.assertEqual(len(report['apk_sha256']), 64)
            self.assertNotIn('synthetic-device', json.dumps(report))


if __name__ == '__main__':
    unittest.main()
