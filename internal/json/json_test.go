package json

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectJsonFiles(t *testing.T) {
	dir := t.TempDir()

	jsonFile := filepath.Join(dir, "test.json")
	os.Create(jsonFile)

	txtFile := filepath.Join(dir, "test.txt")
	os.Create(txtFile)

	noExtFile := filepath.Join(dir, "test")
	os.Create(noExtFile)

	dotFile := filepath.Join(dir, "test.")
	os.Create(dotFile)

	slashFile := filepath.Join(dir, "test/")
	os.Create(slashFile)

	doubleExtFile := filepath.Join(dir, "test.json.txt")
	os.Create(doubleExtFile)

	paths, err := CollectJsonFiles(dir)

	if err != nil {
		t.FailNow()
	}

	if len(paths) != 1 {
		t.FailNow()
	}

	if paths[0] != jsonFile {
		t.FailNow()
	}

	paths, err = CollectJsonFiles(jsonFile)

	if err != nil {
		t.FailNow()
	}

	if paths[0] != jsonFile {
		t.FailNow()
	}

	paths, err = CollectJsonFiles(txtFile)

	if err != nil {
		t.FailNow()
	}

	if paths != nil {
		t.FailNow()
	}

	paths, err = CollectJsonFiles("")

	if err == nil {
		t.FailNow()
	}

	if paths != nil {
		t.FailNow()
	}
}
