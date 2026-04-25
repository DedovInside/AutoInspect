package notify

import "context"

type Notifier interface {
	NotifyJobEvent(ctx context.Context, event *JobEvent) error
}

type Handler func(ctx context.Context, event JobEvent)

type Subscriber interface {
	Subscribe(ctx context.Context, handler Handler) error
}
