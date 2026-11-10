package trip

import "context"

//go:generate mockgen -source=transaction.go -destination=transaction_mock_test.go -package=trip
type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}
