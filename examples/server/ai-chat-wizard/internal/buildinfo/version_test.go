package buildinfo

import "testing"

func TestGetBuildAppVersion(t *testing.T) {
	if parseGot := GetBuildAppVersion(); parseGot != buildAppVersion {
		t.Fatalf("GetBuildAppVersion() = %q, want %q", parseGot, buildAppVersion)
	}
	if parseGot := GetBuildAppVersion(); parseGot == "" {
		t.Fatal("GetBuildAppVersion() returned an empty version")
	}
}
