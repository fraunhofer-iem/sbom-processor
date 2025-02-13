package sbom

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestGetDebVersions(t *testing.T) {
	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"package": "package 1", "result": [{"version": "0.0.1"}, {"version":"0.0.2"}, {"version":"0.0.3"}, {"version":"0.0.4"}]}`))
			},
		))

	defer srv.Close()

	v, err := getDebVersions(srv.URL, "rand")
	if err != nil {
		t.Fatalf("no error expected for valid request, %s", err.Error())
	}

	if len(v) != 4 {
		t.Fatalf("incorrect number of versions returned")
	}

	validVersions := []string{"0.0.1", "0.0.2", "0.0.3", "0.0.4"}

	for _, v := range v {
		contained := false
		for _, tmp := range validVersions {
			if tmp == v {
				contained = true
			}
		}

		if !contained {
			t.Fatalf("invalid version")
		}
		contained = false
	}
}

func TestCyclonedxCollect(t *testing.T) {

	dir := t.TempDir()
	p := filepath.Join(dir, "cyclonedx.json")

	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("unable to create test file %s", err.Error())
	}

	defer f.Close()

	f.WriteString("{\"components\" : [{\"name\": \"name\", \"type\":\"type\", \"id\": \"id\", \"language\":\"language\", \"version\":\"version\"}], \"dependencies\": []}")

	s, err := ReadCyclonedx(p)
	if err != nil {
		t.Fatalf("collect failed with %s", err)
	}

	if len(s.Components) != 1 {
		t.Fatalf("Incorrect component length")
	}

}
