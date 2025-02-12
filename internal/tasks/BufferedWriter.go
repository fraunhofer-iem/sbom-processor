package tasks

import (
	"log/slog"
	"sync"
)

type BufferedWriter[T any] struct {
	DoWrite func(t []*T) error
	Buffer  int
}

type BufferedWriterConfig struct {
	Buffer *int
}

func NewBufferedWriter[T any](
	doWrite func(t []*T) error,
	config BufferedWriterConfig,
) *BufferedWriter[T] {

	buffer := 1
	if config.Buffer != nil && *config.Buffer > 1 {
		buffer = *config.Buffer
	}

	return &BufferedWriter[T]{
		Buffer:  buffer,
		DoWrite: doWrite,
	}
}

func (w *BufferedWriter[T]) Run(in <-chan *T, errc chan error, wg *sync.WaitGroup) {
	defer wg.Done()

	buffer := []*T{}

	for i := range in {
		if len(buffer) > w.Buffer {
			err := w.DoWrite(buffer)
			if err != nil {
				errc <- err
			}
			slog.Default().Info("Wrote buffer to target")
			buffer = []*T{}
		}

		buffer = append(buffer, i)
	}

	if len(buffer) > 0 {
		err := w.DoWrite(buffer)
		if err != nil {
			errc <- err
		}
		slog.Default().Info("Wrote remaining buffer to target")
	}
}
