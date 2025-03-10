package json

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// takes a directory path and returns all *.json files in it
// doesn't travers sub directories.
// if no files are found returns nil
func CollectJsonFiles(p string) ([]string, error) {
	f, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	var paths []string

	if f.IsDir() {

		dirs, err := os.ReadDir(p)
		if err != nil {
			return nil, err
		}

		for _, d := range dirs {
			if d.IsDir() {
				continue
			}

			if !strings.HasSuffix(d.Name(), ".json") {
				continue
			}

			p := filepath.Join(p, d.Name())
			paths = append(paths, p)
		}

	} else {
		if filepath.Ext(p) != ".json" {
			return paths, nil
		}

		paths = append(paths, p)
	}

	return paths, nil
}

func StoreFile(path string, element any) error {
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
