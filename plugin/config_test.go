package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigExpandsEnvironment(t *testing.T) {
	t.Setenv("TEST_RADIUS_SECRET", "expanded-secret")
	path := filepath.Join(t.TempDir(), "plugin.yml")
	data := "radius_auth_server: auth.example:11812\n" +
		"radius_accounting_server: acct.example:11813\n" +
		"radius_secret: ${TEST_RADIUS_SECRET}\n"
	require.NoError(t, os.WriteFile(path, []byte(data), 0600))

	config, err := loadConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "expanded-secret", config.RadiusSecret)
	assert.Equal(t, "auth.example:11812", config.RadiusAuthServer)
	assert.Equal(t, "acct.example:11813", config.RadiusAccountingServer)
	assert.Empty(t, config.PostAccountingAction)
}

func TestLoadConfigPostAccountingAction(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plugin.yml")
	data := "radius_auth_server: auth.example:1812\n" +
		"radius_accounting_server: acct.example:1813\n" +
		"radius_secret: secret\n" +
		"post_accounting_action: notify-gost --reload\n"
	require.NoError(t, os.WriteFile(path, []byte(data), 0600))

	config, err := loadConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "notify-gost --reload", config.PostAccountingAction)
}

func TestLoadConfigRejectsDuplicateBypassNames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plugin.yml")
	data := "radius_auth_server: auth.example:1812\n" +
		"radius_accounting_server: acct.example:1813\n" +
		"radius_secret: secret\n" +
		"bypasses:\n" +
		"  - name: duplicate\n    file: /one\n" +
		"  - name: duplicate\n    url: https://example.com/two\n"
	require.NoError(t, os.WriteFile(path, []byte(data), 0600))

	_, err := loadConfig(path)
	assert.ErrorContains(t, err, "duplicate bypass name")
}

func TestLoadConfigParsesBypassOptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plugin.yml")
	data := "radius_auth_server: auth.example:1812\n" +
		"radius_accounting_server: acct.example:1813\n" +
		"radius_secret: secret\n" +
		"bypasses:\n" +
		"  - name: geo\n    file: /geo.txt\n    whitelist: true\n    prefer-ipv6: true\n    refresh: 1h\n"
	require.NoError(t, os.WriteFile(path, []byte(data), 0600))

	config, err := loadConfig(path)
	require.NoError(t, err)
	require.Len(t, config.Bypasses, 1)
	assert.Equal(t, "1h0m0s", config.Bypasses[0].Refresh.String())
	assert.True(t, config.Bypasses[0].Whitelist)
	assert.True(t, config.Bypasses[0].PreferIPv6)
}
