package lock

import (
	"errors"
	"slices"
)

type Graph struct {
	Lockfile *Lockfile
	Visted  map[string]struct{}
	Results []string
}

func GetRootIDs(lf *Lockfile) []string {
	ids := make([]string, 0)
	for _, id := range lf.RootDependencies {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

func (lf * Lockfile) GetPackage(pkgID string) *Package{
	pkg, exists := lf.Packages[pkgID]
	if !exists{
		return nil
	}
	return &pkg
}

func (lf *Lockfile) GetDepIDs(pkgID string) []string{
	pkg, exists := lf.Packages[pkgID]
	if !exists{
		return nil
	}
	deps := make([]string, 0)
	for _, dep := range pkg.Deps{
		deps = append(deps, dep)
	}
	slices.Sort(deps)	
	return deps
}

func (lf *Lockfile) WalkAll(pkgIds []string) ([]string, error){
	visited := make(map[string]struct{})
	results := make([]string, 0)

	for _, id := range pkgIds{
		err := Walk(id, visited, &results, lf)
		if err != nil{
			return nil, err 
		}
	}
	return results, nil 
}

func Walk(pkgID string, visited map[string]struct{}, results *[]string, lf *Lockfile) (error){
	_, exists := visited[pkgID]
	if exists{
		return nil
	}
	visited[pkgID] = struct{}{}

	pkg, exists := lf.Packages[pkgID]
	if !exists{
		return errors.New("pkg: " + pkgID + " is missing data")
	}
	defer func(){
		if pkgID != ""{	
			*results = append(*results, pkgID)
		}
	}()
	if len(pkg.Deps) == 0{
		return nil
	}
	for _, id := range pkg.Deps{
		err := Walk(id, visited, results, lf)
		if err != nil{
			return err
		}
	}
	return nil
}