package json

import (
	"io/fs"
	"os"
	"path/filepath"
)

func CollectJsonFiles(p string) ([]string, error) {

	f, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	var paths []string

	if f.IsDir() {

		filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			if filepath.Ext(path) != ".json" {
				return nil
			}

			paths = append(paths, path)

			return nil
		})
	} else {
		if filepath.Ext(p) != ".json" {
			return paths, nil
		}

		paths = append(paths, p)
	}

	return paths, nil
}
