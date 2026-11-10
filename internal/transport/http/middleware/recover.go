package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"

	api "github.com/denis-sigma/laboratory-work/internal/generated"
)

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error(
						"паника в HTTP-обработчике",
						"error", recovered,
						"stack", string(debug.Stack()),
					)

					instance := r.URL.Path
					detail := "Внутренняя ошибка сервера"
					w.Header().Set("Content-Type", "application/problem+json")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(api.Problem{
						Type:     "https://tripgo.example/problems/internal_error",
						Title:    "Внутренняя ошибка",
						Status:   http.StatusInternalServerError,
						Detail:   &detail,
						Instance: &instance,
						Code:     "internal_error",
					})
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
