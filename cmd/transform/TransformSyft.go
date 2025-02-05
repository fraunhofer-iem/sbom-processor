package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
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
		fileWriter := tasks.BufferedWriter[json.JsonFileExporter, sbom.CyclonedxSbom]{
			Buffer: 1,
			Store: json.JsonFileExporter{
				Path: *out,
			},
			DoWrite: writeToFile,
		}

		d := tasks.Dispatcher[json.JsonFileExporter, string, sbom.CyclonedxSbom]{
			NoWorker:        runtime.NumCPU(),
			Worker:          worker,
			ResultCollector: fileWriter,
		}

		d.Dispatch(paths)
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

		dbWriter := tasks.BufferedWriter[*mongo.Collection, sbom.CyclonedxSbom]{
			Buffer: 200,
			Store:  coll,
			DoWrite: func(s *mongo.Collection, t []*sbom.CyclonedxSbom) error {
				_, err := s.InsertMany(context.Background(), t)

				return err
			},
		}

		d := tasks.Dispatcher[*mongo.Collection, string, sbom.CyclonedxSbom]{
			NoWorker:        runtime.NumCPU(),
			Worker:          worker,
			ResultCollector: dbWriter,
		}

		d.Dispatch(paths)
	}

	fmt.Println("Finished main")
	elapsed := time.Since(start)
	fmt.Printf("Execution time: %s\n", elapsed)
}

func writeToFile(f json.JsonFileExporter, t []*sbom.CyclonedxSbom) error {
	for _, s := range t {
		ts := time.Now().Format("20060102150405") // Format: YYYYMMDDHHMMSS
		outPath := filepath.Join(f.Path, s.Source.Id+"-"+ts+".json")
		err := f.Store(outPath, s)
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
