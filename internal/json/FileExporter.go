package json

import (
	"encoding/json"
	"os"
	"sbom-processor/internal/sbom"
)

type JsonSbom interface {
	sbom.SyftSbom | sbom.CyclonedxSbom
}

func Store(path string, element any) error {

	outFile, err := os.Create(path)
	if err != nil {
		return err
	}

	defer outFile.Close()

	encoder := json.NewEncoder(outFile)
	err = encoder.Encode(&element)
	if err != nil {
		return err
	}

	return nil
}
