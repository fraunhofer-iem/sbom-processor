package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
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
var dbName = flag.String("db", "sbom_metadata", "database name to connect to")
var collectionName = flag.String("collection", "sboms", "collection name for SBOMs")
var in = flag.String("in", "", "Path to SBOM")
var out = flag.String("out", "", "File to write the SBOM to")
var logLevel = flag.Int("logLevel", 0, "Can be 0 for INFO, -4 for DEBUG, 4 for WARN, or 8 for ERROR. Defaults to INFO.")

func main() {

	start := time.Now()
	// get input path and check for correctness
	flag.Parse()

	var lvl slog.Level

	switch {
	case *logLevel < int(slog.LevelInfo):
		lvl = slog.LevelDebug
	case *logLevel < int(slog.LevelWarn):
		lvl = slog.LevelInfo
	case *logLevel < int(slog.LevelError):
		lvl = slog.LevelWarn
	default:
		lvl = slog.LevelError
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	}))

	slog.SetDefault(logger)

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

	logger.Info("Starting syft to cyclonedx transformation", "path", *in, "mode", *mode)

	var writer *tasks.BufferedWriter[sbom.CyclonedxSbom]

	if *mode == "file" {
		buffer := 100
		writer = tasks.NewBufferedWriter(
			writeToFile,
			tasks.BufferedWriterConfig{Buffer: &buffer})
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

		coll := client.Database(*dbName).Collection(*collectionName)

		buffer := 200
		writer = tasks.NewBufferedWriter(
			func(t []*sbom.CyclonedxSbom) error {
				_, err := coll.InsertMany(context.Background(), t)

				return err
			},
			tasks.BufferedWriterConfig{Buffer: &buffer})
	}

	worker := tasks.Worker[string, sbom.CyclonedxSbom]{
		Do: transformSbom,
	}
	noWorker := runtime.NumCPU()

	d := tasks.NewDispatcher(worker, slices.Values(paths), *writer,
		tasks.DispatcherConfig{NoWorker: &noWorker})

	logger.Debug("Initialized dispatcher", "dispatcher", d)

	d.Dispatch()

	elapsed := time.Since(start)
	logger.Info("Finished syft transform", "time elapsed", elapsed)
}

func writeToFile(t []*sbom.CyclonedxSbom) error {
	var err error
	for _, s := range t {
		ts := time.Now().Format("20060102150405") // Format: YYYYMMDDHHMMSS
		outPath := filepath.Join(*out, s.Source.Id+"-"+ts+".json")
		err = json.Store(outPath, s)
		if err != nil {
			slog.Default().Error("err during file storage", "file", outPath, "error", err)
		}
	}

	return err
}

func transformSbom(p *string) (*sbom.CyclonedxSbom, error) {
	syft, err := sbom.ReadSyft(p)
	if err != nil {
		return nil, err
	}

	return syft.Transform()
}
