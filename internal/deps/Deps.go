package deps

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
)

type Deps struct {
	Name     string    `bson:"name" json:"name"`
	System   string    `bson:"system" json:"system"`
	Versions []Version `bson:"versions" json:"versions"`
}

type Version struct {
	Version     string `bson:"version" json:"version"`
	PublishedAt string `bson:"publishedAt" json:"publishedAt"`
}

type CacheRequest struct {
	Name   string
	System string
}

type DepsApiResponse struct {
	Versions []VersionsApiResponse `json:"versions"`
}

type VersionsApiResponse struct {
	Version     Version `json:"versionKey"`
	PublishedAt string  `bson:"publishedAt" json:"publishedAt"`
}

func DepsWorkerDo(c CacheRequest) (*Deps, error) {
	deps, err := queryApi(c)
	if err != nil {
		// TODO: check if we can retry the request
		return nil, err
	}

	versions := make([]Version, 0)
	for _, v := range deps.Versions {
		v.Version.PublishedAt = v.PublishedAt
		versions = append(versions, v.Version)
	}

	return &Deps{
		Name:     c.Name,
		System:   c.System,
		Versions: versions,
	}, nil
}

func queryApi(c CacheRequest) (*DepsApiResponse, error) {
	encodedName := url.QueryEscape(c.Name)
	encodedSystem := url.QueryEscape(c.System)

	// GET /v3/systems/{packageKey.system}/packages/{packageKey.name}
	url := fmt.Sprintf("https://api.deps.dev/v3/systems/%s/packages/%s", encodedSystem, encodedName)
	resp, err := http.Get(url)
	if err != nil {
		slog.Default().Debug("Request failed with", "url", url, "err", err.Error())
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			retry := resp.Header.Get("Retry-After")
			slog.Default().Debug("Failed due to too many requests.", "retry-after", retry)
		}
		err := fmt.Errorf("request failed with status code %d", resp.StatusCode)
		slog.Default().Debug("Request failed with", "url", url, "err", err.Error())
		return nil, err
	}

	var deps DepsApiResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&deps); err != nil {
		slog.Default().Debug("Decoding of response failed", "url", url, "err", err.Error())
		return nil, err
	}

	return &deps, nil
}
