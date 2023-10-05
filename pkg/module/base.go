package module

import (
	"context"
)

type Module interface {
	Start(ctx context.Context) error
	Cleanup()
}
