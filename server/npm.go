package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/esm-dev/esm.sh/server/storage"
	"github.com/esm-dev/esm.sh/server/tgz"

	"github.com/ije/gox/utils"
	"github.com/ije/gox/valid"
)

// ref https://github.com/npm/validate-npm-package-name
var npmNaming = valid.Validator{valid.FromTo{'a', 'z'}, valid.FromTo{'A', 'Z'}, valid.FromTo{'0', '9'}, valid.Eq('.'), valid.Eq('-'), valid.Eq('_')}

type NameVersion struct {
	Name    string
	Version string
}

func (nv NameVersion) String() string {
	return nv.Name + "@" + nv.Version
}

func (nv NameVersion) IsFullVersion() bool {
	return regexpFullVersion.MatchString(nv.Version)
}

func NewNameVersion(name string, version string) NameVersion {

	a := strings.Split(strings.Trim(name, "/"), "/")
	name = a[0]
	if strings.HasPrefix(name, "@") && len(a) > 1 {
		name = a[0] + "/" + a[1]
	}

	if strings.HasPrefix(version, "=") || strings.HasPrefix(version, "v") {
		version = version[1:]
	}
	if version == "" {
		version = "latest"
	}

	return NameVersion{name, version}
}

// NpmPackageVerions defines versions of a NPM package
type NpmPackageVerions struct {
	DistTags map[string]string         `json:"dist-tags"`
	Versions map[string]NpmPackageInfo `json:"versions"`
}

type Dist struct {
	Shasum  string `json:"shasum"`
	Tarball string `json:"tarball"`
}

type PkgVersion struct {
	Dist Dist `json:"dist"`
}

type Packument struct {
	Name     string                `json:"name"`
	DistTags map[string]string     `json:"dist-tags"`
	Versions map[string]PkgVersion `json:"versions"`
}

// NpmPackageJSON defines the package.json of NPM
type NpmPackageJSON struct {
	Name             string                 `json:"name"`
	Version          string                 `json:"version"`
	Type             string                 `json:"type,omitempty"`
	Main             string                 `json:"main,omitempty"`
	Browser          StringOrMap            `json:"browser,omitempty"`
	Module           StringOrMap            `json:"module,omitempty"`
	ES2015           StringOrMap            `json:"es2015,omitempty"`
	JsNextMain       string                 `json:"jsnext:main,omitempty"`
	Types            string                 `json:"types,omitempty"`
	Typings          string                 `json:"typings,omitempty"`
	SideEffects      interface{}            `json:"sideEffects,omitempty"`
	Dependencies     map[string]string      `json:"dependencies,omitempty"`
	PeerDependencies map[string]string      `json:"peerDependencies,omitempty"`
	Imports          map[string]interface{} `json:"imports,omitempty"`
	TypesVersions    map[string]interface{} `json:"typesVersions,omitempty"`
	PkgExports       json.RawMessage        `json:"exports,omitempty"`
	Deprecated       interface{}            `json:"deprecated,omitempty"`
	ESMConfig        interface{}            `json:"esm.sh,omitempty"`
}

func (a *NpmPackageJSON) ToNpmPackage() *NpmPackageInfo {
	browser := map[string]string{}
	if a.Browser.Str != "" {
		browser["."] = a.Browser.Str
	}
	if a.Browser.Map != nil {
		for k, v := range a.Browser.Map {
			s, isStr := v.(string)
			if isStr {
				browser[k] = s
			} else {
				b, ok := v.(bool)
				if ok && !b {
					browser[k] = ""
				}
			}
		}
	}
	deprecated := ""
	if a.Deprecated != nil {
		if s, ok := a.Deprecated.(string); ok {
			deprecated = s
		}
	}
	esmConfig := map[string]interface{}{}
	if a.ESMConfig != nil {
		if v, ok := a.ESMConfig.(map[string]interface{}); ok {
			esmConfig = v
		}
	}
	var sideEffects *stringSet = nil
	sideEffectsFalse := false
	if a.SideEffects != nil {
		if s, ok := a.SideEffects.(string); ok {
			sideEffectsFalse = s == "false"
		} else if b, ok := a.SideEffects.(bool); ok {
			sideEffectsFalse = !b
		} else if m, ok := a.SideEffects.([]interface{}); ok && len(m) > 0 {
			sideEffects = newStringSet()
			for _, v := range m {
				if name, ok := v.(string); ok {
					sideEffects.Add(name)
				}
			}
		}
	}
	var pkgExports interface{} = nil
	if rawExports := a.PkgExports; rawExports != nil {
		var v interface{}
		if json.Unmarshal(rawExports, &v) == nil {
			if s, ok := v.(string); ok {
				if len(s) > 0 {
					pkgExports = s
				}
			} else if _, ok := v.(map[string]interface{}); ok {
				om := newOrderedMap()
				if om.UnmarshalJSON(rawExports) == nil {
					pkgExports = om
				} else {
					pkgExports = v
				}
			}
		}
	}
	return &NpmPackageInfo{
		Name:             a.Name,
		Version:          a.Version,
		Type:             a.Type,
		Main:             a.Main,
		Module:           a.Module.MainValue(),
		ES2015:           a.ES2015.MainValue(),
		JsNextMain:       a.JsNextMain,
		Types:            a.Types,
		Typings:          a.Typings,
		Browser:          browser,
		SideEffectsFalse: sideEffectsFalse,
		SideEffects:      sideEffects,
		Dependencies:     a.Dependencies,
		PeerDependencies: a.PeerDependencies,
		Imports:          a.Imports,
		TypesVersions:    a.TypesVersions,
		PkgExports:       pkgExports,
		Deprecated:       deprecated,
		ESMConfig:        esmConfig,
	}
}

// NpmPackage defines the package.json
type NpmPackageInfo struct {
	Name             string
	Version          string
	Type             string
	Main             string
	Module           string
	ES2015           string
	JsNextMain       string
	Types            string
	Typings          string
	SideEffectsFalse bool
	SideEffects      *stringSet
	Browser          map[string]string
	Dependencies     map[string]string
	PeerDependencies map[string]string
	Imports          map[string]interface{}
	TypesVersions    map[string]interface{}
	PkgExports       interface{}
	Deprecated       string
	ESMConfig        map[string]interface{}
}

func (a *NpmPackageInfo) UnmarshalJSON(b []byte) error {
	var n NpmPackageJSON
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*a = *n.ToNpmPackage()
	return nil
}

func getPackageInfo(wd string, name string, version string) (info NpmPackageInfo, fromPackageJSON bool, err error) {
	if name == "@types/node" {
		info = NpmPackageInfo{
			Name:    "@types/node",
			Version: nodeTypesVersion,
			Types:   "index.d.ts",
		}
		return
	}

	if wd != "" {
		pkgJsonPath := path.Join(wd, "node_modules", name, "package.json")
		if fileExists(pkgJsonPath) && utils.ParseJSONFile(pkgJsonPath, &info) == nil {
			info, err = fixPkgVersion(info)
			fromPackageJSON = true
			return
		}
	}

	info, err = fetchPackageInfo(name, version)
	if err == nil {
		info, err = fixPkgVersion(info)
	}
	return
}

func fetchPackument(name string) (packument Packument, err error) {

	cacheKey := fmt.Sprintf("npm:packument:%s", name)

	log.Debugf("fetchPackument: %s, cacheKey: %s", name, cacheKey)

	lock := getFetchLock(cacheKey)
	lock.Lock()

	defer lock.Unlock()

	// check cache firstly
	if cache != nil {
		var data []byte
		data, err = cache.Get(cacheKey)
		if err == nil && json.Unmarshal(data, &packument) == nil {
			return
		}
		if err != nil && err != storage.ErrNotFound && err != storage.ErrExpired {
			log.Error("cache:", err)
		}
	}

	url := createUrl(name, "")

	req, err := createFetchRequest(url)

	if err != nil {
		return
	}

	resp, err := httpClient.Do(req)

	if err != nil {
		return
	}

	defer resp.Body.Close()

	log.Debugf("response status code: %d", resp.StatusCode)

	if resp.StatusCode == 404 || resp.StatusCode == 401 {
		ret, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("npm: %v  '%s' (%s: %s)", resp.StatusCode, name, resp.Status, string(ret))
		return
	}

	if resp.StatusCode != 200 {
		ret, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("npm: could not get metadata of package '%s' (%s: %s)", name, resp.Status, string(ret))
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&packument)

	if err != nil {
		return
	}
	log.Debugf("cache.Set .. cacheKey: %s, packument: %s", cacheKey, packument)

	cache.Set(cacheKey, utils.MustEncodeJSON(packument), 2*time.Hour)
	return
}

func createUrl(name string, version string) string {

	log.Debugf("[createUrl]: %s, %s, registry:", name, version)
	log.Debugf("[createUrl]: registry: %s, scope: %s", cfg.NpmRegistry, cfg.NpmRegistryScope)

	url := cfg.NpmRegistry + name

	if cfg.NpmRegistryScope != "" {
		isInScope := strings.HasPrefix(name, cfg.NpmRegistryScope)

		log.Debugf("[createUrl] isInScope: %t, name: %s, scope: %s", isInScope, name, cfg.NpmRegistryScope)
		if !isInScope {
			url = "https://registry.npmjs.org/" + name
		}
	} else {
		log.Debug("no npm registry scope")
	}

	if version != "" {
		url += "/" + version
	}
	log.Debugf("[createUrl] pkg: %s --> url: %s", name, url)
	return url
}

func createFetchRequest(url string) (req *http.Request, err error) {

	/**
	 * Note: using the exact version doesnt work with
	 * github's pkg registry. Instead you need to call the package
	 * then pull the data out of that.
	 * This would also work w/ npmjs?
	 */
	req, err = http.NewRequest("GET", url, nil)

	if err != nil {
		return
	}

	if cfg.NpmToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.NpmToken)
	} else {
		log.Debug("no npm token")
	}

	if cfg.NpmUser != "" && cfg.NpmPassword != "" {
		req.SetBasicAuth(cfg.NpmUser, cfg.NpmPassword)
	} else {
		log.Debug("no npm user or password")
	}

	return
}
func fetchPackageInfo(name string, version string) (info NpmPackageInfo, err error) {

	if cfg.UseNewFetch {
		return newFetchPackageInfo(name, version)
	} else {
		return oldFetchPackageInfo(name, version)
	}
}

// This is the original function
func oldFetchPackageInfo(name string, version string) (info NpmPackageInfo, err error) {
	a := strings.Split(strings.Trim(name, "/"), "/")
	name = a[0]
	if strings.HasPrefix(name, "@") && len(a) > 1 {
		name = a[0] + "/" + a[1]
	}

	if strings.HasPrefix(version, "=") || strings.HasPrefix(version, "v") {
		version = version[1:]
	}
	if version == "" {
		version = "latest"
	}
	isFullVersion := regexpFullVersion.MatchString(version)

	cacheKey := fmt.Sprintf("npm:%s@%s", name, version)
	lock := getFetchLock(cacheKey)
	lock.Lock()
	defer lock.Unlock()

	// check cache firstly
	if cache != nil {
		var data []byte
		data, err = cache.Get(cacheKey)
		if err == nil && json.Unmarshal(data, &info) == nil {
			log.Infof("[old] cache hit: %s", cacheKey)
			return
		}
		if err != nil && err != storage.ErrNotFound && err != storage.ErrExpired {
			log.Error("cache:", err)
		}
	}

	start := time.Now()
	defer func() {
		if err == nil {
			log.Debugf("lookup package(%s@%s) in %v", name, info.Version, time.Since(start))
		}
	}()

	url := cfg.NpmRegistry + name
	if cfg.NpmRegistryScope != "" {
		isInScope := strings.HasPrefix(name, cfg.NpmRegistryScope)
		if !isInScope {
			url = "https://registry.npmjs.org/" + name
		}
	}

	if isFullVersion {
		url += "/" + version
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	if cfg.NpmToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.NpmToken)
	}
	if cfg.NpmUser != "" && cfg.NpmPassword != "" {
		req.SetBasicAuth(cfg.NpmUser, cfg.NpmPassword)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 || resp.StatusCode == 401 {
		if isFullVersion {
			err = fmt.Errorf("npm: version %s of '%s' not found", version, name)
		} else {
			err = fmt.Errorf("npm: package '%s' not found", name)
		}
		return
	}

	if resp.StatusCode != 200 {
		ret, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("npm: could not get metadata of package '%s' (%s: %s)", name, resp.Status, string(ret))
		return
	}

	if isFullVersion {
		err = json.NewDecoder(resp.Body).Decode(&info)
		if err != nil {
			return
		}
		if cache != nil {
			cache.Set(cacheKey, utils.MustEncodeJSON(info), 24*time.Hour)
		}
		return
	}

	var h NpmPackageVerions
	err = json.NewDecoder(resp.Body).Decode(&h)
	if err != nil {
		return
	}

	if len(h.Versions) == 0 {
		err = fmt.Errorf("npm: missing `versions` field")
		return
	}

	distVersion, ok := h.DistTags[version]
	if ok {
		info = h.Versions[distVersion]
	} else {
		var c *semver.Constraints
		c, err = semver.NewConstraint(version)
		if err != nil && version != "latest" {
			return fetchPackageInfo(name, "latest")
		}
		vs := make([]*semver.Version, len(h.Versions))
		i := 0
		for v := range h.Versions {
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
				vs[i] = ver
				i++
			}
		}
		if i > 0 {
			vs = vs[:i]
			if i > 1 {
				sort.Sort(semver.Collection(vs))
			}
			info = h.Versions[vs[i-1].String()]
		}
	}

	if info.Version == "" {
		err = fmt.Errorf("npm: version %s of '%s' not found", version, name)
		return
	}

	// cache package info for 10 minutes
	if cache != nil {
		cache.Set(cacheKey, utils.MustEncodeJSON(info), 10*time.Minute)
	}
	return
}

func fetchFullPackageInfo(nv NameVersion) (info NpmPackageInfo, err error) {

	cacheKey := fmt.Sprintf("npm-fp:%s", nv.String())

	log.Debugf("fetchFullPackageInfo: %s, cacheKey: %s", nv.String(), cacheKey)
	lock := getFetchLock(cacheKey)
	lock.Lock()
	defer lock.Unlock()

	// check cache firstly
	if cache != nil {
		var data []byte
		data, err = cache.Get(cacheKey)
		if err == nil && json.Unmarshal(data, &info) == nil {
			log.Infof("cache hit: %s", cacheKey)
			return
		}
		if err != nil && err != storage.ErrNotFound && err != storage.ErrExpired {
			log.Error("cache:", err)
		}
	}

	start := time.Now()
	defer func() {
		if err == nil {
			log.Debugf("lookup package(%s@%s) in %v", nv.String(), info.Version, time.Since(start))
		}
	}()

	pkgBytes, err := fetchPackageJson(nv)

	if err != nil {
		log.Debugf("fetchPackageJson error: %s", err)
		log.Infof("fetchPackageJson going to try and download the tarball instead")

		tarballBytes, tarballErr := fetchPackageJsonFromTarball(nv)

		if tarballErr != nil {
			log.Debug("fetchPackageJsonFromTarball error: ", tarballErr)
			err = tarballErr
			return
		} else {
			err = nil
		}

		pkgBytes = tarballBytes
	}

	if err != nil {
		return
	}

	err = json.NewDecoder(bytes.NewReader(pkgBytes)).Decode(&info)

	if err != nil {
		return
	}

	if cache != nil {

		// Note (Important!): we store the raw json data in the cache
		// Not the NpmPackageInfo struct because it has derived properties
		// TODO: we could look at making the derived fields getters instead?
		cache.Set(cacheKey, pkgBytes, 24*time.Hour)
	}
	return
}
func downloadFile(filepath string, req *http.Request) (err error) {

	if _, err = os.Stat(filepath); err == nil {
		return
	}

	os.MkdirAll(path.Dir(filepath), os.ModePerm)
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)

	if err != nil {
		return err
	}

	return nil
}

func fetchPackageJsonFromTarball(nv NameVersion) (bytes []byte, err error) {
	// err = fmt.Errorf("todo-----")

	packument, err := fetchPackument(nv.Name)

	if err != nil {
		return
	}

	nv, err = findBestVersionInPackument(nv, packument)

	if err != nil {
		return
	}

	log.Debugf("fetchPackageJsonFromTarball: %s", nv.String())

	tarball := packument.Versions[nv.Version].Dist.Tarball

	req, err := createFetchRequest(tarball)

	if err != nil {
		return
	}

	filepath := path.Join(cfg.WorkDir, "tarballs", FilenameFromUrl(tarball))
	// resp, err := httpClient.Do(req)
	err = downloadFile(filepath, req)

	if err != nil {
		return
	}

	dir, err := tgz.Extract(filepath)

	if err != nil {
		return
	}

	// infos, err := ioutil.ReadDir(dir)

	bytes, err = os.ReadFile(path.Join(dir, "package", "package.json"))

	if err != nil {
		return
	}

	// remove the file once we're done
	defer os.Remove(filepath)
	defer os.RemoveAll(dir)

	// req, err := http.NewRequest("GET", tarball, nil)
	log.Debugf("fetchPackageJsonFromTarball: %s", tarball)

	return
}

func fetchPackageJson(nv NameVersion) (bytes []byte, err error) {

	url := createUrl(nv.Name, nv.Version)

	req, err := createFetchRequest(url)

	log.Debugf("pkg: %s scope %s", nv.String(), url)

	resp, err := httpClient.Do(req)

	if err != nil {
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == 404 || resp.StatusCode == 401 {
		err = fmt.Errorf("npm: version %s of '%s' not found", nv.Version, nv.Name)
		return
	}

	log.Debugf("response status code: %d", resp.StatusCode)

	if resp.StatusCode != 200 {
		ret, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("npm: could not get metadata of package '%s' (%s: %s)", nv.String(), resp.Status, string(ret))
		return
	}

	bytes, err = io.ReadAll(resp.Body)

	return
}

func findBestVersionInPackument(nv NameVersion, packument Packument) (out NameVersion, err error) {

	log.Debugf("findBestVersionInPackument: %s", nv.String())
	distVersion, ok := packument.DistTags[nv.Version]
	if ok {

		out = NameVersion{nv.Name, distVersion}
		log.Debugf("findBestVersionInPackument: out: %s", out.String())
	} else {

		log.Debugf("findBestVersionInPackument: distVersion not found: %s, doing a semver match", nv.Version)
		// find the best version from what's available to match the target version
		bestVersion, versionErr := FindBestVersion(nv.Version, KeysFromMap(packument.Versions))

		if versionErr != nil {
			// TODO: replace this?
			err = versionErr
			return
		}
		out = NameVersion{nv.Name, bestVersion}
	}
	log.Debugf("findBestVersionInPackument: out: %s", out.String())
	return
}

func newFetchPackageInfo(name string, version string) (info NpmPackageInfo, err error) {

	nv := NewNameVersion(name, version)

	if nv.IsFullVersion() {
		return fetchFullPackageInfo(nv)
	} else {
		// We need to look up the best version to load
		packument, pe := fetchPackument(nv.Name)

		if pe != nil {
			err = pe
			return
		}

		nv, err = findBestVersionInPackument(nv, packument)

		if err != nil {
			return
		}

		return fetchFullPackageInfo(nv)

		// eg latest => 1.2.3
		// distVersion, ok := packument.DistTags[nv.Version]
		// if ok {
		// 	return fetchFullPackageInfo(NameVersion{nv.Name, distVersion})
		// } else {

		// 	// find the best version from what's available to match the target version
		// 	bestVersion, versionErr := FindBestVersion(nv.Version, KeysFromMap(packument.Versions))

		// 	if versionErr != nil {
		// 		// TODO: replace this?
		// 		err = versionErr
		// 		return
		// 	}
		// }
	}
}

func installPackage(wd string, pkg Pkg) (err error) {
	pkgVersionName := pkg.VersionName()

	// only one install process allowed at the same time
	lock := getInstallLock(pkgVersionName)
	lock.Lock()
	defer lock.Unlock()

	// ensure package.json file to prevent read up-levels
	packageFilePath := path.Join(wd, "package.json")
	if pkg.FromEsmsh {
		err = copyRawBuildFile(pkg.Name, "package.json", wd)
	} else if pkg.FromGithub || !fileExists(packageFilePath) {
		fileContent := []byte("{}")
		if pkg.FromGithub {
			fileContent = []byte(fmt.Sprintf(
				`{"dependencies": {"%s": "%s"}}`,
				pkg.Name,
				fmt.Sprintf("git+https://github.com/%s.git#%s", pkg.Name, pkg.Version),
			))
		}
		ensureDir(wd)
		err = os.WriteFile(packageFilePath, fileContent, 0644)
	}
	if err != nil {
		return fmt.Errorf("ensure package.json failed: %s", pkgVersionName)
	}

	for i := 0; i < 3; i++ {
		if pkg.FromEsmsh {
			err = pnpmInstall(wd)
			if err == nil {
				installDir := path.Join(wd, "node_modules", pkg.Name)
				for _, name := range []string{"package.json", "index.mjs", "index.d.ts"} {
					err = copyRawBuildFile(pkg.Name, name, installDir)
					if err != nil {
						break
					}
				}
			}
		} else if pkg.FromGithub {
			err = pnpmInstall(wd)
			// pnpm will ignore github package which has been installed without `package.json` file
			if err == nil && !dirExists(path.Join(wd, "node_modules", pkg.Name)) {
				err = ghInstall(wd, pkg.Name, pkg.Version)
			}
		} else if regexpFullVersion.MatchString(pkg.Version) {
			err = pnpmInstall(wd, pkgVersionName, "--prefer-offline")
		} else {
			err = pnpmInstall(wd, pkgVersionName)
		}
		packageFilePath := path.Join(wd, "node_modules", pkg.Name, "package.json")
		if err == nil && !fileExists(packageFilePath) {
			if pkg.FromGithub {
				ensureDir(path.Dir(packageFilePath))
				err = os.WriteFile(packageFilePath, utils.MustEncodeJSON(pkg), 0644)
			} else {
				err = fmt.Errorf("pnpm install %s: package.json not found", pkg)
			}
		}
		if err == nil {
			break
		}
		if i < 2 {
			time.Sleep(100 * time.Millisecond)
		}
	}
	return
}

func pnpmInstall(wd string, packages ...string) (err error) {
	var args []string
	if len(packages) > 0 {
		args = append([]string{"add"}, packages...)
	} else {
		args = []string{"install"}
	}
	args = append(
		args,
		"--ignore-scripts",
		"--loglevel", "error",
	)
	start := time.Now()
	cmd := exec.Command("pnpm", args...)
	cmd.Dir = wd
	if cfg.NpmToken != "" {
		cmd.Env = append(os.Environ(), "ESM_NPM_TOKEN="+cfg.NpmToken)
	}
	if cfg.NpmUser != "" && cfg.NpmPassword != "" {
		data := []byte(cfg.NpmPassword)
		password := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
		base64.StdEncoding.Encode(password, data)
		cmd.Env = append(
			os.Environ(),
			"ESM_NPM_USER="+cfg.NpmUser,
			"ESM_NPM_PASSWORD="+string(password),
		)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pnpm add %s: %s", strings.Join(packages, ","), string(output))
	}
	if len(packages) > 0 {
		log.Debug("pnpm add", strings.Join(packages, ","), "in", time.Since(start))
	} else {
		log.Debug("pnpm install in", time.Since(start))
	}
	return
}

// ref https://github.com/npm/validate-npm-package-name
func validatePackageName(name string) bool {
	scope := ""
	nameWithoutScope := name
	if strings.HasPrefix(name, "@") {
		scope, nameWithoutScope = utils.SplitByFirstByte(name, '/')
		scope = scope[1:]
	}
	if (scope != "" && !npmNaming.Is(scope)) || (nameWithoutScope == "" || !npmNaming.Is(nameWithoutScope)) || len(name) > 214 {
		return false
	}
	return true
}

// added by @jimisaacs
func toTypesPackageName(pkgName string) string {
	if strings.HasPrefix(pkgName, "@") {
		pkgName = strings.Replace(pkgName[1:], "/", "__", 1)
	}
	return "@types/" + pkgName
}

func fixPkgVersion(info NpmPackageInfo) (NpmPackageInfo, error) {
	for prefix, ver := range fixedPkgVersions {
		if strings.HasPrefix(info.Name+"@"+info.Version, prefix) {
			return fetchPackageInfo(info.Name, ver)
		}
	}
	return info, nil
}

func isTypesOnlyPackage(p NpmPackageInfo) bool {
	return p.Main == "" && p.Module == "" && p.Types != ""
}

func getInstallLock(key string) *sync.Mutex {
	v, _ := installLocks.LoadOrStore(key, &sync.Mutex{})
	return v.(*sync.Mutex)
}

func getFetchLock(key string) *sync.Mutex {
	v, _ := fetchLocks.LoadOrStore(key, &sync.Mutex{})
	return v.(*sync.Mutex)
}
