package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Stage string

const (
	StageDev     Stage = "dev"
	StageStaging Stage = "staging"
	StageProd    Stage = "prod"

	minAPISecretKeyLen = 32
	cursorSecretLen    = 32 // AES-256 키 길이
)

type Config struct {
	Stage Stage

	APIHost string
	APIPort string

	DB Database

	// Redis 는 아직 연결하지 않는다 (internal/database/redis)
	RedisHost     string
	RedisPort     string
	RedisPassword string

	APISecretKey string // JWT 서명 키
	CursorSecret string // 페이지네이션 커서 암호화 키
	APIKey       string // 비어 있지 않으면 /api/v1 요청에 X-API-Key 헤더를 요구한다

	CORSAllowedOrigins []string // CSRF 신뢰 출처로도 쓴다
	TrustedProxyHops   int      // 클라이언트 IP를 X-Forwarded-For 오른쪽에서 몇 번째 값으로 읽을지. 0이면 RemoteAddr
	JobsEnabled        bool     // 주기 작업(cron) 실행 여부
	MigrateOnStart     bool     // 시작할 때 DB 마이그레이션(Up) 적용 여부

	IPInfoToken     string
	SMTPUsername    string
	SMTPPassword    string
	SwaggerID       string
	SwaggerPassword string
}

func (c *Config) IsDev() bool { return c.Stage == StageDev }

// Load 환경변수를 읽고 검증한다. 에러가 있어도 Config 는 돌려주므로 logger 생성에 Stage 를 쓸 수 있다.
func Load() (*Config, error) {
	var errs []error

	c := &Config{
		Stage:           Stage(env("STAGE", string(StageDev))),
		APIHost:         os.Getenv("API_HOST"),
		APIPort:         env("API_PORT", "8080"),
		RedisHost:       os.Getenv("REDIS_HOST"),
		RedisPort:       os.Getenv("REDIS_PORT"),
		RedisPassword:   os.Getenv("REDIS_PASSWORD"),
		APISecretKey:    os.Getenv("API_SECRET_KEY"),
		CursorSecret:    os.Getenv("CURSOR_SECRET"),
		APIKey:          os.Getenv("API_X_API_KEY"),
		IPInfoToken:     os.Getenv("IP_INFO_TOKEN"),
		SMTPUsername:    os.Getenv("SMTP_USERNAME"),
		SMTPPassword:    os.Getenv("SMTP_PASSWORD"),
		SwaggerID:       os.Getenv("SWAGGER_ID"),
		SwaggerPassword: os.Getenv("SWAGGER_PASSWORD"),
	}

	var err error
	if c.DB, err = LoadDatabase(); err != nil {
		errs = append(errs, err)
	}
	c.CORSAllowedOrigins = envList("CORS_ALLOWED_ORIGINS")
	if c.TrustedProxyHops, err = envInt("TRUSTED_PROXY_HOPS", 0); err != nil {
		errs = append(errs, err)
	}
	if c.JobsEnabled, err = envBool("JOBS_ENABLED", false); err != nil {
		errs = append(errs, err)
	}
	if c.MigrateOnStart, err = envBool("MIGRATE_ON_START", true); err != nil {
		errs = append(errs, err)
	}

	errs = append(errs, c.validate()...)
	return c, errors.Join(errs...)
}

func (c *Config) validate() []error {
	var errs []error

	switch c.Stage {
	case StageDev, StageStaging, StageProd:
	default:
		errs = append(errs, errors.New("STAGE must be one of dev, staging, prod"))
	}
	if len(c.APISecretKey) < minAPISecretKeyLen {
		errs = append(errs, fmt.Errorf("API_SECRET_KEY must be at least %d characters", minAPISecretKeyLen))
	}
	if len(c.CursorSecret) != cursorSecretLen {
		errs = append(errs, fmt.Errorf("CURSOR_SECRET must be exactly %d characters", cursorSecretLen))
	}
	if c.TrustedProxyHops < 0 {
		errs = append(errs, errors.New("TRUSTED_PROXY_HOPS must be 0 or greater"))
	}

	if !c.IsDev() && len(c.CORSAllowedOrigins) == 0 {
		errs = append(errs, errors.New("CORS_ALLOWED_ORIGINS is required outside dev"))
	}

	return errs
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envList(key string) []string {
	var out []string
	for v := range strings.SplitSeq(os.Getenv(key), ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func envInt(key string, fallback int) (int, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback, fmt.Errorf("%s must be an integer", key)
	}
	return n, nil
}

func envBool(key string, fallback bool) (bool, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback, fmt.Errorf("%s must be true or false", key)
	}
	return b, nil
}
