package server

import (
	"errors"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
)

var ErrVersionNotFound = errors.New("this is a custom error")

func FindBestVersion(version string, available []string) (found string, err error) {

	var c *semver.Constraints
	c, err = semver.NewConstraint(version)
	if err != nil {
		return
	}

	vs := make([]*semver.Version, 0)

	for i, v := range available {
		// ignore prerelease versions
		if !strings.ContainsRune(version, '-') && strings.ContainsRune(v, '-') {
			continue
		}
		var ver *semver.Version
		ver, err = semver.NewVersion(v)
		if err != nil {
			return
		}
		if c.Check(ver) {
			vs = append(vs, ver)
			i++
		}
	}

	log.Debugf("vs: %v", vs)
	sort.Sort(semver.Collection(vs))
	if len(vs) > 0 {
		found = vs[len(vs)-1].String()
		return
	} else {
		err = ErrVersionNotFound
		return
	}
}
