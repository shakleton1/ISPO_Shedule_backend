package push

import (
	"context"
)

type Notifier interface {
	Send(ctx context.Context, token string, data map[string]string) error
}

type noopNotifier struct{}

func (noopNotifier) Send(ctx context.Context, token string, data map[string]string) error {
	return nil
}
