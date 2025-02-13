package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sbom-processor/internal/db"
	"sbom-processor/internal/json"
	"sbom-processor/internal/logging"
	"sbom-processor/internal/tasks"
	"sbom-processor/internal/validator"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type UniqueNames struct {
	Name string `bson:"_id" json:"_id"`
}

var out = flag.String("out", "", "File to write the SBOM to")
var componentType = flag.String("componentType", "java-archive", "Component type to filter on")
var logLevel = flag.Int("logLevel", 0, "Can be 0 for INFO, -4 for DEBUG, 4 for WARN, or 8 for ERROR. Defaults to INFO.")
var dbName = flag.String("db", "sbom_metadata", "database name to connect to")
var collectionName = flag.String("collection", "sboms", "collection name for SBOMs")

func main() {

	start := time.Now()
	flag.Parse()

	logger := logging.SetUpLogging(*logLevel)

	validator.ValidateOutPath(out)

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

	logger.Info("Export unique components called", "db", *dbName, "collection", *collectionName, "componentType", *componentType)

	// prep db query
	pipeline := mongo.Pipeline{
		{
			{Key: "$match", Value: bson.D{{Key: "components.type", Value: *componentType}}},
		},
		{
			{Key: "$unwind", Value: "$components"},
		},
		{{Key: "$match", Value: bson.D{{Key: "components.type", Value: *componentType}}}},
		{
			{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$components.name"},
			}},
		},
	}

	cursor, err := sboms.Aggregate(context.TODO(), pipeline, options.Aggregate().SetBatchSize(1000))
	if err != nil {
		panic(err)
	}
	defer cursor.Close(context.TODO())

	it := db.MongodbIterator[UniqueNames](cursor)

	worker := tasks.Worker[UniqueNames, UniqueNames]{
		Do: tasks.DoNothing[UniqueNames],
	}

	writer := tasks.BufferedWriter[UniqueNames]{
		Buffer: math.MaxInt,
		DoWrite: func(t []*UniqueNames) error {
			outPath := filepath.Join(*out, "uniqueComponentNames.json")
			fmt.Printf("Print path %s\n", outPath)
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
	logger.Info("Finished syft transform", "time elapsed", elapsed)
}
