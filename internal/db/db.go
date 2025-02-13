package db

import (
	"context"
	"iter"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MongoIndex struct {
	Name string `bson:"name"`
}

var logger = slog.Default()

func MongodbIterator[T any](c *mongo.Cursor) iter.Seq[T] {
	ctx := context.Background()
	failed := 0
	counter := 0

	return func(yield func(T) bool) {
		var res T
		for c.Next(ctx) {
			if err := c.Decode(&res); err != nil {
				failed += 1
				continue
			}

			counter += 1

			if counter%100 == 0 {
				logger.Info("processed elements", "processed", counter, "failed", failed)
			}

			if !yield(res) {
				logger.Info("processed all elements", "processed", counter, "failed", failed)
				return
			}
		}
	}
}

// CREATE INDEX IF NOT EXIST
func CreateIdx(coll *mongo.Collection, idxKey string) error {

	ctx := context.Background()

	logger.Debug("Create idx called", "idx name", idxKey)

	// CHECK IF IDX EXISTS
	idx, err := coll.Indexes().List(context.TODO())
	if err != nil {
		return err
	}

	defer idx.Close(ctx)

	idxExists := false

	for idx.Next(ctx) {
		var curr MongoIndex
		err := idx.Decode(&curr)

		if err == nil && curr.Name == idxKey+"_1" {
			idxExists = true
			break
		}
	}

	if idxExists {
		logger.Debug("Index already exists", "idx name", idxKey)
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

	logger.Debug("Index successfully created", "idx name", name)
	return nil
}
