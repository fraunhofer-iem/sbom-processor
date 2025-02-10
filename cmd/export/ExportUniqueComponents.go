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
	"sbom-processor/internal/tasks"
	"sbom-processor/internal/validator"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type UniqueNames struct {
	Name string `bson:"_id" json:"_id"`
}

var out = flag.String("out", "", "File to write the SBOM to")

func main() {

	flag.Parse()
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

	sboms := client.Database("sbom_metadata").Collection("sboms")
	// prep db query
	// TODO: make component type configurable
	pipeline := mongo.Pipeline{
		{
			{Key: "$match", Value: bson.D{{Key: "components.type", Value: "java-archive"}}},
		},
		{
			{Key: "$unwind", Value: "$components"},
		},
		{{Key: "$match", Value: bson.D{{Key: "components.type", Value: "java-archive"}}}},
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
			return json.Store(outPath, t)
		},
	}

	dispatcher := tasks.Dispatcher[UniqueNames, UniqueNames]{
		Worker:          worker,
		NoWorker:        runtime.NumCPU(),
		ResultCollector: writer,
		Producer:        it,
	}

	dispatcher.Dispatch()
}
