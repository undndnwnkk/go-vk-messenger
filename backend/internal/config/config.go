package config

import (
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPConfig     HTTPConfig
	PostgresConfig PostgresConfig
	JWTConfig      JWTConfig
	RedisURL       string
	RedisPrefix    string
	MetricsAddr    string
	// CORSAllowedOrigins lists exact browser origins allowed to call the API.
	// Empty disables CORS headers entirely.
	CORSAllowedOrigins []string
}

type HTTPConfig struct {
	Port string
}

type PostgresConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	DBName   string
	SSLMode  string
}

type JWTConfig struct {
	Secret string
	TTL    time.Duration
}

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) Load() error {
	if err := godotenv.Load(); err != nil {
		log.Println("warn: .env not found")
	}

	httpConf := HTTPConfig{Port: getEnv("HTTP_PORT", "8080")}
	postgresConf := PostgresConfig{
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		Host:     getEnv("DB_HOST", "127.0.0.1"),
		Port:     getEnv("DB_PORT", "5432"),
		DBName:   getEnv("DB_NAME", "messenger_db"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}

	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		return errors.New("JWT_SECRET is required")
	}

	accessTTL, err := parseJWTTTL(getEnv("JWT_TTL", "15"))
	if err != nil {
		return fmt.Errorf("invalid JWT_TTL: %w", err)
	}
	jwtConf := JWTConfig{
		Secret: jwtSecret,
		TTL:    accessTTL,
	}

	c.HTTPConfig = httpConf
	c.PostgresConfig = postgresConf
	c.JWTConfig = jwtConf
	c.RedisURL = getEnv("REDIS_URL", "")
	c.MetricsAddr = getEnv("METRICS_ADDR", "127.0.0.1:9090")
	c.RedisPrefix = getEnv("REDIS_PREFIX", "messenger")
	if strings.ContainsAny(c.RedisPrefix, " \t\r\n") {
		return errors.New("REDIS_PREFIX must not contain whitespace")
	}
	origins, err := parseAllowedOrigins(getEnv("CORS_ALLOWED_ORIGINS", ""))
	if err != nil {
		return fmt.Errorf("invalid CORS_ALLOWED_ORIGINS: %w", err)
	}
	c.CORSAllowedOrigins = origins
	return nil
}

// parseAllowedOrigins parses a comma-separated list of origins such as
// "http://localhost:5173,https://app.example.com". Every entry must be a bare
// http(s) origin; wildcards are rejected so the allowlist stays explicit.
func parseAllowedOrigins(value string) ([]string, error) {
	var origins []string
	seen := make(map[string]bool)
	for _, raw := range strings.Split(value, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		u, err := url.Parse(raw)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" ||
			(u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || u.User != nil ||
			strings.Contains(u.Host, "*") {
			return nil, fmt.Errorf("%q is not an origin like https://app.example.com", raw)
		}
		origin := strings.ToLower(u.Scheme + "://" + u.Host)
		if !seen[origin] {
			seen[origin] = true
			origins = append(origins, origin)
		}
	}
	return origins, nil
}

func getEnv(name, defaultVal string) string {
	if value, ok := os.LookupEnv(name); ok && value != "" {
		return value
	}
	return defaultVal
}

func (c *Config) BuildDSN() string {
	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s?sslmode=%s",
		c.PostgresConfig.User,
		c.PostgresConfig.Password,
		c.PostgresConfig.Host,
		c.PostgresConfig.Port,
		c.PostgresConfig.DBName,
		c.PostgresConfig.SSLMode,
	)
}

func (c *Config) HTTPAddr() string {
	return ":" + c.HTTPConfig.Port
}

func parseJWTTTL(value string) (time.Duration, error) {
	if ttl, err := time.ParseDuration(value); err == nil {
		if ttl <= 0 {
			return 0, errors.New("must be positive")
		}
		return ttl, nil
	}

	minutes, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if minutes <= 0 {
		return 0, errors.New("must be positive")
	}

	return time.Duration(minutes) * time.Minute, nil
}
