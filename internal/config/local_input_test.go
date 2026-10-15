package config

import (
	"strings"
	"testing"
)

func TestLocalInputConfigRejectsDownloadAndStdin(t *testing.T) {
	for _, tc := range []struct{ text, code string }{
		{"config_version=1\n[[specs]]\nname='a'\n[specs.download]\nurl='https://example.test'\n", "E_CONFIG_UNKNOWN_FIELD"},
		{"config_version=1\n[defaults]\ninput='-'\n[[specs]]\nname='a'\ninput='local.json'\n", "E_CONFIG_INVALID"},
		{"config_version=1\n[[specs]]\nname='a'\ninput='-'\n", "E_CONFIG_INVALID"},
		{"config_version=1\n[defaults]\nheaders_from_env={Authorization='SYNTHETIC_PRIVATE'}\n", "E_CONFIG_UNKNOWN_FIELD"},
	} {
		_, e := Parse([]byte(tc.text))
		if e == nil || e.Code != tc.code || strings.Contains(e.Message, "SYNTHETIC_PRIVATE") {
			t.Fatal("local configuration contract", e)
		}
	}
	cfg, e := Parse([]byte("config_version=1\n[[specs]]\nname='a'\ninput='./-'\n"))
	if e != nil || cfg.Specs[0].Settings["input"] != "./-" {
		t.Fatal("literal dash path", cfg, e)
	}
}
