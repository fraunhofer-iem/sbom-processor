package sbom

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadSyftValidEmpty(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "syftTest.json")

	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("unable to create test file %s", err.Error())
	}

	defer f.Close()

	f.WriteString("{\"artifacts\" : [], \"artifactRelationships\": []}")
	s, err := ReadSyft(&p)

	if err != nil {
		t.Fatalf("no error expected for valid Json with empty artifacts")
	}
	if s == nil {
		t.Fatalf("empty artifacts should result in a valid struct")
	}
}

func TestReadSyftValid(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "syftTest.json")

	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("unable to create test file %s", err.Error())
	}

	defer f.Close()

	f.WriteString("{\"artifacts\" : [{\"name\": \"test\", \"id\": \"myId\", \"version\": \"1.0.1\"}], \"artifactRelationships\": [{\"parent\": \"parent\", \"child\": \"child\", \"type\": \"type\"}]}")
	s, err := ReadSyft(&p)

	if err != nil {
		t.Fatalf("no error expected for valid Json with empty artifacts")
	}
	if s == nil {
		t.Fatalf("empty artifacts should result in a valid struct")
	}

	if len(s.Artifacts) != 1 {
		t.Fatalf("exactly one artifact expected")
	}

	if s.Artifacts[0].Name != "test" ||
		s.Artifacts[0].Id != "myId" ||
		s.Artifacts[0].Version != "1.0.1" ||
		s.Artifacts[0].Language != "" {
		t.Fatalf("Unexpected values after parsing JSON")
	}

	if len(s.ArtifactRelationships) != 1 {
		t.Fatalf("exactly one edge expected")
	}

	if s.ArtifactRelationships[0].Child != "child" ||
		s.ArtifactRelationships[0].Parent != "parent" {
		t.Fatalf("Unexpected values after parsing JSON")
	}
}

func TestReadSyftValidJsonMissingKey(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "syftTest.json")

	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("unable to create test file %s", err.Error())
	}

	defer f.Close()
	f.WriteString("{\"artifacts\" : []}")
	s, err := ReadSyft(&p)

	if err == nil {
		t.Fatalf("error expected for valid Json with missing mandatory key")
	}
	if s != nil {
		t.Fatalf("missing mandatory key should result in a nil struct")
	}
}

func TestReadSyftInvalidJson(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "syftTest.json")

	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("unable to create test file %s", err.Error())
	}

	defer f.Close()
	f.WriteString("{\"artifacts : []")
	s, err := ReadSyft(&p)

	if err == nil {
		t.Fatalf("error expected for invalid Json")
	}
	if s != nil {
		t.Fatalf("invalid JSON must result in a nil")
	}
}

func TestReadSyftNoFile(t *testing.T) {

	p := "I don't exist"
	s, err := ReadSyft(&p)

	if err == nil {
		t.Fatalf("error expected for not existing path")
	}
	if s != nil {
		t.Fatalf("not existing path should result in error")
	}
}

func TestTransformValidSbom(t *testing.T) {
	s := SyftSbom{
		ArtifactRelationships: []ArtifactRelationship{
			{
				Parent: "1",
				Child:  "2",
			},
			{
				Parent: "1",
				Child:  "3",
			},
			{
				Parent: "3",
				Child:  "4",
			},
		},
		Artifacts: []Component{
			{
				Name: "Test",
				Id:   "1",
			},
			{
				Name: "Test2",
				Id:   "2",
			},
			{
				Name: "Test3",
				Id:   "3",
			},
			{
				Name: "Test4",
				Id:   "4",
			},
		},
	}

	c, err := s.Transform()

	if err != nil {
		t.Fatalf("transform should succeed %s", err.Error())
	}

	if len(c.Components) != len(s.Artifacts) {
		t.Fatalf("equal amount of components and artifacts expected")
	}

	for _, d := range c.Dependencies {
		if d.Ref == "1" {
			if len(d.DependsOn) != 2 {
				t.Fatalf("Incorrect number of dependencies for %+v", d)
			}
		}
		if d.Ref == "3" {
			if len(d.DependsOn) != 1 {
				t.Fatalf("Incorrect number of dependencies for %+v", d)
			}
		}
	}
}
