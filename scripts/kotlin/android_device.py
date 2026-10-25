"""Execute the synthetic platform APK without opening an Activity."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import xml.etree.ElementTree as ET


def run(command):
    return subprocess.run(command, capture_output=True, text=True, timeout=180)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--apk', type=Path, required=True)
    parser.add_argument('--package', required=True)
    args = parser.parse_args()
    serial = os.environ.get('ANDROID_SERIAL')
    if not serial:
        parser.error('ANDROID_SERIAL must select an authorized device')
    sdk = Path(os.environ.get('ANDROID_HOME') or os.environ.get('ANDROID_SDK_ROOT') or '.')
    adb = os.environ.get('ADB') or str(sdk / 'platform-tools/adb')
    analyzer = os.environ.get('ANDROID_APKANALYZER') or str(sdk / 'cmdline-tools/latest/bin/apkanalyzer')
    runner = 'org.example.platform.RoadmapProbeInstrumentation'
    android = '{http://schemas.android.com/apk/res/android}'
    manifest_result = run([analyzer, 'manifest', 'print', str(args.apk)])
    if manifest_result.returncode:
        raise RuntimeError('APK manifest inspection failed')
    manifest = ET.fromstring(manifest_result.stdout)
    forbidden = {'activity', 'activity-alias', 'service', 'provider', 'receiver'}
    if any(node.tag in forbidden or node.tag.startswith('uses-permission') for node in manifest.iter()):
        raise RuntimeError('Headless APK must contain no UI components or permissions')
    instrumentations = manifest.findall('instrumentation')
    if (manifest.get('package') != args.package or len(instrumentations) != 1
            or instrumentations[0].get(android + 'name') != runner
            or instrumentations[0].get(android + 'targetPackage') != args.package
            or manifest.find('uses-sdk') is None
            or manifest.find('uses-sdk').get(android + 'minSdkVersion') != '23'):
        raise RuntimeError('Unexpected APK package, runner, target or minimum SDK')

    def device(*command):
        return run([adb, '-s', serial, *command])

    if device('get-state').stdout.strip() != 'device':
        raise RuntimeError('Selected device is unavailable')
    existing = device('shell', 'pm', 'path', args.package)
    if existing.stdout.strip():
        raise RuntimeError('Refusing to replace an installed probe package')
    if existing.returncode not in (0, 1):
        raise RuntimeError('Could not inspect installed probe package')
    digest = hashlib.sha256(args.apk.read_bytes()).hexdigest()
    report = {'package': args.package, 'apk_sha256': digest, 'min_sdk': 23,
              'device_api': device('shell', 'getprop', 'ro.build.version.sdk').stdout.strip()}
    installed = False
    try:
        install = device('install', '-t', str(args.apk))
        if install.returncode != 0 or 'Success' not in install.stdout:
            raise RuntimeError('Probe installation failed')
        installed = True
        if hashlib.sha256(args.apk.read_bytes()).hexdigest() != digest:
            raise RuntimeError('APK changed during installation')
        # Raw output retains result bundle keys even when stream is also set.
        runtime = device('shell', 'am', 'instrument', '-w', '-r', args.package + '/' + runner)
        (args.apk.parent / 'android-device-runtime.log').write_text(runtime.stdout + runtime.stderr)
        lines = {line.strip() for line in runtime.stdout.splitlines()}
        report['passed'] = (runtime.returncode == 0
                            and 'INSTRUMENTATION_RESULT: roadmap_result=passed' in lines
                            and 'INSTRUMENTATION_CODE: -1' in lines)
        if not report['passed']:
            raise RuntimeError('Platform contracts failed; see android-device-runtime.log beside APK')
    finally:
        if installed:
            cleanup = device('uninstall', args.package)
            report['uninstalled'] = cleanup.returncode == 0 and 'Success' in cleanup.stdout
        (args.apk.parent / 'android-device-result.json').write_text(json.dumps(report, indent=2) + '\n')
        if installed and not report['uninstalled']:
            raise RuntimeError('Probe cleanup failed; remove only the reported probe package')
    print(json.dumps(report, indent=2))


if __name__ == '__main__':
    main()
