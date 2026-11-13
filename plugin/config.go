package main

import (
	"fmt"
	"os"
	"time"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Listen                 string         `yaml:"listen"`
	RadiusAuthServer       string         `yaml:"radius_auth_server"`
	RadiusAccountingServer string         `yaml:"radius_accounting_server"`
	RadiusSecret           string         `yaml:"radius_secret"`
	NASIdentifier          string         `yaml:"nas_identifier"`
	RadiusTimeout          time.Duration  `yaml:"-"`
	CacheDuration          time.Duration  `yaml:"-"`
	LogPath                string         `yaml:"log_path"`
	Retention              int            `yaml:"retention"`
	HealthPath             string         `yaml:"health_path"`
	PostAccountingAction   string         `yaml:"post_accounting_action"`
	TimeoutText            string         `yaml:"radius_timeout"`
	CacheText              string         `yaml:"cache_duration"`
	Bypasses               []BypassConfig `yaml:"bypasses"`
}

type BypassConfig struct {
	Name        string        `yaml:"name"`
	File        string        `yaml:"file"`
	URL         string        `yaml:"url"`
	Whitelist   bool          `yaml:"whitelist"`
	PreferIPv6  bool          `yaml:"prefer-ipv6"`
	Refresh     time.Duration `yaml:"-"`
	RefreshText string        `yaml:"refresh"`
}

func loadConfig(path string) (Config, error) {
	c := Config{
		Listen:        "127.0.0.1:8000",
		NASIdentifier: "squid", TimeoutText: "3s", CacheText: "30m",
		LogPath: "/var/log/gost/gost.log", Retention: 180,
		HealthPath: "/var/log/gost/accounting.last",
	}
	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal([]byte(os.ExpandEnv(string(data))), &c); err != nil {
			return c, fmt.Errorf("parse config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return c, err
	}
	var err error
	if c.RadiusTimeout, err = time.ParseDuration(c.TimeoutText); err != nil {
		return c, fmt.Errorf("radius_timeout: %w", err)
	}
	if c.CacheDuration, err = time.ParseDuration(c.CacheText); err != nil {
		return c, fmt.Errorf("cache_duration: %w", err)
	}
	seenBypasses := make(map[string]struct{}, len(c.Bypasses))
	for i := range c.Bypasses {
		bypass := &c.Bypasses[i]
		if bypass.Name == "" {
			return c, fmt.Errorf("bypasses[%d].name is required", i)
		}
		if _, ok := seenBypasses[bypass.Name]; ok {
			return c, fmt.Errorf("duplicate bypass name %q", bypass.Name)
		}
		seenBypasses[bypass.Name] = struct{}{}
		if bypass.File == "" && bypass.URL == "" {
			return c, fmt.Errorf("bypass %q requires file or url", bypass.Name)
		}
		if bypass.RefreshText != "" {
			bypass.Refresh, err = time.ParseDuration(bypass.RefreshText)
			if err != nil {
				return c, fmt.Errorf("bypass %q refresh: %w", bypass.Name, err)
			}
			if bypass.Refresh <= 0 {
				return c, fmt.Errorf("bypass %q refresh must be positive", bypass.Name)
			}
		}
	}
	if c.RadiusSecret == "" {
		return c, fmt.Errorf("radius_secret is required")
	}
	if c.RadiusAuthServer == "" {
		return c, fmt.Errorf("radius_auth_server is required")
	}
	if c.RadiusAccountingServer == "" {
		return c, fmt.Errorf("radius_accounting_server is required")
	}
	if c.Retention < 2 {
		return c, fmt.Errorf("retention must be at least 2")
	}
	return c, nil
}
