package nw

import (
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"gon/lock"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type Networker struct {
	Client   *http.Client
	Lockfile *lock.Lockfile
}

func GenerateNetworker(lf *lock.Lockfile, client *http.Client) *Networker {
	return &Networker{
		Client:   client,
		Lockfile: lf,
	}
}

func (nw *Networker) Fetch(pkgID string) (bool, string, string, error) {
	pkg, exists := nw.Lockfile.Packages[pkgID]
	if !exists {
		return false, "", "", errors.New("pkg: " + pkgID + " is missing data ")
	}

	if pkg.Integrity != "" {
		expectedCachePath := fmt.Sprintf("./.gon/cache/tarballs/%s.tgz", pkg.Integrity)
		_, err := os.Stat(expectedCachePath)
		if err == nil {
			log.Println("Cache path used")
			return true, expectedCachePath, pkg.Integrity, nil
		}
	}

	tzURL := pkg.TarballURL
	if tzURL == "" {
		return false, "", "", errors.New("pkg: " + pkgID + " is missing tarball url")
	}

	resp, err := nw.Client.Get(tzURL)
	if err != nil {
		return false, "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return false, "", "", errors.New("pkg: " + pkgID + " status on fetch: " + resp.Status)
	}
	tempFile, err := os.CreateTemp("./.gon/tmp", fmt.Sprintf("download-%s-*.tgz", pkgID))
	if err != nil {
		return false, "", "", err
	}

	hasher := sha512.New()

	defer func() {
		if err != nil {
			os.Remove(tempFile.Name())
		}
	}()
	w := io.MultiWriter(tempFile, hasher)
	_, err = io.Copy(w, resp.Body)
	tempFile.Close()
	if err != nil {
		return false, "", "", err
	}
	hashString := "sha512-" + base64.StdEncoding.EncodeToString(hasher.Sum(nil))

	if pkg.Integrity != "" && pkg.Integrity != hashString {
		return false, "", "", errors.New("pkg: " + pkgID + " failed integrity test")
	}
	pkg.Integrity = hashString
	log.Println("pkg: " + pkgID + " Integrity set: " + hashString)

	return false, tempFile.Name(), hashString, nil
}

func (nw *Networker) Promote(tempPath, hash string) error {
	hash = IntegrityToFilenameKey(hash)
	cachePath := fmt.Sprintf("./.gon/cache/tarballs/%s.tgz", hash)
	_, err := os.Stat(cachePath)
	if err == nil {
		err = os.Remove(tempPath)
		if err != nil {
			return err
		}
		return nil
	}
	err = os.Rename(tempPath, cachePath)
	if err != nil {
		return err
	}

	return nil
}

func (nw *Networker) FetchAll(order []string) error {
	err := os.MkdirAll("./.gon/tmp", 0755)
	if err != nil {
		return err
	}
	err = os.MkdirAll("./.gon/cache/tarballs", 0755)
	if err != nil {
		return err
	}

	for _, pkgID := range order {
		cached, path, hash, err := nw.Fetch(pkgID)
		if err != nil {
			return err
		}
		if !cached {
			err = nw.Promote(path, hash)
			if err != nil {
				return err
			}
		}
	}
	nw.Lockfile.SaveAtomic("gon-lock.json")

	return nil
}

func IntegrityToFilenameKey(integrity string) string {
	const prefix = "sha512-"
	if !strings.HasPrefix(integrity, prefix) {
		return strings.ReplaceAll(integrity, "/", "_") // fallback
	}
	b64 := strings.TrimPrefix(integrity, prefix)

	b64 = strings.ReplaceAll(b64, "/", "_")
	b64 = strings.ReplaceAll(b64, "+", "-")
	b64 = strings.TrimRight(b64, "=")

	return "sha512-" + b64
}
