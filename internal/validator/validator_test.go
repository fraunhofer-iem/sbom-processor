package validator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateOutPathEmpty(t *testing.T) {
	empty := ""
	err := ValidateOutPath(&empty)
	if err != nil {
		t.FailNow()
	}

	if empty != os.TempDir() {
		t.FailNow()
	}
}
func TestValidateOutPathValid(t *testing.T) {
	dir := t.TempDir()

	err := ValidateOutPath(&dir) // nothing should happen. i.e. we should not fail/err

	if err != nil {
		t.FailNow()
	}

}
func TestValidateOutPathNotExist(t *testing.T) {
	dir := t.TempDir()
	nonExisting := filepath.Join(dir, "not", "existing", "dir")

	err := ValidateOutPath(&nonExisting)
	if err == nil {
		t.FailNow()
	}
}

func TestValidateOutPathFile(t *testing.T) {
	dir := t.TempDir()

	fp := filepath.Join(dir, "test.tx")

	err := ValidateOutPath(&fp)
	if err == nil {
		t.FailNow()
	}
}

func TestValidateInEmpty(t *testing.T) {
	empty := ""
	_, err := ValidateInPath(&empty)
	if err != nil {
		t.FailNow()
	}

	wd, err := os.Getwd()
	if err != nil {
		t.FailNow()
	}

	if empty != wd {
		t.FailNow()
	}
}

func TestValidateInDir(t *testing.T) {
	dir := t.TempDir()

	_, err := ValidateInPath(&dir) // nothing should happen. i.e. we should not fail/err

	if err != nil {
		t.FailNow()
	}
}
func TestValidateInFile(t *testing.T) {

	dir := t.TempDir()

	fp := filepath.Join(dir, "test.tx")

	err := ValidateOutPath(&fp)
	if err == nil {
		t.FailNow()
	}
}

func TestValidateInNotExist(t *testing.T) {
	dir := t.TempDir()
	nonExisting := filepath.Join(dir, "not", "existing", "dir")

	f, err := ValidateInPath(&nonExisting)
	if err == nil {
		t.FailNow()
	}
	if f != nil {
		t.FailNow()
	}
}
