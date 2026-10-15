package core_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/testutil"
)

func TestHTTPClientNameAndExclusions(t *testing.T) {
	input, err := os.ReadFile(filepath.Join(testutil.RepoRoot(), "tests", "fixtures", "http", "basic.json"))
	if err != nil {
		t.Fatal(err)
	}
	rq := request(input, "Dto")
	rq.Options.APIClientName = "ChosenClient"
	rq.Options.ExcludeAPIs = []string{"WidgetsApi", "DefaultApi", "WidgetsApi", "AbsentApi"}
	before := append([]string(nil), rq.Options.ExcludeAPIs...)
	r, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, rq.Options.ExcludeAPIs) || len(r.Artifacts.Files) != 4 {
		t.Fatal(r, rq.Options)
	}
	client := source(t, r, "ChosenClient")
	if strings.Contains(client, "WidgetsApi") || strings.Contains(client, "DefaultApi") || !strings.Contains(client, "UserAdminApi") {
		t.Fatal(client)
	}
	rq.Options.ExcludeAPIs = append(rq.Options.ExcludeAPIs, "UserAdminApi")
	r, err = core.GenerateWithDiagnostics(rq)
	if err != nil || len(r.Artifacts.Files) != 3 {
		t.Fatal(r, err)
	}
	rq = request(schemas(`{"Widget":{"type":"object"}}`), "")
	base, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal(err)
	}
	rq.Options.APIClientName = "ChosenClient"
	rq.Options.ExcludeAPIs = []string{"DefaultApi"}
	withOptions, err := core.GenerateWithDiagnostics(rq)
	if err != nil || !reflect.DeepEqual(base, withOptions) {
		t.Fatal("model-only files changed", err)
	}
}

func TestHTTPClientOptionsDoNotHideInvalidOperations(t *testing.T) {
	rq := request(httpDoc(`{"callbacks":{},"responses":{"204":{}}}`), "")
	rq.Options.ExcludeAPIs = []string{"DefaultApi"}
	r, err := core.GenerateWithDiagnostics(rq)
	if err == nil || len(r.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_OPERATIONS_UNSUPPORTED" {
		t.Fatal(r, err)
	}
	for _, bad := range []string{"../escape", "class", "CON", "has space", "123", "`Quoted`"} {
		rq.Options.APIClientName = bad
		_, err = core.GenerateWithDiagnostics(rq)
		if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_INVALID" || err.(core.Diagnostics)[0].Pointer != nil {
			t.Fatal(bad, err)
		}
	}
	// An explicit client name bypasses title-to-name inference, not title typing.
	rq = request([]byte(`{"openapi":"3.1.0","info":{"title":"雪"},"paths":{"/ping":{"get":{"responses":{"204":{}}}}}}`), "")
	rq.Options.APIClientName = "ChosenClient"
	r, err = core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal(err)
	}
	_ = source(t, r, "ChosenClient")
}
