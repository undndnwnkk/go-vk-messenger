package config

import (
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPConfig     HTTPConfig
	PostgresConfig PostgresConfig
	JWTConfig      JWTConfig
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
	return nil
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
