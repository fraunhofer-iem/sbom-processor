package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"runtime"

	"sbom-processor/internal/json"
	"sbom-processor/internal/sbom"
	"sbom-processor/internal/semver"
	"sbom-processor/internal/validator"

	"golang.org/x/sync/semaphore"
)

var in = flag.String("in", "", "Path to SBOM")
var out = flag.String("out", "", "File to write the SBOM to")

func main() {

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

			s, err := sbom.ReadCyclonedx(p)
			if err != nil {
				fmt.Printf("read SBOM failed with %s for %s\n", err.Error(), p)
				return
			}

			vd := make([]*semver.VersionDistance, len(s.Components))

			for i, c := range s.Components {
				ver, err := c.GetVersions()
				if err != nil {
					fmt.Printf("query for %+v failed with %s", c, err)
					continue
				}
				v, err := semver.GetVersionDistance(c.Version, ver)
				if err != nil {
					continue

				}
				vd[i] = v
			}

			var avg int64 = 0
			for _, v := range vd {
				avg += v.MissedReleases
			}
			avg = avg / int64(len(vd))
			fmt.Printf("Avg missed releases for %s is %d", p, avg)

			fmt.Printf("Finished SBOM processing for path %s\n", p)
		}()
	}

	// Acquire all of the tokens to wait for any remaining workers to finish.
	if err := sem.Acquire(ctx, int64(maxWorkers)); err != nil {
		log.Printf("Failed to acquire semaphore: %v", err)
	}

	fmt.Println("Finished main")
}
