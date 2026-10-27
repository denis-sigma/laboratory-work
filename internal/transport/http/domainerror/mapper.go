package domainerror

import (
	"errors"

	"github.com/denis-sigma/laboratory-work/internal/domain/trip"
)

type Details struct {
	Status int32
	Code   string
	Title  string
	Detail string
}

func Map(err error) Details {
	if errors.Is(err, trip.ErrTripNotFound) {
		return Details{
			404,
			"trip_not_found",
			"Поездка не найдена",
			"Поездка с указанным идентификатором не существует",
		}
	}

	if errors.Is(err, trip.ErrDriverBusy) {
		return Details{
			409,
			"driver_busy",
			"Водитель занят",
			"У водителя уже есть активная поездка",
		}
	}

	if errors.Is(err, trip.ErrTripCompleted) {
		return Details{
			409,
			"trip_completed",
			"Поездка уже завершена",
			"Нельзя повторно завершить поездку",
		}
	}

	return Details{
		500,
		"internal_error",
		"Внутренняя ошибка",
		"Внутренняя ошибка сервера",
	}
}
