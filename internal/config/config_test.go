package config

import (
	"os"
	"testing"
	"time"
)

func TestParseDurationEnv(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		want      time.Duration
		wantError bool
	}{
		{
			name:  "положительная длительность",
			value: "2s",
			want:  2 * time.Second,
		},
		{
			name:      "нулевая длительность",
			value:     "0s",
			wantError: true,
		},
		{
			name:      "отрицательная длительность",
			value:     "-1s",
			wantError: true,
		},
		{
			name:      "неверный формат",
			value:     "two seconds",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_DURATION", tt.value)

			got, err := parseDurationEnv("TEST_DURATION")
			if tt.wantError {
				if err == nil {
					t.Fatal("ожидалась ошибка")
				}
				return
			}

			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if got != tt.want {
				t.Errorf("получено %s, ожидалось %s", got, tt.want)
			}
		})
	}
}

func TestParseDurationEnvMissing(t *testing.T) {
	os.Unsetenv("TEST_DURATION")

	if _, err := parseDurationEnv("TEST_DURATION"); err == nil {
		t.Fatal("ожидалась ошибка для отсутствующей переменной")
	}
}
