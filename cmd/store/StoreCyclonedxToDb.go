package main

import (
	"context"
	"fmt"
	"log"
	"sbom-processor/internal/json"
	"sbom-processor/internal/sbom"
	"sbom-processor/internal/validator"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

func StoreCyclonedx(in *string, client *mongo.Client) {
	_, err := validator.ValidateInPath(in)
	if err != nil {
		log.Fatal(err)
	}

	coll := client.Database("sbom_metadata").Collection("sboms")

	paths, err := json.CollectJsonFiles(*in)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.TODO()
	sboms := []*sbom.CyclonedxSbom{}
	collectedPaths := []string{}

	for _, p := range paths {

		if len(sboms) > 200 {
			_, err = coll.InsertMany(ctx, sboms)
			if err != nil {
				for _, c := range collectedPaths {
					fmt.Printf("Insert failed for %s\n", c)
				}
			}
			for _, c := range collectedPaths {
				fmt.Printf("Insert success for %s\n", c)
			}
			collectedPaths = []string{}
			sboms = []*sbom.CyclonedxSbom{}
		} else {
			s, err := sbom.ReadCyclonedx(p)
			if err != nil {
				fmt.Printf("Read syft failed with %s for %s\n", err.Error(), p)
				continue
				// return
			}
			collectedPaths = append(collectedPaths, p)
			sboms = append(sboms, s)
		}

	}
}
