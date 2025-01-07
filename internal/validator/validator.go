package validator

import (
	"log"
	"os"
)

func validateFilePath(p string) os.FileInfo {

	f, err := os.Stat(p)
	if err != nil {
		log.Fatal(err)
	}

	return f
}

// validates output path and sets default tmpdir
// as default value if *p == ""
// p - path
// exits the program on fail
func ValidateOutPath(p *string) {
	// set default value if needed
	if *p == "" {
		dir := os.TempDir()
		p = &dir
	}

	f := validateFilePath(*p)

	if !f.IsDir() {
		log.Fatal("out path must be a directory.")
	}
}

// validates input path and sets a current working dir
// as default value if *p == ""
// p - path
// exits the program on fail
func ValidateInPath(p *string) os.FileInfo {

	// set default value if needed
	if *p == "" {
		dir, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		p = &dir
	}

	return validateFilePath(*p)
}
