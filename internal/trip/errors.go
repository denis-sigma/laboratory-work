package trip

import "errors"

var (
	ErrTripNotFound  = errors.New("поездка не найдена")
	ErrDriverBusy    = errors.New("водитель занят")
	ErrTripCompleted = errors.New("поездка уже завершена")
)