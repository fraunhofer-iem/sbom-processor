package db

import (
	"context"
	"fmt"
	"iter"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MongoIndex struct {
	Name string `bson:"name"`
}

// type Seq[V any] func(yield func(V) bool)
func MongodbIterator[T any](c *mongo.Cursor) iter.Seq[T] {
	ctx := context.Background()
	failed := 0
	counter := 0

	return func(yield func(T) bool) {
		var res T
		for c.Next(ctx) {
			if err := c.Decode(&res); err != nil {
				failed += 1
				if failed%10 == 0 {
					fmt.Printf("database response decode failed. Failed for %d elements\n", failed)
				}
				continue
			}

			counter += 1

			if counter%100 == 0 {
				fmt.Printf("processed %d elements\n", counter)
			}

			if !yield(res) {
				fmt.Printf("processed all elements\n")
				return
			}

		}
	}
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
