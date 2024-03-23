package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/esm-dev/esm.sh/server/config"
	"github.com/esm-dev/esm.sh/server/storage"
	"github.com/google/go-cmp/cmp"
	"github.com/ije/gox/utils"
)

func jsonString(d NpmPackageInfo) string {
	one, _ := json.MarshalIndent(d, "", "  ")
	return string(one)
}

// TestGithubPackage tests the fetchPackageInfo function
// This will fallback to dling the tarball.
func TestGithubPackage(t *testing.T) {
	cfg = config.Default()
	// Make sure we use the new fetch
	cfg.UseNewFetch = true
	cache, _ = storage.OpenCache(cfg.Cache)

	cfg.NpmRegistryScope = "@pips-framework"
	cfg.NpmRegistry = "https://npm.pkg.github.com/"
	cfg.NpmToken = os.Getenv("GH_TOKEN")

	info, err := fetchPackageInfo("@pips-framework/pips-mc", "")

	if err != nil {
		t.Errorf(err.Error())
	}

	if info.Name != "@pips-framework/pips-mc" {
		t.Errorf("error!!!!!!!")
	}
}

func TestFetchPackageJsonFromTarball(t *testing.T) {

	cfg = config.Default()
	cache, _ = storage.OpenCache(cfg.Cache)
	cfg.NpmRegistry = "https://registry.npmjs.org/"

	testCases := []NameVersion{
		NewNameVersion("smallest", ""),
	}

	for _, tc := range testCases {
		t.Run(tc.String(), func(t *testing.T) {
			pkgBytes, err := fetchPackageJsonFromTarball(tc)

			if err != nil {
				t.Errorf(err.Error())
			}
			// one, _ := json.MarshalIndent(info, "", "  ")

			info := NpmPackageInfo{}
			err = json.NewDecoder(bytes.NewReader(pkgBytes)).Decode(&info)

			if err != nil {
				t.Errorf("error!!!!!!!")
			}
			t.Log(info)
		})
	}
}

// The json encoding is breaking here ...
func TestCacheNpmPackageInfo(t *testing.T) {

	n := NpmPackageInfo{
		SideEffectsFalse: true,
	}

	encoded := utils.MustEncodeJSON(n)

	newPkg := NpmPackageInfo{}
	json.Unmarshal(encoded, &newPkg)

	if diff := cmp.Diff(jsonString(n), jsonString(newPkg)); diff != "" {
		t.Errorf("MakeGatewayInfo() mismatch (-want +got):\n%s", diff)
	}
}

// TestNewFetchPkgInfo tests the newFetchPackageInfo function make sure all still works
// Once happy -  move to using snapshot testing
func TestNewFetchPkgInfo(t *testing.T) {

	cfg = config.Default()
	cache, _ = storage.OpenCache(cfg.Cache)

	cfg.NpmRegistry = "https://registry.npmjs.org/" //npm.pkg.github.com/"

	testCases := []struct {
		name    string
		version string
	}{
		{"smallest", ""},
		{"smallest", "1.0.1"},
		{"smallest", "~1.0.1"},
		{"memoize", ""},
		{"memoize", "10.0.0"},
		{"conf", ""},
		{"conf", "12.0.0"},
		{"conf", "12.x"},
		{"react", ""},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s@%s", tc.name, tc.version), func(t *testing.T) {
			old, _ := fetchPackageInfo(tc.name, tc.version)
			ni, _ := newFetchPackageInfo(tc.name, tc.version)
			if diff := cmp.Diff(jsonString(old), jsonString(ni)); diff != "" {
				t.Errorf("MakeGatewayInfo() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
