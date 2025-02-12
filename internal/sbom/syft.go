package sbom

import (
	"encoding/json"
	"fmt"
	"os"
)

type SyftSbom struct {
	Artifacts             []Component            `json:"artifacts"`
	ArtifactRelationships []ArtifactRelationship `json:"artifactRelationships"`
	Source                Source                 `json:"source"`
	Distro                Distro                 `json:"distro"`
}

type Source struct {
	Id       string   `json:"id"`
	Name     string   `json:"name"`
	Version  string   `json:"version"` // in our current example a SHA256
	Metadata Metadata `json:"metadata"`
}

type Metadata struct {
	Labels  map[string]string `json:"labels"`
	ImageId string            `json:"imageID"`
}

type Distro struct {
	Id      string `json:"id"`
	Version string `json:"versionID"`
}

type ArtifactRelationship struct {
	Parent string `json:"parent"`
	Child  string `json:"child"`
	Type   string `json:"type"`
}

type Component struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Id       string `json:"id"`
	Language string `json:"language"`
	Version  string `json:"version"`
}

func ReadSyft(p *string) (*SyftSbom, error) {
	file, err := os.Open(*p)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	decoder := json.NewDecoder(file)
	var sbom SyftSbom

	if err := decoder.Decode(&sbom); err != nil {
		return nil, err
	}

	if sbom.Artifacts == nil ||
		sbom.ArtifactRelationships == nil {
		return nil, fmt.Errorf("incomplete syft sbom")
	}

	return &sbom, nil
}

func (s *SyftSbom) Transform() (*CyclonedxSbom, error) {

	// transform syft to cyclonedx
	parentChild := make(map[string][]Target, len(s.Artifacts))
	for _, r := range s.ArtifactRelationships {
		child := Target{
			Child: r.Child,
			Type:  r.Type,
		}

		// Append directly to the map
		parentChild[r.Parent] = append(parentChild[r.Parent], child)
	}

	var dependencies []Dependency

	for key, value := range parentChild {
		dependencies = append(dependencies, Dependency{
			Ref:       key,
			DependsOn: value,
		})
	}

	return &CyclonedxSbom{
		Components:   s.Artifacts,
		Dependencies: dependencies,
		Source:       s.Source,
		Distro:       s.Distro,
	}, nil

}
