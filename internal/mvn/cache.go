package mvn

import (
	"context"
	"fmt"
	"sbom-processor/internal/sbom"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MvnCacheEntry struct {
	ComponentId       string `bson:"component_id"`
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

	ctx context.Context
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
func componentWorker(mvnCache *MvnCache, components <-chan *sbom.Component, cache chan *MvnCacheEntry, blacklist, multiResult chan string) {

	for c := range components {
		if mvnCache.IsInCache(c.Id) {
			fmt.Printf("%s %s found in cache\n", c.Id, c.Name)
			continue
		}

		mvnRes, err := queryApi(c.Name)
		if err != nil {
			blacklist <- c.Id
		}

		switch {
		case mvnRes.NumFound == 1:
			cache <- &MvnCacheEntry{ComponentId: c.Id, MvnSearchResponse: *mvnRes}
		case mvnRes.NumFound > 1:
			multiResult <- c.Id
		case mvnRes.NumFound < 1:
			blacklist <- c.Id
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
				cache.MvnMirror.InsertMany(cache.ctx, mirrorBuffer)
				mirrorBuffer = []MvnCacheEntry{}
			}
		case f := <-blacklist:
			blackListBuffer = append(blackListBuffer, f)
			if len(blackListBuffer) > 200 {
				cache.Blacklist.InsertMany(cache.ctx, blackListBuffer)
				blackListBuffer = []string{}
			}
		case m := <-multiResult:
			multiBuffer = append(multiBuffer, m)
			if len(multiBuffer) > 200 {
				cache.MultiResult.InsertMany(cache.ctx, multiBuffer)
				multiBuffer = []string{}
			}
		case <-done:
			// empty all
			if len(multiBuffer) > 0 {
				cache.MultiResult.InsertMany(cache.ctx, multiBuffer)
			}
			if len(blackListBuffer) > 0 {
				cache.Blacklist.InsertMany(cache.ctx, blackListBuffer)
			}
			if len(mirrorBuffer) > 0 {
				cache.MvnMirror.InsertMany(cache.ctx, mirrorBuffer)
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
		{{Key: "$unwind", Value: "$components"}},
		{{Key: "$match", Value: bson.D{{Key: "components.language", Value: "java"}}}},
		{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$components.id"}}}},
	}

	cursor, err := sboms.Aggregate(context.TODO(), pipeline, options.Aggregate().SetBatchSize(1000))
	if err != nil {
		return err
	}
	defer cursor.Close(cache.ctx)

	// setup worker
	components := make(chan *sbom.Component)
	blacklist := make(chan string)
	multiResult := make(chan string)
	mirror := make(chan *MvnCacheEntry)

	done := make(chan int)

	go resultCollector(cache, mirror, multiResult, blacklist, done)

	for i := 0; i < 7; i++ {
		go componentWorker(cache, components, mirror, blacklist, multiResult)
	}

	// iterate db results
	for cursor.Next(cache.ctx) {

		var component sbom.Component
		if err = cursor.Decode(&component); err != nil {
			continue
		}

		components <- &component
	}

	// closing components ends the workers
	close(components)
	done <- 0

	return nil
}

func (cache *MvnCache) IsInCache(componentId string) bool {

	var inCache = func(coll *mongo.Collection, key string) bool {
		filter := bson.D{{Key: key, Value: componentId}}
		err := coll.FindOne(cache.ctx, filter).Err()
		return err == mongo.ErrNoDocuments
	}

	return inCache(cache.Blacklist, "component_id") &&
		inCache(cache.MultiResult, "component_id") &&
		inCache(cache.MvnMirror, "component_id")
}
