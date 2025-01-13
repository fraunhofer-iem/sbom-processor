package sbom

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
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

func ReadCyclonedx(p string) (*CyclonedxSbom, error) {
	file, err := os.Open(p)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	decoder := json.NewDecoder(file)
	var sbom CyclonedxSbom

	if err := decoder.Decode(&sbom); err != nil {
		return nil, err
	}

	if sbom.Components == nil ||
		sbom.Dependencies == nil {
		return nil, fmt.Errorf("incomplete sbom")
	}

	return &sbom, nil
}

func (c *Component) GetVersions() ([]string, error) {
	if c.Type == deb {
		return getDebVersions(debBasePath, c.Name)
	}

	return nil, fmt.Errorf("unkown component type for %+v", *c)
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

func (c *CyclonedxSbom) Store(out string) error {

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
