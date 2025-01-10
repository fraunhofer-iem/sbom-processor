package json

import (
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

		// filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
		// 	if err != nil {
		// 		return err
		// 	}

		// 	if d.IsDir() {
		// 		return nil
		// 	}

		// 	if filepath.Ext(path) != ".json" {
		// 		return nil
		// 	}

		// 	paths = append(paths, path)

		// 	return nil
		// })
	} else {
		if filepath.Ext(p) != ".json" {
			return paths, nil
		}

		paths = append(paths, p)
	}

	return paths, nil
}
