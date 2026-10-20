from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parents[2]


def job_body(text, name):
    match = re.search(r'^  ' + re.escape(name) + r':\n(.*?)(?=^  [A-Za-z_][A-Za-z0-9_-]*:\n|\Z)', text, re.MULTILINE | re.DOTALL)
    if match is None:
        raise AssertionError('Required CI job is missing: ' + name)
    return match.group(1)


class CIWorkflowTests(unittest.TestCase):
    def test_source_minimum_and_every_ci_toolchain_use_the_declared_stable_version(self):
        module = (ROOT / 'go.mod').read_text()
        minimum = re.search(r'(?m)^go ([0-9]+\.[0-9]+\.[0-9]+)$', module)
        self.assertIsNotNone(minimum)
        self.assertEqual(minimum.group(1), '1.27.2')
        ci = (ROOT / '.github/workflows/ci.yml').read_text()
        release = (ROOT / '.github/workflows/release.yml').read_text()
        for body in (job_body(ci, 'go'), job_body(ci, 'gen_kt'), job_body(ci, 'kotlin'), job_body(release, 'package')):
            self.assertIn('go-version-file: go.mod', body)
            self.assertNotRegex(body, r'(?m)^\s+go-version:')

    def test_default_go_gate_contains_only_go_lint_and_tests(self):
        text = (ROOT / '.github/workflows/ci.yml').read_text()
        go = job_body(text, 'go')
        for command in ('gofmt', 'go vet ./...', 'go test ./...', 'go test -race ./...'):
            self.assertIn(command, go)
        for functional_command in ('scripts/test.sh unit', 'python', 'generate', 'gradle'):
            self.assertNotIn(functional_command, go)
        self.assertIn('fail-fast: false', go)
        self.assertIn('[ubuntu-latest, macos-latest, windows-latest]', go)

    def test_dependency_preparation_precedes_offline_checks_in_every_build_job(self):
        ci = (ROOT / '.github/workflows/ci.yml').read_text()
        release = (ROOT / '.github/workflows/release.yml').read_text()
        for body, first_check in ((job_body(ci, 'go'), 'Go format'),
                                  (job_body(ci, 'gen_kt'), 'Source installation'),
                                  (job_body(ci, 'kotlin'), 'Explicit Kotlin dependency preparation'),
                                  (job_body(release, 'package'), 'Build allowlisted')):
            self.assertIn('GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org go mod download', body)
            self.assertIn('go mod verify', body)
            self.assertLess(body.index('go mod download'), body.index('go mod verify'))
            self.assertLess(body.index('go mod verify'), body.index(first_check))
        for body in (job_body(ci, 'go'), job_body(ci, 'gen_kt'), job_body(ci, 'kotlin')):
            self.assertIn('GOPROXY: "off"', body)
            self.assertIn('GOSUMDB: "off"', body)

    def test_kotlin_runtime_gate_waits_for_go_and_prepares_dependencies_explicitly(self):
        text = (ROOT / '.github/workflows/ci.yml').read_text()
        body = job_body(text, 'kotlin')
        self.assertRegex(body, r'(?m)^    needs: go$')
        self.assertIn('suite: [models, http, profiles]', body)
        self.assertIn('java-version: "21"', body)
        self.assertIn('gradle-version: "9.3.1"', body)
        preparation = 'bash scripts/kotlin/prepare-dependencies.sh'
        execution = 'bash scripts/test.sh "$KOTLIN_SUITE"'
        self.assertLess(body.index(preparation), body.index(execution))
        self.assertIn('feature directional-models', body)
        self.assertIn('feature raw-response-media', body)
        self.assertNotIn('continue-on-error:', body)
        self.assertNotIn('always()', body)

    def test_native_package_gate_checks_current_version_and_installation(self):
        text = (ROOT / '.github/workflows/ci.yml').read_text()
        generated = job_body(text, 'gen_kt')
        source = (ROOT / 'internal/cli/arguments.go').read_text()
        version = re.search(r'^const version = "([^"]+)"$', source, re.MULTILINE).group(1)
        self.assertIn("version = '" + version + "'", generated)
        self.assertIn('scripts/release/package.py', generated)
        self.assertIn('scripts/release/verify.py', generated)
        self.assertIn('verify.native_target()', generated)
        self.assertIn('hashlib.sha256(archive.read_bytes())', generated)
        self.assertNotIn('continue-on-error:', generated)

    def test_generation_is_blocked_until_entire_go_matrix_passes(self):
        text = (ROOT / '.github/workflows/ci.yml').read_text()
        generated = job_body(text, 'gen_kt')
        self.assertRegex(generated, r'(?m)^    needs: go$')
        self.assertNotIn('always()', generated)
        self.assertNotIn('continue-on-error:', generated)
        self.assertIn("'generate'", generated)
        self.assertIn("'--check'", generated)
        self.assertIn("unittest discover -s scripts/tests", generated)
        self.assertLess(generated.index("'generate'"), generated.index('unittest discover'))
        self.assertIn('scripts/publication/check.py', generated)
        release = (ROOT / '.github/workflows/release.yml').read_text()
        self.assertIn('uses: ./.github/workflows/ci.yml', job_body(release, 'verify'))
        self.assertRegex(job_body(release, 'package'), r'(?m)^    needs: verify$')

    def test_release_verifies_published_archives_on_their_own_hosts(self):
        # The published assets are cross-built on one runner, so the packaged
        # binary is only exercised if a later job runs it natively.
        release = (ROOT / '.github/workflows/release.yml').read_text()
        body = job_body(release, 'verify-native')
        self.assertRegex(body, r'(?m)^    needs: package$')
        self.assertIn('package-${{ matrix.target }}', body)
        self.assertIn('scripts/release/verify.py', body)
        self.assertNotIn('continue-on-error:', body)
        for target, host in (('darwin-arm64', 'macos-latest'),
                             ('darwin-amd64', 'macos-15-intel'),
                             ('linux-amd64', 'ubuntu-latest'),
                             ('linux-arm64', 'ubuntu-24.04-arm'),
                             ('windows-amd64', 'windows-latest')):
            self.assertIn('target: ' + target, body)
            self.assertIn('os: ' + host, body)
        # A draft must not be created before the native checks pass.
        self.assertIn('needs: [package, verify-native]', job_body(release, 'draft'))


if __name__ == '__main__':
    unittest.main()
