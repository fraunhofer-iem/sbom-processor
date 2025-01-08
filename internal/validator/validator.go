package validator

import (
	"errors"
	"os"
)

// validates output path and sets default tmpdir
// as default value if *p == ""
// p - path
func ValidateOutPath(p *string) error {
	// set default value if needed
	if *p == "" {
		dir := os.TempDir()
		*p = dir
	}

	f, err := os.Stat(*p)
	if err != nil {
		return err
	}

	if !f.IsDir() {
		return errors.New("out path must be a directory")
	}

	return nil
}

// validates input path and sets a current working dir
// as default value if *p == ""
// p - path
func ValidateInPath(p *string) (os.FileInfo, error) {

	// set default value if needed
	if *p == "" {
		dir, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		*p = dir
	}
	f, err := os.Stat(*p)
	if err != nil {
		return nil, err
	}

	return f, nil
}
