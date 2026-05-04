package application

import "context"

// Notifier sends a formatted message to a messenger service.
// Implementations live in internal/infrastructure/notification/.
type Notifier interface {
	Send(ctx context.Context, message string) error
}
