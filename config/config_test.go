package config

import (
	"strings"
	"testing"
)

func setValidEnv(t *testing.T) {
	t.Helper()
	for k, v := range map[string]string{
		"STAGE":          "dev",
		"DB_USER":        "u",
		"DB_PASSWORD":    "p",
		"DB_NAME":        "n",
		"DB_HOST":        "h",
		"API_SECRET_KEY": strings.Repeat("a", 32),
		"CURSOR_SECRET":  strings.Repeat("b", 32),
	} {
		t.Setenv(k, v)
	}
}

func TestLoadValid(t *testing.T) {
	setValidEnv(t)
	t.Setenv("CORS_ALLOWED_ORIGINS", " https://a.com , ,https://b.com")

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(c.CORSAllowedOrigins) != 2 || c.CORSAllowedOrigins[1] != "https://b.com" {
		t.Errorf("origins parsed wrong: %v", c.CORSAllowedOrigins)
	}
	if !c.MigrateOnStart || c.JobsEnabled || c.APIPort != "8080" {
		t.Errorf("defaults wrong: %+v", c)
	}
}

func TestLoadInvalid(t *testing.T) {
	tests := map[string]map[string]string{
		"bad stage":           {"STAGE": "local"},
		"short secret":        {"API_SECRET_KEY": "short"},
		"cursor not 32":       {"CURSOR_SECRET": "short"},
		"missing db":          {"DB_HOST": ""},
		"bad bool":            {"JOBS_ENABLED": "yes please"},
		"negative hops":       {"TRUSTED_PROXY_HOPS": "-1"},
		"prod without origin": {"STAGE": "prod"},
	}
	for name, override := range tests {
		t.Run(name, func(t *testing.T) {
			setValidEnv(t)
			for k, v := range override {
				t.Setenv(k, v)
			}
			if _, err := Load(); err == nil {
				t.Error("expected error")
			}
		})
	}
}
