package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigExpandsEnvironment(t *testing.T) {
	t.Setenv("TEST_RADIUS_SECRET", "expanded-secret")
	path := filepath.Join(t.TempDir(), "plugin.yml")
	data := "radius_auth_server: auth.example:11812\n" +
		"radius_accounting_server: acct.example:11813\n" +
		"radius_secret: ${TEST_RADIUS_SECRET}\n"
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}

	config, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if config.RadiusSecret != "expanded-secret" {
		t.Fatalf("radius_secret = %q", config.RadiusSecret)
	}
	if config.RadiusAuthServer != "auth.example:11812" {
		t.Fatalf("radius_auth_server = %q", config.RadiusAuthServer)
	}
	if config.RadiusAccountingServer != "acct.example:11813" {
		t.Fatalf("radius_accounting_server = %q", config.RadiusAccountingServer)
	}
}

func TestLoadConfigRejectsDuplicateBypassNames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plugin.yml")
	data := "radius_auth_server: auth.example:1812\n" +
		"radius_accounting_server: acct.example:1813\n" +
		"radius_secret: secret\n" +
		"bypasses:\n" +
		"  - name: duplicate\n    file: /one\n" +
		"  - name: duplicate\n    url: https://example.com/two\n"
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := loadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "duplicate bypass name") {
		t.Fatalf("error = %v", err)
	}
}

func TestLoadConfigParsesBypassOptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plugin.yml")
	data := "radius_auth_server: auth.example:1812\n" +
		"radius_accounting_server: acct.example:1813\n" +
		"radius_secret: secret\n" +
		"bypasses:\n" +
		"  - name: geo\n    file: /geo.txt\n    prefer-ipv6: true\n    refresh: 1h\n"
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}

	config, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Bypasses) != 1 || config.Bypasses[0].Refresh.String() != "1h0m0s" || !config.Bypasses[0].PreferIPv6 {
		t.Fatalf("bypasses = %+v", config.Bypasses)
	}
}
