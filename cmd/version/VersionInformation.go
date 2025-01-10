package main

import (
	"flag"
	"fmt"
	"log"
	"sync"

	"sbom-processor/internal/json"
	"sbom-processor/internal/sbom"
	"sbom-processor/internal/semver"
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

			fmt.Printf("started SBOM process for path %s\n", p)

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

			// fmt.Printf("Found %d artifacts.\n", len(s.Artifacts))

			// t, err := s.Transform()
			// if err != nil {
			// 	fmt.Printf("Transform failed with %s for %s\n", err.Error(), p)
			// 	return
			// }

			// err = t.Store(*out)
			// if err != nil {
			// 	fmt.Printf("Store failed with %s for %s\n", err.Error(), p)
			// 	return
			// }

			fmt.Printf("Finished SBOM processing for path %s\n", p)
		}()
	}

	wg.Wait()
	fmt.Println("Finished main")
}
