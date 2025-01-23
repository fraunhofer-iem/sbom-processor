package store

import (
	"context"
	"fmt"
	"log"
	"runtime"

	"sbom-processor/internal/db"
	"sbom-processor/internal/sbom"
	"sbom-processor/internal/semver"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/sync/semaphore"
)

func StoreVersionInformation(client *mongo.Client) {

	sboms := client.Database("sbom_metadata").Collection("sboms")
	versions := client.Database("sbom_metadata").Collection("versions")
	blackList := client.Database("sbom_metadata").Collection("blacklist")

	err := db.CreateIdx(versions, "component_id")
	if err != nil {
		fmt.Printf("Index creation failed with %s\n", err)
	}

	err = db.CreateIdx(blackList, "id")
	if err != nil {
		fmt.Printf("Index creation failed with %s\n", err)
	}

	// ASYNC ITERATION OF SBOMs AND STORE VERSIONS IN DB
	var (
		maxWorkers = runtime.GOMAXPROCS(0) // 0 = default = maxNumProc
		sem        = semaphore.NewWeighted(int64(maxWorkers))
		ctx        = context.TODO()
	)

	fmt.Printf("Starting up to %d workers\n", maxWorkers)

	cursor, err := sboms.Find(ctx, bson.D{})
	if err != nil {
		panic(err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		fmt.Print("Cursor next call\n")
		var sbom sbom.CyclonedxSbom
		if err = cursor.Decode(&sbom); err != nil {
			continue
		}
		fmt.Printf("Retrieved %s\n", sbom.Source.Name)

		// When maxWorkers goroutines are in flight, Acquire blocks until one of the
		// workers finishes.
		if err := sem.Acquire(ctx, 1); err != nil {
			log.Printf("Failed to acquire semaphore: %v", err)
			break
		}

		go StoreVersions(sbom, sem, versions, blackList)

	}

	// Acquire all of the tokens to wait for any remaining workers to finish.
	if err := sem.Acquire(ctx, int64(maxWorkers)); err != nil {
		log.Printf("Failed to acquire semaphore: %v", err)
	}

}

func StoreVersions(s sbom.CyclonedxSbom, sem *semaphore.Weighted, versionsColl, blackListColl *mongo.Collection) {
	defer sem.Release(1)
	fmt.Printf("Store versions for %s\n", s.Source.Name)

	var (
		cacheCounter = 0
		errCounter   = 0
	)

	versions := make([]*semver.ComponentVersions, 202)

	// GET ALL VERSIONS FOR EACH COMPONENT AND INSERT TO DB
	for _, c := range s.Components {

		if len(versions) > 200 {
			_, err := versionsColl.InsertMany(context.TODO(), versions)
			if err != nil {
				fmt.Printf("db store failed with %s\n", err)
			}
			versions = make([]*semver.ComponentVersions, 202)
		}

		// check if versions are in db before continue
		if c.IsInCache(versionsColl, blackListColl) {
			fmt.Printf("Versions for %+v in cache or blacklist db\n", c)
			cacheCounter += 1
			continue
		}

		ver, err := c.GetVersions()
		if err != nil {
			errCounter += 1
			fmt.Printf("query for %+v failed with %s\n", c, err)
			_, err = blackListColl.InsertOne(context.TODO(), bson.M{"id": c.Id})
			if err != nil {
				fmt.Printf("db store failed with %s\n", err)
			}
			continue
		}

		versions = append(versions, ver)

	}

	fmt.Printf("%d of %d querries found in cash\n", cacheCounter, len(s.Components))
	fmt.Printf("%d of %d querries failed \n", errCounter, len(s.Components))

	fmt.Printf("Finished SBOM processing %s\n", s.Source.Name)
}
