package httptransport

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domain "github.com/denis-sigma/laboratory-work/internal/domain/trip"
	api "github.com/denis-sigma/laboratory-work/internal/generated"
	usecase "github.com/denis-sigma/laboratory-work/internal/usecase/trip"
)

const problemContentType = "application/problem+json"

func TestValidateCreateTrip(t *testing.T) {
	t.Run("корректный запрос", func(t *testing.T) {
		body := api.TripData{
			UserId:   uuid.MustParse("5cb72c04-7650-45c9-a79b-bcdba0631e0c"),
			DriverId: uuid.MustParse("8860b315-ec86-42eb-a17c-7c163d721ff5"),
			StartPoint: api.Coordinates{
				Latitude:  59.9398,
				Longitude: 30.3146,
			},
			EndPoint: api.Coordinates{
				Latitude:  59.9290,
				Longitude: 30.3626,
			},
			Price: 1450,
		}

		if err := validateCreateTrip(body); err != nil {
			t.Fatalf("ожидалась успешная валидация, получена ошибка: %v", err)
		}
	})

	t.Run("отрицательная цена", func(t *testing.T) {
		body := api.TripData{
			UserId:   uuid.MustParse("5cb72c04-7650-45c9-a79b-bcdba0631e0c"),
			DriverId: uuid.MustParse("8860b315-ec86-42eb-a17c-7c163d721ff5"),
			StartPoint: api.Coordinates{
				Latitude:  59.9398,
				Longitude: 30.3146,
			},
			EndPoint: api.Coordinates{
				Latitude:  59.9290,
				Longitude: 30.3626,
			},
			Price: -1,
		}

		if err := validateCreateTrip(body); err == nil {
			t.Fatal("ожидалась ошибка для отрицательной цены")
		}
	})

	t.Run("пустой user_id", func(t *testing.T) {
		body := api.TripData{
			DriverId: uuid.MustParse("8860b315-ec86-42eb-a17c-7c163d721ff5"),
			StartPoint: api.Coordinates{
				Latitude:  59.9398,
				Longitude: 30.3146,
			},
			EndPoint: api.Coordinates{
				Latitude:  59.9290,
				Longitude: 30.3626,
			},
			Price: 1450,
		}

		if err := validateCreateTrip(body); err == nil {
			t.Fatal("ожидалась ошибка для пустого user_id")
		}
	})

	t.Run("пустой driver_id", func(t *testing.T) {
		body := api.TripData{
			UserId: uuid.MustParse("5cb72c04-7650-45c9-a79b-bcdba0631e0c"),
			StartPoint: api.Coordinates{
				Latitude:  59.9398,
				Longitude: 30.3146,
			},
			EndPoint: api.Coordinates{
				Latitude:  59.9290,
				Longitude: 30.3626,
			},
			Price: 1450,
		}

		if err := validateCreateTrip(body); err == nil {
			t.Fatal("ожидалась ошибка для пустого driver_id")
		}
	})

	t.Run("недопустимая широта", func(t *testing.T) {
		body := api.TripData{
			UserId:   uuid.MustParse("5cb72c04-7650-45c9-a79b-bcdba0631e0c"),
			DriverId: uuid.MustParse("8860b315-ec86-42eb-a17c-7c163d721ff5"),
			StartPoint: api.Coordinates{
				Latitude:  91,
				Longitude: 30.3146,
			},
			EndPoint: api.Coordinates{
				Latitude:  59.9290,
				Longitude: 30.3626,
			},
			Price: 1450,
		}

		if err := validateCreateTrip(body); err == nil {
			t.Fatal("ожидалась ошибка для недопустимой широты")
		}
	})

	t.Run("недопустимая долгота", func(t *testing.T) {
		body := api.TripData{
			UserId:   uuid.MustParse("5cb72c04-7650-45c9-a79b-bcdba0631e0c"),
			DriverId: uuid.MustParse("8860b315-ec86-42eb-a17c-7c163d721ff5"),
			StartPoint: api.Coordinates{
				Latitude:  59.9398,
				Longitude: 181,
			},
			EndPoint: api.Coordinates{
				Latitude:  59.9290,
				Longitude: 30.3626,
			},
			Price: 1450,
		}

		if err := validateCreateTrip(body); err == nil {
			t.Fatal("ожидалась ошибка для недопустимой долготы")
		}
	})
}

func TestHandlerHealth(t *testing.T) {
	handler := &Handler{}

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	handler.Health(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
	}

	if body := recorder.Body.String(); body != "{\"status\":\"ok\"}\n" {
		t.Errorf("body = %q, want %q", body, "{\"status\":\"ok\"}\n")
	}
}

func TestHandlerCreateTrip(t *testing.T) {
	ctrl := gomock.NewController(t)

	service := NewMockTripService(ctrl)
	handler := NewHandler(nil, 0, service)

	requestBody := `{
		"user_id": "5cb72c04-7650-45c9-a79b-bcdba0631e0c",
		"driver_id": "8860b315-ec86-42eb-a17c-7c163d721ff5",
		"start_point": {
			"latitude": 59.9398,
			"longitude": 30.3146
		},
		"end_point": {
			"latitude": 59.9290,
			"longitude": 30.3626
		},
		"price": 1450
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/trips",
		strings.NewReader(requestBody),
	)
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	_ = domain.Trip{}
	_ = usecase.CreateTripInput{}

	tripID := uuid.New()

	service.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(domain.Trip{
			ID:     tripID,
			Status: domain.TripStatusActive,
		}, nil)

	handler.CreateTrip(recorder, request, api.CreateTripParams{})

	if recorder.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}

	wantLocation := "/api/v1/trips/" + tripID.String()
	if location := recorder.Header().Get("Location"); location != wantLocation {
		t.Errorf("Location = %q, want %q", location, wantLocation)
	}
}

func TestHandlerCreateTripInvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)

	service := NewMockTripService(ctrl)
	handler := NewHandler(nil, 0, service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/trips",
		strings.NewReader(`{"user_id":`),
	)
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	handler.CreateTrip(recorder, request, api.CreateTripParams{})

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != problemContentType {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/problem+json")
	}
}

func TestHandlerCreateTripRejectsIncompleteBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	service := NewMockTripService(ctrl)
	handler := NewHandler(nil, 0, service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/trips",
		strings.NewReader(`{
			"user_id": "5cb72c04-7650-45c9-a79b-bcdba0631e0c",
			"driver_id": "8860b315-ec86-42eb-a17c-7c163d721ff5"
		}`),
	)
	recorder := httptest.NewRecorder()

	handler.CreateTrip(recorder, request, api.CreateTripParams{})

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestHandlerCreateTripRejectsTrailingJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	service := NewMockTripService(ctrl)
	handler := NewHandler(nil, 0, service)

	requestBody := `{
		"user_id": "5cb72c04-7650-45c9-a79b-bcdba0631e0c",
		"driver_id": "8860b315-ec86-42eb-a17c-7c163d721ff5",
		"start_point": {"latitude": 59.9398, "longitude": 30.3146},
		"end_point": {"latitude": 59.9290, "longitude": 30.3626},
		"price": 1450
	} {}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/trips",
		strings.NewReader(requestBody),
	)
	recorder := httptest.NewRecorder()

	handler.CreateTrip(recorder, request, api.CreateTripParams{})

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestHandlerGetTripNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)

	service := NewMockTripService(ctrl)
	handler := NewHandler(nil, 0, service)

	tripID := uuid.New()

	service.EXPECT().
		GetByID(gomock.Any(), tripID).
		Return(domain.Trip{}, domain.ErrTripNotFound)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/trips/"+tripID.String(),
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.GetTrip(recorder, request, tripID)

	if recorder.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != problemContentType {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/problem+json")
	}
}

func TestHandlerFinishTripCompleted(t *testing.T) {
	ctrl := gomock.NewController(t)

	service := NewMockTripService(ctrl)
	handler := NewHandler(nil, 0, service)

	tripID := uuid.New()

	service.EXPECT().
		Finish(gomock.Any(), tripID).
		Return(domain.Trip{}, domain.ErrTripCompleted)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/trips/"+tripID.String()+"/finish",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.FinishTrip(recorder, request, tripID)

	if recorder.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusConflict)
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != problemContentType {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/problem+json")
	}
}
