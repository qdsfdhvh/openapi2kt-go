package naming

import "testing"

func TestOperationNames(t *testing.T) {
	for _, tc := range []struct{ id, method, path, want string }{
		{"widgets.get", "get", "/ignored", "getWidgets"},
		{"widgets.me.profile.update", "patch", "/ignored", "updateMeProfile"},
		{"widgets.search", "get", "/ignored", "search"},
		{"batchGet", "post", "/ignored", "batchGet"},
		{"batch_get", "get", "/ignored", "batch_get"},
		{"users:batchGet", "post", "/ignored", "usersBatchGet"},
		{"list-users", "get", "/ignored", "listUsers"},
		{"widgets.when", "get", "/ignored", "whenWidgets"},
		{"get", "get", "/ignored", "getGet"},
		{"", "get", "/widgets/{id}/@me/items", "getWidgetsItems"},
	} {
		var id *string
		if tc.id != "" {
			id = &tc.id
		}
		got, err := MethodName(id, tc.method, tc.path, "/operationId")
		if err != nil || got != tc.want {
			t.Errorf("%+v: got %q, %v", tc, got, err)
		}
	}
	for _, tc := range []struct{ in, want string }{{"user_name", "userName"}, {"X-Trace-Id", "xTraceId"}, {"UserAdmin", "userAdmin"}, {"API", "api"}, {"X_TRACE_ID", "x_trace_id"}} {
		got, err := APICamelName(tc.in, "/name")
		if err != nil || got != tc.want {
			t.Errorf("%+v: %q %v", tc, got, err)
		}
	}
	for _, bad := range []string{"", "雪", "123", "X-TRACE-ID", "a`b"} {
		if _, err := APICamelName(bad, "/name"); err == nil {
			t.Errorf("accepted unsafe/unsupported API spelling %q", bad)
		}
	}
	if CodeName("when") != "`when`" || CodeName("requestBuilder") != "requestBuilder" {
		t.Fatal("keyword escaping")
	}
}
