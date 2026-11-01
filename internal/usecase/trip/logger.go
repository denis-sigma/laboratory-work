package trip

import "context"

type Logger interface {
	InfoContext(ctx context.Context, msg string, args ...any)
}
