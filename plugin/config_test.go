package main

import (
	"os"
	"path/filepath"
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
