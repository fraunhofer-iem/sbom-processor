package semver

import (
	"fmt"
	"testing"
)

func TestVersionDistanceValidUsedVersionNotContained(t *testing.T) {
	usedVersion := "1.0.0"
	versions := []string{"1.0.1", "0.1.0", "0.2.0", "2.0.3", "1.2.0", "1.2.3"}

	d, err := GetVersionDistance(usedVersion, versions)
	if err != nil {
		t.Fatalf("no error expected")
	}

	if d.MissedReleases != 4 {
		t.Fatalf("unexpected number of missed releases. Expected 4, got %d", d.MissedReleases)
	}

	if d.MissedMajor != 1 {
		t.Fatalf("unexpected number of missed major releases. Expected 1, got %d", d.MissedMajor)
	}
	if d.MissedMinor != 0 {
		t.Fatalf("unexpected number of missed minor releases. Expected 0, got %d", d.MissedMinor)
	}

	if d.MissedPatch != 3 {
		t.Fatalf("unexpected number of missed patch releases. Expected 3, got %d", d.MissedPatch)
	}
}

func TestVersionDistanceValidUsedVersion(t *testing.T) {
	usedVersion := "1.0.0"
	versions := []string{"0.1.0", "1.0.0", "0.2.0", "1.0.1", "1.2.0", "1.2.3", "2.0.3"}

	d, err := GetVersionDistance(usedVersion, versions)
	if err != nil {
		t.Fatalf("no error expected")
	}

	if d.MissedReleases != 4 {
		t.Fatalf("unexpected number of missed releases. Expected 4, got %d", d.MissedReleases)
	}

	if d.MissedMajor != 1 {
		t.Fatalf("unexpected number of missed major releases. Expected 1, got %d", d.MissedMajor)
	}
	if d.MissedMinor != 0 {
		t.Fatalf("unexpected number of missed minor releases. Expected 0, got %d", d.MissedMinor)
	}

	if d.MissedPatch != 3 {
		t.Fatalf("unexpected number of missed patch releases. Expected 3, got %d", d.MissedPatch)
	}
}

func TestVersionDistanceWeiredVersions(t *testing.T) {
	usedVersion := "1.0.0"
	// 1.6.5+git20160407+5e5d3-1 <- fails with NewVersion
	// 2.5.1.ds1-4 <- fails with NewVersion and NewSemver
	// 4.6.0+git+20171230-2 <- fails with NewVersion
	// 4.2+dfsg-0.1+deb7u4 <- fails with NewVersion

	versions := []string{"0.1.4.bo", "1.0.0+123", "v0.2.0", "1.0.1-meta.bla", "1.2.0.sha253.214.dsf", "1.2.3", "2.0.3"}

	d, err := GetVersionDistance(usedVersion, versions)
	if err != nil {
		t.Fatalf("no error expected")
	}

	if d.MissedReleases != 4 {
		t.Fatalf("unexpected number of missed releases. Expected 4, got %d", d.MissedReleases)
	}

	if d.MissedMajor != 1 {
		t.Fatalf("unexpected number of missed major releases. Expected 1, got %d", d.MissedMajor)
	}
	if d.MissedMinor != 0 {
		t.Fatalf("unexpected number of missed minor releases. Expected 0, got %d", d.MissedMinor)
	}

	if d.MissedPatch != 3 {
		t.Fatalf("unexpected number of missed patch releases. Expected 3, got %d", d.MissedPatch)
	}
}

func TestVersionParsing(t *testing.T) {
	// versions from our test set
	// 1.6.5+git20160407+5e5d3-1 <- fails with NewVersion
	// 2.5.1.ds1-4 <- fails with NewVersion and NewSemver
	// 4.6.0+git+20171230-2 <- fails with NewVersion
	// 4.2+dfsg-0.1+deb7u4 <- fails with NewVersion
	versions := []string{"0.1.4.bo", "2.5.1.ds1-4", "v0.2.0", "1.0.1-meta.bla", "1.2.0.sha253.214.dsf", "4.6.0+git+20171230-2", "4.2+dfsg-0.1+deb7u4"}

	for _, v := range versions {

		v, err := newRelaxedSemver(v)
		if err != nil {
			t.Fatalf("Parsing failed %s\n", err)
		}

		fmt.Printf("%+v\n", v)

	}

}
