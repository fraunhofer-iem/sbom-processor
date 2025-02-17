package main

import (
	"context"
	"flag"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sbom-processor/internal/json"
	"sbom-processor/internal/logging"
	"sbom-processor/internal/tasks"
	"sbom-processor/internal/validator"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var in = flag.String("in", "", "path to input file containing mvn metadata")
var logLevel = flag.Int("logLevel", 0, "Can be 0 for INFO, -4 for DEBUG, 4 for WARN, or 8 for ERROR. Defaults to INFO.")
var dbName = flag.String("db", "sbom_metadata", "database name to connect to")
var collectionName = flag.String("collection", "mvn_metadata", "collection name for SBOMs")

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

	sboms := client.Database(*dbName).Collection(*collectionName)

	logger.Info("Import unique components called", "db", *dbName, "collection", *collectionName, "componentType", *componentType)

	worker := tasks.Worker[MvnIdentifier]{
		Do: tasks.DoNothing[UniqueNames],
	}

	writer := tasks.BufferedWriter[UniqueNames]{
		Buffer: math.MaxInt,
		DoWrite: func(t []*UniqueNames) error {
			outPath := filepath.Join(*out, "uniqueComponentNames.json")
			logger.Info("write output called", "out path", outPath)
			return json.StoreFile(outPath, t)
		},
	}

	dispatcher := tasks.Dispatcher[UniqueNames, UniqueNames]{
		Worker:          worker,
		NoWorker:        runtime.NumCPU(),
		ResultCollector: writer,
		Producer:        it,
	}

	dispatcher.Dispatch()

	elapsed := time.Since(start)
	logger.Info("Execution finished", "runtime", elapsed)
}
