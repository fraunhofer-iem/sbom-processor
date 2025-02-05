package tasks

import (
	"fmt"
	"sbom-processor/internal/json"
	"sync"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Store interface {
	json.JsonFileExporter | *mongo.Collection
}

type BufferedWriter[S Store, T any] struct {
	DoWrite func(s S, t []*T) error
	Store   S
	Buffer  int
}

type Worker[T, E any] struct {
	Do func(*T) (*E, error)
}

type Dispatcher[S Store, T, E any] struct {
	NoWorker        int
	Worker          Worker[T, E]
	ResultCollector BufferedWriter[S, E]
}

func (w *BufferedWriter[S, T]) Run(in <-chan *T, errc chan error, wg *sync.WaitGroup) {
	defer wg.Done()

	buffer := []*T{}

	for i := range in {
		if len(buffer) > w.Buffer {
			err := w.DoWrite(w.Store, buffer)
			if err != nil {
				errc <- err
			}
			buffer = []*T{}
		}

		buffer = append(buffer, i)
	}

	if len(buffer) > 0 {
		err := w.DoWrite(w.Store, buffer)
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

func (d *Dispatcher[S, T, E]) Dispatch(inputs []T) {

	in := make(chan *T)
	out := make(chan *E, d.ResultCollector.Buffer/d.NoWorker)
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
	for _, p := range inputs {
		in <- &p
	}
	close(in)

	processWg.Wait()
	close(out)
	writeWg.Wait()

	close(errc)
}
