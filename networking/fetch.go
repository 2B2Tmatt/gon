package nw

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"gon/lock"
	"io"
	"net/http"
	"os"
	"time"
)

type Networker struct{ 
	Client *http.Client
	Lockfile *lock.Lockfile	
}

func GenerateNetworker(lf *lock.Lockfile) *Networker{
	return &Networker{
		Client: &http.Client{Timeout: time.Duration(15) * time.Second}, 
		Lockfile: lf,
	}
}

func (nw *Networker) Fetch(pkgID string) error{
	pkg, exists := nw.Lockfile.Packages[pkgID]
	if !exists{
		return errors.New("pkg: " + pkgID + " is missing data ")
	}
	tzURL := pkg.TarballURL
	if tzURL == ""{
		return errors.New("pkg: " + pkgID + " is missing tarball url")
	}

	resp, err := nw.Client.Get(tzURL)
	if err != nil{
		return err 
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200{
		return errors.New("pkg: " + pkgID + " status on fetch: " + resp.Status)
	}
	tempFile, err := os.CreateTemp("./.gon/tmp", fmt.Sprintf("download-%s-*.tgz", pkgID))
	if err != nil{
		return err
	}
	defer tempFile.Close()

	hasher := sha256.New()

 	w := io.MultiWriter(tempFile, hasher)
	_, err = io.Copy(w, resp.Body)
	if err != nil{
		return err 
	}
	hashString := hex.EncodeToString(hasher.Sum(nil))
	if pkg.Integrity != "" && pkg.Integrity != hashString{
		return errors.New("pkg: " + pkgID + " failed integrity test")
	}
	pkg.Integrity = hashString

	return nil
}

func (nw *Networker) FetchAll(order []string) error{
	err := os.MkdirAll("./.gon/tmp", 0755) 
	if err != nil{
		return err
	}
	
	for _, pkgID := range order{
		err = nw.Fetch(pkgID)
		if err != nil{
			return err
		}
	}

	return nil
}