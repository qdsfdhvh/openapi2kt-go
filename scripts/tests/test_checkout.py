import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]


class CheckoutTests(unittest.TestCase):
    def test_windows_autocrlf_keeps_text_lf_and_binary_bytes(self):
        environment = dict(os.environ, GIT_CONFIG_NOSYSTEM='1', GIT_CONFIG_GLOBAL=os.devnull)
        source = b'package example\n\nfunc answer() int {\n\treturn 42\n}\n'
        binary = b'\x00synthetic\r\nbinary\n'
        with tempfile.TemporaryDirectory() as directory:
            directory = Path(directory)
            repository = directory / 'repository'
            repository.mkdir()
            subprocess.run(['git', 'init', '-q', str(repository)], env=environment, check=True, capture_output=True)
            (repository / 'probe.go').write_bytes(source)
            (repository / 'probe.sh').write_bytes(b'#!/usr/bin/env bash\nset -euo pipefail\n')
            (repository / 'probe.json').write_bytes(b'{"synthetic":true}\n')
            (repository / 'probe.bin').write_bytes(binary)
            attributes = ROOT / '.gitattributes'
            if attributes.exists():
                (repository / '.gitattributes').write_bytes(attributes.read_bytes())
            subprocess.run(['git', '-C', str(repository), '-c', 'core.autocrlf=false', 'add', '--all'], env=environment, check=True, capture_output=True)
            checkout = directory / 'checkout'
            subprocess.run(['git', '-C', str(repository), '-c', 'core.autocrlf=true', 'checkout-index', '--all', '--prefix=' + checkout.as_posix() + '/'], env=environment, check=True, capture_output=True)
            self.assertEqual((checkout / 'probe.go').read_bytes(), source, 'Windows-style checkout must not convert formatted Go source to CRLF')
            for name in ('probe.sh', 'probe.json'):
                self.assertEqual((checkout / name).read_bytes(), (repository / name).read_bytes())
            self.assertEqual((checkout / 'probe.bin').read_bytes(), binary, 'Auto text detection must leave binary files unchanged')
            formatting = subprocess.run([os.environ.get('GOFMT', 'gofmt'), '-l', str(checkout / 'probe.go')], env=environment, capture_output=True, check=True)
            self.assertEqual(formatting.stdout, b'', 'Checked-out source must pass the real gofmt gate')


if __name__ == '__main__':
    unittest.main()
