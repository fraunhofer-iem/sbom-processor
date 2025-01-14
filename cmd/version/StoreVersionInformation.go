package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"sbom-processor/internal/json"
	"sbom-processor/internal/sbom"
	"sbom-processor/internal/semver"
	"sbom-processor/internal/validator"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/sync/semaphore"
)

var in = flag.String("in", "", "Path to SBOM")

func main() {

	// INPUT VALIDATION
	uri := os.Getenv("MONGO_URI")
	usr := os.Getenv("MONGO_USERNAME")
	pwd := os.Getenv("MONGO_PWD")

	if usr == "" || pwd == "" || uri == "" {
		log.Fatalf("username or password not found. Make sure MONGO_USERNAME, MONGO_PWD, and MONGO_URI are set\n")
	}

	// get input path and check for correctness
	flag.Parse()
	_, err := validator.ValidateInPath(in)
	if err != nil {
		panic(err)
	}

	// DB CONNECTION
	client, err := mongo.Connect(options.Client().
		ApplyURI(uri).
		SetAuth(options.Credential{
			Username: usr,
			Password: pwd,
		}))
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

	coll := client.Database("sbom_metadata").Collection("versions")

	// CREATE INDEX
	indexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "component_id", Value: 1}},
	}
	name, err := coll.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		panic(err)
	}
	fmt.Println("Name of Index Created: " + name)

	// GET JSON FILE PATHS
	paths, err := json.CollectJsonFiles(*in)
	if err != nil {
		panic(err)
	}

	// ASYNC ITERATION OF FILES AND STORE TO DB
	var (
		maxWorkers = runtime.GOMAXPROCS(0) // 0 = default = maxNumProc
		sem        = semaphore.NewWeighted(int64(maxWorkers))
		ctx        = context.TODO()
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

			errCounter := 0

			// GET ALL VERSIONS FOR EACH COMPONENT AND INSERT TO DB
			for _, c := range s.Components {
				ver, err := c.GetVersions()
				if err != nil {
					errCounter += 1
					fmt.Printf("query for %+v failed with %s\n", c, err)
					continue
				}

				compVer := make([]semver.ComponentVersion, len(ver))
				for i, v := range ver {
					compVer[i] = semver.ComponentVersion{
						Version:     v,
						ReleaseDate: "",
					}
				}
				compVers := semver.ComponentVersions{
					ComponentId: c.Id,
					Versions:    compVer,
				}
				_, err = coll.InsertOne(ctx, compVers)
				if err != nil {
					fmt.Printf("db store failed with %s\n", err)
				}
			}

			fmt.Printf("%d of %d querries failed \n", errCounter, len(s.Components))

			fmt.Printf("Finished SBOM processing for path %s\n", p)
		}()
	}

	// Acquire all of the tokens to wait for any remaining workers to finish.
	if err := sem.Acquire(ctx, int64(maxWorkers)); err != nil {
		log.Printf("Failed to acquire semaphore: %v", err)
	}

	fmt.Println("Finished main")
}
