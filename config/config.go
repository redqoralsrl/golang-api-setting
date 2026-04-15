package config

import (
	"os"
	"strings"
)

type Config struct {
	Stage           string
	DBUser          string
	DBPassword      string
	DBName          string
	DBHost          string
	DBPort          string
	APIHost         string
	APIPort         string
	APIXApikey      string
	RedisHost       string
	RedisPort       string
	CursorSecret    string
	APISecretKey    string
	IPInfoToken     string
	SMTPUsername    string
	SMTPPassword    string
	SwaggerID       string
	SwaggerPassword string
}

type StageType string

const (
	stageDev StageType = "dev"
)

func (c *Config) IsDev() bool { return StageType(c.Stage) == stageDev }

func (c *Config) JobsEnabled() bool {
	switch strings.TrimSpace(strings.ToLower(c.APIHost)) {
	case "localhost":
		return false
	default:
		return true
	}
}

func LoadConfig() *Config {
	return &Config{
		Stage:           os.Getenv("STAGE"),
		DBUser:          os.Getenv("DB_USER"),
		DBPassword:      os.Getenv("DB_PASSWORD"),
		DBName:          os.Getenv("DB_NAME"),
		DBHost:          os.Getenv("DB_HOST"),
		DBPort:          os.Getenv("DB_PORT"),
		APIHost:         os.Getenv("API_HOST"),
		APIPort:         os.Getenv("API_PORT"),
		APIXApikey:      os.Getenv("API_X_API_KEY"),
		RedisHost:       os.Getenv("REDIS_HOST"),
		RedisPort:       os.Getenv("REDIS_PORT"),
		CursorSecret:    os.Getenv("CURSOR_SECRET"),
		APISecretKey:    os.Getenv("API_SECRET_KEY"),
		IPInfoToken:     os.Getenv("IP_INFO_TOKEN"),
		SMTPUsername:    os.Getenv("SMTP_USERNAME"),
		SMTPPassword:    os.Getenv("SMTP_PASSWORD"),
		SwaggerID:       os.Getenv("SWAGGER_ID"),
		SwaggerPassword: os.Getenv("SWAGGER_PASSWORD"),
	}
}
