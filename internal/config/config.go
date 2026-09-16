package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
)

type Config struct {
	HTTPConfig     HTTPConfig
	PostgresConfig PostgresConfig
}

type HTTPConfig struct {
	Addr string
}

type PostgresConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	DBName   string
	SSLMode  string
}

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) Load() {
	if err := godotenv.Load(); err != nil {
		log.Println("warn: .env not found")
	}

	httpConf := HTTPConfig{Addr: getEnv("HTTP_PORT", ":8080")}
	postgresConf := PostgresConfig{
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		Host:     getEnv("DB_HOST", "127.0.0.1"),
		Port:     getEnv("DB_PORT", "5432"),
		DBName:   getEnv("DB_NAME", "messenger_db"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}

	c.HTTPConfig = httpConf
	c.PostgresConfig = postgresConf
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
