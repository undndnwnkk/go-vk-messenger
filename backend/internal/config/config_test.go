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

func TestParseAllowedOrigins(t *testing.T) {
	got, err := parseAllowedOrigins(" http://localhost:5173, HTTPS://App.Example.com/ ,,http://localhost:5173,http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("parseAllowedOrigins returned error: %v", err)
	}
	want := []string{"http://localhost:5173", "https://app.example.com", "http://127.0.0.1:8080"}
	if len(got) != len(want) {
		t.Fatalf("origins = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("origins = %v, want %v", got, want)
		}
	}

	if got, err := parseAllowedOrigins(""); err != nil || got != nil {
		t.Fatalf("empty value: origins=%v err=%v, want nil, nil", got, err)
	}
}

func TestParseAllowedOriginsRejectsNonOrigins(t *testing.T) {
	for _, value := range []string{
		"*",
		"localhost:5173",
		"ftp://example.com",
		"https://*.example.com",
		"https://example.com/app",
		"https://example.com?x=1",
		"https://user@example.com",
		"null",
		"http://ok.example,not-an-origin",
	} {
		if _, err := parseAllowedOrigins(value); err == nil {
			t.Errorf("parseAllowedOrigins(%q) returned nil error", value)
		}
	}
}

func TestLoadReadsCORSAllowedOrigins(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("JWT_TTL", "15")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")
	chdirTemp(t)

	conf := NewConfig()
	if err := conf.Load(); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(conf.CORSAllowedOrigins) != 1 || conf.CORSAllowedOrigins[0] != "http://localhost:5173" {
		t.Fatalf("CORSAllowedOrigins = %v", conf.CORSAllowedOrigins)
	}

	t.Setenv("CORS_ALLOWED_ORIGINS", "*")
	if err := NewConfig().Load(); err == nil {
		t.Fatal("Load accepted wildcard CORS origin")
	}
}
