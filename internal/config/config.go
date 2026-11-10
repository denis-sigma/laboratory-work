package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTP            HTTPConfig
	Database        DatabaseConfig
	LogLevel        string
	ShutdownTimeout time.Duration
	OTel            OTelConfig
}

type HTTPConfig struct {
	Addr              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

type DatabaseConfig struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	ConnectTimeout  time.Duration
	QueryTimeout    time.Duration
}

type OTelConfig struct {
	ServiceName string
	Endpoint    string
}

func Load() (Config, error) {
	var cfg Config
	var err error

	cfg.HTTP.Addr, err = requiredEnv("HTTP_ADDR")
	if err != nil {
		return Config{}, err
	}

	cfg.Database.URL, err = requiredEnv("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}

	cfg.LogLevel, err = requiredEnv("LOG_LEVEL")
	if err != nil {
		return Config{}, err
	}

	cfg.ShutdownTimeout, err = parseDurationEnv("SHUTDOWN_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.OTel.ServiceName, err = requiredEnv("OTEL_SERVICE_NAME")
	if err != nil {
		return Config{}, err
	}

	cfg.OTel.Endpoint, err = requiredEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if err != nil {
		return Config{}, err
	}

	cfg.Database.MaxConns, err = parseInt32Env("DATABASE_MAX_CONNS")
	if err != nil {
		return Config{}, err
	}

	cfg.Database.MinConns, err = parseInt32Env("DATABASE_MIN_CONNS")
	if err != nil {
		return Config{}, err
	}

	cfg.Database.MaxConnLifetime, err = parseDurationEnv("DATABASE_MAX_CONN_LIFETIME")
	if err != nil {
		return Config{}, err
	}

	cfg.Database.ConnectTimeout, err = parseDurationEnv("DATABASE_CONNECT_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.Database.QueryTimeout, err = parseDurationEnv("DATABASE_QUERY_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.HTTP.ReadTimeout, err = parseDurationEnv("HTTP_READ_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.HTTP.ReadHeaderTimeout, err = parseDurationEnv("HTTP_READ_HEADER_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.HTTP.WriteTimeout, err = parseDurationEnv("HTTP_WRITE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.HTTP.IdleTimeout, err = parseDurationEnv("HTTP_IDLE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func requiredEnv(name string) (string, error) {
	value, exists := os.LookupEnv(name)
	if !exists || value == "" {
		return "", fmt.Errorf("требуемая переменная окружения %s не установлена", name)
	}

	return value, nil
}

func parseDurationEnv(name string) (time.Duration, error) {
	value, err := requiredEnv(name)
	if err != nil {
		return 0, err
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("неверная продолжительность в %s: %w", name, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("продолжительность в %s должна быть больше нуля", name)
	}

	return duration, nil
}

func parseInt32Env(name string) (int32, error) {
	value, err := requiredEnv(name)
	if err != nil {
		return 0, err
	}

	number, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("неверное целое число в %s: %w", name, err)
	}

	return int32(number), nil
}
