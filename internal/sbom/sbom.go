package sbom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sbom-processor/internal/semver"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Target struct {
	Child string `json:"child"`
	Type  string `json:"type"`
}

type Dependency struct {
	Ref       string   `json:"ref"`
	DependsOn []Target `json:"dependsOn"`
}

type CyclonedxSbom struct {
	Components   []Component  `json:"components"`
	Dependencies []Dependency `json:"dependencies"`
	Source       Source       `json:"source"`
	Distro       Distro       `json:"distro"`
}

type DebVersionResponse struct {
	Package string    `json:"package"`
	Result  []Version `json:"result"`
}

type Version struct {
	Version string `json:"version"`
}

const deb string = "deb"
const debBasePath string = "https://snapshot.debian.org/mr/package/"

type CyclonedxReader struct {
	FileReader CycloneDxFileReader
	DbReader   CycloneDxDbReader
}

type CycloneDxFileReader struct{}
type CycloneDxDbReader struct {
	Sboms mongo.Collection
}

type CycloneDxDbStore struct {
	Sboms mongo.Collection
}

func (r *CycloneDxFileReader) Collect(i <-chan string, out chan *CyclonedxSbom, errc chan error) {
	for p := range i {
		var v CyclonedxSbom
		func() {
			file, err := os.Open(p)
			if err != nil {
				errc <- err
			}
			defer file.Close()

			decoder := json.NewDecoder(file)

			if err := decoder.Decode(&v); err != nil {
				errc <- err
			}

			if v.Components == nil ||
				v.Dependencies == nil {
				errc <- fmt.Errorf("incomplete sbom")
			}
		}()

		out <- &v
	}
}

func (s *CycloneDxDbStore) StoreBulk(c context.Context, e []*CyclonedxSbom) error {
	_, err := s.Sboms.InsertMany(c, e)
	return err
}

func (c *Component) IsInCache(cache, blackList *mongo.Collection) bool {
	// check if versions are in db before continue
	err := blackList.FindOne(context.TODO(), bson.D{{Key: "id", Value: c.Id}}).Err()
	if err == nil || err != mongo.ErrNoDocuments {
		fmt.Printf("Versions for %+v in blacklist db\n", c)
		return true
	}

	filter := bson.D{{Key: "component_id", Value: c.Id}}
	err = cache.FindOne(context.TODO(), filter).Err()
	if err == nil || err != mongo.ErrNoDocuments {
		fmt.Printf("Versions for %+v already in db\n", c)
		return true
	}

	return false
}

func (c *Component) GetVersions() (*semver.ComponentVersions, error) {
	var raw []string
	var err error

	switch c.Type {
	case deb:
		raw, err = getDebVersions(debBasePath, c.Name)
	default:
		raw = nil
		err = fmt.Errorf("unkown component type")
	}

	if err != nil {
		return nil, err
	}

	compVer := make([]semver.ComponentVersion, len(raw))
	for i, v := range raw {
		compVer[i] = semver.ComponentVersion{
			Version:     v,
			ReleaseDate: "",
		}
	}
	compVers := semver.ComponentVersions{
		ComponentId: c.Id,
		Versions:    compVer,
	}

	return &compVers, nil
}

func getDebVersions(basePath string, n string) ([]string, error) {

	if n == "" {
		return nil, fmt.Errorf("can't get version information for empty package name")
	}

	encodedName := url.QueryEscape(n)

	// Ensure the basePath ends with a slash
	if !strings.HasSuffix(basePath, "/") {
		basePath += "/"
	}

	url := basePath + encodedName

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var versionResponse DebVersionResponse
	if err := decoder.Decode(&versionResponse); err != nil {
		return nil, fmt.Errorf("decode of response for %s failed. Response status %s", encodedName, resp.Status)
	}

	if versionResponse.Result == nil {
		return nil, fmt.Errorf("empty result for %s", encodedName)
	}

	var versions = make([]string, len(versionResponse.Result))
	for i, r := range versionResponse.Result {
		versions[i] = r.Version
	}

	return versions, nil
}

func (c *CyclonedxSbom) StoreToFile(out string) error {

	outFile, err := os.Create(out)
	if err != nil {
		return err
	}

	defer outFile.Close()

	encoder := json.NewEncoder(outFile)
	err = encoder.Encode(&c)
	if err != nil {
		return err
	}

	return nil
}
