package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"sbom-processor/internal/deps"
	"sbom-processor/internal/logging"
	"sbom-processor/internal/tasks"
	"sbom-processor/internal/validator"
	"slices"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var in = flag.String("in", "", "path to input file containing mvn metadata")
var logLevel = flag.Int("logLevel", 0, "Can be 0 for INFO, -4 for DEBUG, 4 for WARN, or 8 for ERROR. Defaults to INFO.")
var dbName = flag.String("db", "sbom_metadata", "database name to connect to")
var collectionName = flag.String("collection", "deps_metadata", "collection name for SBOMs")

type MvnIdentifier struct {
	// e.g. "u":"org.apache.pdfbox|pdfbox-io|3.0.0-beta1|NA|jar"
	Idenfitier string `bson:"u" json:"u"`
}

func main() {

	start := time.Now()

	flag.Parse()

	validator.ValidateInPath(in)
	logger := logging.SetUpLogging(*logLevel)

	// INPUT VALIDATION
	uri := os.Getenv("MONGO_URI")
	usr := os.Getenv("MONGO_USERNAME")
	pwd := os.Getenv("MONGO_PWD")

	if usr == "" || pwd == "" || uri == "" {
		log.Fatalf("uri, username or password not found. Make sure MONGO_USERNAME, MONGO_PWD, and MONGO_URI are set\n")
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
		if err := client.Disconnect(context.Background()); err != nil {
			panic(err)
		}
	}()

	coll := client.Database(*dbName).Collection(*collectionName)

	logger.Info("Import unique components called", "db", *dbName, "collection", *collectionName)

	file, err := os.Open(*in)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	decoder := json.NewDecoder(file)
	var mvn []MvnIdentifier

	if err := decoder.Decode(&mvn); err != nil {
		panic(err)
	}

	worker := tasks.Worker[MvnIdentifier, deps.Deps]{
		Do: func(t *MvnIdentifier) (*deps.Deps, error) {
			// first tranform u to CacheRequest
			// the format is system dependent
			split := strings.Split(t.Idenfitier, "|")
			if len(split) < 2 {
				return nil, fmt.Errorf("invalid identifier. identfier is to short: %s", t.Idenfitier)
			}

			urlIdent := split[0] + ":" + split[1]
			c := deps.CacheRequest{
				Name:   urlIdent,
				System: "MAVEN",
			}

			// then query api
			logger.Debug("Querying deps.dev", "name", c.Name, "system", c.System)
			dep, err := deps.DepsWorkerDo(c)

			logger.Debug("Query result", "dep", dep, "err", err)

			return dep, err
		},
	}

	writer := tasks.BufferedWriter[deps.Deps]{
		Buffer: math.MaxInt,
		DoWrite: func(t []*deps.Deps) error {
			_, err := coll.InsertMany(context.TODO(), t)
			return err
		},
	}

	throttle := time.Second / 10
	dispatcher := tasks.NewDispatcher(worker, slices.Values(mvn), writer, tasks.DispatcherConfig{RateLimit: &throttle})

	dispatcher.Dispatch()

	elapsed := time.Since(start)
	logger.Info("Execution finished", "runtime", elapsed)
}
