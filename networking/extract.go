package nw

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Tgz struct {
	Name string
	Path string
}

func ExtractAll(dest string, tgzFiles []Tgz) error {
	for _, tgzFile := range tgzFiles {
		err := Extract(tgzFile.Path, filepath.Join(dest, tgzFile.Name))
		if err != nil {
			os.RemoveAll(dest)
			return err
		}
	}
	return nil
}

func Extract(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)

	destAbs, err := filepath.Abs(dest)
	if err != nil {
		return err
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		destPath := filepath.Clean(filepath.Join(destAbs, header.Name))

		if !strings.HasPrefix(destPath, destAbs+string(os.PathSeparator)) && destPath != destAbs {
			return errors.New("filepath contains escape")
		}
		switch header.Typeflag {
		case tar.TypeDir:
			_, err := os.Stat(destPath)
			if os.IsNotExist(err) {
				err := os.MkdirAll(destPath, os.FileMode(header.Mode))
				if err != nil {
					return err
				}
			}
		case tar.TypeReg:
			err := os.MkdirAll(filepath.Dir(destPath), os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			newFile, err := os.Create(destPath)
			if err != nil {
				return err
			}
			_, err = io.Copy(newFile, tarReader)
			if err != nil {
				return err
			}
			err = newFile.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
