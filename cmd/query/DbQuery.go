package main

import (
	"context"
	"flag"
	"log"
	"math"
	"os"
	"path/filepath"
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
	Name  string `bson:"_id" json:"_id"`
	Count int    `bson:"count" json:"count"`
}

var out = flag.String("out", "", "File to write the SBOM to")
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

	logger.Info("DB Query", "db", *dbName, "collection", *collectionName)

	// prep db query
	pipeline := mongo.Pipeline{
		{{Key: "$project", Value: bson.D{
			{Key: "nameWithoutPostfix", Value: bson.D{
				{Key: "$arrayElemAt", Value: bson.A{
					bson.D{{Key: "$split", Value: bson.A{"$source.name", ":"}}},
					0,
				}},
			}},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$nameWithoutPostfix"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			// Add other aggregations if needed
		}}},
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
	DoWrite := func(t []*UniqueNames) error {
		outPath := filepath.Join(*out, "productNames.json")
		logger.Info("write output called", "out path", outPath)
		return json.StoreFile(outPath, t)
	}

	buffer := math.MaxInt

	writer := tasks.NewBufferedWriter(DoWrite, tasks.BufferedWriterConfig{Buffer: &buffer})
	dispatcher := tasks.NewDispatcher(worker, it, *writer, tasks.DispatcherConfig{})

	dispatcher.Dispatch()

	elapsed := time.Since(start)
	logger.Info("Finished syft transform", "time elapsed", elapsed)
}
