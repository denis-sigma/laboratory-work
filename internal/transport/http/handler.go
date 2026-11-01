package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	domain "github.com/denis-sigma/laboratory-work/internal/domain/trip"
	"github.com/denis-sigma/laboratory-work/internal/generated"
	"github.com/denis-sigma/laboratory-work/internal/transport/http/domainerror"
	usecase "github.com/denis-sigma/laboratory-work/internal/usecase/trip"
	"github.com/google/uuid"
)

type Handler struct {
	api.Unimplemented
	readiness    Readiness
	queryTimeout time.Duration
	service      TripService
}

func NewHandler(
	readiness Readiness,
	queryTimeout time.Duration,
	service TripService,
) *Handler {
	return &Handler{
		readiness:    readiness,
		queryTimeout: queryTimeout,
		service:      service,
	}
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(api.HealthResponse{
		Status: api.Ok,
	})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	status := api.Ok
	httpStatus := http.StatusOK

	if err := h.readiness.Ping(ctx); err != nil {
		status = api.Unavailable
		httpStatus = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	_ = json.NewEncoder(w).Encode(api.HealthResponse{
		Status: status,
	})
}

func (h *Handler) CreateTrip(
	w http.ResponseWriter,
	r *http.Request,
	_ api.CreateTripParams,
) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Некорректный запрос", "Не удалось прочитать тело запроса")
		return
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		writeProblem(
			w,
			r,
			http.StatusBadRequest,
			"invalid_request",
			"Некорректный запрос",
			"Тело запроса имеет неверный формат или содержит неизвестное поле",
		)
		return
	}

	if err := validateRequiredCreateTripFields(raw); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Некорректный запрос", err.Error())
		return
	}

	var body api.CreateTripJSONRequestBody
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Некорректный запрос", "Тело запроса имеет неверный формат или содержит неизвестное поле")
		return
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Некорректный запрос", "После объекта запроса обнаружены лишние данные")
		return
	}

	if err := validateCreateTrip(body); err != nil {
		writeProblem(
			w,
			r,
			http.StatusBadRequest,
			"invalid_request",
			"Некорректный запрос",
			err.Error(),
		)
		return
	}

	result, err := h.service.Create(
		r.Context(),
		usecase.CreateTripInput{
			UserID:   body.UserId,
			DriverID: body.DriverId,
			StartPoint: domain.Coordinates{
				Latitude:  body.StartPoint.Latitude,
				Longitude: body.StartPoint.Longitude,
			},
			EndPoint: domain.Coordinates{
				Latitude:  body.EndPoint.Latitude,
				Longitude: body.EndPoint.Longitude,
			},
			Price: body.Price,
		},
	)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	response := api.Trip{
		Id:       result.ID,
		UserId:   result.UserID,
		DriverId: result.DriverID,
		StartPoint: api.Coordinates{
			Latitude:  result.StartPoint.Latitude,
			Longitude: result.StartPoint.Longitude,
		},
		EndPoint: api.Coordinates{
			Latitude:  result.EndPoint.Latitude,
			Longitude: result.EndPoint.Longitude,
		},
		Price:     result.Price,
		Status:    api.TripStatus(result.Status),
		StartedAt: result.StartedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/api/v1/trips/"+result.ID.String())
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(response)
}

func writeProblem(
	w http.ResponseWriter,
	r *http.Request,
	status int32,
	code string,
	title string,
	detail string,
) {
	instance := r.URL.Path

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(int(status))

	_ = json.NewEncoder(w).Encode(api.Problem{
		Type:     "https://tripgo.example/problems/" + code,
		Title:    title,
		Status:   status,
		Detail:   &detail,
		Instance: &instance,
		Code:     code,
	})
}

func writeDomainError(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	details := domainerror.Map(err)

	writeProblem(
		w,
		r,
		details.Status,
		details.Code,
		details.Title,
		details.Detail,
	)
}

func (h *Handler) GetTrip(
	w http.ResponseWriter,
	r *http.Request,
	id api.TripId,
) {
	result, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	response := api.Trip{
		Id:       result.ID,
		UserId:   result.UserID,
		DriverId: result.DriverID,
		StartPoint: api.Coordinates{
			Latitude:  result.StartPoint.Latitude,
			Longitude: result.StartPoint.Longitude,
		},
		EndPoint: api.Coordinates{
			Latitude:  result.EndPoint.Latitude,
			Longitude: result.EndPoint.Longitude,
		},
		Price:      result.Price,
		Status:     api.TripStatus(result.Status),
		StartedAt:  result.StartedAt,
		FinishedAt: result.FinishedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handler) FinishTrip(
	w http.ResponseWriter,
	r *http.Request,
	id api.TripId,
) {
	result, err := h.service.Finish(r.Context(), id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	response := api.Trip{
		Id:       result.ID,
		UserId:   result.UserID,
		DriverId: result.DriverID,
		StartPoint: api.Coordinates{
			Latitude:  result.StartPoint.Latitude,
			Longitude: result.StartPoint.Longitude,
		},
		EndPoint: api.Coordinates{
			Latitude:  result.EndPoint.Latitude,
			Longitude: result.EndPoint.Longitude,
		},
		Price:      result.Price,
		Status:     api.TripStatus(result.Status),
		StartedAt:  result.StartedAt,
		FinishedAt: result.FinishedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(response)
}

func validateCreateTrip(body api.TripData) error {
	if body.UserId == uuid.Nil {
		return fmt.Errorf("user_id обязателен")
	}

	if body.DriverId == uuid.Nil {
		return fmt.Errorf("driver_id обязателен")
	}

	if body.StartPoint.Latitude < -90 ||
		body.StartPoint.Latitude > 90 {
		return fmt.Errorf("широта начальной точки должна быть от -90 до 90")
	}

	if body.StartPoint.Longitude < -180 ||
		body.StartPoint.Longitude > 180 {
		return fmt.Errorf("долгота начальной точки должна быть от -180 до 180")
	}

	if body.EndPoint.Latitude < -90 ||
		body.EndPoint.Latitude > 90 {
		return fmt.Errorf("широта конечной точки должна быть от -90 до 90")
	}

	if body.EndPoint.Longitude < -180 ||
		body.EndPoint.Longitude > 180 {
		return fmt.Errorf("долгота конечной точки должна быть от -180 до 180")
	}

	if body.Price < 0 {
		return fmt.Errorf("цена не может быть отрицательной")
	}

	return nil
}

func validateRequiredCreateTripFields(raw map[string]json.RawMessage) error {
	for _, field := range []string{"user_id", "driver_id", "start_point", "end_point", "price"} {
		value, ok := raw[field]
		if !ok || string(value) == "null" {
			return fmt.Errorf("поле %s обязательно и не может быть null", field)
		}
	}

	for _, field := range []string{"start_point", "end_point"} {
		var point map[string]json.RawMessage
		if err := json.Unmarshal(raw[field], &point); err != nil {
			return fmt.Errorf("поле %s должно быть объектом", field)
		}

		for _, coordinate := range []string{"latitude", "longitude"} {
			value, ok := point[coordinate]
			if !ok || string(value) == "null" {
				return fmt.Errorf("поле %s.%s обязательно и не может быть null", field, coordinate)
			}
		}
	}

	return nil
}

func OpenAPIErrorHandler(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	writeProblem(
		w,
		r,
		http.StatusBadRequest,
		"invalid_request",
		"Некорректный запрос",
		err.Error(),
	)
}
