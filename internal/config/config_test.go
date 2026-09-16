package config

import (
	"errors"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestLoadRequiresJWTSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("JWT_TTL", "15")
	chdirTemp(t)

	conf := NewConfig()
	if err := conf.Load(); err == nil {
		t.Fatal("Load returned nil error without JWT_SECRET")
	}
}

func TestLoadRejectsInvalidJWTTTL(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("JWT_TTL", "abc")
	chdirTemp(t)

	conf := NewConfig()
	if err := conf.Load(); err == nil {
		t.Fatal("Load returned nil error for invalid JWT_TTL")
	}
}

func TestLoadUsesHTTPPortAsPort(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("JWT_TTL", "15")
	t.Setenv("HTTP_PORT", "8080")
	chdirTemp(t)

	conf := NewConfig()
	if err := conf.Load(); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if conf.HTTPConfig.Port != "8080" {
		t.Fatalf("HTTP port = %q, want %q", conf.HTTPConfig.Port, "8080")
	}
	if conf.HTTPAddr() != ":8080" {
		t.Fatalf("HTTP addr = %q, want %q", conf.HTTPAddr(), ":8080")
	}
}

func TestParseJWTTTL(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{name: "minutes", value: "15", want: 15 * time.Minute},
		{name: "duration", value: "30m", want: 30 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseJWTTTL(tt.value)
			if err != nil {
				t.Fatalf("parseJWTTTL returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("parseJWTTTL = %v, want %v", got, tt.want)
			}
		})
	}
}

func chdirTemp(t *testing.T) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("restore working directory: %v", err)
		}
	})
}

func TestParseJWTTTLRejectsNonPositiveValues(t *testing.T) {
	for _, value := range []string{"0", "-1", strconv.Itoa(-15)} {
		if _, err := parseJWTTTL(value); err == nil {
			t.Fatalf("parseJWTTTL(%q) returned nil error", value)
		}
	}
}
