package main

import (
	"fmt"
	"gon/lock"
	nw "gon/networking"
)

func main() {
	lf, err := lock.ReadLockFile("gon-lock.json")
	if err != nil{
		fmt.Println(err)
		return
	}

	data, err := lock.EncodeLockFile(lf)
	if err != nil{
		fmt.Println(err)
		return
	}
	fmt.Println(string(data))
	fmt.Println("\n\n")

	rootIDs := lock.GetRootIDs(lf)
	fmt.Println("Base pkIDs: ")
	for _, rootID := range rootIDs{
		fmt.Println("id: " + rootID)
	}

	pkgIDs, err := lf.WalkAll(rootIDs)
	if err != nil{
		fmt.Println(err)
		return 
	}
	fmt.Println("\nWalking pkgIDs: ") 
	for _, pkgID := range pkgIDs{
		fmt.Println("pkgID: " + pkgID)
	}

	networker := nw.GenerateNetworker(lf)
	err = networker.FetchAll(pkgIDs)
	if err != nil{
		fmt.Println(err)
		return
	}

	tgzSrc := make([]nw.Tgz, 0) 
	for _, pkg := range lf.Packages{
		tgzSrc = append(tgzSrc, nw.Tgz{Name:pkg.Name, Path: fmt.Sprintf("./.gon/cache/tarballs/%s.tgz", pkg.Integrity)})
	}
	err = nw.ExtractAll("./.gon/extract/", tgzSrc)
	if err != nil{
		fmt.Println(err)
		return
	}
}