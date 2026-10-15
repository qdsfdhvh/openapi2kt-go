# Third-party notices

The generator uses the unmodified vendored `go.yaml.in/yaml/v3` **v3.0.4** module
from [yaml/go-yaml](https://github.com/yaml/go-yaml/tree/v3.0.4), maintained by the
YAML organization. The module requires Go 1.16, uses pure Go (no CGO/platform
adapter), and has no runtime dependency beyond the standard library. Its upstream
`gopkg.in/check.v1` requirement is test-only and is not vendored or used by this
project's tests. Versions and checksums are declared in `go.mod` / `go.sum`;
ordinary builds use `vendor/` and can run with module network access disabled.

The TOML dependency already declared by the project is also vendored unchanged;
its attribution and complete MIT license remain in [THIRD-PARTY-NOTICES](THIRD-PARTY-NOTICES).
New binary archives retain both notice files.

The parser is a Go port of libyaml. Its files retain upstream copyright notices,
[MIT and Apache licensing](vendor/go.yaml.in/yaml/v3/LICENSE), and
[NOTICE](vendor/go.yaml.in/yaml/v3/NOTICE). The complete
[Apache License 2.0](third-party/APACHE-2.0.txt) is also included. Binary archives
must include these notices and license texts alongside the project and Go
licenses. Vendoring does not make upstream YAML decoding policy the product
contract: the loader applies the bounded [YAML input profile](docs/yaml-input.md),
including its own alias expansion budgets and a preflight rejection of `%TAG`
directives before upstream URI-prefix allocation.
