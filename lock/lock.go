package lock

import (
	"encoding/json"
	"errors"
	"os"
)

type Lockfile struct {
	LockFileVersion  int                `json:"lockfileVersion"`
	Registry         string             `json:"registry"`
	RootDependencies map[string]string  `json:"rootDeps"`
	Packages         map[string]Package `json:"packages"`
}

type Package struct {
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	TarballURL string            `json:"tarballURL"`
	Integrity  string            `json:"integrity"`
	Deps       map[string]string `json:"deps"`
}

func ReadLockFile(filepath string) (*Lockfile, error) {
	bytes, err := os.ReadFile(filepath)
	if err != nil{ 
		return nil, err
	}
	var lf Lockfile
	err = json.Unmarshal(bytes, &lf)
	if err != nil{ 
		return nil, err
	}
	if lf.RootDependencies == nil{
		lf.RootDependencies = make(map[string]string)
	}
	if lf.Packages == nil{
		lf.Packages = make(map[string]Package)
	}
	err = ValidateLockfile(&lf)
	if err != nil{ 
		return nil, err
	}
	return &lf, nil
}

func EncodeLockFile(lf *Lockfile) ([]byte, error) {
	err := ValidateLockfile(lf)
	if err != nil{
		return nil, err
	}
	bytes, err := json.MarshalIndent(lf, "", " ")
	if err != nil{ 
		return nil, err
	}
	

	return bytes, nil 
}

func ValidateLockfile(lf *Lockfile) error{
	if lf.LockFileVersion <= 0{ 
		return errors.New("invalid lockfile version")
	}
	if lf.RootDependencies == nil{
		return errors.New("rootDeps missing")
	}
	if lf.Packages == nil{
		return errors.New("packages missing")
	}
	for name, pkgID := range lf.RootDependencies{
		pkg, exists := lf.Packages[pkgID]
		if !exists{
			return errors.New("root dependency " + name + " points to missing package " + pkgID)
		}
		if pkg.Name != name{
			return errors.New("root dependency" + name + " points to package with name " + pkg.Name)
		}
	}
	for pkgID, pkg := range lf.Packages{
		expectedID := pkg.Name+"@"+pkg.Version	
		if pkgID != expectedID{
			return errors.New("package key " + pkgID + " does not match " + expectedID)
		}
		if pkg.Name == "" || pkg.Version == ""{
			return errors.New("package " + pkgID + " has empty name or version")
		}
		if pkg.TarballURL == ""{
			return errors.New("package " + pkgID + " has missing tarbalURL")
		}
		if pkg.Integrity == ""{
			return errors.New("package " + pkgID + " missing integrity")
		}
		for depName, depID := range pkg.Deps{
			depPkg, exists := lf.Packages[depID]
			if !exists{
				return errors.New("package " + pkgID + " depends on missing package " + depID)
			}
			if depPkg.Name != depName{
				return errors.New("package " + pkgID + " dependency name mismatch: expected " + depName + ", got " + depPkg.Name)
			}
		}
	}


	return nil
}