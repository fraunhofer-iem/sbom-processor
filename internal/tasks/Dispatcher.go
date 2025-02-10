package tasks

import (
	"fmt"
	"iter"
	"log/slog"
	"math"
	"os"
	"runtime"
	"sync"
)

type Dispatcher[T, E any] struct {
	NoWorker        int
	Worker          Worker[T, E]
	Producer        iter.Seq[T]
	ResultCollector BufferedWriter[E]
	Logger          slog.Logger
	channelBuffer   int
}

type DispatcherConfig struct {
	NoWorker *int         // optional, defaults to runtime.NumCPU()
	Logger   *slog.Logger // optional, defaults to error only log
}

func NewDispatcher[T, E any](Worker Worker[T, E],
	Producer iter.Seq[T],
	ResultCollector BufferedWriter[E],
	config DispatcherConfig) *Dispatcher[T, E] {

	d := Dispatcher[T, E]{
		Worker:          Worker,
		Producer:        Producer,
		ResultCollector: ResultCollector,
	}

	// set number of workers
	var noWorker int
	if config.NoWorker != nil {
		noWorker = *config.NoWorker
	} else {
		noWorker = runtime.NumCPU()
	}
	d.NoWorker = noWorker

	// calculate channel buffer size
	channelBuffer := d.ResultCollector.Buffer
	if channelBuffer == math.MaxInt {
		channelBuffer = 100 // limit channel buffer size
	}
	d.channelBuffer = channelBuffer / noWorker

	// create logger
	var logger *slog.Logger
	if config.Logger != nil {
		logger = config.Logger
	} else {
		slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelError,
		}))
	}
	d.Logger = *logger

	return &d
}

func (d *Dispatcher[T, E]) Dispatch() {

	in := make(chan *T)
	out := make(chan *E, d.channelBuffer)
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
