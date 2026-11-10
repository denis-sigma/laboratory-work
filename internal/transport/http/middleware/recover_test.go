package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecover(t *testing.T) {
	logger := slog.New(
		slog.NewTextHandler(io.Discard, nil),
	)

	handler := Recover(logger)(
		http.HandlerFunc(func(
			http.ResponseWriter,
			*http.Request,
		) {
			panic("тестовая паника")
		}),
	)

	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"status = %d, want %d",
			recorder.Code,
			http.StatusInternalServerError,
		)
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/problem+json" {
		t.Fatalf("Content-Type = %q, want %q", contentType, "application/problem+json")
	}

	if body := recorder.Body.String(); !strings.Contains(body, `"code":"internal_error"`) {
		t.Fatalf("body = %q, want internal_error", body)
	}
}

func TestMiddlewareChainFinishesLoggingOnPanic(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	metrics, err := Metrics()
	if err != nil {
		t.Fatalf("создание metrics middleware: %v", err)
	}

	panicHandler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("test panic")
	})
	handler := Recover(logger)(Logging(logger)(metrics(panicHandler)))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/unknown/one", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(logs.String(), "status=500") {
		t.Errorf("logs = %q, want final status 500", logs.String())
	}
}
