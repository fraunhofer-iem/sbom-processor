package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"time"

	"sbom-processor/internal/json"
	"sbom-processor/internal/sbom"
	"sbom-processor/internal/validator"

	"golang.org/x/sync/semaphore"
)

var in = flag.String("in", "", "Path to SBOM")
var out = flag.String("out", "", "File to write the SBOM to")

func main() {

	start := time.Now()
	// get input path and check for correctness
	flag.Parse()
	_, err := validator.ValidateInPath(in)
	if err != nil {
		log.Fatal(err)
	}
	err = validator.ValidateOutPath(out)
	if err != nil {
		log.Fatal(err)
	}

	paths, err := json.CollectJsonFiles(*in)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.TODO()

	var (
		maxWorkers = runtime.GOMAXPROCS(0) // 0 = default = maxNumProc
		sem        = semaphore.NewWeighted(int64(maxWorkers))
	)

	fmt.Printf("Starting up to %d workers\n", maxWorkers)

	for _, p := range paths {
		// When maxWorkers goroutines are in flight, Acquire blocks until one of the
		// workers finishes.
		if err := sem.Acquire(ctx, 1); err != nil {
			log.Printf("Failed to acquire semaphore: %v", err)
			break
		}

		go func() {

			defer sem.Release(1)
			fmt.Printf("Started SBOM minify for path %s\n", p)

			s, err := sbom.ReadSyft(p)
			if err != nil {
				fmt.Printf("Read syft failed with %s for %s\n", err.Error(), p)
				return
			}

			fmt.Printf("Found %d artifacts.\n", len(s.Artifacts))

			t, err := s.Transform()
			if err != nil {
				fmt.Printf("Transform failed with %s for %s\n", err.Error(), p)
				return
			}

			ts := time.Now().Format("20060102150405") // Format: YYYYMMDDHHMMSS
			outPath := filepath.Join(*out, s.Source.Id+"-"+ts+".json")

			err = t.Store(outPath)
			if err != nil {
				fmt.Printf("Store failed with %s for %s\n", err.Error(), p)
				return
			}

			fmt.Printf("Finished SBOM minify for path %s\n", p)
		}()
	}

	// Acquire all of the tokens to wait for any remaining workers to finish.
	if err := sem.Acquire(ctx, int64(maxWorkers)); err != nil {
		log.Printf("Failed to acquire semaphore: %v", err)
	}

	fmt.Println("Finished main")
	elapsed := time.Since(start)
	fmt.Printf("Execution time: %s\n", elapsed)
}
