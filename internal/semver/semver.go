package semver

import (
	"fmt"
	"slices"
	"sort"

	"github.com/hashicorp/go-version"
)

type VersionDistance struct {
	MissedReleases int64
	MissedMajor    int64
	MissedMinor    int64
	MissedPatch    int64
}

func GetVersionDistance(usedVersion string, versions []string) (*VersionDistance, error) {

	usedSemver, err := version.NewVersion(usedVersion)
	if err != nil {
		return nil, err
	}

	var semVers []*version.Version
	for _, v := range versions {
		semVer, err := version.NewVersion(v)
		if err != nil {
			fmt.Printf("can't parse %s to semver\n", v)
			continue
		}
		semVers = append(semVers, semVer)
	}

	slices.SortFunc(semVers, func(a *version.Version, b *version.Version) int {
		if a == nil || b == nil {
			return 0
		}
		return a.Compare(b)
	})

	i := sort.Search(len(semVers),
		func(i int) bool { return semVers[i].GreaterThanOrEqual(usedSemver) })

	if i <= len(semVers) && !semVers[i].Equal(usedSemver) {
		// x is not present in data,
		// but i is the index where it would be inserted.
		semVers = slices.Insert(semVers, i, usedSemver)
	}

	largestVersion := semVers[len(semVers)-1]

	// semVers[i] == usedSemver
	missedReleases := (len(semVers) - 1) - i

	return &VersionDistance{
		MissedReleases: int64(missedReleases),
		MissedMajor:    largestVersion.Segments64()[0] - usedSemver.Segments64()[0],
		MissedMinor:    largestVersion.Segments64()[1] - usedSemver.Segments64()[1],
		MissedPatch:    largestVersion.Segments64()[2] - usedSemver.Segments64()[2],
	}, nil
}
