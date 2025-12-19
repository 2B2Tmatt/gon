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
}