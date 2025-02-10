package tasks

import (
	"fmt"
	"iter"
	"math"
	"sync"
)

type BufferedWriter[T any] struct {
	DoWrite func(t []*T) error
	Buffer  int
}

type Worker[T, E any] struct {
	Do func(*T) (*E, error)
}

type Dispatcher[T, E any] struct {
	NoWorker        int
	Worker          Worker[T, E]
	Producer        iter.Seq[T]
	ResultCollector BufferedWriter[E]
}

// helper function for workers that should just pass through their
// input to the output. This is useful to use the workerpool to read
// data from one source and store it to another source (file to database)
// without further modification
func DoNothing[T any](t *T) (*T, error) {
	return t, nil
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
			buffer = []*T{}
		}

		buffer = append(buffer, i)
	}

	if len(buffer) > 0 {
		err := w.DoWrite(buffer)
		if err != nil {
			errc <- err
		}
	}
}

func (w *Worker[T, E]) Run(in <-chan *T, out chan *E, errc chan error, wg *sync.WaitGroup) {
	defer wg.Done()
	for i := range in {
		res, err := w.Do(i)
		if err != nil {
			errc <- err
			continue
		}
		out <- res
	}
}

func (d *Dispatcher[T, E]) Dispatch() {

	channelBuffer := d.ResultCollector.Buffer
	if channelBuffer == math.MaxInt {
		channelBuffer = 100
	}

	in := make(chan *T)
	out := make(chan *E, channelBuffer/d.NoWorker)
	errc := make(chan error)

	var processWg sync.WaitGroup
	var writeWg sync.WaitGroup

	// setup workers
	for i := 0; i < d.NoWorker; i++ {
		processWg.Add(1)
		go d.Worker.Run(in, out, errc, &processWg)
	}

	// setup error logging
	go func() {
		for err := range errc {
			fmt.Printf("error occured %s\n", err)
		}
	}()

	// setup writer
	writeWg.Add(1)
	go d.ResultCollector.Run(out, errc, &writeWg)

	// sent paths to collector worker
	for e := range d.Producer {
		in <- &e
	}
	close(in)

	processWg.Wait()
	close(out)
	writeWg.Wait()

	close(errc)
}
