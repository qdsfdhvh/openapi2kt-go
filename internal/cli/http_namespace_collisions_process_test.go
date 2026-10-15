package cli_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestHTTPNamespaceCollisionNativeDeterminism(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, read(t, filepath.Join(repo, "tests", "fixtures", "http", "namespace-collisions.json")))
	result(t, p.generate(t), 0)
	before := snapshot(t, p.Output)
	api := string(read(t, filepath.Join(p.Output, "org/example/catalog/api/ShadowsApi.kt")))
	if !strings.Contains(api, "import io.ktor.http.HttpMethod as OpenapiKtHttpMethodImport") || !strings.Contains(api, "import kotlinx.coroutines.flow.flow as OpenapiKtFlowImport") {
		t.Fatal(api)
	}
	for _, extra := range [][]string{nil, {"--check"}} {
		result(t, p.generate(t, extra...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("namespace alias no-op/check changed bytes or mtime")
		}
	}
	result(t, invoke(t, "", "validate", "--input", p.Input, "--diagnostic-format", "json"), 0)
}
