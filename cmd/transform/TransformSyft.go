package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"time"

	"sbom-processor/internal/json"
	"sbom-processor/internal/sbom"
	"sbom-processor/internal/tasks"
	"sbom-processor/internal/validator"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var mode = flag.String("mode", "file", "file or db. defines whether to store results in db or file.")
var in = flag.String("in", "", "Path to SBOM")
var out = flag.String("out", "", "File to write the SBOM to")

func main() {

	start := time.Now()
	// get input path and check for correctness
	flag.Parse()

	if *mode != "file" && *mode != "db" {
		panic("Unkown operation mode, choose file or db")
	}

	// INPUT VALIDATION
	uri := os.Getenv("MONGO_URI")
	usr := os.Getenv("MONGO_USERNAME")
	pwd := os.Getenv("MONGO_PWD")

	if *mode == "db" && (usr == "" || pwd == "" || uri == "") {
		log.Fatalf("uri, username or password not found. Make sure MONGO_USERNAME, MONGO_PWD, and MONGO_URI are set\n")
	}

	_, err := validator.ValidateInPath(in)
	if err != nil {
		log.Fatal(err)
	}

	if *mode == "file" {
		err = validator.ValidateOutPath(out)
		if err != nil {
			log.Fatal(err)
		}
	}

	paths, err := json.CollectJsonFiles(*in)
	if err != nil {
		log.Fatal(err)
	}

	worker := tasks.Worker[string, sbom.CyclonedxSbom]{
		Do: transformSbom,
	}

	if *mode == "file" {
		fileWriter := tasks.BufferedWriter[sbom.CyclonedxSbom]{
			Buffer:  1,
			DoWrite: writeToFile,
		}

		d := tasks.Dispatcher[string, sbom.CyclonedxSbom]{
			NoWorker:        runtime.NumCPU(),
			Worker:          worker,
			ResultCollector: fileWriter,
			Producer:        slices.Values(paths),
		}

		d.Dispatch()
	} else {

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

		coll := client.Database("sbom_metadata").Collection("sboms")

		buffer := 200
		dbWriter := tasks.NewBufferedWriter(
			func(t []*sbom.CyclonedxSbom) error {
				_, err := coll.InsertMany(context.Background(), t)

				return err
			},
			tasks.BufferedWriterConfig{Buffer: &buffer})

		d := tasks.NewDispatcher(worker, slices.Values(paths), *dbWriter, tasks.DispatcherConfig{})

		d.Dispatch()
	}

	fmt.Println("Finished main")
	elapsed := time.Since(start)
	fmt.Printf("Execution time: %s\n", elapsed)
}

func writeToFile(t []*sbom.CyclonedxSbom) error {
	for _, s := range t {
		ts := time.Now().Format("20060102150405") // Format: YYYYMMDDHHMMSS
		outPath := filepath.Join(*out, s.Source.Id+"-"+ts+".json")
		err := json.Store(outPath, s)
		if err != nil {
			fmt.Printf("err during file storage %s\n", err)
		}
	}

	return nil
}

func transformSbom(p *string) (*sbom.CyclonedxSbom, error) {
	syft, err := sbom.ReadSyft(p)
	if err != nil {
		return nil, err
	}

	return syft.Transform()
}
