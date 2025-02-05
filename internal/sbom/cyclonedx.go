package sbom

import (
	"encoding/json"
	"fmt"
	"os"
)

func ReadCyclonedx(p string) (*CyclonedxSbom, error) {
	var sbom CyclonedxSbom
	file, err := os.Open(p)

	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	if err := decoder.Decode(&sbom); err != nil {
		return nil, err
	}

	if sbom.Components == nil ||
		sbom.Dependencies == nil {
		return nil, fmt.Errorf("incomplete sbom")
	}

	return &sbom, nil
}
