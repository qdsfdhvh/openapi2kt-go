package config

import (
	"reflect"
	"testing"
)

func TestConfigWarningPolicyPresenceAndTypes(t *testing.T) {
	cfg, err := Parse([]byte("config_version=1\n[defaults]\ndeny_warnings=true\nallow_warnings=['W_TUPLE_APPROXIMATION']\n[[specs]]\nname='a'\ndeny_warnings=false\nallow_warnings=[]\n"))
	if err != nil {
		t.Fatal(err)
	}
	effective := Overlay(cfg.Defaults, cfg.Specs[0].Settings)
	if effective["deny_warnings"] != false || !reflect.DeepEqual(effective["allow_warnings"], []string{}) {
		t.Fatal("lost explicit false/empty policy", effective)
	}
	for _, field := range []string{"deny_warnings='true'", "allow_warnings=[true]", "allow_warnings='W_TUPLE_APPROXIMATION'", "allow_warnings=['W_*']", "allow_warnings=['W_lowercase']"} {
		if _, err := Parse([]byte("config_version=1\n[defaults]\n" + field)); err == nil {
			t.Fatal("coerced warning policy", field)
		}
	}
}

func TestConfigWarningPolicyRejectsInvalidOverriddenCode(t *testing.T) {
	_, err := Parse([]byte("config_version=1\n[defaults]\nallow_warnings=['W_*']\n[[specs]]\nname='a'\nallow_warnings=[]\n"))
	if err == nil || err.Code != "E_CONFIG_INVALID" || err.Pointer == nil || *err.Pointer != "/defaults/allow_warnings" {
		t.Fatal("hidden invalid original warning code", err)
	}
}
