/**
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package pkg

import (
	"bytes"
	"fmt"
	"mynewt.apache.org/newt/newt/cfgv"
	"os"
	"path/filepath"
	"strings"

	"mynewt.apache.org/newt/newt/config"
	"mynewt.apache.org/newt/newt/interfaces"
	"mynewt.apache.org/newt/newt/newtutil"
	"mynewt.apache.org/newt/newt/repo"
	"mynewt.apache.org/newt/newt/ycfg"
	"mynewt.apache.org/newt/util"
	"mynewt.apache.org/newt/yaml"
)

var PackageHashIgnoreDirs = map[string]bool{
	"obj": true,
	"bin": true,
	".":   true,
}

var LocalPackageSpecialNames = map[string]bool{
	"src":     true,
	"include": true,
	"bin":     true,
}

type LocalPackage struct {
	repo        *repo.Repo
	name        string
	basePath    string
	packageType interfaces.PackageType
	subPriority int
	linkedName  string

	// General information about the package
	desc *PackageDesc

	// Extra package-specific settings that don't come from syscfg.  For
	// example, SELFTEST gets set when the newt test command is used.
	injectedSettings *cfgv.Settings

	// Settings read from pkg.yml.
	PkgY ycfg.YCfg

	// Settings read from syscfg.yml.
	SyscfgY ycfg.YCfg

	// Names of all source yml files; used to determine if rebuild required.
	cfgFilenames []string
}

func NewLocalPackage(r *repo.Repo, pkgDir string) *LocalPackage {
	pkg := &LocalPackage{
		desc:             &PackageDesc{},
		repo:             r,
		basePath:         filepath.ToSlash(filepath.Clean(pkgDir)),
		injectedSettings: cfgv.NewSettings(nil),
	}

	pkg.PkgY = ycfg.NewYCfg(pkg.PkgYamlPath())
	pkg.SyscfgY = ycfg.NewYCfg(pkg.SyscfgYamlPath())
	return pkg
}

func (pkg *LocalPackage) Name() string {
	return pkg.name
}

func (pkg *LocalPackage) NameWithRepo() string {
	r := pkg.Repo()
	return newtutil.BuildPackageString(r.Name(), pkg.Name())
}

func (pkg *LocalPackage) FullName() string {
	r := pkg.Repo()
	if r.IsLocal() {
		return pkg.Name()
	} else {
		return pkg.NameWithRepo()
	}
}

func (pkg *LocalPackage) LinkedName() string {
	return pkg.linkedName
}

func (pkg *LocalPackage) BasePath() string {
	return pkg.basePath
}

func (pkg *LocalPackage) PkgConfig() *ycfg.YCfg {
	return &pkg.PkgY
}

func (pkg *LocalPackage) RelativePath() string {
	proj := interfaces.GetProject()
	return strings.TrimPrefix(pkg.BasePath(), proj.Path())
}

func (pkg *LocalPackage) PkgYamlPath() string {
	return fmt.Sprintf("%s/%s", pkg.BasePath(), PACKAGE_FILE_NAME)
}

func (pkg *LocalPackage) SyscfgYamlPath() string {
	return fmt.Sprintf("%s/%s", pkg.BasePath(), SYSCFG_YAML_FILENAME)
}

func (pkg *LocalPackage) Type() interfaces.PackageType {
	return pkg.packageType
}

func (pkg *LocalPackage) SubPriority() int {
	return pkg.subPriority
}

func (pkg *LocalPackage) Repo() interfaces.RepoInterface {
	return pkg.repo
}

func (pkg *LocalPackage) Desc() *PackageDesc {
	return pkg.desc
}

func (pkg *LocalPackage) SetName(name string) {
	pkg.name = name
}

func (pkg *LocalPackage) SetBasePath(basePath string) {
	pkg.basePath = filepath.ToSlash(filepath.Clean(basePath))
}

func (pkg *LocalPackage) SetType(packageType interfaces.PackageType) {
	pkg.packageType = packageType
}

func (pkg *LocalPackage) SetSubPriority(prio int) {
	pkg.subPriority = prio
}

func (pkg *LocalPackage) SetDesc(desc *PackageDesc) {
	pkg.desc = desc
}

func (pkg *LocalPackage) SetRepo(r *repo.Repo) {
	pkg.repo = r
}

func (pkg *LocalPackage) CfgFilenames() []string {
	return pkg.cfgFilenames
}

func (pkg *LocalPackage) AddCfgFilename(cfgFilename string) {
	pkg.cfgFilenames = append(pkg.cfgFilenames, cfgFilename)
}

func (pkg *LocalPackage) readDesc(yc ycfg.YCfg) (*PackageDesc, error) {
	pdesc := &PackageDesc{}

	var err error

	pdesc.Author, err = yc.GetValString("pkg.author", nil)
	util.OneTimeWarningError(err)

	pdesc.Homepage, err = yc.GetValString("pkg.homepage", nil)
	util.OneTimeWarningError(err)

	pdesc.Description, err = yc.GetValString("pkg.description", nil)
	util.OneTimeWarningError(err)

	pdesc.Keywords, err = yc.GetValStringSlice("pkg.keywords", nil)
	util.OneTimeWarningError(err)

	return pdesc, nil
}

func (pkg *LocalPackage) sequenceString(key string) string {
	var buffer bytes.Buffer

	vals, err := pkg.PkgY.GetValStringSlice(key, nil)
	util.OneTimeWarningError(err)
	for _, f := range vals {
		buffer.WriteString("    - " + yaml.EscapeString(f) + "\n")
	}

	if buffer.Len() == 0 {
		return ""
	} else {
		return key + ":\n" + buffer.String()
	}
}

func (lpkg *LocalPackage) SaveSyscfg() error {
	dirpath := lpkg.BasePath()
	if err := os.MkdirAll(dirpath, 0755); err != nil {
		return util.NewNewtError(err.Error())
	}

	file, err := os.Create(lpkg.SyscfgYamlPath())
	if err != nil {
		return util.NewNewtError(err.Error())
	}
	defer file.Close()

	s := lpkg.SyscfgY.YAML()
	file.WriteString(s)

	return nil
}

// Saves the package's pkg.yml file.
// NOTE: This does not save every field in the package.  Only the fields
// necessary for creating a new target get saved.
func (pkg *LocalPackage) Save() error {
	dirpath := pkg.BasePath()
	if err := os.MkdirAll(dirpath, 0755); err != nil {
		return util.NewNewtError(err.Error())
	}

	file, err := os.Create(pkg.PkgYamlPath())
	if err != nil {
		return util.NewNewtError(err.Error())
	}
	defer file.Close()

	// XXX: Just iterate ycfg object's settings rather than calling out
	// cached settings individually.
	file.WriteString("pkg.name: " + yaml.EscapeString(pkg.Name()) + "\n")
	file.WriteString("pkg.type: " +
		yaml.EscapeString(PackageTypeNames[pkg.Type()]) + "\n")
	file.WriteString("pkg.description: " +
		yaml.EscapeString(pkg.Desc().Description) + "\n")
	file.WriteString("pkg.author: " +
		yaml.EscapeString(pkg.Desc().Author) + "\n")
	file.WriteString("pkg.homepage: " +
		yaml.EscapeString(pkg.Desc().Homepage) + "\n")

	file.WriteString("\n")

	file.WriteString(pkg.sequenceString("pkg.deps"))
	file.WriteString(pkg.sequenceString("pkg.aflags"))
	file.WriteString(pkg.sequenceString("pkg.cflags"))
	file.WriteString(pkg.sequenceString("pkg.cxxflags"))
	file.WriteString(pkg.sequenceString("pkg.lflags"))

	return nil
}

func matchNamePath(name, path string) bool {
	// assure that name and path use the same path separator...
	names := strings.Split(name, "/")
	name = strings.Join(names, "/")

	if strings.HasSuffix(path, name) {
		return true
	}
	return false
}

// Load reads everything that isn't identity specific into the package
func (pkg *LocalPackage) Load() error {
	var err error

	pkg.PkgY, err = config.ReadFile(pkg.PkgYamlPath())
	if err != nil {
		return err
	}
	pkg.AddCfgFilename(pkg.PkgYamlPath())

	// Set package name from the package
	pkg.name, err = pkg.PkgY.GetValString("pkg.name", nil)
	util.OneTimeWarningError(err)
	if pkg.name == "" {
		return util.FmtNewtError(
			"Package \"%s\" missing \"pkg.name\" field in its `pkg.yml` file",
			pkg.basePath)
	}

	if !matchNamePath(pkg.name, pkg.basePath) {
		return util.FmtNewtError(
			"Package \"%s\" has incorrect \"pkg.name\" field in its "+
				"`pkg.yml` file (pkg.name=%s)", pkg.basePath, pkg.name)
	}

	typeString, err := pkg.PkgY.GetValString("pkg.type", nil)
	util.OneTimeWarningError(err)
	pkg.packageType = PACKAGE_TYPE_LIB
	if len(typeString) > 0 {
		found := false
		for t, n := range PackageTypeNames {
			if typeString == n {
				pkg.packageType = t
				found = true
				break
			}
		}

		if !found {
			return util.FmtNewtError(
				"Package \"%s\" has incorrect \"pkg.type\" field in its "+
					"`pkg.yml` file (pkg.type=%s)", pkg.basePath, typeString)
		}
	}

	if pkg.packageType == PACKAGE_TYPE_TRANSIENT {
		n, err := pkg.PkgY.GetValString("pkg.link", nil)
		util.OneTimeWarningError(err)
		if len(n) == 0 {
			return util.FmtNewtError(
				"Transient package \"%s\" does not specify target "+
					"package in its `pkg.yml` file (pkg.name=%s)",
				pkg.basePath, pkg.name)
		}

		pkg.linkedName = n

		// We don't really want anything else for this package
		return nil
	}

	subPriority, err := pkg.PkgY.GetValInt("pkg.subpriority", nil)
	util.OneTimeWarningError(err)
	if subPriority >= PACKAGE_SUBPRIO_NUM {
		return util.FmtNewtError(
			"Package \"%s\" subpriority value \"%d\" is out of range (0 - \"%d\")",
			pkg.basePath, subPriority, PACKAGE_SUBPRIO_NUM-1)
	}
	if subPriority > 0 && pkg.packageType >= PACKAGE_TYPE_BSP {
		return util.FmtNewtError(
			"Package \"%s\" of type \"%s\" does not support subpriorities",
			pkg.basePath, typeString)
	}
	pkg.subPriority = subPriority

	// Read the package description from the file
	pkg.desc, err = pkg.readDesc(pkg.PkgY)
	if err != nil {
		return err
	}

	// Load syscfg settings.
	pkg.SyscfgY, err = config.ReadFile(pkg.SyscfgYamlPath())
	if err != nil && !util.IsNotExist(err) {
		return err
	}

	pkg.AddCfgFilename(pkg.SyscfgYamlPath())

	return nil
}

func (pkg *LocalPackage) InitFuncs(
	settings *cfgv.Settings) map[string]interface{} {

	vals, err := pkg.PkgY.GetValStringMap("pkg.init", settings)
	util.OneTimeWarningError(err)
	return vals
}

// DownFuncs retrieves the package's shutdown functions.  The returned map has:
// key=C-function-name, value=numeric-stage.
func (pkg *LocalPackage) DownFuncs(
	settings *cfgv.Settings) map[string]string {

	vals, err := pkg.PkgY.GetValStringMapString("pkg.down", settings)
	util.OneTimeWarningError(err)
	return vals
}

func (pkg *LocalPackage) PreBuildCmds(
	settings *cfgv.Settings) map[string]string {

	vals, err := pkg.PkgY.GetValStringMapString("pkg.pre_build_cmds", settings)
	util.OneTimeWarningError(err)
	return vals
}

func (pkg *LocalPackage) PreLinkCmds(
	settings *cfgv.Settings) map[string]string {

	vals, err := pkg.PkgY.GetValStringMapString("pkg.pre_link_cmds", settings)
	util.OneTimeWarningError(err)
	return vals
}

func (pkg *LocalPackage) PostLinkCmds(
	settings *cfgv.Settings) map[string]string {

	vals, err := pkg.PkgY.GetValStringMapString("pkg.post_link_cmds", settings)
	util.OneTimeWarningError(err)
	return vals
}

func (pkg *LocalPackage) InjectedSettings() *cfgv.Settings {
	return pkg.injectedSettings
}

func (pkg *LocalPackage) Clone(newRepo *repo.Repo,
	newName string) *LocalPackage {

	// XXX: Validate name.

	// Copy the package.
	newPkg := *pkg
	newPkg.repo = newRepo
	newPkg.name = newName
	newPkg.basePath = newRepo.Path() + "/" + newPkg.name

	// Insert the clone into the global package map.
	proj := interfaces.GetProject()
	pMap := proj.PackageList()

	(*pMap[newRepo.Name()])[newPkg.name] = &newPkg

	return &newPkg
}

func LoadLocalPackage(repo *repo.Repo, pkgDir string) (*LocalPackage, error) {
	pkg := NewLocalPackage(repo, pkgDir)
	err := pkg.Load()
	if err != nil {
		err = util.FmtNewtError("%s; ignoring package %s.",
			err.Error(), pkgDir)
		return nil, err
	}

	return pkg, err
}

func LocalPackageSpecialName(dirName string) bool {
	_, ok := LocalPackageSpecialNames[dirName]
	return ok
}

func ReadLocalPackageRecursive(repo *repo.Repo,
	pkgList map[string]interfaces.PackageInterface, basePath string,
	pkgName string, searchedMap map[string]struct{}) ([]string, error) {

	var warnings []string

	dirList, err := repo.FilteredSearchList(pkgName, searchedMap)
	if err != nil {
		return append(warnings, err.Error()), nil
	}

	for _, name := range dirList {
		if LocalPackageSpecialName(name) || strings.HasPrefix(name, ".") {
			continue
		}

		subWarnings, err := ReadLocalPackageRecursive(repo, pkgList,
			basePath, filepath.Join(pkgName, name), searchedMap)
		warnings = append(warnings, subWarnings...)
		if err != nil {
			return warnings, err
		}
	}

	if util.NodeNotExist(filepath.Join(basePath, pkgName, PACKAGE_FILE_NAME)) {
		return warnings, nil
	}

	pkg, err := LoadLocalPackage(repo, filepath.Join(basePath, pkgName))
	if err != nil {
		warnings = append(warnings, err.Error())
		return warnings, nil
	}

	if oldPkg, ok := pkgList[pkg.Name()]; ok {
		oldlPkg := oldPkg.(*LocalPackage)
		warnings = append(warnings,
			fmt.Sprintf("Multiple packages with same pkg.name=%s "+
				"in repo %s; path1=%s path2=%s", oldlPkg.Name(), repo.Name(),
				oldlPkg.BasePath(), pkg.BasePath()))

		return warnings, nil
	}

	pkgList[pkg.Name()] = pkg

	return warnings, nil
}

func ReadLocalPackages(repo *repo.Repo, basePath string) (
	*map[string]interfaces.PackageInterface, []string, error) {

	pkgMap := &map[string]interfaces.PackageInterface{}

	// Keep track of which directories we have traversed.  Prevent infinite
	// loops caused by symlink cycles by not inspecting the same directory
	// twice.
	searchedMap := map[string]struct{}{}

	warnings, err := ReadLocalPackageRecursive(repo, *pkgMap,
		basePath, "", searchedMap)

	return pkgMap, warnings, err
}
