package mvn

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type QueryResult struct {
	Name string `bson:"_id" json:"_id"`
}

type MvnCacheEntry struct {
	Name              string `bson:"name"`
	MvnSearchResponse `bson:"mvn_search_response"`
}

type MvnSearchResponse struct {
	NumFound int   `json:"numFound" bson:"num_found"`
	Start    int   `json:"start" bson:"start"`
	Docs     []Doc `json:"docs" bson:"docs"`
}

type Doc struct {
	Id            string `json:"id" bson:"id"`
	Group         string `json:"g" bson:"group"`
	Artifact      string `json:"a" bson:"artifact"`
	LatestVersion string `json:"latestVersion" bson:"latestVersion"`
	RepositoryId  string `json:"repositoryId" bson:"repositoryId"`
	P             string `json:"p" bson:"p"`
	TimeStamp     int    `json:"timestamp" bson:"time_stamp"`
	VersionCount  int    `json:"versionCount" bson:"version_count"`
}

type MvnCache struct {
	// raw search url:
	// https://search.maven.org/solrsearch/select?q=a:cloudevents-api&rows=20&wt=json

	// single results (high confidence to found the right package)
	MvnMirror *mongo.Collection

	// multiple results
	MultiResult *mongo.Collection

	// no results
	// type java && not contained in mvn central
	Blacklist *mongo.Collection

	Ctx context.Context
}

// create X channels to sent responses to
// create Y worker to perform:
// - cache check
// - API query
// - sent the response to the corresponding channel (failed, success, furtherInvestigation)
// collect the results from the channels until a threshold is reached
// InsertMany(results) into database
// repeat until all workers finished
// insert remaining elements
func componentWorker(mvnCache *MvnCache, components <-chan string, cache chan *MvnCacheEntry, blacklist, multiResult chan string) {

	for c := range components {
		if mvnCache.isInCache(c) {
			fmt.Printf("%s found in cache\n", c)
			continue
		}

		mvnRes, err := queryApi(c)
		if err != nil {
			blacklist <- c
			continue
		}

		switch {
		case mvnRes.NumFound == 1:
			cache <- &MvnCacheEntry{Name: c, MvnSearchResponse: *mvnRes}
		case mvnRes.NumFound > 1:
			multiResult <- c
		case mvnRes.NumFound < 1:
			blacklist <- c
		}
	}
}

func resultCollector(cache *MvnCache, mirror <-chan *MvnCacheEntry, multiResult, blacklist <-chan string, done <-chan int) {
	mirrorBuffer := []MvnCacheEntry{}
	blackListBuffer := []string{}
	multiBuffer := []string{}

	for {
		select {
		case ce := <-mirror:
			mirrorBuffer = append(mirrorBuffer, *ce)
			if len(mirrorBuffer) > 200 {
				cache.MvnMirror.InsertMany(cache.Ctx, mirrorBuffer)
				mirrorBuffer = []MvnCacheEntry{}
				fmt.Println("successfully inserted 200 elements in mirror")
			}
		case f := <-blacklist:
			blackListBuffer = append(blackListBuffer, f)
			if len(blackListBuffer) > 200 {
				cache.Blacklist.InsertMany(cache.Ctx, blackListBuffer)
				blackListBuffer = []string{}
				fmt.Println("successfully inserted 200 elements in blacklist")
			}
		case m := <-multiResult:
			multiBuffer = append(multiBuffer, m)
			if len(multiBuffer) > 200 {
				cache.MultiResult.InsertMany(cache.Ctx, multiBuffer)
				multiBuffer = []string{}
				fmt.Println("successfully inserted 200 elements in multi")
			}
		case <-done:
			// empty all
			if len(multiBuffer) > 0 {
				cache.MultiResult.InsertMany(cache.Ctx, multiBuffer)
			}
			if len(blackListBuffer) > 0 {
				cache.Blacklist.InsertMany(cache.Ctx, blackListBuffer)
			}
			if len(mirrorBuffer) > 0 {
				cache.MvnMirror.InsertMany(cache.Ctx, mirrorBuffer)
			}
			return
		}
	}
}

// batch querry all unique components from collection
// push components into components channel to distribute them between worker
// collect results and wait until we have enough results for InsertMany
func (cache *MvnCache) FillCache(sboms *mongo.Collection) error {

	// prep db query
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
		return err
	}
	defer cursor.Close(cache.Ctx)

	// setup worker
	components := make(chan string)
	blacklist := make(chan string)
	multiResult := make(chan string)
	mirror := make(chan *MvnCacheEntry)

	done := make(chan int)

	go resultCollector(cache, mirror, multiResult, blacklist, done)

	for i := 0; i < 7; i++ {
		go componentWorker(cache, components, mirror, blacklist, multiResult)
	}

	failed := 0
	counter := 0

	// iterate db results
	for cursor.Next(cache.Ctx) {
		var res QueryResult
		if err := cursor.Decode(&res); err != nil {
			fmt.Printf("decode failed\n")
			failed += 1
			if failed%10 == 0 {
				fmt.Printf("failed to process %d components", failed)
			}
			continue
		}

		components <- res.Name
		counter += 1
		if counter%100 == 0 {
			fmt.Printf("processed %d components", counter)
		}
	}

	// closing components ends the workers
	close(components)
	done <- 0

	return nil
}

func (cache *MvnCache) isInCache(name string) bool {

	var inCache = func(coll *mongo.Collection, key string) bool {
		filter := bson.D{{Key: key, Value: name}}
		err := coll.FindOne(cache.Ctx, filter).Err()
		return err != mongo.ErrNoDocuments
	}

	return inCache(cache.Blacklist, "name") &&
		inCache(cache.MultiResult, "name") &&
		inCache(cache.MvnMirror, "name")
}
