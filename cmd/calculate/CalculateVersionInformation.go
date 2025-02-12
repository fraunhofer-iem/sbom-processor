package main

import (
	"context"
	"log"
	"os"
	"sbom-processor/internal/mvn"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {

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

	db := client.Database("sbom_metadata")

	cache := mvn.MvnCache{
		MvnMirror:   db.Collection("mvn_mirror"),
		MultiResult: db.Collection("multi_result"),
		Blacklist:   db.Collection("blacklist"),
		Ctx:         context.Background(),
	}

	sbomsColl := db.Collection("sboms")

	cache.FillCache(sbomsColl)

	// var results []bson.M
	// if err = cursor.All(context.TODO(), &results); err != nil {
	// 	panic(err)
	// }

	// fmt.Printf("%+v\n", results)
	// fmt.Printf("length %d", len(results))

}
