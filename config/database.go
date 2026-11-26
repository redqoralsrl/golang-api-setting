package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Database PostgreSQL 접속 정보. 마이그레이션 CLI 처럼 DB만 필요한 곳은 LoadDatabase 만 쓴다.
type Database struct {
	User     string
	Password string
	Name     string
	Host     string
	Port     string
	SSLMode  string // disable, require, verify-full ... 비우면 dev 는 disable, 그 외 require
}

func LoadDatabase() (Database, error) {
	d := Database{
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Name:     os.Getenv("DB_NAME"),
		Host:     os.Getenv("DB_HOST"),
		Port:     env("DB_PORT", "5432"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}
	if d.SSLMode == "" {
		d.SSLMode = "require"
		if Stage(env("STAGE", string(StageDev))) == StageDev {
			d.SSLMode = "disable"
		}
	}

	var errs []error
	for _, r := range []struct{ name, value string }{
		{"DB_USER", d.User},
		{"DB_PASSWORD", d.Password},
		{"DB_NAME", d.Name},
		{"DB_HOST", d.Host},
	} {
		if strings.TrimSpace(r.value) == "" {
			errs = append(errs, fmt.Errorf("%s is required", r.name))
		}
	}
	return d, errors.Join(errs...)
}
