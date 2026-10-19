"""Release documentation must not drift from what the packager actually emits.

The archive allowlist is already pinned by test_release_packaging. What went
wrong once is that a *document* claimed a different member count, so a reader
verifying a package by member list would have concluded it was wrong. These
checks tie the prose to the implementation and to go.mod, without a network or
toolchain dependency.
"""

import importlib.util
import re
from pathlib import Path
import tarfile
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('release_package', ROOT / 'scripts/release/package.py')
release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(release)

NUMBER_WORDS = {
    'one': 1, 'two': 2, 'three': 3, 'four': 4, 'five': 5, 'six': 6,
    'seven': 7, 'eight': 8, 'nine': 9, 'ten': 10, 'eleven': 11, 'twelve': 12,
}


def packaged_members():
    """Build one throwaway archive and return its exact member set."""
    (ROOT / 'build').mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(dir=ROOT / 'build') as directory:
        root = Path(directory)
        inputs = []
        for name in ('binary', 'README.md', 'LICENSE', 'GO-LICENSE'):
            path = root / name
            path.write_bytes(b'Synthetic release documentation test')
            inputs.append(path)
        inputs.append(ROOT / 'THIRD-PARTY-NOTICES')
        archive = release.archive(*inputs, 'v0.0.0-test', 'linux-amd64', root / 'out')
        with tarfile.open(archive) as opened:
            return {Path(name).name for name in opened.getnames() if not name.endswith('/')}


def required_modules():
    """Read the direct module requirements from go.mod without invoking Go."""
    text = (ROOT / 'go.mod').read_text()
    block = re.search(r'require\s*\(([^)]*)\)', text)
    if block:
        lines = block.group(1).splitlines()
    else:
        lines = re.findall(r'^require\s+(.+)$', text, re.MULTILINE)
    modules = []
    for line in lines:
        line = line.split('//')[0].strip()
        if not line:
            continue
        modules.append(line.split()[0])
    return modules


class ReleaseDocumentationTests(unittest.TestCase):
    def test_documented_member_count_and_names_match_the_packager(self):
        members = packaged_members()
        prose = (ROOT / 'docs/releases/next-alpha.md').read_text()
        match = re.search(r'the exact (\w+) members \(([^)]*)\)', prose)
        self.assertIsNotNone(match, 'next-alpha.md no longer states an exact member list')
        claimed_count = NUMBER_WORDS[match.group(1).lower()]
        claimed = {name.strip().strip('`') for name in match.group(2).split(',')}
        self.assertEqual(claimed_count, len(members), 'documented member count is stale')
        self.assertEqual(claimed, members, 'documented members differ from the packaged archive')

    def test_every_required_module_is_attributed(self):
        notices = (ROOT / 'THIRD-PARTY-NOTICES').read_text()
        notices += (ROOT / 'THIRD-PARTY-NOTICES.md').read_text()
        for module in required_modules():
            self.assertIn(module, notices, f'{module} has no third-party attribution')

    def test_vendor_tree_matches_the_declared_requirements(self):
        text = (ROOT / 'vendor/modules.txt').read_text()
        vendored = {line.split()[1] for line in text.splitlines() if line.startswith('# ')}
        self.assertEqual(vendored, set(required_modules()), 'vendor/modules.txt drifted from go.mod')


if __name__ == '__main__':
    unittest.main()
