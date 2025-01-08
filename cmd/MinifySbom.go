package main

import (
	"flag"
	"fmt"
	"log"
	"sync"

	"sbom-processor/internal/json"
	"sbom-processor/internal/sbom"
	"sbom-processor/internal/validator"
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

	var wg sync.WaitGroup
	for _, p := range paths {
		wg.Add(1)
		go func() {

			defer wg.Done()

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

			err = t.Store(*out)
			if err != nil {
				fmt.Printf("Store failed with %s for %s\n", err.Error(), p)
				return
			}

			fmt.Printf("Finished SBOM minify for path %s\n", p)
		}()
	}

	wg.Wait()
	fmt.Println("Finished main")
}
