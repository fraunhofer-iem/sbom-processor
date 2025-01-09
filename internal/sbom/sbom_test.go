package sbom

import (
	"net/http"
	"net/http/httptest"
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
