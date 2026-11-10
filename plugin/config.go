package main

import (
	"fmt"
	"os"
	"time"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Listen                 string        `yaml:"listen"`
	RadiusAuthServer       string        `yaml:"radius_auth_server"`
	RadiusAccountingServer string        `yaml:"radius_accounting_server"`
	RadiusSecret           string        `yaml:"radius_secret"`
	NASIdentifier          string        `yaml:"nas_identifier"`
	RadiusTimeout          time.Duration `yaml:"-"`
	CacheDuration          time.Duration `yaml:"-"`
	LogPath                string        `yaml:"log_path"`
	Retention              int           `yaml:"retention"`
	HealthPath             string        `yaml:"health_path"`
	TimeoutText            string        `yaml:"radius_timeout"`
	CacheText              string        `yaml:"cache_duration"`
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
