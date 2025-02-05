package json

import (
	"encoding/json"
	"os"
)

type JsonFileExporter struct {
	Path string
}

func (e *JsonFileExporter) Store(path string, element any) error {

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
