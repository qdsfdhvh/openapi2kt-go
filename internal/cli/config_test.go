package cli

import (
	"openapi-kt/core"
	"openapi-kt/internal/config"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfigEffectiveOverlay(t *testing.T) {
	base := arguments{command: "generate", packageName: "org.example.cli", suffix: "CLI", excludeAPIs: []string{"CLI"}, explicit: map[string]bool{"--package-name": true, "--model-name-suffix": true, "--exclude-api": true}}
	s := config.Overlay(config.Settings{"input": "spec.json", "output_dir": "generated", "package_name": "org.example.default", "validation": true, "exclude_apis": []string{"Default"}}, config.Settings{"package_name": "org.example.spec", "validation": false, "exclude_apis": []string{}})
	a := configArguments(base, s, "/config-dir")
	if a.packageName != "org.example.cli" || a.suffix != "CLI" || a.validation || !reflect.DeepEqual(a.excludeAPIs, []string{"CLI"}) || a.input != filepath.Join("/config-dir", "spec.json") || a.output != filepath.Join("/config-dir", "generated") {
		t.Fatal(a)
	}
	base.explicit = nil
	a = configArguments(base, s, "/config-dir")
	if a.packageName != "org.example.spec" || a.validation || !reflect.DeepEqual(a.excludeAPIs, []string{}) || a.format != core.Auto {
		t.Fatal(a)
	}
}
func TestConfigCLIClosedArgumentSurface(t *testing.T) {
	for _, args := range [][]string{{"generate", "--config=x", "--spec=a", "--check"}, {"generate", "--config=x", "--reconcile"}, {"validate", "--config=x", "--spec=a"}} {
		if _, _, ok := parse(args); !ok {
			t.Fatal(args)
		}
	}
	for _, args := range [][]string{{"generate", "--config=x", "--input=x"}, {"generate", "--config="}, {"generate", "--config=x", "--spec="}, {"generate", "--config=x", "--spec=a", "--reconcile"}, {"validate", "--config=x", "--reconcile"}, {"generate", "--spec=a"}, {"download", "--config=x", "--spec=a"}, {"download", "--config=x", "--output=x"}, {"generate", "--config=x", "--config=x"}, {"generate", "--config=x", "--reconcile=true"}} {
		if _, _, ok := parse(args); ok {
			t.Fatal(args)
		}
	}
}
