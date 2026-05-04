// Package console implements application.Notifier by writing to an io.Writer
// (defaults to os.Stdout). Useful as a fallback notifier or in tests.
package console

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/ducminhgd/weather-bot/internal/application"
)

var _ application.Notifier = (*Notifier)(nil)

// Notifier writes messages to a writer.
type Notifier struct {
	w io.Writer
}

// New returns a Notifier that writes to os.Stdout.
func New() *Notifier {
	return &Notifier{w: os.Stdout}
}

// NewWithWriter returns a Notifier that writes to w.
func NewWithWriter(w io.Writer) *Notifier {
	return &Notifier{w: w}
}

// Send writes message followed by a newline to the writer.
func (n *Notifier) Send(_ context.Context, message string) error {
	_, err := fmt.Fprintln(n.w, message)
	return err
}
