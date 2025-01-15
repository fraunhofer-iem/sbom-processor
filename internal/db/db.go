package db

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MongoIndex struct {
	Name string `bson:"name"`
}

// CREATE INDEX IF NOT EXIST
func CreateIdx(coll *mongo.Collection, idxKey string) error {

	ctx := context.Background()

	// CHECK IF IDX EXISTS
	idx, err := coll.Indexes().List(context.TODO())
	if err != nil {
		return err
	}

	defer idx.Close(ctx)

	idxExists := false

	for {
		var curr MongoIndex
		err := idx.Decode(&curr)

		if err == nil {
			if curr.Name == idxKey+"_1" {
				idxExists = true
				break
			}
		}

		if !idx.Next(ctx) {
			break
		}
	}

	if idxExists {
		return nil
	}

	// CREATE IDX
	indexModel := mongo.IndexModel{
		Keys: bson.D{{Key: idxKey, Value: 1}},
	}
	name, err := coll.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		return err
	}

	fmt.Println("Name of created index : " + name)
	return nil
}
